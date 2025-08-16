package manifest_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Avalanche-io/c4"
	"github.com/Avalanche-io/c4/manifest"
)

// BenchmarkManifestOperations benchmarks core manifest operations
func BenchmarkManifestOperations(b *testing.B) {
	// Create a manifest with test data
	m := manifest.NewManifest()
	
	// Add files to manifest
	for i := 0; i < 1000; i++ {
		fi := &mockFileInfo{
			name:    fmt.Sprintf("file%d.txt", i),
			size:    int64(100 + i),
			mode:    0644,
			modTime: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			isDir:   false,
		}
		id := c4.Identify(strings.NewReader(fmt.Sprintf("content%d", i)))
		mfi := manifest.NewFileInfo(fi, id)
		m.SetFileInfo(fmt.Sprintf("/path/to/file%d.txt", i), mfi)
	}
	
	b.Run("Get", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			path := fmt.Sprintf("/path/to/file%d.txt", i%1000)
			_ = m.Get(path)
		}
	})
	
	b.Run("Paths", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = m.Paths()
		}
	})
	
	b.Run("Len", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = m.Len()
		}
	})
	
	b.Run("Marshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := m.Marshal()
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkManifestCreation benchmarks creating manifests of different sizes
func BenchmarkManifestCreation(b *testing.B) {
	sizes := []int{10, 100, 1000, 10000}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				m := manifest.NewManifest()
				
				for j := 0; j < size; j++ {
					fi := &mockFileInfo{
						name:    fmt.Sprintf("file%d.txt", j),
						size:    int64(100 + j),
						mode:    0644,
						modTime: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
						isDir:   false,
					}
					id := c4.Identify(strings.NewReader(fmt.Sprintf("content%d", j)))
					mfi := manifest.NewFileInfo(fi, id)
					m.SetFileInfo(fmt.Sprintf("/file%d.txt", j), mfi)
				}
			}
		})
	}
}

// BenchmarkManifestSerialization benchmarks manifest serialization
func BenchmarkManifestSerialization(b *testing.B) {
	// Create test manifests of different sizes
	manifests := make([]*manifest.M, 0)
	sizes := []int{100, 1000, 5000}
	
	for _, size := range sizes {
		m := manifest.NewManifest()
		for i := 0; i < size; i++ {
			fi := &mockFileInfo{
				name:    fmt.Sprintf("file%d.txt", i),
				size:    int64(100 + i),
				mode:    0644,
				modTime: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				isDir:   false,
			}
			id := c4.Identify(strings.NewReader(fmt.Sprintf("content%d", i)))
			mfi := manifest.NewFileInfo(fi, id)
			m.SetFileInfo(fmt.Sprintf("/file%d.txt", i), mfi)
		}
		manifests = append(manifests, m)
	}
	
	// Benchmark Marshal
	for i, m := range manifests {
		size := sizes[i]
		b.Run(fmt.Sprintf("Marshal_size_%d", size), func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				data, err := m.Marshal()
				if err != nil {
					b.Fatal(err)
				}
				b.SetBytes(int64(len(data)))
			}
		})
	}
	
	// Benchmark Unmarshal
	for i, m := range manifests {
		size := sizes[i]
		data, err := m.Marshal()
		if err != nil {
			b.Fatal(err)
		}
		
		b.Run(fmt.Sprintf("Unmarshal_size_%d", size), func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				m2 := manifest.NewManifest()
				err := m2.Unmarshal(bytes.NewReader(data))
				if err != nil {
					b.Fatal(err)
				}
				b.SetBytes(int64(len(data)))
			}
		})
	}
}

// BenchmarkFileInfoOperations benchmarks FileInfo operations
func BenchmarkFileInfoOperations(b *testing.B) {
	fi := &mockFileInfo{
		name:    "testfile.txt",
		size:    1024,
		mode:    0644,
		modTime: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		isDir:   false,
	}
	id := c4.Identify(strings.NewReader("test content"))
	metadata := c4.Identify(strings.NewReader("metadata"))
	mfi := manifest.NewFileInfo(fi, id, metadata)
	
	b.Run("Name", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = mfi.Name()
		}
	})
	
	b.Run("Size", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = mfi.Size()
		}
	})
	
	b.Run("Mode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = mfi.Mode()
		}
	})
	
	b.Run("ModTime", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = mfi.ModTime()
		}
	})
	
	b.Run("ID", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = mfi.ID()
		}
	})
	
	b.Run("Metadata", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = mfi.Metadata()
		}
	})
	
	b.Run("MarshalJson", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			data, err := mfi.MarshalJson()
			if err != nil {
				b.Fatal(err)
			}
			b.SetBytes(int64(len(data)))
		}
	})
}

// BenchmarkParseFileInfo benchmarks parsing file info from strings
func BenchmarkParseFileInfo(b *testing.B) {
	testLines := []string{
		"\t-rw-r--r--    6148 2019-11-06T20:01:22Z .DS_Store                                         c458Yt9m2xPHH8jxfyipfqD9qsXpZh2fGD9HpbfwSFfAFgX9nWHQp1LG94SsEron2GteyvxfYmQcsUjvJCbxPuRTj6\n",
		"\tdrwxr-xr-x       0 2023-01-01T12:00:00Z somedir/                                          \n",
		"\t-rwxrwxrwx    1024 2023-05-15T10:30:45Z executable.sh                                     c459DdnZNhjY9JzJbJ6mF5pJhVBXpq7m8aBTgCrq36jMKxE8hHtNJLqJn2YCTFbCUbZzchNSwqJTbm1U3ZAiuVJ2\n",
	}
	
	for i, line := range testLines {
		b.Run(fmt.Sprintf("line_%d", i), func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				_, err := manifest.ParseFileInfo(line)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkManifestSetOperations benchmarks Set operations on manifests
func BenchmarkManifestSetOperations(b *testing.B) {
	m := manifest.NewManifest()
	
	b.Run("SetFileInfo", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			fi := &mockFileInfo{
				name:    fmt.Sprintf("file%d.txt", i),
				size:    int64(100 + i%1000),
				mode:    0644,
				modTime: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				isDir:   false,
			}
			id := c4.Identify(strings.NewReader(fmt.Sprintf("content%d", i)))
			mfi := manifest.NewFileInfo(fi, id)
			m.SetFileInfo(fmt.Sprintf("/file%d.txt", i), mfi)
		}
	})
	
	// Add some files first
	for i := 0; i < 1000; i++ {
		fi := &mockFileInfo{
			name:    fmt.Sprintf("file%d.txt", i),
			size:    int64(100 + i),
			mode:    0644,
			modTime: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			isDir:   false,
		}
		id := c4.Identify(strings.NewReader(fmt.Sprintf("content%d", i)))
		mfi := manifest.NewFileInfo(fi, id)
		m.SetFileInfo(fmt.Sprintf("/file%d.txt", i), mfi)
	}
	
	b.Run("SetId", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			path := fmt.Sprintf("/file%d.txt", i%1000)
			newId := c4.Identify(strings.NewReader(fmt.Sprintf("new_content%d", i)))
			m.SetId(path, newId)
		}
	})
	
	b.Run("SetMetadata", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			path := fmt.Sprintf("/file%d.txt", i%1000)
			metadata := c4.Identify(strings.NewReader(fmt.Sprintf("metadata%d", i)))
			m.SetMetadata(path, metadata)
		}
	})
}

// BenchmarkMakeFileInfo benchmarks FileInfo creation
func BenchmarkMakeFileInfo(b *testing.B) {
	testTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	id := c4.Identify(strings.NewReader("test content"))
	metadata := c4.Identify(strings.NewReader("metadata"))
	
	b.Run("MakeFileInfo", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = manifest.MakeFileInfo(0644, 1024, testTime, "test.txt", id, metadata)
		}
	})
	
	b.Run("NewFileInfo", func(b *testing.B) {
		fi := &mockFileInfo{
			name:    "test.txt",
			size:    1024,
			mode:    0644,
			modTime: testTime,
			isDir:   false,
		}
		
		for i := 0; i < b.N; i++ {
			_ = manifest.NewFileInfo(fi, id, metadata)
		}
	})
}