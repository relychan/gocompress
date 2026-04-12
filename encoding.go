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
	QualityValue float64
}

func parseQualityParam(params []string) (float64, error) {
	var (
		quantity float64 = 1
		err      error
	)

	for _, param := range params {
		key, value, present := strings.Cut(param, "=")
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

	return quantity, nil
}
