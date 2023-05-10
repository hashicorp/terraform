// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package addrs

import "fmt"

// Check is the address of a check block within a module.
//
// For now, checks do not support meta arguments such as "count" or "for_each"
// so this address uniquely describes a single check within a module.
type Check struct {
	referenceable
	Name string
}

func (c Check) String() string {
	return fmt.Sprintf("check.%s", c.Name)
}

// InModule returns a ConfigCheck from the receiver and the given module
// address.
func (c Check) InModule(modAddr Module) ConfigCheck {
	return ConfigCheck{
		Module: modAddr,
		Check:  c,
	}
}

// Absolute returns an AbsCheck from the receiver and the given module instance
// address.
func (c Check) Absolute(modAddr ModuleInstance) AbsCheck {
	return AbsCheck{
		Module: modAddr,
		Check:  c,
	}
}

func (c Check) Equal(o Check) bool {
	return c.Name == o.Name
}

func (c Check) UniqueKey() UniqueKey {
	return c // A Check is its own UniqueKey
}

func (c Check) uniqueKeySigil() {}

// ConfigCheck is an address for a check block within a configuration.
//
// This contains a Check address and a Module address, meaning this describes
// a check block within the entire configuration.
type ConfigCheck struct {
	Module Module
	Check  Check
}

var _ ConfigCheckable = ConfigCheck{}

func (c ConfigCheck) UniqueKey() UniqueKey {
	return configCheckUniqueKey(c.String())
}

func (c ConfigCheck) configCheckableSigil() {}

func (c ConfigCheck) CheckableKind() CheckableKind {
	return CheckableCheck
}

func (c ConfigCheck) String() string {
	if len(c.Module) == 0 {
		return c.Check.String()
	}
	return fmt.Sprintf("%s.%s", c.Module, c.Check)
}

// AbsCheck is an absolute address for a check block under a given module path.
//
// This contains an actual ModuleInstance address (compared to the Module within
// a ConfigCheck), meaning this uniquely describes a check block within the
// entire configuration after any "count" or "foreach" meta arguments have been
// evaluated on the containing module.
type AbsCheck struct {
	Module ModuleInstance
	Check  Check
}

var _ Checkable = AbsCheck{}

func (c AbsCheck) UniqueKey() UniqueKey {
	return absCheckUniqueKey(c.String())
}

func (c AbsCheck) checkableSigil() {}

// CheckRule returns an address for a given rule type within the check block.
//
// There will be at most one CheckDataResource rule within a check block (with
// an index of 0). There will be at least one, but potentially many,
// CheckAssertion rules within a check block.
func (c AbsCheck) CheckRule(typ CheckRuleType, i int) CheckRule {
	return CheckRule{
		Container: c,
		Type:      typ,
		Index:     i,
	}
}

// ConfigCheckable returns the ConfigCheck address for this absolute reference.
func (c AbsCheck) ConfigCheckable() ConfigCheckable {
	return ConfigCheck{
		Module: c.Module.Module(),
		Check:  c.Check,
	}
}

func (c AbsCheck) CheckableKind() CheckableKind {
	return CheckableCheck
}

func (c AbsCheck) String() string {
	if len(c.Module) == 0 {
		return c.Check.String()
	}
	return fmt.Sprintf("%s.%s", c.Module, c.Check)
}

type configCheckUniqueKey string

func (k configCheckUniqueKey) uniqueKeySigil() {}

type absCheckUniqueKey string

func (k absCheckUniqueKey) uniqueKeySigil() {}
