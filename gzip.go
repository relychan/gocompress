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
	"io"

	"github.com/klauspost/compress/gzip"
)

const (
	// EncodingGzip represents the gzip compression format.
	EncodingGzip CompressionFormat = "gzip"
)

// GzipCompressor implements the compression handler for gzip encoding.
type GzipCompressor struct{}

var _ Compressor = (*GzipCompressor)(nil)

// NewWriter will create a new compression encoder.
func (gc GzipCompressor) NewWriter(w io.Writer) (io.WriteCloser, error) {
	return gzip.NewWriter(w), nil
}

// Compress the reader content with gzip encoding.
func (gc GzipCompressor) Compress(w io.Writer, src io.Reader) (int64, error) {
	zw := gzip.NewWriter(w)

	size, err := io.Copy(zw, src)
	_ = zw.Close()

	return size, err
}

// Decompress the reader content with gzip encoding.
func (gc GzipCompressor) Decompress(reader io.ReadCloser) (io.ReadCloser, error) {
	compressionReader, err := gzip.NewReader(reader)
	if err != nil {
		return nil, err
	}

	return readCloserWrapper{
		CompressionReader: compressionReader,
		OriginalReader:    reader,
	}, nil
}
