package addrs

import (
	"fmt"
)

type SmokeTest struct {
	Name string
}

func (st SmokeTest) String() string {
	return fmt.Sprintf("smoke_test.%s", st.Name)
}

func (st SmokeTest) InModule(modAddr Module) ConfigSmokeTest {
	return ConfigSmokeTest{
		Module:    modAddr,
		SmokeTest: st,
	}
}

func (st SmokeTest) Absolute(modAddr ModuleInstance) AbsSmokeTest {
	return AbsSmokeTest{
		Module:    modAddr,
		SmokeTest: st,
	}
}

type ConfigSmokeTest struct {
	Module    Module
	SmokeTest SmokeTest
}

var _ ConfigCheckable = ConfigSmokeTest{}

func (st ConfigSmokeTest) String() string {
	if len(st.Module) == 0 {
		return st.SmokeTest.String()
	}
	return st.Module.String() + "." + st.SmokeTest.String()
}

func (st ConfigSmokeTest) UniqueKey() UniqueKey {
	return configSmokeTestUniqueKey(st.String())
}

func (st ConfigSmokeTest) configCheckableSigil() {}

func (st ConfigSmokeTest) CheckableKind() CheckableKind {
	return CheckableSmokeTest
}

type AbsSmokeTest struct {
	Module    ModuleInstance
	SmokeTest SmokeTest
}

var _ Checkable = AbsSmokeTest{}

func (st AbsSmokeTest) String() string {
	if len(st.Module) == 0 {
		return st.SmokeTest.String()
	}
	return st.Module.String() + "." + st.SmokeTest.String()
}

func (st AbsSmokeTest) UniqueKey() UniqueKey {
	return absSmokeTestUniqueKey(st.String())
}

func (st AbsSmokeTest) checkableSigil() {}

func (st AbsSmokeTest) CheckableKind() CheckableKind {
	return CheckableSmokeTest
}

func (st AbsSmokeTest) ConfigCheckable() ConfigCheckable {
	return ConfigSmokeTest{
		Module:    st.Module.Module(),
		SmokeTest: st.SmokeTest,
	}
}

func (st AbsSmokeTest) Check(typ CheckType, idx int) Check {
	return Check{
		Container: st,
		Type:      typ,
		Index:     idx,
	}
}

type configSmokeTestUniqueKey string

func (k configSmokeTestUniqueKey) uniqueKeySigil() {}

type absSmokeTestUniqueKey string

func (k absSmokeTestUniqueKey) uniqueKeySigil() {}
