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

	"github.com/klauspost/compress/flate"
)

const (
	// EncodingDeflate represents the deflate compression format.
	EncodingDeflate CompressionFormat = "deflate"
)

// DeflateCompressor implements the compression handler for deflate encoding.
type DeflateCompressor struct{}

// Compress the reader content with deflate encoding.
func (dc DeflateCompressor) Compress(w io.Writer, src io.Reader) (int64, error) {
	fw, err := flate.NewWriter(w, flate.DefaultCompression)
	if err != nil {
		return 0, err
	}

	size, copyErr := io.Copy(fw, src)
	closeErr := fw.Close()

	if copyErr != nil {
		return 0, copyErr
	}

	if closeErr != nil {
		return 0, closeErr
	}

	return size, nil
}

// Decompress the reader content with deflate encoding.
func (dc DeflateCompressor) Decompress(reader io.ReadCloser) (io.ReadCloser, error) {
	compressionReader := flate.NewReader(reader)

	return readCloserWrapper{
		CompressionReader: compressionReader,
		OriginalReader:    reader,
	}, nil
}
