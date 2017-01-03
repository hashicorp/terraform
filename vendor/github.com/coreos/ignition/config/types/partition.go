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
	"fmt"
	"regexp"

	"github.com/coreos/ignition/config/validate/report"
)

type Partition struct {
	Label    PartitionLabel     `json:"label,omitempty"`
	Number   int                `json:"number"`
	Size     PartitionDimension `json:"size"`
	Start    PartitionDimension `json:"start"`
	TypeGUID PartitionTypeGUID  `json:"typeGuid,omitempty"`
}

type PartitionLabel string

func (n PartitionLabel) Validate() report.Report {
	// http://en.wikipedia.org/wiki/GUID_Partition_Table#Partition_entries:
	// 56 (0x38) 	72 bytes 	Partition name (36 UTF-16LE code units)

	// XXX(vc): note GPT calls it a name, we're using label for consistency
	// with udev naming /dev/disk/by-partlabel/*.
	if len(string(n)) > 36 {
		return report.ReportFromError(fmt.Errorf("partition labels may not exceed 36 characters"), report.EntryError)
	}
	return report.Report{}
}

type PartitionDimension uint64

type PartitionTypeGUID string

func (d PartitionTypeGUID) Validate() report.Report {
	ok, err := regexp.MatchString("^(|[[:xdigit:]]{8}-[[:xdigit:]]{4}-[[:xdigit:]]{4}-[[:xdigit:]]{4}-[[:xdigit:]]{12})$", string(d))
	if err != nil {
		return report.ReportFromError(fmt.Errorf("error matching type-guid regexp: %v", err), report.EntryError)
	}
	if !ok {
		return report.ReportFromError(fmt.Errorf(`partition type-guid must have the form "01234567-89AB-CDEF-EDCB-A98765432101", got: %q`, string(d)), report.EntryError)
	}
	return report.Report{}
}
