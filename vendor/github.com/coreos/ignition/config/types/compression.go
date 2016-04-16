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
	"encoding/json"
	"errors"
)

var (
	ErrCompressionInvalid = errors.New("invalid compression method")
)

type Compression string

func (c *Compression) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return c.unmarshal(unmarshal)
}

func (c *Compression) UnmarshalJSON(data []byte) error {
	return c.unmarshal(func(tc interface{}) error {
		return json.Unmarshal(data, tc)
	})
}

func (c *Compression) unmarshal(unmarshal func(interface{}) error) error {
	var tc string
	if err := unmarshal(&tc); err != nil {
		return err
	}
	*c = Compression(tc)
	return c.assertValid()
}

func (c Compression) assertValid() error {
	switch c {
	case "gzip":
	default:
		return ErrCompressionInvalid
	}
	return nil
}
