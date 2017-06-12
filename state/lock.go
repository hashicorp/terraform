package state

import (
	"github.com/hashicorp/terraform/terraform"
)

// LockDisabled implements State and Locker but disables state locking.
// If State doesn't support locking, this is a no-op. This is useful for
// easily disabling locking of an existing state or for tests.
type LockDisabled struct {
	// We can't embed State directly since Go dislikes that a field is
	// State and State interface has a method State
	Inner State
}

func (s *LockDisabled) State() *terraform.State {
	return s.Inner.State()
}

func (s *LockDisabled) WriteState(v *terraform.State) error {
	return s.Inner.WriteState(v)
}

func (s *LockDisabled) RefreshState() error {
	return s.Inner.RefreshState()
}

func (s *LockDisabled) PersistState() error {
	return s.Inner.PersistState()
}

func (s *LockDisabled) Lock(info *LockInfo) (string, error) {
	return "", nil
}

func (s *LockDisabled) Unlock(id string) error {
	return nil
}

// Because LockDisabled is a wrapper for remote state need to recall wrapped object.
func (s *LockDisabled) WriteRecoveryLog(data []byte) error {
	if recoveryWriter, ok := s.Inner.(RecoveryLogWriter); ok {
		return recoveryWriter.WriteRecoveryLog(data)
	}
	return nil
}
func (s *LockDisabled) WriteLostResourceLog(data []byte) error {
	if recoveryWriter, ok := s.Inner.(RecoveryLogWriter); ok {
		return recoveryWriter.WriteLostResourceLog(data)
	}
	return nil
}

func (s *LockDisabled) DeleteRecoveryLog() error {
	if recoveryWriter, ok := s.Inner.(RecoveryLogWriter); ok {
		return recoveryWriter.DeleteRecoveryLog()
	}
	return nil
}

func (s *LockDisabled) ReadRecoveryLog() (map[string]Instance, error) {
	if recoveryReader, ok := s.Inner.(RecoveryLogReader); ok {
		return recoveryReader.ReadRecoveryLog()
	}
	return nil, nil
}
