package statemgr

import (
	"testing"
)

func TestLockDisabled_impl(t *testing.T) {
	var _ Full = new(LockDisabled)
	var _ Locker = new(LockDisabled)
}
