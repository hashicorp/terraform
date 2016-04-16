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
	"github.com/coreos/go-semver/semver"
)

var (
	MaxVersion = semver.Version{
		Major: 2,
		Minor: 0,
	}
)

type Config struct {
	Ignition Ignition `json:"ignition"           yaml:"ignition"`
	Storage  Storage  `json:"storage,omitempty"  yaml:"storage"`
	Systemd  Systemd  `json:"systemd,omitempty"  yaml:"systemd"`
	Networkd Networkd `json:"networkd,omitempty" yaml:"networkd"`
	Passwd   Passwd   `json:"passwd,omitempty"   yaml:"passwd"`
}
