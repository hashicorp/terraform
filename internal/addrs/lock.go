// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"fmt"
)

// Lock is the address of a lock.
type Lock struct {
	referenceable
	Name string
}

func (v Lock) String() string {
	return "lock." + v.Name
}

func (v Lock) UniqueKey() UniqueKey {
	return v // A Lock is its own UniqueKey
}

func (v Lock) uniqueKeySigil() {}

// Absolute converts the receiver into an absolute address within the given
// module instance.
func (v Lock) Absolute(m ModuleInstance) AbsLock {
	return AbsLock{
		Module: m,
		Lock:   v,
	}
}

// AbsLock is the absolute address of a lock within a module instance.
type AbsLock struct {
	Module ModuleInstance
	Lock   Lock
}

// Lock returns the absolute address of a lock of the given
// name within the receiving module instance.
func (m ModuleInstance) Lock(name string) AbsLock {
	return AbsLock{
		Module: m,
		Lock: Lock{
			Name: name,
		},
	}
}

func (v AbsLock) String() string {
	if v.Module.Equal(RootModuleInstance) {
		return v.Lock.String()
	}
	return fmt.Sprintf("%s.%s", v.Module.String(), v.Lock.String())
}

func (v AbsLock) UniqueKey() UniqueKey {
	return absLockKey{
		moduleKey: v.Module.UniqueKey(),
		valueKey:  v.Lock.UniqueKey(),
	}
}

type absLockKey struct {
	moduleKey UniqueKey
	valueKey  UniqueKey
}

// uniqueKeySigil implements UniqueKey.
func (absLockKey) uniqueKeySigil() {}
