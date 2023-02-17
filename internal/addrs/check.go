package addrs

import "fmt"

type Check struct {
	referenceable
	Name string
}

func (c Check) String() string {
	return fmt.Sprintf("check.%s", c.Name)
}

func (c Check) InModule(modAddr Module) ConfigCheck {
	return ConfigCheck{
		Module: modAddr,
		Check:  c,
	}
}

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

type AbsCheck struct {
	Module ModuleInstance
	Check  Check
}

var _ Checkable = AbsCheck{}

func (c AbsCheck) UniqueKey() UniqueKey {
	return absCheckUniqueKey(c.String())
}

func (c AbsCheck) checkableSigil() {}

func (c AbsCheck) CheckRule(typ CheckRuleType, i int) CheckRule {
	return CheckRule{
		Container: c,
		Type:      typ,
		Index:     i,
	}
}

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
