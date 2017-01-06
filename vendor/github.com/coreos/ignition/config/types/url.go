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
	"net/url"

	"github.com/coreos/ignition/config/validate/report"
	"github.com/vincent-petithory/dataurl"
)

var (
	ErrInvalidScheme = errors.New("invalid url scheme")
)

type Url url.URL

func (u *Url) UnmarshalJSON(data []byte) error {
	var tu string
	if err := json.Unmarshal(data, &tu); err != nil {
		return err
	}

	pu, err := url.Parse(tu)
	if err != nil {
		return err
	}

	*u = Url(*pu)
	return nil
}

func (u Url) MarshalJSON() ([]byte, error) {
	return []byte(`"` + u.String() + `"`), nil
}

func (u Url) String() string {
	tu := url.URL(u)
	return (&tu).String()
}

func (u Url) Validate() report.Report {
	// Empty url is valid, indicates an empty file
	if u.String() == "" {
		return report.Report{}
	}
	switch url.URL(u).Scheme {
	case "http", "https", "oem":
		return report.Report{}
	case "data":
		if _, err := dataurl.DecodeString(u.String()); err != nil {
			return report.ReportFromError(err, report.EntryError)
		}
		return report.Report{}
	default:
		return report.ReportFromError(ErrInvalidScheme, report.EntryError)
	}
}
