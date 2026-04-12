// Copyright 2026 RelyChan Pte. Ltd
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gocompress

import (
	"bytes"
	"io"
	"slices"
	"strings"
	"testing"
)

func TestNewCompressors(t *testing.T) {
	c := NewCompressors()
	if c == nil {
		t.Fatal("NewCompressors returned nil")
	}

	if c.acceptEncoding == "" {
		t.Error("acceptEncoding should not be empty")
	}

	if len(c.compressors) != 3 {
		t.Errorf("expected 3 compressors, got %d", len(c.compressors))
	}

	// Check that all expected compressors are present
	expectedFormats := []CompressionFormat{EncodingGzip, EncodingDeflate, EncodingZstd}
	for _, format := range expectedFormats {
		if _, ok := c.compressors[format]; !ok {
			t.Errorf("compressor for %s not found", format)
		}
	}
}

func TestAcceptEncoding(t *testing.T) {
	c := NewCompressors()
	acceptEncoding := c.AcceptEncoding()

	if acceptEncoding == "" {
		t.Error("AcceptEncoding returned empty string")
	}

	// Check that all encodings are present
	if !strings.Contains(acceptEncoding, "zstd") {
		t.Error("AcceptEncoding should contain 'zstd'")
	}
	if !strings.Contains(acceptEncoding, "gzip") {
		t.Error("AcceptEncoding should contain 'gzip'")
	}
	if !strings.Contains(acceptEncoding, "deflate") {
		t.Error("AcceptEncoding should contain 'deflate'")
	}
}

func TestIsEncodingSupported(t *testing.T) {
	c := NewCompressors()

	tests := []struct {
		name     string
		encoding string
		want     bool
	}{
		{"gzip supported", "gzip", true},
		{"deflate supported", "deflate", true},
		{"zstd supported", "zstd", true},
		{"unsupported encoding", "brotli", false},
		{"empty encoding", "", false},
		{"uppercase gzip", "GZIP", true}, // FindSupportedEncoding converts to lowercase
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.IsEncodingSupported(tt.encoding)
			if got != tt.want {
				t.Errorf("IsEncodingSupported(%q) = %v, want %v", tt.encoding, got, tt.want)
			}
		})
	}
}

func TestParseSupportedEncoding(t *testing.T) {
	c := NewCompressors()

	tests := []struct {
		name     string
		encoding string
		want     []CompressionFormat
		errorMsg string
	}{
		{"exact gzip", "gzip", []CompressionFormat{EncodingGzip}, ""},
		{"exact deflate", "deflate", []CompressionFormat{EncodingDeflate}, ""},
		{"exact zstd", "zstd", []CompressionFormat{EncodingZstd}, ""},
		{"multiple encodings - first supported", "gzip, deflate", []CompressionFormat{EncodingGzip, EncodingDeflate}, ""},
		{"multiple encodings - second supported", "deflate, brotli", []CompressionFormat{EncodingDeflate}, ErrUnsupportedCompressionFormat.Error()},
		{"with spaces", " gzip ", []CompressionFormat{EncodingGzip}, ""},
		{"multiple with spaces", "brotli, deflate, gzip", []CompressionFormat{EncodingDeflate, EncodingGzip}, ErrUnsupportedCompressionFormat.Error()},
		{"unsupported", "brotli", nil, ErrUnsupportedCompressionFormat.Error()},
		{"empty", "", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.ParseSupportedEncoding(tt.encoding)
			if tt.errorMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("FindSupportedEncoding(%q): Expected error, got: %s", tt.encoding, err)
				}
			} else if err != nil {
				t.Errorf("FindSupportedEncoding(%q): Expected nil error, got: %s", tt.encoding, err)
			}

			if !slices.Equal(got, tt.want) {
				t.Errorf("FindSupportedEncoding(%q) = %v, want %v", tt.encoding, got, tt.want)
			}
		})
	}
}

func TestCompressDecompress(t *testing.T) {
	c := NewCompressors()
	testData := "Hello, World! This is a test string for compression."

	encodings := []string{"gzip", "deflate", "zstd"}

	for _, encoding := range encodings {
		t.Run(encoding, func(t *testing.T) {
			// Compress
			var compressed bytes.Buffer
			reader := strings.NewReader(testData)
			n, err := c.Compress(&compressed, reader, encoding)
			if err != nil {
				t.Fatalf("Compress failed: %v", err)
			}

			if n != int64(len(testData)) {
				t.Errorf("Compress returned size %d, want %d", n, len(testData))
			}

			// Decompress
			decompressed, err := c.Decompress(io.NopCloser(&compressed), encoding)
			if err != nil {
				t.Fatalf("Decompress failed: %v", err)
			}
			defer decompressed.Close()

			result, err := io.ReadAll(decompressed)
			if err != nil {
				t.Fatalf("ReadAll failed: %v", err)
			}

			if string(result) != testData {
				t.Errorf("Decompressed data = %q, want %q", string(result), testData)
			}
		})
	}
}

func TestCompressUnsupportedEncoding(t *testing.T) {
	c := NewCompressors()
	testData := "Hello, World!"

	var buf bytes.Buffer
	reader := strings.NewReader(testData)
	_, err := c.Compress(&buf, reader, "unsupported")
	if err == nil {
		t.Fatal("Expected unsupported error, got nil")
	}
}

func TestDecompressUnsupportedEncoding(t *testing.T) {
	c := NewCompressors()
	testData := "Hello, World!"

	reader := io.NopCloser(strings.NewReader(testData))
	_, err := c.Decompress(reader, "unsupported")
	if err == nil {
		t.Fatal("Expected unsupported error, got nil")
	}
}

func TestCompressEmptyData(t *testing.T) {
	c := NewCompressors()
	encodings := []string{"gzip", "deflate", "zstd"}

	for _, encoding := range encodings {
		t.Run(encoding, func(t *testing.T) {
			var compressed bytes.Buffer
			reader := strings.NewReader("")
			n, err := c.Compress(&compressed, reader, encoding)
			if err != nil {
				t.Fatalf("Compress empty data failed: %v", err)
			}

			if n != 0 {
				t.Errorf("Compress empty data returned size %d, want 0", n)
			}

			// Decompress
			decompressed, err := c.Decompress(io.NopCloser(&compressed), encoding)
			if err != nil {
				t.Fatalf("Decompress empty data failed: %v", err)
			}
			defer decompressed.Close()

			result, err := io.ReadAll(decompressed)
			if err != nil {
				t.Fatalf("ReadAll failed: %v", err)
			}

			if len(result) != 0 {
				t.Errorf("Decompressed empty data length = %d, want 0", len(result))
			}
		})
	}
}

func TestCompressLargeData(t *testing.T) {
	c := NewCompressors()
	// Create a large test string (1MB)
	testData := strings.Repeat("Hello, World! This is a test string for compression. ", 20000)

	encodings := []string{"gzip", "deflate", "zstd"}

	for _, encoding := range encodings {
		t.Run(encoding, func(t *testing.T) {
			var compressed bytes.Buffer
			reader := strings.NewReader(testData)
			n, err := c.Compress(&compressed, reader, encoding)
			if err != nil {
				t.Fatalf("Compress large data failed: %v", err)
			}

			if n != int64(len(testData)) {
				t.Errorf("Compress returned size %d, want %d", n, len(testData))
			}

			// Verify compression actually reduced size
			if compressed.Len() >= len(testData) {
				t.Errorf("Compressed size %d >= original size %d", compressed.Len(), len(testData))
			}

			// Decompress
			decompressed, err := c.Decompress(io.NopCloser(&compressed), encoding)
			if err != nil {
				t.Fatalf("Decompress large data failed: %v", err)
			}
			defer decompressed.Close()

			result, err := io.ReadAll(decompressed)
			if err != nil {
				t.Fatalf("ReadAll failed: %v", err)
			}

			if string(result) != testData {
				t.Errorf("Decompressed data length = %d, want %d", len(result), len(testData))
			}
		})
	}
}

func TestDefaultCompressor(t *testing.T) {
	if DefaultCompressor == nil {
		t.Fatal("DefaultCompressor is nil")
	}

	// Test that DefaultCompressor works
	testData := "Test with default compressor"
	var compressed bytes.Buffer
	reader := strings.NewReader(testData)
	_, err := DefaultCompressor.Compress(&compressed, reader, "gzip")
	if err != nil {
		t.Fatalf("DefaultCompressor.Compress failed: %v", err)
	}

	decompressed, err := DefaultCompressor.Decompress(io.NopCloser(&compressed), "gzip")
	if err != nil {
		t.Fatalf("DefaultCompressor.Decompress failed: %v", err)
	}
	defer decompressed.Close()

	result, err := io.ReadAll(decompressed)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if string(result) != testData {
		t.Errorf("DefaultCompressor result = %q, want %q", string(result), testData)
	}
}
