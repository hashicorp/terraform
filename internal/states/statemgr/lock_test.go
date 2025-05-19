// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package statemgr

import (
	"testing"
)

func TestLockDisabled_impl(t *testing.T) {
	var _ Full = new(LockDisabled)
	var _ Locker = new(LockDisabled)
}
