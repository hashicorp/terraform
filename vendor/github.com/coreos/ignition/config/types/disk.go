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

type Disk struct {
	Device     Path        `json:"device,omitempty"     yaml:"device"`
	WipeTable  bool        `json:"wipeTable,omitempty"  yaml:"wipe_table"`
	Partitions []Partition `json:"partitions,omitempty" yaml:"partitions"`
}

func (n *Disk) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := n.unmarshal(unmarshal); err != nil {
		return err
	}
	if err := n.preparePartitions(); err != nil {
		return err
	}
	return n.assertValid()
}

func (n *Disk) UnmarshalJSON(data []byte) error {
	err := n.unmarshal(func(tn interface{}) error {
		return json.Unmarshal(data, tn)
	})
	if err != nil {
		return err
	}
	return n.assertValid()
}

type disk Disk

func (n *Disk) unmarshal(unmarshal func(interface{}) error) error {
	tn := disk(*n)
	if err := unmarshal(&tn); err != nil {
		return err
	}
	*n = Disk(tn)
	return nil
}

func (n Disk) assertValid() error {
	// This applies to YAML (post-prepare) and JSON unmarshals equally:
	if len(n.Device) == 0 {
		return fmt.Errorf("disk device is required")
	}
	if n.partitionNumbersCollide() {
		return fmt.Errorf("disk %q: partition numbers collide", n.Device)
	}
	if n.partitionsOverlap() {
		return fmt.Errorf("disk %q: partitions overlap", n.Device)
	}
	if n.partitionsMisaligned() {
		return fmt.Errorf("disk %q: partitions misaligned", n.Device)
	}
	// Disks which get to this point will likely succeed in sgdisk
	return nil
}

// partitionNumbersCollide returns true if partition numbers in n.Partitions are not unique.
func (n Disk) partitionNumbersCollide() bool {
	m := map[int][]Partition{}
	for _, p := range n.Partitions {
		m[p.Number] = append(m[p.Number], p)
	}
	for _, n := range m {
		if len(n) > 1 {
			// TODO(vc): return information describing the collision for logging
			return true
		}
	}
	return false
}

// end returns the last sector of a partition.
func (p Partition) end() PartitionDimension {
	if p.Size == 0 {
		// a size of 0 means "fill available", just return the start as the end for those.
		return p.Start
	}
	return p.Start + p.Size - 1
}

// partitionsOverlap returns true if any explicitly dimensioned partitions overlap
func (n Disk) partitionsOverlap() bool {
	for _, p := range n.Partitions {
		// Starts of 0 are placed by sgdisk into the "largest available block" at that time.
		// We aren't going to check those for overlap since we don't have the disk geometry.
		if p.Start == 0 {
			continue
		}

		for _, o := range n.Partitions {
			if p == o || o.Start == 0 {
				continue
			}

			// is p.Start within o?
			if p.Start >= o.Start && p.Start <= o.end() {
				return true
			}

			// is p.end() within o?
			if p.end() >= o.Start && p.end() <= o.end() {
				return true
			}

			// do p.Start and p.end() straddle o?
			if p.Start < o.Start && p.end() > o.end() {
				return true
			}
		}
	}
	return false
}

// partitionsMisaligned returns true if any of the partitions don't start on a 2048-sector (1MiB) boundary.
func (n Disk) partitionsMisaligned() bool {
	for _, p := range n.Partitions {
		if (p.Start & (2048 - 1)) != 0 {
			return true
		}
	}
	return false
}

// preparePartitions performs some checks and potentially adjusts the partitions for alignment.
// This is only invoked when unmarshalling YAML, since there we parse human-friendly units.
func (n *Disk) preparePartitions() error {
	// On the YAML side we accept human-friendly units which may require massaging.

	// align partition starts
	for i := range n.Partitions {
		// skip automatically placed partitions
		if n.Partitions[i].Start == 0 {
			continue
		}

		// keep partitions out of the first 2048 sectors
		if n.Partitions[i].Start < 2048 {
			n.Partitions[i].Start = 2048
		}

		// toss the bottom 11 bits
		n.Partitions[i].Start &= ^PartitionDimension(2048 - 1)
	}

	// TODO(vc): may be interesting to do something about potentially overlapping partitions
	return nil
}
