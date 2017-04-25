package state

import (
	"testing"
)

func TestLockDisabled_impl(t *testing.T) {
	var _ State = new(LockDisabled)
	var _ Locker = new(LockDisabled)
}
