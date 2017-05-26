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
	"errors"
	"os"

	"github.com/coreos/ignition/config/validate/report"
)

var (
	ErrFileIllegalMode = errors.New("illegal file mode")
	ErrNoFilesystem    = errors.New("no filesystem specified")
)

type File struct {
	Filesystem string       `json:"filesystem,omitempty"`
	Path       Path         `json:"path,omitempty"`
	Contents   FileContents `json:"contents,omitempty"`
	Mode       FileMode     `json:"mode,omitempty"`
	User       FileUser     `json:"user,omitempty"`
	Group      FileGroup    `json:"group,omitempty"`
}

func (f File) Validate() report.Report {
	if f.Filesystem == "" {
		return report.ReportFromError(ErrNoFilesystem, report.EntryError)
	}
	return report.Report{}
}

type FileUser struct {
	Id int `json:"id,omitempty"`
}

type FileGroup struct {
	Id int `json:"id,omitempty"`
}

type FileContents struct {
	Compression  Compression  `json:"compression,omitempty"`
	Source       Url          `json:"source,omitempty"`
	Verification Verification `json:"verification,omitempty"`
}

type FileMode os.FileMode

func (m FileMode) Validate() report.Report {
	if (m &^ 07777) != 0 {
		return report.ReportFromError(ErrFileIllegalMode, report.EntryError)
	}
	return report.Report{}
}
