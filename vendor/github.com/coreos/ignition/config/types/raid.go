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
	"fmt"
)

type Raid struct {
	Name    string `json:"name"              yaml:"name"`
	Level   string `json:"level"             yaml:"level"`
	Devices []Path `json:"devices,omitempty" yaml:"devices"`
	Spares  int    `json:"spares,omitempty"  yaml:"spares"`
}

func (n *Raid) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return n.unmarshal(unmarshal)
}

func (n *Raid) UnmarshalJSON(data []byte) error {
	return n.unmarshal(func(tn interface{}) error {
		return json.Unmarshal(data, tn)
	})
}

type raid Raid

func (n *Raid) unmarshal(unmarshal func(interface{}) error) error {
	tn := raid(*n)
	if err := unmarshal(&tn); err != nil {
		return err
	}
	*n = Raid(tn)
	return n.assertValid()
}

func (n Raid) assertValid() error {
	switch n.Level {
	case "linear", "raid0", "0", "stripe":
		if n.Spares != 0 {
			return fmt.Errorf("spares unsupported for %q arrays", n.Level)
		}
	case "raid1", "1", "mirror":
	case "raid4", "4":
	case "raid5", "5":
	case "raid6", "6":
	case "raid10", "10":
	default:
		return fmt.Errorf("unrecognized raid level: %q", n.Level)
	}
	return nil
}
