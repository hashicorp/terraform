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
	"path/filepath"
)

var (
	ErrPathRelative = errors.New("path not absolute")
)

type Path string

func (p *Path) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return p.unmarshal(unmarshal)
}

func (p *Path) UnmarshalJSON(data []byte) error {
	return p.unmarshal(func(td interface{}) error {
		return json.Unmarshal(data, td)
	})
}

func (p Path) MarshalJSON() ([]byte, error) {
	return []byte(`"` + string(p) + `"`), nil
}

type path Path

func (p *Path) unmarshal(unmarshal func(interface{}) error) error {
	td := path(*p)
	if err := unmarshal(&td); err != nil {
		return err
	}
	*p = Path(td)
	return p.assertValid()
}

func (p Path) assertValid() error {
	if !filepath.IsAbs(string(p)) {
		return ErrPathRelative
	}
	return nil
}
