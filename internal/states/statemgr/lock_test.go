// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package statemgr

import (
	"testing"
)

func TestLockDisabled_impl(t *testing.T) {
	var _ Full = new(LockDisabled)
	var _ Locker = new(LockDisabled)
}
