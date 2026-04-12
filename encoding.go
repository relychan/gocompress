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
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// CompressionFormat represents a compression format enumeration.
type CompressionFormat string

const (
	// EncodingIdentity indicates the identity function (that is, without modification or compression).
	// This value is always considered as acceptable, even if omitted.
	EncodingIdentity = "identity"
	// EncodingWildcard matches any content encoding not already listed in the header.
	// This is the default value if the header is not present.
	EncodingWildcard = "*"
)

// ErrUnsupportedCompressionFormat occurs when the compression format is not supported.
var ErrUnsupportedCompressionFormat = errors.New("unsupported compression format")

// CompressionEncoding represents the parsed compression encoding from string.
type CompressionEncoding struct {
	Format       CompressionFormat
	QualityValue float32
	Index        int32
}

// AcceptEncoding returns the Accept-Encoding header with supported compression encodings.
func (c Compressors) AcceptEncoding() string {
	return c.acceptEncoding
}

// IsEncodingSupported checks if the input encoding is supported.
func (c Compressors) IsEncodingSupported(encoding string) bool {
	results, _ := c.ParseSupportedEncoding(encoding)

	return len(results) > 0
}

// ParseSupportedEncoding returns the supported encodings from the input string.
// The priority order is determined by quality value, or the left-to-right order for the same weight in the header.
// The server generally selects the first encoding listed that it also supports.
// Return the first error if there is any.
func (c Compressors) ParseSupportedEncoding( //nolint:cyclop,funlen
	encoding string,
) ([]CompressionFormat, error) {
	encoding = strings.TrimSpace(encoding)
	if encoding == "" {
		return nil, nil
	}

	encoding = strings.ToLower(encoding)
	if encoding == EncodingIdentity {
		return nil, nil
	}

	if encoding == EncodingWildcard {
		return []CompressionFormat{
			EncodingDeflate,
			EncodingGzip,
			EncodingZstd,
		}, nil
	}

	compressionFormat := CompressionFormat(encoding)

	_, ok := c.compressors[compressionFormat]
	if ok {
		return []CompressionFormat{compressionFormat}, nil
	}

	var err error

	parts := strings.Split(encoding, ",")
	encodings := make([]CompressionEncoding, 0, len(parts))

	for i, part := range parts {
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
				Index:        int32(i),
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
			Index:        int32(i),
		})
	}

	slices.SortFunc(encodings, func(a, b CompressionEncoding) int {
		if a.QualityValue == b.QualityValue {
			return int(a.Index - b.Index)
		}

		if a.QualityValue < b.QualityValue {
			return 1
		}

		return -1
	})

	results := make([]CompressionFormat, len(encodings))

	for i, enc := range encodings {
		results[i] = enc.Format
	}

	return results, err
}

func parseQualityParam(params []string) (float32, error) {
	var (
		quantity float64 = 1
		err      error
	)

	for _, param := range params {
		key, value, present := strings.Cut(param, "=")
		if !present {
			continue
		}

		key = strings.TrimSpace(key)

		value = strings.TrimSpace(value)
		if !present || key != "q" || value == "" {
			continue
		}

		quantity, err = strconv.ParseFloat(value, 32)
		if err != nil {
			return 0, err
		}
	}

	return float32(quantity), nil
}
