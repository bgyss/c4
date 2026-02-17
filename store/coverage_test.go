package store

import (
	"bytes"
	"crypto/sha512"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bgyss/c4"
)

type stubReadCloser struct {
	data     []byte
	readErr  error
	closeErr error
	readDone bool
}

func (s *stubReadCloser) Read(p []byte) (int, error) {
	if s.readDone {
		return 0, io.EOF
	}
	s.readDone = true
	n := copy(p, s.data)
	return n, s.readErr
}

func (s *stubReadCloser) Close() error { return s.closeErr }

type stubWriteCloser struct {
	writeErr   error
	closeErr   error
	closeCalls int
}

func (s *stubWriteCloser) Write(p []byte) (int, error) { return len(p), s.writeErr }
func (s *stubWriteCloser) Close() error {
	s.closeCalls++
	return s.closeErr
}

func TestValidatingReaderAndWriterEOFPaths(t *testing.T) {
	id := c4.Identify(strings.NewReader("expected"))

	vr := &validatingReader{
		h:  sha512.New(),
		id: id,
		r: &stubReadCloser{
			data:    []byte("different"),
			readErr: io.EOF,
		},
	}
	buf := make([]byte, 64)
	if _, err := vr.Read(buf); err != ErrInvalidID {
		t.Fatalf("expected ErrInvalidID from validatingReader on EOF mismatch, got %v", err)
	}

	removed := false
	sw := &stubWriteCloser{writeErr: io.EOF}
	vw := &validatingWriter{
		h:  sha512.New(),
		id: id,
		w:  sw,
		remove: func(_ c4.ID) error {
			removed = true
			return nil
		},
	}
	if _, err := vw.Write([]byte("different")); err != ErrInvalidID {
		t.Fatalf("expected ErrInvalidID from validatingWriter on EOF mismatch, got %v", err)
	}
	if sw.closeCalls == 0 || !removed {
		t.Fatalf("expected writer close and remove on invalid EOF write")
	}
}

func TestLoggerCloseAndErrorFiltering(t *testing.T) {
	id := c4.Identify(strings.NewReader("logger-id")).String()
	buf := new(bytes.Buffer)

	// ErrInvalidID with LogInvalidID disabled should return the error without logging it.
	lr := &loggingReader{
		r:      &stubReadCloser{closeErr: ErrInvalidID},
		logout: buf,
		idstr:  id,
		flags:  LogClose,
	}
	if err := lr.Close(); err != ErrInvalidID {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}
	if got := buf.String(); got != id+" Close\n" {
		t.Fatalf("unexpected log output: %q", got)
	}

	// Generic close error with LogError disabled should not emit an error line.
	buf.Reset()
	lw := &loggingWriter{
		r:      &stubWriteCloser{closeErr: errors.New("close-fail")},
		logout: buf,
		idstr:  id,
		flags:  LogClose,
	}
	if err := lw.Close(); err == nil {
		t.Fatalf("expected close error")
	}
	if got := buf.String(); got != id+" Close\n" {
		t.Fatalf("unexpected writer log output: %q", got)
	}

	// Read/Write ErrInvalidID filtering with LogInvalidID disabled.
	buf.Reset()
	lr2 := &loggingReader{
		r:      &stubReadCloser{data: []byte("x"), readErr: ErrInvalidID},
		logout: buf,
		idstr:  id,
		flags:  LogRead,
	}
	if _, err := lr2.Read(make([]byte, 8)); err != ErrInvalidID {
		t.Fatalf("expected ErrInvalidID from loggingReader.Read, got %v", err)
	}
	if got := buf.String(); got != id+" Read 1\n" {
		t.Fatalf("unexpected read log output: %q", got)
	}

	buf.Reset()
	lw2 := &loggingWriter{
		r:      &stubWriteCloser{writeErr: ErrInvalidID},
		logout: buf,
		idstr:  id,
		flags:  LogWrite,
	}
	if _, err := lw2.Write([]byte("x")); err != ErrInvalidID {
		t.Fatalf("expected ErrInvalidID from loggingWriter.Write, got %v", err)
	}
	if got := buf.String(); got != id+" Write 1\n" {
		t.Fatalf("unexpected write log output: %q", got)
	}

	// Generic errors with LogError enabled should include an error log line.
	buf.Reset()
	lr3 := &loggingReader{
		r:      &stubReadCloser{data: []byte("x"), readErr: errors.New("read-fail")},
		logout: buf,
		idstr:  id,
		flags:  LogRead | LogError,
	}
	if _, err := lr3.Read(make([]byte, 8)); err == nil {
		t.Fatalf("expected generic read error")
	}
	if !strings.Contains(buf.String(), "Read error read-fail") {
		t.Fatalf("expected read error line in logs, got %q", buf.String())
	}

	buf.Reset()
	lw3 := &loggingWriter{
		r:      &stubWriteCloser{writeErr: errors.New("write-fail")},
		logout: buf,
		idstr:  id,
		flags:  LogWrite | LogError,
	}
	if _, err := lw3.Write([]byte("x")); err == nil {
		t.Fatalf("expected generic write error")
	}
	if !strings.Contains(buf.String(), "Write error write-fail") {
		t.Fatalf("expected write error line in logs, got %q", buf.String())
	}

	// ErrInvalidID with LogInvalidID enabled should log the error.
	buf.Reset()
	lr4 := &loggingReader{
		r:      &stubReadCloser{closeErr: ErrInvalidID},
		logout: buf,
		idstr:  id,
		flags:  LogClose | LogInvalidID,
	}
	if err := lr4.Close(); err != ErrInvalidID {
		t.Fatalf("expected ErrInvalidID from close, got %v", err)
	}
	if !strings.Contains(buf.String(), "Close error c4 id does not match data") {
		t.Fatalf("expected invalid-id close error log, got %q", buf.String())
	}

	buf.Reset()
	lw4 := &loggingWriter{
		r:      &stubWriteCloser{closeErr: errors.New("close-fail")},
		logout: buf,
		idstr:  id,
		flags:  LogClose | LogError,
	}
	if err := lw4.Close(); err == nil {
		t.Fatalf("expected writer close error")
	}
	if !strings.Contains(buf.String(), "Close error close-fail") {
		t.Fatalf("expected close error line in logs, got %q", buf.String())
	}

	// EOF with LogEof disabled should not include an error line.
	buf.Reset()
	lrEOF := &loggingReader{
		r:      &stubReadCloser{data: []byte("x"), readErr: io.EOF},
		logout: buf,
		idstr:  id,
		flags:  LogRead,
	}
	if _, err := lrEOF.Read(make([]byte, 8)); err != io.EOF {
		t.Fatalf("expected EOF from loggingReader, got %v", err)
	}
	if strings.Contains(buf.String(), "error EOF") {
		t.Fatalf("did not expect EOF error log with LogEof disabled")
	}

	// EOF with LogEof enabled should include an error line.
	buf.Reset()
	lwEOF := &loggingWriter{
		r:      &stubWriteCloser{writeErr: io.EOF},
		logout: buf,
		idstr:  id,
		flags:  LogWrite | LogEof,
	}
	if _, err := lwEOF.Write([]byte("x")); err != io.EOF {
		t.Fatalf("expected EOF from loggingWriter, got %v", err)
	}
	if !strings.Contains(buf.String(), "Write error EOF") {
		t.Fatalf("expected EOF write error log, got %q", buf.String())
	}

	// Writer ErrInvalidID with LogInvalidID enabled should include an error line.
	buf.Reset()
	lwInvalid := &loggingWriter{
		r:      &stubWriteCloser{closeErr: ErrInvalidID},
		logout: buf,
		idstr:  id,
		flags:  LogClose | LogInvalidID,
	}
	if err := lwInvalid.Close(); err != ErrInvalidID {
		t.Fatalf("expected ErrInvalidID close error, got %v", err)
	}
	if !strings.Contains(buf.String(), "Close error c4 id does not match data") {
		t.Fatalf("expected invalid-id close log for writer, got %q", buf.String())
	}
}

func TestRAMAndFolderErrorBranches(t *testing.T) {
	ram := NewRAM()
	id := c4.Identify(strings.NewReader("ram-id"))
	rc, err := ram.Create(id)
	if err != nil {
		t.Fatalf("create ram file error: %v", err)
	}
	if _, err := rc.Write([]byte("abc")); err != nil {
		t.Fatalf("write ram file error: %v", err)
	}
	if err := rc.Close(); err != nil {
		t.Fatalf("close ram file error: %v", err)
	}

	r, err := ram.Open(id)
	if err != nil {
		t.Fatalf("open ram file error: %v", err)
	}
	data := make([]byte, 8)
	if _, err := r.Read(data); err != nil {
		t.Fatalf("first ram read error: %v", err)
	}
	if _, err := r.Read(data); err != io.EOF {
		t.Fatalf("expected EOF on second ram read, got %v", err)
	}
	_ = r.Close()

	if _, err := ram.Create(id); err == nil {
		t.Fatalf("expected create-existing error")
	}

	if err := ram.Remove(id); err != nil {
		t.Fatalf("remove ram file error: %v", err)
	}
	if err := ram.Remove(id); err == nil {
		t.Fatalf("expected remove missing error")
	}

	readonly := &ramfile{readonly: true}
	if _, err := readonly.Write([]byte("x")); err != os.ErrPermission {
		t.Fatalf("expected os.ErrPermission on readonly write, got %v", err)
	}

	var badID c4.ID
	f := Folder(string([]byte{0}))
	if _, err := f.Open(badID); err == nil {
		t.Fatalf("expected Folder.Open path validation error")
	}
	if _, err := f.Create(badID); err == nil {
		t.Fatalf("expected Folder.Create path validation error")
	}
	if err := f.Remove(badID); err == nil {
		t.Fatalf("expected Folder.Remove path validation error")
	}

	rootFolder := Folder(string(filepath.Separator))
	if _, err := rootFolder.Open(id); err == nil {
		t.Fatalf("expected root-folder traversal validation error")
	}
}

func TestValidatingCreateAndErrorForwarding(t *testing.T) {
	vs := NewValidating(&failingStore{})
	id := c4.Identify(strings.NewReader("id"))
	if _, err := vs.Create(id); err == nil {
		t.Fatalf("expected create error passthrough")
	}

	vr := &validatingReader{
		h:  sha512.New(),
		id: id,
		r: &stubReadCloser{
			data:    []byte("abc"),
			readErr: errors.New("read-fail"),
		},
	}
	if _, err := vr.Read(make([]byte, 8)); err == nil || err.Error() != "read-fail" {
		t.Fatalf("expected read-fail passthrough, got %v", err)
	}

	vw := &validatingWriter{
		h:      sha512.New(),
		id:     id,
		w:      &stubWriteCloser{writeErr: errors.New("write-fail")},
		remove: func(_ c4.ID) error { return nil },
	}
	if _, err := vw.Write([]byte("abc")); err == nil || err.Error() != "write-fail" {
		t.Fatalf("expected write-fail passthrough, got %v", err)
	}
}

func TestFolderValidatePathCwdError(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd error: %v", err)
	}
	base := t.TempDir()
	if err := os.Chdir(base); err != nil {
		t.Fatalf("Chdir error: %v", err)
	}
	if err := os.RemoveAll(base); err != nil {
		t.Fatalf("RemoveAll error: %v", err)
	}
	defer func() { _ = os.Chdir(wd) }()

	id := c4.Identify(strings.NewReader("cwd-err-id"))
	f := Folder("relative-folder")
	if _, err := f.Open(id); err == nil {
		t.Fatalf("expected Open validation error when cwd is gone")
	}
	if _, err := f.Create(id); err == nil {
		t.Fatalf("expected Create validation error when cwd is gone")
	}
	if err := f.Remove(id); err == nil {
		t.Fatalf("expected Remove validation error when cwd is gone")
	}
}

type failingStore struct{}

func (f *failingStore) Open(c4.ID) (io.ReadCloser, error)    { return nil, errors.New("open-fail") }
func (f *failingStore) Create(c4.ID) (io.WriteCloser, error) { return nil, errors.New("create-fail") }
func (f *failingStore) Remove(c4.ID) error                   { return errors.New("remove-fail") }
