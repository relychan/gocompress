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

// Package gocompress wraps compressors with a generic interface.
package gocompress

import (
	"io"
	"strings"
)

// DefaultCompressor the default compressors.
var DefaultCompressor = NewCompressors() //nolint:gochecknoglobals

// Compressor abstracts the interface for a compression handler.
type Compressor interface {
	NewWriter(w io.Writer) (io.WriteCloser, error)
	Compress(w io.Writer, src io.Reader) (int64, error)
	Decompress(reader io.ReadCloser) (io.ReadCloser, error)
}

// Compressors is a general helper for web compression.
type Compressors struct {
	acceptEncoding string
	compressors    map[CompressionFormat]Compressor
}

// NewCompressors create a Compressors instance.
func NewCompressors() *Compressors {
	compressors := map[CompressionFormat]Compressor{
		EncodingGzip:    GzipCompressor{},
		EncodingDeflate: DeflateCompressor{},
		EncodingZstd:    ZstdCompressor{},
	}

	return &Compressors{
		acceptEncoding: strings.Join([]string{
			string(EncodingZstd),
			string(EncodingGzip),
			string(EncodingDeflate),
		}, ", "),
		compressors: compressors,
	}
}

// Compress and writes compressed data by a raw content encoding value.
// When multiple encodings are used, the client must encode the data in the order listed in the header.
// For example, Content-Encoding: deflate, gzip means the data was first deflated, then gzipped.
func (c Compressors) Compress(w io.Writer, data io.Reader, encoding string) (int64, error) {
	formats, err := c.ParseSupportedEncoding(encoding)
	if err != nil {
		return 0, err
	}

	return c.CompressFormat(w, data, formats...)
}

// CompressFormat and writes compressed data by a compression format enum.
func (c Compressors) CompressFormat(
	w io.Writer,
	data io.Reader,
	formats ...CompressionFormat,
) (int64, error) {
	if len(formats) == 0 {
		return io.Copy(w, data)
	}

	// Compress and write the stream directly.
	if len(formats) == 1 {
		compressor, ok := c.compressors[formats[0]]
		if !ok {
			return io.Copy(w, data)
		}

		return compressor.Compress(w, data)
	}

	return c.compressMultipleFormats(w, data, formats)
}

// Decompress reads and decompresses the reader with an equivalent content encoding.
// When multiple encodings are used, the client must decode the data in the reverse order to that listed in the header.
// For example, Content-Encoding: deflate, gzip means the data was first deflated, then gzipped.
// To decode, you must gunzip first, then inflate.
func (c Compressors) Decompress(reader io.ReadCloser, encoding string) (io.ReadCloser, error) {
	formats, err := c.ParseSupportedEncoding(encoding)
	if err != nil {
		return nil, err
	}

	return c.DecompressFormat(reader, formats...)
}

// DecompressFormat reads and decompresses the reader with a compression format.
// Because the compression is lazy, the original reader should be closed on error.
func (c Compressors) DecompressFormat(
	reader io.ReadCloser,
	formats ...CompressionFormat,
) (io.ReadCloser, error) {
	var (
		result = reader
		err    error
	)

	for i := len(formats) - 1; i >= 0; i-- {
		format := formats[i]
		if format == "" {
			continue
		}

		compressor, ok := c.compressors[format]
		if !ok {
			continue
		}

		result, err = compressor.Decompress(result)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (c Compressors) compressMultipleFormats( //nolint:funlen
	w io.Writer,
	data io.Reader,
	formats []CompressionFormat,
) (int64, error) {
	pr, pw := io.Pipe()

	var writer io.Writer = pw

	closers := make([]io.Closer, 0, len(formats))

	closeFunc := func() {
		for i := len(closers) - 1; i >= 0; i-- {
			_ = closers[i].Close()
		}
	}

	for i := range len(formats) - 1 {
		format := formats[i]
		if format == "" {
			continue
		}

		compressor, ok := c.compressors[format]
		if !ok {
			continue
		}

		cw, err := compressor.NewWriter(writer)
		if err != nil {
			closeFunc()

			return 0, err
		}

		closers = append(closers, cw)
		writer = cw
	}

	if len(closers) == 0 {
		_ = pr.Close()
		_ = pw.Close()

		return io.Copy(w, data)
	}

	go func() {
		_, err := io.Copy(writer, data)
		if err != nil {
			closeFunc()

			_ = pw.CloseWithError(err)

			return
		}

		closeFunc()

		_ = pw.Close()
	}()

	written, err := io.Copy(w, pr)

	_ = pr.Close()

	return written, err
}

type readCloserWrapper struct {
	CompressionReader io.ReadCloser
	OriginalReader    io.ReadCloser
}

func (rcw readCloserWrapper) Close() error {
	_ = rcw.OriginalReader.Close()

	return rcw.CompressionReader.Close()
}

func (rcw readCloserWrapper) Read(p []byte) (int, error) {
	return rcw.CompressionReader.Read(p)
}
