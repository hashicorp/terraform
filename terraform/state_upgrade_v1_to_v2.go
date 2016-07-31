package terraform

import (
	"fmt"

	"github.com/mitchellh/copystructure"
)

// upgradeStateV1ToV2 is used to upgrade a V1 state representation
// into a V2 state representation
func upgradeStateV1ToV2(old *stateV1) (*State, error) {
	if old == nil {
		return nil, nil
	}

	remote, err := old.Remote.upgradeToV2()
	if err != nil {
		return nil, fmt.Errorf("Error upgrading State V1: %v", err)
	}

	modules := make([]*ModuleState, len(old.Modules))
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

	newState := &State{
		Version: 2,
		Serial:  old.Serial,
		Remote:  remote,
		Modules: modules,
	}

	newState.sort()

	return newState, nil
}

func (old *remoteStateV1) upgradeToV2() (*RemoteState, error) {
	if old == nil {
		return nil, nil
	}

	config, err := copystructure.Copy(old.Config)
	if err != nil {
		return nil, fmt.Errorf("Error upgrading RemoteState V1: %v", err)
	}

	return &RemoteState{
		Type:   old.Type,
		Config: config.(map[string]string),
	}, nil
}

func (old *moduleStateV1) upgradeToV2() (*ModuleState, error) {
	if old == nil {
		return nil, nil
	}

	path, err := copystructure.Copy(old.Path)
	if err != nil {
		return nil, fmt.Errorf("Error upgrading ModuleState V1: %v", err)
	}

	// Outputs needs upgrading to use the new structure
	outputs := make(map[string]*OutputState)
	for key, output := range old.Outputs {
		outputs[key] = &OutputState{
			Type:      "string",
			Value:     output,
			Sensitive: false,
		}
	}

	resources := make(map[string]*ResourceState)
	for key, oldResource := range old.Resources {
		upgraded, err := oldResource.upgradeToV2()
		if err != nil {
			return nil, fmt.Errorf("Error upgrading ModuleState V1: %v", err)
		}
		resources[key] = upgraded
	}

	dependencies, err := copystructure.Copy(old.Dependencies)
	if err != nil {
		return nil, fmt.Errorf("Error upgrading ModuleState V1: %v", err)
	}

	return &ModuleState{
		Path:         path.([]string),
		Outputs:      outputs,
		Resources:    resources,
		Dependencies: dependencies.([]string),
	}, nil
}

func (old *resourceStateV1) upgradeToV2() (*ResourceState, error) {
	if old == nil {
		return nil, nil
	}

	dependencies, err := copystructure.Copy(old.Dependencies)
	if err != nil {
		return nil, fmt.Errorf("Error upgrading ResourceState V1: %v", err)
	}

	primary, err := old.Primary.upgradeToV2()
	if err != nil {
		return nil, fmt.Errorf("Error upgrading ResourceState V1: %v", err)
	}

	deposed := make([]*InstanceState, len(old.Deposed))
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

	return &ResourceState{
		Type:         old.Type,
		Dependencies: dependencies.([]string),
		Primary:      primary,
		Deposed:      deposed,
		Provider:     old.Provider,
	}, nil
}

func (old *instanceStateV1) upgradeToV2() (*InstanceState, error) {
	if old == nil {
		return nil, nil
	}

	attributes, err := copystructure.Copy(old.Attributes)
	if err != nil {
		return nil, fmt.Errorf("Error upgrading InstanceState V1: %v", err)
	}
	ephemeral, err := old.Ephemeral.upgradeToV2()
	if err != nil {
		return nil, fmt.Errorf("Error upgrading InstanceState V1: %v", err)
	}
	meta, err := copystructure.Copy(old.Meta)
	if err != nil {
		return nil, fmt.Errorf("Error upgrading InstanceState V1: %v", err)
	}

	return &InstanceState{
		ID:         old.ID,
		Attributes: attributes.(map[string]string),
		Ephemeral:  *ephemeral,
		Meta:       meta.(map[string]string),
	}, nil
}

func (old *ephemeralStateV1) upgradeToV2() (*EphemeralState, error) {
	connInfo, err := copystructure.Copy(old.ConnInfo)
	if err != nil {
		return nil, fmt.Errorf("Error upgrading EphemeralState V1: %v", err)
	}
	return &EphemeralState{
		ConnInfo: connInfo.(map[string]string),
	}, nil
}
