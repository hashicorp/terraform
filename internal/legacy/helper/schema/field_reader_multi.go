// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package schema

import (
	"fmt"
)

// MultiLevelFieldReader reads from other field readers,
// merging their results along the way in a specific order. You can specify
// "levels" and name them in order to read only an exact level or up to
// a specific level.
//
// This is useful for saying things such as "read the field from the state
// and config and merge them" or "read the latest value of the field".
type MultiLevelFieldReader struct {
	Readers map[string]FieldReader
	Levels  []string
}

func (r *MultiLevelFieldReader) ReadField(address []string) (FieldReadResult, error) {
	return r.ReadFieldMerge(address, r.Levels[len(r.Levels)-1])
}

func (r *MultiLevelFieldReader) ReadFieldExact(
	address []string, level string) (FieldReadResult, error) {
	reader, ok := r.Readers[level]
	if !ok {
		return FieldReadResult{}, fmt.Errorf(
			"Unknown reader level: %s", level)
	}

	result, err := reader.ReadField(address)
	if err != nil {
		return FieldReadResult{}, fmt.Errorf(
			"Error reading level %s: %s", level, err)
	}

	return result, nil
}

func (r *MultiLevelFieldReader) ReadFieldMerge(
	address []string, level string) (FieldReadResult, error) {
	var result FieldReadResult
	for _, l := range r.Levels {
		if r, ok := r.Readers[l]; ok {
			out, err := r.ReadField(address)
			if err != nil {
				return FieldReadResult{}, fmt.Errorf(
					"Error reading level %s: %s", l, err)
			}

			// TODO: computed
			if out.Exists {
				result = out
			}
		}

		if l == level {
			break
		}
	}

	return result, nil
}
