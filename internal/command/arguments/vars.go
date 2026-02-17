// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

// Vars describes arguments which specify non-default variable values. This
// interface is unfortunately obscure, because the order of the CLI arguments
// determines the final value of the gathered variables. In future it might be
// desirable for the arguments package to handle the gathering of variables
// directly, returning a map of variable values.
type Vars struct {
	vars     *FlagNameValueSlice
	varFiles *FlagNameValueSlice
}

func (v *Vars) All() []FlagNameValue {
	if v.vars == nil {
		return nil
	}
	return v.vars.AllItems()
}

func (v *Vars) Empty() bool {
	if v.vars == nil {
		return true
	}
	return v.vars.Empty()
}
