// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package addrs

import "fmt"

// Run is the address of a run block within a testing file.
//
// Run blocks are only accessible from within the same testing file, and they
// do not support any meta-arguments like "count" or "for_each". So this address
// uniquely describes a run block from within a single testing file.
type Run struct {
	referenceable
	Name string
}

func (r Run) String() string {
	return fmt.Sprintf("run.%s", r.Name)
}

func (r Run) Equal(run Run) bool {
	return r.Name == run.Name
}

func (r Run) UniqueKey() UniqueKey {
	return r // A Run is its own UniqueKey
}

func (r Run) uniqueKeySigil() {}
