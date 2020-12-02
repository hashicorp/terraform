package terraform

import (
	"bytes"
	"strings"
	"testing"
)

// TestReadUpgradeStateV2toV3 tests the state upgrade process from the V2 state
// to the current version, and needs editing each time. This means it tests the
// entire pipeline of upgrades (which migrate version to version).
func TestReadUpgradeStateV2toV3(t *testing.T) {
	// ReadState should transparently detect the old version but will upgrade
	// it on Write.
	upgraded, err := ReadState(strings.NewReader(testV2State))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	buf := new(bytes.Buffer)
	if err := WriteState(upgraded, buf); err != nil {
		t.Fatalf("err: %s", err)
	}

	if upgraded.Version != 3 {
		t.Fatalf("bad: State version not incremented; is %d", upgraded.Version)
	}

	// For this test we cannot assert that we match the round trip because an
	// empty map has been removed from state. Instead we make assertions against
	// some of the key fields in the _upgraded_ state.
	instanceState, ok := upgraded.RootModule().Resources["test_resource.main"]
	if !ok {
		t.Fatalf("Instance state for test_resource.main was removed from state during upgrade")
	}

	primary := instanceState.Primary
	if primary == nil {
		t.Fatalf("Primary instance was removed from state for test_resource.main")
	}

	// Non-empty computed map is moved from .# to .%
	if _, ok := primary.Attributes["computed_map.#"]; ok {
		t.Fatalf("Count was not upgraded from .# to .%% for computed_map")
	}
	if count, ok := primary.Attributes["computed_map.%"]; !ok || count != "1" {
		t.Fatalf("Count was not in .%% or was not 2 for computed_map")
	}

	// list_of_map top level retains .#
	if count, ok := primary.Attributes["list_of_map.#"]; !ok || count != "2" {
		t.Fatal("Count for list_of_map was migrated incorrectly")
	}

	// list_of_map.0 is moved from .# to .%
	if _, ok := primary.Attributes["list_of_map.0.#"]; ok {
		t.Fatalf("Count was not upgraded from .# to .%% for list_of_map.0")
	}
	if count, ok := primary.Attributes["list_of_map.0.%"]; !ok || count != "2" {
		t.Fatalf("Count was not in .%% or was not 2 for list_of_map.0")
	}

	// list_of_map.1 is moved from .# to .%
	if _, ok := primary.Attributes["list_of_map.1.#"]; ok {
		t.Fatalf("Count was not upgraded from .# to .%% for list_of_map.1")
	}
	if count, ok := primary.Attributes["list_of_map.1.%"]; !ok || count != "2" {
		t.Fatalf("Count was not in .%% or was not 2 for list_of_map.1")
	}

	// map is moved from .# to .%
	if _, ok := primary.Attributes["map.#"]; ok {
		t.Fatalf("Count was not upgraded from .# to .%% for map")
	}
	if count, ok := primary.Attributes["map.%"]; !ok || count != "2" {
		t.Fatalf("Count was not in .%% or was not 2 for map")
	}

	// optional_computed_map should be removed from state
	if _, ok := primary.Attributes["optional_computed_map"]; ok {
		t.Fatal("optional_computed_map was not removed from state")
	}

	// required_map is moved from .# to .%
	if _, ok := primary.Attributes["required_map.#"]; ok {
		t.Fatalf("Count was not upgraded from .# to .%% for required_map")
	}
	if count, ok := primary.Attributes["required_map.%"]; !ok || count != "3" {
		t.Fatalf("Count was not in .%% or was not 3 for map")
	}

	// computed_list keeps .#
	if count, ok := primary.Attributes["computed_list.#"]; !ok || count != "2" {
		t.Fatal("Count was migrated incorrectly for computed_list")
	}

	// computed_set keeps .#
	if count, ok := primary.Attributes["computed_set.#"]; !ok || count != "2" {
		t.Fatal("Count was migrated incorrectly for computed_set")
	}
	if val, ok := primary.Attributes["computed_set.2337322984"]; !ok || val != "setval1" {
		t.Fatal("Set item for computed_set.2337322984 changed or moved")
	}
	if val, ok := primary.Attributes["computed_set.307881554"]; !ok || val != "setval2" {
		t.Fatal("Set item for computed_set.307881554 changed or moved")
	}

	// string properties are unaffected
	if val, ok := primary.Attributes["id"]; !ok || val != "testId" {
		t.Fatal("id was not set correctly after migration")
	}
}

const testV2State = `{
    "version": 2,
    "terraform_version": "0.7.0",
    "serial": 2,
    "modules": [
        {
            "path": [
                "root"
            ],
            "outputs": {
                "computed_map": {
                    "sensitive": false,
                    "type": "map",
                    "value": {
                        "key1": "value1"
                    }
                },
                "computed_set": {
                    "sensitive": false,
                    "type": "list",
                    "value": [
                        "setval1",
                        "setval2"
                    ]
                },
                "map": {
                    "sensitive": false,
                    "type": "map",
                    "value": {
                        "key": "test",
                        "test": "test"
                    }
                },
                "set": {
                    "sensitive": false,
                    "type": "list",
                    "value": [
                        "test1",
                        "test2"
                    ]
                }
            },
            "resources": {
                "test_resource.main": {
                    "type": "test_resource",
                    "primary": {
                        "id": "testId",
                        "attributes": {
                            "computed_list.#": "2",
                            "computed_list.0": "listval1",
                            "computed_list.1": "listval2",
                            "computed_map.#": "1",
                            "computed_map.key1": "value1",
                            "computed_read_only": "value_from_api",
                            "computed_read_only_force_new": "value_from_api",
                            "computed_set.#": "2",
                            "computed_set.2337322984": "setval1",
                            "computed_set.307881554": "setval2",
                            "id": "testId",
                            "list_of_map.#": "2",
                            "list_of_map.0.#": "2",
                            "list_of_map.0.key1": "value1",
                            "list_of_map.0.key2": "value2",
                            "list_of_map.1.#": "2",
                            "list_of_map.1.key3": "value3",
                            "list_of_map.1.key4": "value4",
                            "map.#": "2",
                            "map.key": "test",
                            "map.test": "test",
                            "map_that_look_like_set.#": "2",
                            "map_that_look_like_set.12352223": "hello",
                            "map_that_look_like_set.36234341": "world",
                            "optional_computed_map.#": "0",
                            "required": "Hello World",
                            "required_map.#": "3",
                            "required_map.key1": "value1",
                            "required_map.key2": "value2",
                            "required_map.key3": "value3",
                            "set.#": "2",
                            "set.2326977762": "test1",
                            "set.331058520": "test2"
                        }
                    }
                }
            }
        }
    ]
}
`
