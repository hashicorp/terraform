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
	"regexp"

	"github.com/alecthomas/units"
)

type Partition struct {
	Label    PartitionLabel     `json:"label,omitempty"    yaml:"label"`
	Number   int                `json:"number"             yaml:"number"`
	Size     PartitionDimension `json:"size"               yaml:"size"`
	Start    PartitionDimension `json:"start"              yaml:"start"`
	TypeGUID PartitionTypeGUID  `json:"typeGuid,omitempty" yaml:"type_guid"`
}

type PartitionLabel string

func (n *PartitionLabel) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return n.unmarshal(unmarshal)
}

func (n *PartitionLabel) UnmarshalJSON(data []byte) error {
	return n.unmarshal(func(tn interface{}) error {
		return json.Unmarshal(data, tn)
	})
}

type partitionLabel PartitionLabel

func (n *PartitionLabel) unmarshal(unmarshal func(interface{}) error) error {
	tn := partitionLabel(*n)
	if err := unmarshal(&tn); err != nil {
		return err
	}
	*n = PartitionLabel(tn)
	return n.assertValid()
}

func (n PartitionLabel) assertValid() error {
	// http://en.wikipedia.org/wiki/GUID_Partition_Table#Partition_entries:
	// 56 (0x38) 	72 bytes 	Partition name (36 UTF-16LE code units)

	// XXX(vc): note GPT calls it a name, we're using label for consistency
	// with udev naming /dev/disk/by-partlabel/*.
	if len(string(n)) > 36 {
		return fmt.Errorf("partition labels may not exceed 36 characters")
	}
	return nil
}

type PartitionDimension uint64

func (n *PartitionDimension) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// In YAML we allow human-readable dimensions like GiB/TiB etc.
	var str string
	if err := unmarshal(&str); err != nil {
		return err
	}

	b2b, err := units.ParseBase2Bytes(str) // TODO(vc): replace the units package
	if err != nil {
		return err
	}
	if b2b < 0 {
		return fmt.Errorf("negative value inappropriate: %q", str)
	}

	// Translate bytes into sectors
	sectors := (b2b / 512)
	if b2b%512 != 0 {
		sectors++
	}
	*n = PartitionDimension(uint64(sectors))
	return nil
}

func (n *PartitionDimension) UnmarshalJSON(data []byte) error {
	// In JSON we expect plain integral sectors.
	// The YAML->JSON conversion is responsible for normalizing human units to sectors.
	var pd uint64
	if err := json.Unmarshal(data, &pd); err != nil {
		return err
	}
	*n = PartitionDimension(pd)
	return nil
}

type PartitionTypeGUID string

func (d *PartitionTypeGUID) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return d.unmarshal(unmarshal)
}

func (d *PartitionTypeGUID) UnmarshalJSON(data []byte) error {
	return d.unmarshal(func(td interface{}) error {
		return json.Unmarshal(data, td)
	})
}

type partitionTypeGUID PartitionTypeGUID

func (d *PartitionTypeGUID) unmarshal(unmarshal func(interface{}) error) error {
	td := partitionTypeGUID(*d)
	if err := unmarshal(&td); err != nil {
		return err
	}
	*d = PartitionTypeGUID(td)
	return d.assertValid()
}

func (d PartitionTypeGUID) assertValid() error {
	ok, err := regexp.MatchString("[[:xdigit:]]{8}-[[:xdigit:]]{4}-[[:xdigit:]]{4}-[[:xdigit:]]{4}-[[:xdigit:]]{12}", string(d))
	if err != nil {
		return fmt.Errorf("error matching type-guid regexp: %v", err)
	}
	if !ok {
		return fmt.Errorf(`partition type-guid must have the form "01234567-89AB-CDEF-EDCB-A98765432101", got: %q`, string(d))
	}
	return nil
}
