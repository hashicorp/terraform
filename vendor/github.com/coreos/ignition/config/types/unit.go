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

type SystemdUnit struct {
	Name     SystemdUnitName     `json:"name,omitempty"     yaml:"name"`
	Enable   bool                `json:"enable,omitempty"   yaml:"enable"`
	Mask     bool                `json:"mask,omitempty"     yaml:"mask"`
	Contents string              `json:"contents,omitempty" yaml:"contents"`
	DropIns  []SystemdUnitDropIn `json:"dropins,omitempty"  yaml:"dropins"`
}

type SystemdUnitDropIn struct {
	Name     SystemdUnitDropInName `json:"name,omitempty"     yaml:"name"`
	Contents string                `json:"contents,omitempty" yaml:"contents"`
}

type SystemdUnitName string

func (n *SystemdUnitName) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return n.unmarshal(unmarshal)
}

func (n *SystemdUnitName) UnmarshalJSON(data []byte) error {
	return n.unmarshal(func(tn interface{}) error {
		return json.Unmarshal(data, tn)
	})
}

type systemdUnitName SystemdUnitName

func (n *SystemdUnitName) unmarshal(unmarshal func(interface{}) error) error {
	tn := systemdUnitName(*n)
	if err := unmarshal(&tn); err != nil {
		return err
	}
	*n = SystemdUnitName(tn)
	return n.assertValid()
}

func (n SystemdUnitName) assertValid() error {
	switch filepath.Ext(string(n)) {
	case ".service", ".socket", ".device", ".mount", ".automount", ".swap", ".target", ".path", ".timer", ".snapshot", ".slice", ".scope":
		return nil
	default:
		return errors.New("invalid systemd unit extension")
	}
}

type SystemdUnitDropInName string

func (n *SystemdUnitDropInName) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return n.unmarshal(unmarshal)
}

func (n *SystemdUnitDropInName) UnmarshalJSON(data []byte) error {
	return n.unmarshal(func(tn interface{}) error {
		return json.Unmarshal(data, tn)
	})
}

type systemdUnitDropInName SystemdUnitDropInName

func (n *SystemdUnitDropInName) unmarshal(unmarshal func(interface{}) error) error {
	tn := systemdUnitDropInName(*n)
	if err := unmarshal(&tn); err != nil {
		return err
	}
	*n = SystemdUnitDropInName(tn)
	return n.assertValid()
}

func (n SystemdUnitDropInName) assertValid() error {
	switch filepath.Ext(string(n)) {
	case ".conf":
		return nil
	default:
		return errors.New("invalid systemd unit drop-in extension")
	}
}

type NetworkdUnit struct {
	Name     NetworkdUnitName `json:"name,omitempty"     yaml:"name"`
	Contents string           `json:"contents,omitempty" yaml:"contents"`
}

type NetworkdUnitName string

func (n *NetworkdUnitName) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return n.unmarshal(unmarshal)
}

func (n *NetworkdUnitName) UnmarshalJSON(data []byte) error {
	return n.unmarshal(func(tn interface{}) error {
		return json.Unmarshal(data, tn)
	})
}

type networkdUnitName NetworkdUnitName

func (n *NetworkdUnitName) unmarshal(unmarshal func(interface{}) error) error {
	tn := networkdUnitName(*n)
	if err := unmarshal(&tn); err != nil {
		return err
	}
	*n = NetworkdUnitName(tn)
	return n.assertValid()
}

func (n NetworkdUnitName) assertValid() error {
	switch filepath.Ext(string(n)) {
	case ".link", ".netdev", ".network":
		return nil
	default:
		return errors.New("invalid networkd unit extension")
	}
}
