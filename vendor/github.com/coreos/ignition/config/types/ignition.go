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

	"github.com/coreos/ignition/config/validate/report"
)

var (
	ErrOldVersion = errors.New("incorrect config version (too old)")
	ErrNewVersion = errors.New("incorrect config version (too new)")
)

type Ignition struct {
	Version IgnitionVersion `json:"version,omitempty" merge:"old"`
	Config  IgnitionConfig  `json:"config,omitempty"  merge:"new"`
}

type IgnitionConfig struct {
	Append  []ConfigReference `json:"append,omitempty"`
	Replace *ConfigReference  `json:"replace,omitempty"`
}

type ConfigReference struct {
	Source       Url          `json:"source,omitempty"`
	Verification Verification `json:"verification,omitempty"`
}

type IgnitionVersion semver.Version

func (v *IgnitionVersion) UnmarshalJSON(data []byte) error {
	tv := semver.Version(*v)
	if err := json.Unmarshal(data, &tv); err != nil {
		return err
	}
	*v = IgnitionVersion(tv)
	return nil
}

func (v IgnitionVersion) MarshalJSON() ([]byte, error) {
	return semver.Version(v).MarshalJSON()
}

func (v IgnitionVersion) Validate() report.Report {
	if MaxVersion.Major > v.Major {
		return report.ReportFromError(ErrOldVersion, report.EntryError)
	}
	if MaxVersion.LessThan(semver.Version(v)) {
		return report.ReportFromError(ErrNewVersion, report.EntryError)
	}
	return report.Report{}
}
