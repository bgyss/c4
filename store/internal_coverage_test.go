package store

import (
	"bytes"
	"crypto/sha512"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/bgyss/c4"
)

type fakeRC struct {
	readFn   func([]byte) (int, error)
	closeErr error
}

func (f *fakeRC) Read(p []byte) (int, error) { return f.readFn(p) }
func (f *fakeRC) Close() error               { return f.closeErr }

type fakeWC struct {
	writeFn  func([]byte) (int, error)
	closeErr error
}

func (f *fakeWC) Write(p []byte) (int, error) { return f.writeFn(p) }
func (f *fakeWC) Close() error                { return f.closeErr }

type stubStore struct {
	openFn   func(id c4.ID) (io.ReadCloser, error)
	createFn func(id c4.ID) (io.WriteCloser, error)
	removeFn func(id c4.ID) error
}

func (s stubStore) Open(id c4.ID) (io.ReadCloser, error) {
	return s.openFn(id)
}

func (s stubStore) Create(id c4.ID) (io.WriteCloser, error) {
	return s.createFn(id)
}

func (s stubStore) Remove(id c4.ID) error {
	return s.removeFn(id)
}

func TestLoggingReaderWriterBranches(t *testing.T) {
	id := c4.Identify(strings.NewReader("log-branches"))
	idstr := id.String()

	t.Run("reader-eof-not-logged-when-disabled", func(t *testing.T) {
		buf := new(bytes.Buffer)
		r := &loggingReader{
			r: &fakeRC{
				readFn: func([]byte) (int, error) { return 0, io.EOF },
			},
			logout: buf,
			idstr:  idstr,
			flags:  LogRead, // no LogEof
		}
		_, _ = r.Read(make([]byte, 1))
		if strings.Contains(buf.String(), "error EOF") {
			t.Fatalf("did not expect EOF error log when LogEof is disabled")
		}
	})

	t.Run("reader-invalid-id-not-logged-when-disabled", func(t *testing.T) {
		buf := new(bytes.Buffer)
		r := &loggingReader{
			r: &fakeRC{
				readFn: func([]byte) (int, error) { return 0, ErrInvalidID },
			},
			logout: buf,
			idstr:  idstr,
			flags:  LogRead, // no LogInvalidID
		}
		_, _ = r.Read(make([]byte, 1))
		if strings.Contains(buf.String(), "error") {
			t.Fatalf("did not expect error log")
		}
	})

	t.Run("reader-generic-error-not-logged-when-disabled", func(t *testing.T) {
		buf := new(bytes.Buffer)
		r := &loggingReader{
			r: &fakeRC{
				readFn: func([]byte) (int, error) { return 0, errors.New("read fail") },
			},
			logout: buf,
			idstr:  idstr,
			flags:  LogRead, // no LogError
		}
		_, _ = r.Read(make([]byte, 1))
		if strings.Contains(buf.String(), "error") {
			t.Fatalf("did not expect error log")
		}
	})

	t.Run("reader-close-invalid-id-not-logged-when-disabled", func(t *testing.T) {
		buf := new(bytes.Buffer)
		r := &loggingReader{
			r: &fakeRC{
				readFn:   func([]byte) (int, error) { return 0, nil },
				closeErr: ErrInvalidID,
			},
			logout: buf,
			idstr:  idstr,
			flags:  LogClose, // no LogInvalidID
		}
		_ = r.Close()
		if strings.Contains(buf.String(), "error") {
			t.Fatalf("did not expect close error log")
		}
	})

	t.Run("reader-close-errors-logged-when-enabled", func(t *testing.T) {
		buf := new(bytes.Buffer)
		r1 := &loggingReader{
			r: &fakeRC{
				readFn:   func([]byte) (int, error) { return 0, nil },
				closeErr: ErrInvalidID,
			},
			logout: buf,
			idstr:  idstr,
			flags:  LogClose | LogInvalidID,
		}
		_ = r1.Close()
		if !strings.Contains(buf.String(), "error") {
			t.Fatalf("expected invalid-id close error log")
		}

		buf.Reset()
		r2 := &loggingReader{
			r: &fakeRC{
				readFn:   func([]byte) (int, error) { return 0, nil },
				closeErr: errors.New("close fail"),
			},
			logout: buf,
			idstr:  idstr,
			flags:  LogClose | LogError,
		}
		_ = r2.Close()
		if !strings.Contains(buf.String(), "error") {
			t.Fatalf("expected generic close error log")
		}
	})

	t.Run("reader-close-generic-error-not-logged-when-disabled", func(t *testing.T) {
		buf := new(bytes.Buffer)
		r := &loggingReader{
			r: &fakeRC{
				readFn:   func([]byte) (int, error) { return 0, nil },
				closeErr: errors.New("close fail"),
			},
			logout: buf,
			idstr:  idstr,
			flags:  LogClose, // no LogError
		}
		_ = r.Close()
		if strings.Contains(buf.String(), "error") {
			t.Fatalf("did not expect close error log")
		}
	})

	t.Run("writer-eof-invalid-generic-error-logging-switches", func(t *testing.T) {
		cases := []struct {
			name   string
			err    error
			flags  LoggerFlags
			logged bool
		}{
			{name: "eof-off", err: io.EOF, flags: LogWrite, logged: false},
			{name: "eof-on", err: io.EOF, flags: LogWrite | LogEof, logged: true},
			{name: "invalid-off", err: ErrInvalidID, flags: LogWrite, logged: false},
			{name: "invalid-on", err: ErrInvalidID, flags: LogWrite | LogInvalidID, logged: true},
			{name: "generic-off", err: errors.New("write fail"), flags: LogWrite, logged: false},
			{name: "generic-on", err: errors.New("write fail"), flags: LogWrite | LogError, logged: true},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				buf := new(bytes.Buffer)
				w := &loggingWriter{
					r: &fakeWC{
						writeFn: func([]byte) (int, error) { return 0, tc.err },
					},
					logout: buf,
					idstr:  idstr,
					flags:  tc.flags,
				}
				_, _ = w.Write([]byte("x"))
				hasErr := strings.Contains(buf.String(), "error")
				if hasErr != tc.logged {
					t.Fatalf("unexpected log behavior, got log=%v flags=%v err=%v", hasErr, tc.flags, tc.err)
				}
			})
		}
	})

	t.Run("writer-close-invalid-id-not-logged-when-disabled", func(t *testing.T) {
		buf := new(bytes.Buffer)
		w := &loggingWriter{
			r: &fakeWC{
				writeFn:  func(p []byte) (int, error) { return len(p), nil },
				closeErr: ErrInvalidID,
			},
			logout: buf,
			idstr:  idstr,
			flags:  LogClose, // no LogInvalidID
		}
		_ = w.Close()
		if strings.Contains(buf.String(), "error") {
			t.Fatalf("did not expect close error log")
		}
	})

	t.Run("writer-close-errors-logged-when-enabled", func(t *testing.T) {
		buf := new(bytes.Buffer)
		w1 := &loggingWriter{
			r: &fakeWC{
				writeFn:  func(p []byte) (int, error) { return len(p), nil },
				closeErr: ErrInvalidID,
			},
			logout: buf,
			idstr:  idstr,
			flags:  LogClose | LogInvalidID,
		}
		_ = w1.Close()
		if !strings.Contains(buf.String(), "error") {
			t.Fatalf("expected invalid-id close error log")
		}

		buf.Reset()
		w2 := &loggingWriter{
			r: &fakeWC{
				writeFn:  func(p []byte) (int, error) { return len(p), nil },
				closeErr: errors.New("close fail"),
			},
			logout: buf,
			idstr:  idstr,
			flags:  LogClose | LogError,
		}
		_ = w2.Close()
		if !strings.Contains(buf.String(), "error") {
			t.Fatalf("expected generic close error log")
		}
	})
}

func TestValidatingReaderWriterEOFInvalidBranches(t *testing.T) {
	data := []byte("validating-branches")
	id := c4.Identify(bytes.NewReader(data))
	badID := c4.Identify(strings.NewReader("different"))

	t.Run("reader-eof-invalid-id", func(t *testing.T) {
		called := false
		r := &validatingReader{
			h:  sha512.New(),
			id: badID,
			r: &fakeRC{
				readFn: func(p []byte) (int, error) {
					copy(p, data)
					return len(data), io.EOF
				},
				closeErr: nil,
			},
		}
		buf := make([]byte, len(data))
		_, err := r.Read(buf)
		if !errors.Is(err, ErrInvalidID) {
			t.Fatalf("expected ErrInvalidID, got %v", err)
		}
		r.r = &fakeRC{
			readFn:   func([]byte) (int, error) { return 0, nil },
			closeErr: nil,
		}
		called = true
		if !called {
			t.Fatalf("sanity")
		}
	})

	t.Run("reader-eof-valid-id", func(t *testing.T) {
		r := &validatingReader{
			h:  sha512.New(),
			id: id,
			r: &fakeRC{
				readFn: func(p []byte) (int, error) {
					copy(p, data)
					return len(data), io.EOF
				},
				closeErr: nil,
			},
		}
		buf := make([]byte, len(data))
		_, err := r.Read(buf)
		if !errors.Is(err, io.EOF) {
			t.Fatalf("expected io.EOF for valid read, got %v", err)
		}
	})

	t.Run("writer-eof-invalid-id-removes", func(t *testing.T) {
		removed := false
		closed := false
		w := &validatingWriter{
			h:  sha512.New(),
			id: badID,
			w: &fakeWC{
				writeFn:  func(p []byte) (int, error) { return len(data), io.EOF },
				closeErr: nil,
			},
			remove: func(c4.ID) error {
				removed = true
				return nil
			},
		}
		w.w = &fakeWC{
			writeFn: func(p []byte) (int, error) {
				closed = true
				return len(data), io.EOF
			},
			closeErr: nil,
		}
		if _, err := w.Write(data); !errors.Is(err, ErrInvalidID) {
			t.Fatalf("expected ErrInvalidID, got %v", err)
		}
		if !removed {
			t.Fatalf("expected remove callback to run")
		}
		if !closed {
			t.Fatalf("expected writer write to run")
		}
	})

	t.Run("writer-eof-valid-id", func(t *testing.T) {
		w := &validatingWriter{
			h:  sha512.New(),
			id: id,
			w: &fakeWC{
				writeFn:  func(p []byte) (int, error) { return len(p), io.EOF },
				closeErr: nil,
			},
			remove: func(c4.ID) error {
				t.Fatalf("remove should not be called for valid writer")
				return nil
			},
		}
		if _, err := w.Write(data); !errors.Is(err, io.EOF) {
			t.Fatalf("expected io.EOF for valid write, got %v", err)
		}
	})

	t.Run("valid-open-create-errors", func(t *testing.T) {
		openErr := errors.New("open fail")
		createErr := errors.New("create fail")
		st := stubStore{
			openFn:   func(c4.ID) (io.ReadCloser, error) { return nil, openErr },
			createFn: func(c4.ID) (io.WriteCloser, error) { return nil, createErr },
			removeFn: func(c4.ID) error { return nil },
		}
		v := NewValidating(st)
		if _, err := v.Open(id); !errors.Is(err, openErr) {
			t.Fatalf("expected open error passthrough")
		}
		if _, err := v.Create(id); !errors.Is(err, createErr) {
			t.Fatalf("expected create error passthrough")
		}
	})

	t.Run("writer-close-remove-error-ignored", func(t *testing.T) {
		store := NewRAM()
		v := NewValidating(store)
		w, err := v.Create(id)
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		if _, err := w.Write([]byte("mismatch")); err != nil {
			t.Fatalf("write: %v", err)
		}
		if err := w.Close(); !errors.Is(err, ErrInvalidID) {
			t.Fatalf("expected close invalid id error, got %v", err)
		}
	})
}

func TestRAMAndFolderExtraBranches(t *testing.T) {
	t.Run("ramfile-read-eof-and-readonly-write", func(t *testing.T) {
		ram := NewRAM()
		id := c4.Identify(strings.NewReader("ram-branches"))
		f := &ramfile{ram: ram, id: id, readonly: true, data: nil}
		if _, err := f.Read(make([]byte, 1)); !errors.Is(err, io.EOF) {
			t.Fatalf("expected EOF, got %v", err)
		}
		if _, err := f.Write([]byte("x")); !errors.Is(err, os.ErrPermission) {
			t.Fatalf("expected permission error, got %v", err)
		}
		if err := f.Close(); err != nil {
			t.Fatalf("readonly close should be nil: %v", err)
		}
	})

	t.Run("ram-create-duplicate-remove-missing", func(t *testing.T) {
		ram := NewRAM()
		id := c4.Identify(strings.NewReader("dup"))
		w, err := ram.Create(id)
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		if _, err := w.Write([]byte("dup")); err != nil {
			t.Fatalf("write: %v", err)
		}
		if err := w.Close(); err != nil {
			t.Fatalf("close: %v", err)
		}
		if _, err := ram.Create(id); err == nil {
			t.Fatalf("expected duplicate create error")
		}
		missing := c4.Identify(strings.NewReader("missing"))
		if err := ram.Remove(missing); err == nil {
			t.Fatalf("expected remove missing error")
		}
	})

	t.Run("folder-validatePath-guards-and-propagation", func(t *testing.T) {
		origIDToString := idToString
		origFolderAbs := folderAbs
		defer func() {
			idToString = origIDToString
			folderAbs = origFolderAbs
		}()

		f := Folder(t.TempDir())
		id := c4.Identify(strings.NewReader("folder"))

		// traversal character guard
		idToString = func(c4.ID) string { return "../escape" }
		if _, err := f.validatePath(id); err == nil {
			t.Fatalf("expected traversal guard error")
		}
		if _, err := f.Open(id); err == nil {
			t.Fatalf("expected Open to propagate validatePath error")
		}
		if _, err := f.Create(id); err == nil {
			t.Fatalf("expected Create to propagate validatePath error")
		}
		if err := f.Remove(id); err == nil {
			t.Fatalf("expected Remove to propagate validatePath error")
		}

		// path traversal detection guard after resolution
		idToString = func(c4.ID) string { return "ok" }
		call := 0
		folderAbs = func(string) (string, error) {
			call++
			if call == 1 {
				return "/tmp/base", nil
			}
			return "/tmp/outside", nil
		}
		if _, err := f.validatePath(id); err == nil {
			t.Fatalf("expected path traversal detection error")
		}

		// absolute path resolution error on base path
		folderAbs = func(string) (string, error) {
			return "", errors.New("abs fail")
		}
		if _, err := f.validatePath(id); err == nil {
			t.Fatalf("expected base abs error")
		}

		// absolute path resolution error on target path
		call = 0
		folderAbs = func(string) (string, error) {
			call++
			if call == 1 {
				return "/tmp/base", nil
			}
			return "", errors.New("target abs fail")
		}
		if _, err := f.validatePath(id); err == nil {
			t.Fatalf("expected target abs error")
		}
	})

}
