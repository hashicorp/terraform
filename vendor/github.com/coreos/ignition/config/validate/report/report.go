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

package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
)

type Report struct {
	Entries []Entry
}

func (into *Report) Merge(from Report) {
	into.Entries = append(into.Entries, from.Entries...)
}

func ReportFromError(err error, severity entryKind) Report {
	if err == nil {
		return Report{}
	}
	return Report{
		Entries: []Entry{
			{
				Kind:    severity,
				Message: err.Error(),
			},
		},
	}
}

// Sort sorts the entries by line number, then column number
func (r *Report) Sort() {
	sort.Sort(entries(r.Entries))
}

type entries []Entry

func (e entries) Len() int {
	return len(e)
}

func (e entries) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func (e entries) Less(i, j int) bool {
	if e[i].Line != e[j].Line {
		return e[i].Line < e[j].Line
	}
	return e[i].Column < e[j].Column
}

const (
	EntryError entryKind = iota
	EntryWarning
	EntryInfo
	EntryDeprecated
)

// AddPosition updates all the entries with Line equal to 0 and sets the Line/Column fields to line/column. This is useful for
// when a type has a custom unmarshaller and thus can't determine an exact offset of the error with the type. In this case
// the offset for the entire chunk of json that got unmarshalled to the type can be used instead, which is still pretty good.
func (r *Report) AddPosition(line, col int, highlight string) {
	for i, e := range r.Entries {
		if e.Line == 0 {
			r.Entries[i].Line = line
			r.Entries[i].Column = col
			r.Entries[i].Highlight = highlight
		}
	}
}

func (r *Report) Add(e Entry) {
	r.Entries = append(r.Entries, e)
}

func (r Report) String() string {
	var errs bytes.Buffer
	for i, entry := range r.Entries {
		if i != 0 {
			// Only add line breaks on multiline reports
			errs.WriteString("\n")
		}
		errs.WriteString(entry.String())
	}
	return errs.String()
}

// IsFatal returns if there were any errors that make the config invalid
func (r Report) IsFatal() bool {
	for _, entry := range r.Entries {
		if entry.Kind == EntryError {
			return true
		}
	}
	return false
}

// IsDeprecated returns if the report has deprecations
func (r Report) IsDeprecated() bool {
	for _, entry := range r.Entries {
		if entry.Kind == EntryDeprecated {
			return true
		}
	}
	return false
}

type Entry struct {
	Kind      entryKind `json:"kind"`
	Message   string    `json:"message"`
	Line      int       `json:"line,omitempty"`
	Column    int       `json:"column,omitempty"`
	Highlight string    `json:"-"`
}

func (e Entry) String() string {
	if e.Line != 0 {
		return fmt.Sprintf("%s at line %d, column %d\n%s%v", e.Kind.String(), e.Line, e.Column, e.Highlight, e.Message)
	}
	return fmt.Sprintf("%s: %v", e.Kind.String(), e.Message)
}

type entryKind int

func (e entryKind) String() string {
	switch e {
	case EntryError:
		return "error"
	case EntryWarning:
		return "warning"
	case EntryInfo:
		return "info"
	case EntryDeprecated:
		return "deprecated"
	default:
		return "unknown error"
	}
}

func (e entryKind) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.String())
}
