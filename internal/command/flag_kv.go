// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"fmt"
	"strings"
)

// FlagStringKV is a flag.Value implementation for parsing user variables
// from the command-line in the format of '-var key=value', where value is
// only ever a primitive.
type FlagStringKV map[string]string

func (v *FlagStringKV) String() string {
	return ""
}

func (v *FlagStringKV) Set(raw string) error {
	idx := strings.Index(raw, "=")
	if idx == -1 {
		return fmt.Errorf("No '=' value in arg: %s", raw)
	}

	if *v == nil {
		*v = make(map[string]string)
	}

	key, value := raw[0:idx], raw[idx+1:]
	(*v)[key] = value
	return nil
}
