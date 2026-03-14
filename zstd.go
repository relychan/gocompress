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
	"fmt"
	"io"

	"github.com/klauspost/compress/zstd"
)

const (
	// EncodingZstd represents the Zstandard compression format.
	EncodingZstd CompressionFormat = "zstd"
)

// ZstdCompressor implements the compression handler for zstandard encoding.
type ZstdCompressor struct{}

// Compress the reader content with zstd encoding.
func (zc ZstdCompressor) Compress(w io.Writer, src io.Reader) (int64, error) {
	zw, err := zstd.NewWriter(w)
	if err != nil {
		return 0, fmt.Errorf("failed to create the zstd writer: %w", err)
	}

	size, err := io.Copy(zw, src)
	_ = zw.Close()

	return size, err
}

// Decompress the reader content with zstd encoding.
func (zc ZstdCompressor) Decompress(reader io.ReadCloser) (io.ReadCloser, error) {
	compressionReader, err := zstd.NewReader(reader)
	if err != nil {
		return nil, err
	}

	return readCloserWrapper{
		CompressionReader: &zstdReader{
			Decoder: compressionReader,
		},
		OriginalReader: reader,
	}, nil
}

type zstdReader struct {
	*zstd.Decoder
}

func (zd *zstdReader) Close() error {
	zd.Decoder.Close()

	return nil
}
