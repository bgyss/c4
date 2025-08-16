package c4_test

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"sort"
	"strings"
	"testing"

	"github.com/Avalanche-io/c4"
)

// BenchmarkIdentify benchmarks the core C4 Identify function with various data sizes
func BenchmarkIdentify(b *testing.B) {
	sizes := []int{1, 100, 1024, 10240, 102400, 1048576} // 1B to 1MB

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			data := make([]byte, size)
			_, _ = rand.Read(data)
			reader := bytes.NewReader(data)

			b.ResetTimer()
			b.SetBytes(int64(size))

			for i := 0; i < b.N; i++ {
				_, _ = reader.Seek(0, 0)
				c4.Identify(reader)
			}
		})
	}
}

// BenchmarkIdentifyString benchmarks C4 identification of string data
func BenchmarkIdentifyString(b *testing.B) {
	testStrings := []string{
		"hello",
		"hello world",
		strings.Repeat("a", 100),
		strings.Repeat("test data ", 100),
		strings.Repeat("Lorem ipsum dolor sit amet, consectetur adipiscing elit. ", 100),
	}

	for i, str := range testStrings {
		b.Run(fmt.Sprintf("string_%d_len_%d", i, len(str)), func(b *testing.B) {
			b.SetBytes(int64(len(str)))
			for n := 0; n < b.N; n++ {
				c4.Identify(strings.NewReader(str))
			}
		})
	}
}

// BenchmarkIDOperations benchmarks various ID operations
func BenchmarkIDOperations(b *testing.B) {
	id := c4.Identify(strings.NewReader("test data for benchmarking"))
	
	b.Run("String", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = id.String()
		}
	})
	
	b.Run("IsNil", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = id.IsNil()
		}
	})
	
	b.Run("Digest", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = id.Digest()
		}
	})
}

// BenchmarkIDSliceOperations benchmarks c4.IDs slice operations
func BenchmarkIDSliceOperations(b *testing.B) {
	// Create test IDs
	var ids c4.IDs
	for i := 0; i < 100; i++ {
		id := c4.Identify(strings.NewReader(fmt.Sprintf("test data %d", i)))
		ids = append(ids, id)
	}
	
	b.Run("Sort", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			testIDs := make(c4.IDs, len(ids))
			copy(testIDs, ids)
			sort.Sort(testIDs)
		}
	})
	
	b.Run("ID", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = ids.ID()
		}
	})
	
	b.Run("Tree", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = ids.Tree()
		}
	})
}

// BenchmarkParse benchmarks ID parsing from strings
func BenchmarkParse(b *testing.B) {
	validID := "c459DdnZNhjY9JzJbJ6mF5pJhVBXpq7m8aBTgCrq36jMKxE8hHtNJLqJn2YCTFbCUbZzchNSwqJTbm1U3ZAiuVJ2"
	
	b.Run("ValidID", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := c4.Parse(validID)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	
	invalidIDs := []string{
		"invalid",
		"c459DdnZNhjY9JzJbJ6mF5pJhVBXpq7m8aBTgCrq36jMKxE8hHtNJLqJn2YCTFbCUbZzchNSwqJTbm1U3ZAiuVJ", // too short
		"x459DdnZNhjY9JzJbJ6mF5pJhVBXpq7m8aBTgCrq36jMKxE8hHtNJLqJn2YCTFbCUbZzchNSwqJTbm1U3ZAiuVJ2", // wrong prefix
	}
	
	for i, invalidID := range invalidIDs {
		b.Run(fmt.Sprintf("InvalidID_%d", i), func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				_, _ = c4.Parse(invalidID)
			}
		})
	}
}

// BenchmarkConcurrentIdentify benchmarks concurrent C4 identification
func BenchmarkConcurrentIdentify(b *testing.B) {
	data := make([]byte, 1024)
	_, _ = rand.Read(data)
	
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			reader := bytes.NewReader(data)
			c4.Identify(reader)
		}
	})
}

// BenchmarkDigestOperations benchmarks digest creation and manipulation
func BenchmarkDigestOperations(b *testing.B) {
	id := c4.Identify(strings.NewReader("test data"))
	digest := id.Digest()
	
	b.Run("Digest", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = id.Digest()
		}
	})
	
	b.Run("DigestBytes", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = digest[:]
		}
	})
}

// BenchmarkLargeFileIdentify benchmarks identification of large files
func BenchmarkLargeFileIdentify(b *testing.B) {
	sizes := []int{1024 * 1024, 10 * 1024 * 1024, 100 * 1024 * 1024} // 1MB, 10MB, 100MB
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%dMB", size/(1024*1024)), func(b *testing.B) {
			b.StopTimer()
			
			// Create a large data reader
			data := make([]byte, size)
			_, _ = rand.Read(data)
			
			b.StartTimer()
			b.SetBytes(int64(size))
			
			for i := 0; i < b.N; i++ {
				reader := bytes.NewReader(data)
				c4.Identify(reader)
			}
		})
	}
}

// BenchmarkStreamingIdentify benchmarks streaming identification
func BenchmarkStreamingIdentify(b *testing.B) {
	b.Run("ChunkedReader", func(b *testing.B) {
		data := make([]byte, 10*1024) // 10KB
		_, _ = rand.Read(data)
		
		b.SetBytes(int64(len(data)))
		
		for i := 0; i < b.N; i++ {
			reader := &chunkReader{data: data, chunkSize: 512}
			c4.Identify(reader)
		}
	})
}

// chunkReader simulates reading data in chunks to test streaming performance
type chunkReader struct {
	data      []byte
	pos       int
	chunkSize int
}

func (r *chunkReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	
	end := r.pos + r.chunkSize
	if end > len(r.data) {
		end = len(r.data)
	}
	
	copy(p, r.data[r.pos:end])
	n = end - r.pos
	r.pos = end
	
	return n, nil
}

// BenchmarkMemoryAllocation benchmarks memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	b.Run("MultipleSmallIdentify", func(b *testing.B) {
		data := "small test string"
		
		for i := 0; i < b.N; i++ {
			for j := 0; j < 100; j++ {
				c4.Identify(strings.NewReader(data))
			}
		}
	})
	
	b.Run("IDSliceCreation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var ids c4.IDs
			for j := 0; j < 100; j++ {
				id := c4.Identify(strings.NewReader(fmt.Sprintf("data %d", j)))
				ids = append(ids, id)
			}
		}
	})
}