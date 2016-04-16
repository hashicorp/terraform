// Copyright 2015 CoreOS, Inc.
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

	"github.com/coreos/go-semver/semver"
)

var (
	ErrOldVersion = errors.New("incorrect config version (too old)")
	ErrNewVersion = errors.New("incorrect config version (too new)")
)

type Ignition struct {
	Version IgnitionVersion `json:"version,omitempty" yaml:"version" merge:"old"`
	Config  IgnitionConfig  `json:"config,omitempty"  yaml:"config"  merge:"new"`
}

type IgnitionConfig struct {
	Append  []ConfigReference `json:"append,omitempty"  yaml:"append"`
	Replace *ConfigReference  `json:"replace,omitempty" yaml:"replace"`
}

type ConfigReference struct {
	Source       Url          `json:"source,omitempty"       yaml:"source"`
	Verification Verification `json:"verification,omitempty" yaml:"verification"`
}

type IgnitionVersion semver.Version

func (v *IgnitionVersion) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return v.unmarshal(unmarshal)
}

func (v *IgnitionVersion) UnmarshalJSON(data []byte) error {
	return v.unmarshal(func(tv interface{}) error {
		return json.Unmarshal(data, tv)
	})
}

func (v IgnitionVersion) MarshalJSON() ([]byte, error) {
	return semver.Version(v).MarshalJSON()
}

func (v *IgnitionVersion) unmarshal(unmarshal func(interface{}) error) error {
	tv := semver.Version(*v)
	if err := unmarshal(&tv); err != nil {
		return err
	}
	*v = IgnitionVersion(tv)
	return nil
}

func (v IgnitionVersion) AssertValid() error {
	if MaxVersion.Major > v.Major {
		return ErrOldVersion
	}
	if MaxVersion.LessThan(semver.Version(v)) {
		return ErrNewVersion
	}
	return nil
}
