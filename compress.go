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
	"bytes"
	"fmt"
	"io"
	"slices"
	"strings"
)

// DefaultCompressor the default compressors.
var DefaultCompressor = NewCompressors() //nolint:gochecknoglobals

// Compressor abstracts the interface for a compression handler.
type Compressor interface {
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

// AcceptEncoding returns the Accept-Encoding header with supported compression encodings.
func (c Compressors) AcceptEncoding() string {
	return c.acceptEncoding
}

// IsEncodingSupported checks if the input encoding is supported.
func (c Compressors) IsEncodingSupported(encoding string) bool {
	return c.FindSupportedEncoding(encoding) != ""
}

// FindSupportedEncoding returns the supported encoding from the input string.
func (c Compressors) FindSupportedEncoding(encoding string) CompressionFormat {
	results, _ := c.ParseSupportedEncoding(encoding)
	if len(results) == 0 {
		return ""
	}

	return results[len(results)-1]
}

// ParseSupportedEncoding returns the supported encodings from the input string.
func (c Compressors) ParseSupportedEncoding( //nolint:cyclop,funlen
	encoding string,
) ([]CompressionFormat, error) {
	encoding = strings.TrimSpace(encoding)
	if encoding == "" {
		return nil, nil
	}

	encoding = strings.ToLower(encoding)
	if encoding == EncodingWildcard || encoding == EncodingIdentity {
		return nil, nil
	}

	compressionFormat := CompressionFormat(encoding)

	_, ok := c.compressors[compressionFormat]
	if ok {
		return []CompressionFormat{compressionFormat}, nil
	}

	var err error

	parts := strings.Split(encoding, ",")
	encodings := make([]CompressionEncoding, 0, len(parts))

	for _, part := range parts {
		if part == "" {
			continue
		}

		part = strings.TrimSpace(part)
		if part == "" || part == EncodingIdentity || part == EncodingWildcard {
			continue
		}

		partFormat := CompressionFormat(part)

		_, ok := c.compressors[partFormat]
		if ok {
			encodings = append(encodings, CompressionEncoding{
				Format:       partFormat,
				QualityValue: 1,
			})

			continue
		}

		rawParams := strings.Split(part, ";")

		enc := strings.TrimSpace(rawParams[0])
		if enc == "" || enc == EncodingIdentity || enc == EncodingWildcard {
			continue
		}

		partFormat = CompressionFormat(enc)

		_, ok = c.compressors[partFormat]
		if !ok {
			if err == nil {
				err = fmt.Errorf("%w: %s", ErrUnsupportedCompressionFormat, enc)
			}

			continue
		}

		quantity, qErr := parseQualityParam(rawParams[1:])
		if qErr != nil && err == nil {
			// Relax the parse error and make the priority lowest.
			err = fmt.Errorf(
				"failed to parse quantity value of compression format %s: %w",
				enc,
				err,
			)
		}

		encodings = append(encodings, CompressionEncoding{
			Format:       partFormat,
			QualityValue: quantity,
		})
	}

	slices.SortFunc(encodings, func(a, b CompressionEncoding) int {
		if a.QualityValue == b.QualityValue {
			return 0
		}

		if a.QualityValue < b.QualityValue {
			return -1
		}

		return 1
	})

	results := make([]CompressionFormat, len(encodings))

	for i, enc := range encodings {
		results[i] = enc.Format
	}

	return results, err
}

// Compress and writes compressed data by a raw content encoding value.
// When multiple encodings are used, the client must end the data in the order to that listed in the header.
// For example, Content-Encoding: deflate, gzip means the data was first deflated, then gzipped.
func (c Compressors) Compress(w io.Writer, encoding string, data io.Reader) (int64, error) {
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

	for i := range len(formats) - 1 {
		format := formats[i]
		if format == "" {
			continue
		}

		compressor, ok := c.compressors[format]
		if !ok {
			continue
		}

		buf := new(bytes.Buffer)

		_, err := compressor.Compress(buf, data)
		if err != nil {
			return 0, err
		}

		data = buf
	}

	lastFormat := formats[len(formats)-1]

	compressor, ok := c.compressors[lastFormat]
	if !ok {
		return io.Copy(w, data)
	}

	return compressor.Compress(w, data)
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

	result := reader

	for i := len(formats) - 1; i >= 0; i-- {
		result, err = c.DecompressFormat(reader, formats[i])
		if err != nil {
			return nil, err
		}
	}

	return result, nil
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

		result, err = compressor.Decompress(reader)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
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
