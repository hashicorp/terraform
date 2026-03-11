// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package statefile

import (
	"fmt"
	"log"
)

// upgradeStateV1ToV2 is used to upgrade a V1 state representation
// into a V2 state representation
func upgradeStateV1ToV2(old *stateV1) (*stateV2, error) {
	log.Printf("[TRACE] statefile.Read: upgrading format from v1 to v2")
	if old == nil {
		return nil, nil
	}

	remote, err := old.Remote.upgradeToV2()
	if err != nil {
		return nil, fmt.Errorf("Error upgrading State V1: %v", err)
	}

	modules := make([]*moduleStateV2, len(old.Modules))
	for i, module := range old.Modules {
		upgraded, err := module.upgradeToV2()
		if err != nil {
			return nil, fmt.Errorf("Error upgrading State V1: %v", err)
		}
		modules[i] = upgraded
	}
	if len(modules) == 0 {
		modules = nil
	}

	newState := &stateV2{
		Version: 2,
		Serial:  old.Serial,
		Remote:  remote,
		Modules: modules,
	}

	return newState, nil
}

func (old *remoteStateV1) upgradeToV2() (*remoteStateV2, error) {
	if old == nil {
		return nil, nil
	}

	return &remoteStateV2{
		Type:   old.Type,
		Config: shallowCopyMap(old.Config),
	}, nil
}

func (old *moduleStateV1) upgradeToV2() (*moduleStateV2, error) {
	if old == nil {
		return nil, nil
	}

	path := shallowCopySlice(old.Path)
	if len(path) == 0 {
		// We found some V1 states with a nil path. Assume root.
		path = []string{"root"}
	}

	// Outputs needs upgrading to use the new structure
	outputs := make(map[string]*outputStateV2)
	for key, output := range old.Outputs {
		outputs[key] = &outputStateV2{
			Type:      "string",
			Value:     output,
			Sensitive: false,
		}
	}

	resources := make(map[string]*resourceStateV2)
	for key, oldResource := range old.Resources {
		upgraded, err := oldResource.upgradeToV2()
		if err != nil {
			return nil, fmt.Errorf("Error upgrading ModuleState V1: %v", err)
		}
		resources[key] = upgraded
	}

	return &moduleStateV2{
		Path:         path,
		Outputs:      outputs,
		Resources:    resources,
		Dependencies: shallowCopySlice(old.Dependencies),
	}, nil
}

func (old *resourceStateV1) upgradeToV2() (*resourceStateV2, error) {
	if old == nil {
		return nil, nil
	}

	primary, err := old.Primary.upgradeToV2()
	if err != nil {
		return nil, fmt.Errorf("Error upgrading ResourceState V1: %v", err)
	}

	deposed := make([]*instanceStateV2, len(old.Deposed))
	for i, v := range old.Deposed {
		upgraded, err := v.upgradeToV2()
		if err != nil {
			return nil, fmt.Errorf("Error upgrading ResourceState V1: %v", err)
		}
		deposed[i] = upgraded
	}
	if len(deposed) == 0 {
		deposed = nil
	}

	return &resourceStateV2{
		Type:         old.Type,
		Dependencies: shallowCopySlice(old.Dependencies),
		Primary:      primary,
		Deposed:      deposed,
		Provider:     old.Provider,
	}, nil
}

func (old *instanceStateV1) upgradeToV2() (*instanceStateV2, error) {
	if old == nil {
		return nil, nil
	}

	// "Meta" changed from map[string]string to map[string]interface{},
	// so we'll need to wrap all of the prior strings as interface values.
	var newMeta map[string]interface{}
	if old.Meta != nil {
		newMeta = make(map[string]interface{}, len(old.Meta))
		for k, v := range old.Meta {
			newMeta[k] = v
		}
	}

	return &instanceStateV2{
		ID:         old.ID,
		Attributes: shallowCopyMap(old.Attributes),
		Meta:       newMeta,
	}, nil
}
