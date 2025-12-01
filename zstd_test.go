package gocompress

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestZstdCompressor_Compress(t *testing.T) {
	zc := ZstdCompressor{}
	testData := "Hello, World! This is a test for zstd compression."

	var compressed bytes.Buffer
	reader := strings.NewReader(testData)
	n, err := zc.Compress(&compressed, reader)
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
	_, err = zc.Compress(&largeCompressed, largeReader)
	if err != nil {
		t.Fatalf("Compress large data failed: %v", err)
	}

	if largeCompressed.Len() >= len(largeData) {
		t.Errorf("Compressed size %d >= original size %d", largeCompressed.Len(), len(largeData))
	}
}

func TestZstdCompressor_Decompress(t *testing.T) {
	zc := ZstdCompressor{}
	testData := "Hello, World! This is a test for zstd decompression."

	// First compress the data
	var compressed bytes.Buffer
	reader := strings.NewReader(testData)
	_, err := zc.Compress(&compressed, reader)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	// Now decompress it
	decompressed, err := zc.Decompress(io.NopCloser(&compressed))
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

func TestZstdCompressor_DecompressInvalidData(t *testing.T) {
	zc := ZstdCompressor{}
	invalidData := "This is not zstd compressed data"

	reader := io.NopCloser(strings.NewReader(invalidData))
	decompressed, err := zc.Decompress(reader)
	if err != nil {
		// Some invalid data may fail immediately
		return
	}
	defer decompressed.Close()

	// Error should occur when trying to read
	_, err = io.ReadAll(decompressed)
	if err == nil {
		t.Error("ReadAll should fail with invalid zstd data")
	}
}

func TestZstdCompressor_CompressEmpty(t *testing.T) {
	zc := ZstdCompressor{}

	var compressed bytes.Buffer
	reader := strings.NewReader("")
	n, err := zc.Compress(&compressed, reader)
	if err != nil {
		t.Fatalf("Compress empty data failed: %v", err)
	}

	if n != 0 {
		t.Errorf("Compress empty data returned size %d, want 0", n)
	}

	// Decompress empty compressed data
	decompressed, err := zc.Decompress(io.NopCloser(&compressed))
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

func TestZstdCompressor_RoundTrip(t *testing.T) {
	zc := ZstdCompressor{}
	testCases := []string{
		"Short string",
		strings.Repeat("C", 1000),
		"String with special characters: !@#$%^&*()_+-=[]{}|;:',.<>?/~`",
		"Multi\nline\nstring\nwith\nnewlines",
		"Unicode: 你好世界 🌍🚀",
	}

	for _, testData := range testCases {
		t.Run(testData[:min(20, len(testData))], func(t *testing.T) {
			var compressed bytes.Buffer
			reader := strings.NewReader(testData)
			_, err := zc.Compress(&compressed, reader)
			if err != nil {
				t.Fatalf("Compress failed: %v", err)
			}

			decompressed, err := zc.Decompress(io.NopCloser(&compressed))
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

func TestZstdReader_Close(t *testing.T) {
	zc := ZstdCompressor{}
	testData := "Test zstd reader close"

	var compressed bytes.Buffer
	reader := strings.NewReader(testData)
	_, err := zc.Compress(&compressed, reader)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	decompressed, err := zc.Decompress(io.NopCloser(&compressed))
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}

	// Close should not return an error
	err = decompressed.Close()
	if err != nil {
		t.Errorf("Close returned error: %v", err)
	}
}
