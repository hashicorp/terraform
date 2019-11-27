// Copyright 2018 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package codec

import (
	"bytes"

	"github.com/pkg/errors"
)

const (
	encGroupSize = 8
	encMarker    = byte(0xFF)
	encPad       = byte(0x0)
)

var pads = make([]byte, encGroupSize)

// DecodeBytes decodes a TiDB encoded byte slice.
func DecodeBytes(b []byte) ([]byte, []byte, error) {
	buf := make([]byte, 0, len(b)/(encGroupSize+1)*encGroupSize)
	for {
		if len(b) < encGroupSize+1 {
			return nil, nil, errors.New("insufficient bytes to decode value")
		}

		groupBytes := b[:encGroupSize+1]

		group := groupBytes[:encGroupSize]
		marker := groupBytes[encGroupSize]

		padCount := encMarker - marker
		if padCount > encGroupSize {
			return nil, nil, errors.Errorf("invalid marker byte, group bytes %q", groupBytes)
		}

		realGroupSize := encGroupSize - padCount
		buf = append(buf, group[:realGroupSize]...)
		b = b[encGroupSize+1:]

		if padCount != 0 {
			// Check validity of padding bytes.
			if !bytes.Equal(group[realGroupSize:], pads[:padCount]) {
				return nil, nil, errors.Errorf("invalid padding byte, group bytes %q", groupBytes)
			}
			break
		}
	}
	return b, buf, nil
}

// EncodeBytes encodes a byte slice into TiDB's encoded form.
func EncodeBytes(b []byte) []byte {
	dLen := len(b)
	reallocSize := (dLen/encGroupSize + 1) * (encGroupSize + 1)
	result := make([]byte, 0, reallocSize)
	for idx := 0; idx <= dLen; idx += encGroupSize {
		remain := dLen - idx
		padCount := 0
		if remain >= encGroupSize {
			result = append(result, b[idx:idx+encGroupSize]...)
		} else {
			padCount = encGroupSize - remain
			result = append(result, b[idx:]...)
			result = append(result, pads[:padCount]...)
		}

		marker := encMarker - byte(padCount)
		result = append(result, marker)
	}
	return result
}
