package checks

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
)

func initialStatuses(cfg *configs.Config) addrs.Map[addrs.ConfigCheckable, *configCheckableState] {
	ret := addrs.MakeMap[addrs.ConfigCheckable, *configCheckableState]()
	if cfg == nil {
		// This should not happen in normal use, but can arise in some
		// unit tests that are not working with a full configuration and
		// don't care about checks.
		return ret
	}

	collectInitialStatuses(ret, cfg)

	return ret
}

func collectInitialStatuses(into addrs.Map[addrs.ConfigCheckable, *configCheckableState], cfg *configs.Config) {
	moduleAddr := cfg.Path

	for _, rc := range cfg.Module.ManagedResources {
		addr := rc.Addr().InModule(moduleAddr)
		collectInitialStatusForResource(into, addr, rc)
	}
	for _, rc := range cfg.Module.DataResources {
		addr := rc.Addr().InModule(moduleAddr)
		collectInitialStatusForResource(into, addr, rc)
	}

	for _, oc := range cfg.Module.Outputs {
		addr := oc.Addr().InModule(moduleAddr)

		ct := len(oc.Preconditions)
		if ct == 0 {
			// We just ignore output values that don't declare any checks.
			continue
		}

		st := &configCheckableState{}

		st.checkTypes = map[addrs.CheckType]int{
			addrs.OutputPrecondition: ct,
		}

		into.Put(addr, st)
	}

	for _, test := range cfg.Module.SmokeTests {
		addr := test.Addr().InModule(moduleAddr)
		// All smoke_test blocks are automatically checkable objects, because
		// they exist only to represent checks.
		st := &configCheckableState{
			checkTypes: map[addrs.CheckType]int{
				addrs.SmokeTestPrecondition:  len(test.Preconditions),
				addrs.SmokeTestPostcondition: len(test.Postconditions),
				addrs.SmokeTestDataResource:  len(test.DataResources),
			},
		}
		into.Put(addr, st)
	}

	// Must also visit child modules to collect everything
	for _, child := range cfg.Children {
		collectInitialStatuses(into, child)
	}
}

func collectInitialStatusForResource(into addrs.Map[addrs.ConfigCheckable, *configCheckableState], addr addrs.ConfigResource, rc *configs.Resource) {
	if (len(rc.Preconditions) + len(rc.Postconditions)) == 0 {
		// Don't bother with any resource that doesn't have at least
		// one condition.
		return
	}

	st := &configCheckableState{
		checkTypes: make(map[addrs.CheckType]int),
	}

	if ct := len(rc.Preconditions); ct > 0 {
		st.checkTypes[addrs.ResourcePrecondition] = ct
	}
	if ct := len(rc.Postconditions); ct > 0 {
		st.checkTypes[addrs.ResourcePostcondition] = ct
	}

	into.Put(addr, st)
}
