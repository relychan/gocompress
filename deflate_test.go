package gocompress

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestDeflateCompressor_Compress(t *testing.T) {
	dc := DeflateCompressor{}
	testData := "Hello, World! This is a test for deflate compression."

	var compressed bytes.Buffer
	reader := strings.NewReader(testData)
	n, err := dc.Compress(&compressed, reader)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	if n != int64(len(testData)) {
		t.Errorf("Compress returned size %d, want %d", n, len(testData))
	}

	if compressed.Len() == 0 {
		t.Error("Compressed data is empty")
	}

	// Verify it's actually compressed (should be smaller for repetitive data)
	largeData := strings.Repeat("test ", 1000)
	var largeCompressed bytes.Buffer
	largeReader := strings.NewReader(largeData)
	_, err = dc.Compress(&largeCompressed, largeReader)
	if err != nil {
		t.Fatalf("Compress large data failed: %v", err)
	}

	if largeCompressed.Len() >= len(largeData) {
		t.Errorf("Compressed size %d >= original size %d", largeCompressed.Len(), len(largeData))
	}
}

func TestDeflateCompressor_Decompress(t *testing.T) {
	dc := DeflateCompressor{}
	testData := "Hello, World! This is a test for deflate decompression."

	// First compress the data
	var compressed bytes.Buffer
	reader := strings.NewReader(testData)
	_, err := dc.Compress(&compressed, reader)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	// Now decompress it
	decompressed, err := dc.Decompress(io.NopCloser(&compressed))
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
}

func TestDeflateCompressor_DecompressInvalidData(t *testing.T) {
	dc := DeflateCompressor{}
	invalidData := "This is not deflate compressed data"

	reader := io.NopCloser(strings.NewReader(invalidData))
	decompressed, err := dc.Decompress(reader)
	if err != nil {
		t.Fatalf("Decompress should not fail immediately: %v", err)
	}
	defer decompressed.Close()

	// Error should occur when trying to read
	_, err = io.ReadAll(decompressed)
	if err == nil {
		t.Error("ReadAll should fail with invalid deflate data")
	}
}

func TestDeflateCompressor_CompressEmpty(t *testing.T) {
	dc := DeflateCompressor{}

	var compressed bytes.Buffer
	reader := strings.NewReader("")
	n, err := dc.Compress(&compressed, reader)
	if err != nil {
		t.Fatalf("Compress empty data failed: %v", err)
	}

	if n != 0 {
		t.Errorf("Compress empty data returned size %d, want 0", n)
	}

	// Decompress empty compressed data
	decompressed, err := dc.Decompress(io.NopCloser(&compressed))
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
}

func TestDeflateCompressor_RoundTrip(t *testing.T) {
	dc := DeflateCompressor{}
	testCases := []string{
		"Short string",
		strings.Repeat("B", 1000),
		"String with special characters: !@#$%^&*()_+-=[]{}|;:',.<>?/~`",
		"Multi\nline\nstring\nwith\nnewlines",
		"Unicode: 你好世界 🌍🚀",
	}

	for _, testData := range testCases {
		t.Run(testData[:min(20, len(testData))], func(t *testing.T) {
			var compressed bytes.Buffer
			reader := strings.NewReader(testData)
			_, err := dc.Compress(&compressed, reader)
			if err != nil {
				t.Fatalf("Compress failed: %v", err)
			}

			decompressed, err := dc.Decompress(io.NopCloser(&compressed))
			if err != nil {
				t.Fatalf("Decompress failed: %v", err)
			}
			defer decompressed.Close()

			result, err := io.ReadAll(decompressed)
			if err != nil {
				t.Fatalf("ReadAll failed: %v", err)
			}

			if string(result) != testData {
				t.Errorf("Round trip failed: got %q, want %q", string(result), testData)
			}
		})
	}
}
