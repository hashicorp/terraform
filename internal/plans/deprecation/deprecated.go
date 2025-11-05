// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package deprecation

import (
	"sync"
)

type Deprecated struct {
	// Must hold this lock when accessing all fields after this one.
	mu sync.Mutex
}

func NewDeprecated() *Deprecated {
	return &Deprecated{}
}
