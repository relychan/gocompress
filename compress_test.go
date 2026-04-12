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
	"errors"
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
	if err == nil || !errors.Is(err, ErrUnsupportedCompressionFormat) {
		t.Fatal("Expected unsupported error, got nil")
	}
}

func TestDecompressUnsupportedEncoding(t *testing.T) {
	c := NewCompressors()
	testData := "Hello, World!"

	reader := io.NopCloser(strings.NewReader(testData))
	_, err := c.Decompress(reader, "unsupported")
	if err == nil || !errors.Is(err, ErrUnsupportedCompressionFormat) {
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

func TestParseSupportedEncodingSpecialValues(t *testing.T) {
	c := NewCompressors()

	tests := []struct {
		name     string
		encoding string
		wantNil  bool
		wantAll  bool
	}{
		{"identity returns nil", "identity", true, false},
		{"IDENTITY case-insensitive", "IDENTITY", true, false},
		{"wildcard returns all", "*", false, true},
		{"empty returns nil", "", true, false},
		{"whitespace-only returns nil", "   ", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.ParseSupportedEncoding(tt.encoding)
			if err != nil {
				t.Errorf("ParseSupportedEncoding(%q) unexpected error: %v", tt.encoding, err)
			}

			if tt.wantNil && got != nil {
				t.Errorf("ParseSupportedEncoding(%q) = %v, want nil", tt.encoding, got)
			}

			if tt.wantAll {
				if len(got) == 0 {
					t.Errorf("ParseSupportedEncoding(%q): expected all formats, got none", tt.encoding)
				}

				// Wildcard must include all three supported formats.
				wantFormats := []CompressionFormat{EncodingDeflate, EncodingGzip, EncodingZstd}
				for _, f := range wantFormats {
					found := slices.Contains(got, f)
					if !found {
						t.Errorf("ParseSupportedEncoding(%q): missing format %q in %v", tt.encoding, f, got)
					}
				}
			}
		})
	}
}

func TestParseSupportedEncodingQualityValues(t *testing.T) {
	c := NewCompressors()

	tests := []struct {
		name     string
		encoding string
		want     []CompressionFormat
	}{
		{
			"quality ordering highest first",
			"gzip;q=0.5, deflate;q=0.9, zstd;q=1.0",
			[]CompressionFormat{EncodingZstd, EncodingDeflate, EncodingGzip},
		},
		{
			"equal quality preserves left-to-right order",
			"gzip;q=0.8, deflate;q=0.8",
			[]CompressionFormat{EncodingGzip, EncodingDeflate},
		},
		{
			"quality with spaces",
			"gzip ; q=0.5 , zstd ; q=0.9",
			[]CompressionFormat{EncodingZstd, EncodingGzip},
		},
		{
			"identity in multi-value is skipped",
			"gzip, identity, deflate",
			[]CompressionFormat{EncodingGzip, EncodingDeflate},
		},
		{
			"wildcard in multi-value is skipped",
			"gzip, *, deflate",
			[]CompressionFormat{EncodingGzip, EncodingDeflate},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.ParseSupportedEncoding(tt.encoding)
			if err != nil {
				t.Errorf("ParseSupportedEncoding(%q) unexpected error: %v", tt.encoding, err)
			}

			if !slices.Equal(got, tt.want) {
				t.Errorf("ParseSupportedEncoding(%q) = %v, want %v", tt.encoding, got, tt.want)
			}
		})
	}
}

func TestCompressFormatVariadic(t *testing.T) {
	c := NewCompressors()
	testData := "Hello, compression world!"

	t.Run("no formats copies data verbatim", func(t *testing.T) {
		var buf bytes.Buffer
		n, err := c.CompressFormat(&buf, strings.NewReader(testData))
		if err != nil {
			t.Fatalf("CompressFormat() error: %v", err)
		}

		if n != int64(len(testData)) {
			t.Errorf("CompressFormat() n = %d, want %d", n, len(testData))
		}

		if buf.String() != testData {
			t.Errorf("CompressFormat() data mismatch")
		}
	})

	t.Run("single unsupported format copies data verbatim", func(t *testing.T) {
		var buf bytes.Buffer
		_, err := c.CompressFormat(&buf, strings.NewReader(testData), "brotli")
		if err == nil || !errors.Is(err, ErrUnsupportedCompressionFormat) {
			t.Fatalf("DecompressFormat() expected error: %v", ErrUnsupportedCompressionFormat)
		}
	})

	t.Run("single gzip format compresses", func(t *testing.T) {
		var buf bytes.Buffer
		n, err := c.CompressFormat(&buf, strings.NewReader(testData), EncodingGzip)
		if err != nil {
			t.Fatalf("CompressFormat() error: %v", err)
		}

		if n != int64(len(testData)) {
			t.Errorf("CompressFormat() n = %d, want %d", n, len(testData))
		}

		if buf.String() == testData {
			t.Error("CompressFormat() with gzip: data should be compressed, not identical")
		}
	})
}

func TestDecompressFormatVariadic(t *testing.T) {
	c := NewCompressors()
	testData := "Hello, decompression world!"

	t.Run("no formats returns original reader", func(t *testing.T) {
		orig := io.NopCloser(strings.NewReader(testData))
		result, err := c.DecompressFormat(orig)
		if err != nil {
			t.Fatalf("DecompressFormat() error: %v", err)
		}

		data, err := io.ReadAll(result)
		if err != nil {
			t.Fatalf("ReadAll error: %v", err)
		}

		if string(data) != testData {
			t.Errorf("DecompressFormat() = %q, want %q", string(data), testData)
		}
	})

	t.Run("unknown format error", func(t *testing.T) {
		orig := io.NopCloser(strings.NewReader(testData))
		_, err := c.DecompressFormat(orig, "brotli")
		if err == nil || !errors.Is(err, ErrUnsupportedCompressionFormat) {
			t.Fatalf("DecompressFormat() expected error: %v", ErrUnsupportedCompressionFormat)
		}
	})

	t.Run("empty format string skipped", func(t *testing.T) {
		orig := io.NopCloser(strings.NewReader(testData))
		result, err := c.DecompressFormat(orig, "")
		if err != nil {
			t.Fatalf("DecompressFormat() error: %v", err)
		}

		data, err := io.ReadAll(result)
		if err != nil {
			t.Fatalf("ReadAll error: %v", err)
		}

		if string(data) != testData {
			t.Errorf("DecompressFormat() empty format = %q, want %q", string(data), testData)
		}
	})
}

func TestMultiEncodingRoundtrip(t *testing.T) {
	c := NewCompressors()
	testData := "Hello, multi-encoding world! " + strings.Repeat("compress me", 100)

	// compressMultipleFormats applies formats[0..len-2] via NewWriter and the
	// final format via the last compressor.Compress call in CompressFormat.
	// Decompress reverses the order. Only pairs where both sides are handled
	// correctly produce valid roundtrips.
	tests := []struct {
		name     string
		encoding string
	}{
		{"gzip then deflate", "gzip, deflate"},
		{"deflate then gzip", "deflate, gzip"},
		{"deflate then zstd", "deflate, zstd"},
		{"gzip then zstd", "gzip, zstd"},
		{"three encodings", "gzip, deflate, zstd"},
		{"four encodings", "gzip, deflate, zstd, gzip"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Compress
			var compressed bytes.Buffer
			_, err := c.Compress(&compressed, strings.NewReader(testData), tt.encoding)
			if err != nil {
				t.Fatalf("Compress(%q) error: %v", tt.encoding, err)
			}

			if compressed.Len() == 0 {
				t.Errorf("Compress(%q) produced no output", tt.encoding)
			}

			// Decompress
			decompressed, err := c.Decompress(io.NopCloser(&compressed), tt.encoding)
			if err != nil {
				t.Fatalf("Decompress(%q) error: %v", tt.encoding, err)
			}
			defer decompressed.Close()

			result, err := io.ReadAll(decompressed)
			if err != nil {
				t.Fatalf("ReadAll(%q) error: %v", tt.encoding, err)
			}

			if string(result) != testData {
				t.Errorf("Roundtrip(%q): data mismatch, got len=%d want len=%d", tt.encoding, len(result), len(testData))
			}
		})
	}
}

func TestCompressMultipleFormatsAllUnsupported(t *testing.T) {
	c := NewCompressors()
	testData := "Hello, World!"

	// When all formats in a multi-encoding are unsupported, CompressFormat falls
	// back to a plain copy (no closers are created).
	var buf bytes.Buffer
	_, err := c.CompressFormat(&buf, strings.NewReader(testData), "brotli", "lzma")
	if err == nil || !errors.Is(err, ErrUnsupportedCompressionFormat) {
		t.Fatalf("CompressFormat() expected error: %v, got nil", ErrUnsupportedCompressionFormat)
	}
}

func TestCompressDecompressWildcard(t *testing.T) {
	c := NewCompressors()
	testData := "Hello, wildcard world!"

	// Wildcard should pick all supported formats and produce something we can decompress.
	formats, err := c.ParseSupportedEncoding("*")
	if err != nil {
		t.Fatalf("ParseSupportedEncoding(*) error: %v", err)
	}

	var compressed bytes.Buffer
	n, err := c.CompressFormat(&compressed, strings.NewReader(testData), formats[0])
	if err != nil {
		t.Fatalf("CompressFormat(*) error: %v", err)
	}

	if n != int64(len(testData)) {
		t.Errorf("CompressFormat(*) n = %d, want %d", n, len(testData))
	}

	decompressed, err := c.DecompressFormat(io.NopCloser(&compressed), formats[0])
	if err != nil {
		t.Fatalf("DecompressFormat(*) error: %v", err)
	}
	defer decompressed.Close()

	result, err := io.ReadAll(decompressed)
	if err != nil {
		t.Fatalf("ReadAll(*) error: %v", err)
	}

	if string(result) != testData {
		t.Errorf("Roundtrip(*): data mismatch, got len=%d want len=%d", len(result), len(testData))
	}
}

func TestCompressErrorPropagation(t *testing.T) {
	c := NewCompressors()
	testData := "Hello, World!"

	// Compress returns the ParseSupportedEncoding error immediately (no partial write).
	var buf bytes.Buffer
	_, err := c.Compress(&buf, strings.NewReader(testData), "brotli")
	if err == nil || !errors.Is(err, ErrUnsupportedCompressionFormat) {
		t.Fatal("Compress() expected error for unsupported encoding, got nil")
	}
}

func TestDecompressErrorPropagation(t *testing.T) {
	c := NewCompressors()
	testData := "Hello, World!"

	reader := io.NopCloser(strings.NewReader(testData))
	_, err := c.Decompress(reader, "brotli")
	if err == nil || !errors.Is(err, ErrUnsupportedCompressionFormat) {
		t.Fatal("Decompress() expected error for unsupported encoding, got nil")
	}
}

func TestIsEncodingSupportedPartialMatch(t *testing.T) {
	c := NewCompressors()

	// "deflate, brotli" — deflate is supported even though brotli is not.
	// IsEncodingSupported returns true because at least one format matched.
	if !c.IsEncodingSupported("deflate, brotli") {
		t.Error("IsEncodingSupported(\"deflate, brotli\") = false, want true")
	}

	// Fully unsupported.
	if c.IsEncodingSupported("brotli, lzma") {
		t.Error("IsEncodingSupported(\"brotli, lzma\") = true, want false")
	}
}

func BenchmarkCompressionPipelines(b *testing.B) {
	c := NewCompressors()
	testData := "Hello, multi-encoding world! " + strings.Repeat("compress me", 100)

	b.Run("pipe2", func(b *testing.B) {
		for b.Loop() {
			// Compress
			var compressed bytes.Buffer
			_, err := c.compressMultipleFormats(&compressed, strings.NewReader(testData), []CompressionFormat{
				EncodingDeflate,
				EncodingGzip,
			})
			if err != nil {
				panic(err)
			}
		}
	})

	b.Run("sync2", func(b *testing.B) {
		for b.Loop() {
			// Compress
			var compressed bytes.Buffer
			_, err := c.compressMultipleFormatsSync(&compressed, strings.NewReader(testData), []CompressionFormat{
				EncodingDeflate,
				EncodingGzip,
			})
			if err != nil {
				panic(err)
			}
		}
	})

	b.Run("pipe3", func(b *testing.B) {
		for b.Loop() {
			// Compress
			var compressed bytes.Buffer
			_, err := c.compressMultipleFormats(&compressed, strings.NewReader(testData), []CompressionFormat{
				EncodingDeflate,
				EncodingGzip,
				EncodingZstd,
			})
			if err != nil {
				panic(err)
			}
		}
	})

	b.Run("sync3", func(b *testing.B) {
		for b.Loop() {
			// Compress
			var compressed bytes.Buffer
			_, err := c.compressMultipleFormatsSync(&compressed, strings.NewReader(testData), []CompressionFormat{
				EncodingDeflate,
				EncodingGzip,
				EncodingZstd,
			})
			if err != nil {
				panic(err)
			}
		}
	})
}
