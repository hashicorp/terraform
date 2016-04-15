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
	"net/url"
)

type Url url.URL

func (u *Url) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return u.unmarshal(unmarshal)
}

func (u *Url) UnmarshalJSON(data []byte) error {
	return u.unmarshal(func(tu interface{}) error {
		return json.Unmarshal(data, tu)
	})
}

func (u Url) MarshalJSON() ([]byte, error) {
	return []byte(`"` + u.String() + `"`), nil
}

func (u *Url) unmarshal(unmarshal func(interface{}) error) error {
	var tu string
	if err := unmarshal(&tu); err != nil {
		return err
	}

	pu, err := url.Parse(tu)
	*u = Url(*pu)
	return err
}

func (u Url) String() string {
	tu := url.URL(u)
	return (&tu).String()
}
