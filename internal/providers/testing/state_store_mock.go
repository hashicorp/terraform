// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package testing

import (
	"fmt"
	"maps"
	"slices"
)

// MockStateBytes is a map where the keys are state store types, i.e. names of store implementations in the mock provider. The value is a map of workspace name to bytes data.
type MockStateBytes map[string]map[string][]byte

func NewMultipleMockStateBytes(types []string) MockStateBytes {
	m := map[string]map[string][]byte{}
	for _, t := range types {
		m[t] = map[string][]byte{}
	}
	return m
}

func NewMockStateBytesWithSingleState(typeName, stateId string, b []byte) MockStateBytes {
	m := map[string]map[string][]byte{}
	m[typeName] = map[string][]byte{
		stateId: b,
	}
	return m
}

func NewMockStateBytesWithTypes(typeNames []string) MockStateBytes {
	m := map[string]map[string][]byte{}
	for _, typeName := range typeNames {
		m[typeName] = map[string][]byte{}
	}
	return m
}

func NewMockStateBytesWithStateIds(typeName string, stateIds []string) MockStateBytes {
	m := map[string]map[string][]byte{}
	m[typeName] = map[string][]byte{}
	for _, stateId := range stateIds {
		m[typeName][stateId] = []byte{}
	}
	return m
}

func NewMockStateBytesWithTypesAndStateIds(typeNames []string, stateIds []string) MockStateBytes {
	m := map[string]map[string][]byte{}
	for _, typeName := range typeNames {
		m[typeName] = map[string][]byte{}
		for _, stateId := range stateIds {
			m[typeName][stateId] = []byte{}
		}
	}
	return m
}

func (msb MockStateBytes) Write(typeName string, stateId string, b []byte) error {
	_, ok := msb[typeName]
	if !ok {
		return fmt.Errorf("state store %q not declared", typeName)
	}

	msb[typeName][stateId] = b
	return nil
}

type StateNotFoundErr struct {
	TypeName string
	StateId  string
}

func (e StateNotFoundErr) Is(target error) bool {
	return target == e
}

func (e StateNotFoundErr) Error() string {
	return fmt.Sprintf("state not found for state ID %q (%q)", e.StateId, e.TypeName)
}

func (msb MockStateBytes) Read(typeName string, stateId string) ([]byte, error) {
	_, ok := msb[typeName]
	if !ok {
		return nil, fmt.Errorf("state store %q not declared", typeName)
	}
	_, ok = msb[typeName][stateId]
	if !ok {
		return nil, StateNotFoundErr{
			TypeName: typeName,
			StateId:  stateId,
		}
	}

	return msb[typeName][stateId], nil
}

func (msb MockStateBytes) StateIds(typeName string) ([]string, error) {
	_, ok := msb[typeName]
	if !ok {
		return nil, fmt.Errorf("state store %q not declared", typeName)
	}
	return slices.Sorted(maps.Keys(msb[typeName])), nil
}

func (msb MockStateBytes) Delete(typeName, stateId string) error {
	_, ok := msb[typeName]
	if !ok {
		return fmt.Errorf("state store %q not declared", typeName)
	}

	_, ok = msb[typeName][stateId]
	if !ok {
		return fmt.Errorf("state ID %q (%q) does not exist so cannot be deleted", stateId, typeName)
	}

	delete(msb[typeName], stateId)
	return nil
}

func (msb MockStateBytes) StateIdExists(typeName, stateId string) (bool, error) {
	_, ok := msb[typeName]
	if !ok {
		return false, fmt.Errorf("state store %q not declared", typeName)
	}

	_, ok = msb[typeName][stateId]
	return ok, nil
}
