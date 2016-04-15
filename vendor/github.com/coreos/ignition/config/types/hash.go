// Copyright 2016 CoreOS, Inc.
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

package types

import (
	"crypto"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
)

var (
	ErrHashMalformed    = errors.New("malformed hash specifier")
	ErrHashWrongSize    = errors.New("incorrect size for hash sum")
	ErrHashUnrecognized = errors.New("unrecognized hash function")
)

type Hash struct {
	Function string
	Sum      string
}

func (h *Hash) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return h.unmarshal(unmarshal)
}

func (h *Hash) UnmarshalJSON(data []byte) error {
	return h.unmarshal(func(th interface{}) error {
		return json.Unmarshal(data, th)
	})
}

func (h Hash) MarshalJSON() ([]byte, error) {
	return []byte(`"` + h.Function + "-" + h.Sum + `"`), nil
}

func (h *Hash) unmarshal(unmarshal func(interface{}) error) error {
	var th string
	if err := unmarshal(&th); err != nil {
		return err
	}

	parts := strings.SplitN(th, "-", 2)
	if len(parts) != 2 {
		return ErrHashMalformed
	}

	h.Function = parts[0]
	h.Sum = parts[1]

	return h.assertValid()
}

func (h Hash) assertValid() error {
	var hash crypto.Hash
	switch h.Function {
	case "sha512":
		hash = crypto.SHA512
	default:
		return ErrHashUnrecognized
	}

	if len(h.Sum) != hex.EncodedLen(hash.Size()) {
		return ErrHashWrongSize
	}

	return nil
}
