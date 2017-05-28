package schema

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/hil"
	"github.com/hashicorp/hil/ast"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/terraform"
)

func TestEnvDefaultFunc(t *testing.T) {
	key := "TF_TEST_ENV_DEFAULT_FUNC"
	defer os.Unsetenv(key)

	f := EnvDefaultFunc(key, "42")
	if err := os.Setenv(key, "foo"); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual, err := f()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if actual != "foo" {
		t.Fatalf("bad: %#v", actual)
	}

	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual, err = f()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if actual != "42" {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestMultiEnvDefaultFunc(t *testing.T) {
	keys := []string{
		"TF_TEST_MULTI_ENV_DEFAULT_FUNC1",
		"TF_TEST_MULTI_ENV_DEFAULT_FUNC2",
	}
	defer func() {
		for _, k := range keys {
			os.Unsetenv(k)
		}
	}()

	// Test that the first key is returned first
	f := MultiEnvDefaultFunc(keys, "42")
	if err := os.Setenv(keys[0], "foo"); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual, err := f()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if actual != "foo" {
		t.Fatalf("bad: %#v", actual)
	}

	if err := os.Unsetenv(keys[0]); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Test that the second key is returned if the first one is empty
	f = MultiEnvDefaultFunc(keys, "42")
	if err := os.Setenv(keys[1], "foo"); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual, err = f()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if actual != "foo" {
		t.Fatalf("bad: %#v", actual)
	}

	if err := os.Unsetenv(keys[1]); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Test that the default value is returned when no keys are set
	actual, err = f()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if actual != "42" {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestValueType_Zero(t *testing.T) {
	cases := []struct {
		Type  ValueType
		Value interface{}
	}{
		{TypeBool, false},
		{TypeInt, 0},
		{TypeFloat, 0.0},
		{TypeString, ""},
		{TypeList, []interface{}{}},
		{TypeMap, map[string]interface{}{}},
		{TypeSet, new(Set)},
	}

	for i, tc := range cases {
		actual := tc.Type.Zero()
		if !reflect.DeepEqual(actual, tc.Value) {
			t.Fatalf("%d: %#v != %#v", i, actual, tc.Value)
		}
	}
}

func interfaceToVariableSwallowError(input interface{}) ast.Variable {
	variable, _ := hil.InterfaceToVariable(input)
	return variable
}

func TestSchemaMap_Diff(t *testing.T) {
	cases := []struct {
		Name            string
		Schema          map[string]*Schema
		State           *terraform.InstanceState
		Config          map[string]interface{}
		ConfigVariables map[string]ast.Variable
		Diff            *terraform.InstanceDiff
		Err             bool
	}{
		{
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"availability_zone": "foo",
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "foo",
						RequiresNew: true,
					},
				},
			},

			Err: false,
		},

		{
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: nil,

			Config: map[string]interface{}{},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": &terraform.ResourceAttrDiff{
						Old:         "",
						NewComputed: true,
						RequiresNew: true,
					},
				},
			},

			Err: false,
		},

		{
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: &terraform.InstanceState{
				ID: "foo",
			},

			Config: map[string]interface{}{},

			Diff: nil,

			Err: false,
		},

		{
			Name: "Computed, but set in config",
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Optional: true,
					Computed: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"availability_zone": "foo",
				},
			},

			Config: map[string]interface{}{
				"availability_zone": "bar",
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": &terraform.ResourceAttrDiff{
						Old: "foo",
						New: "bar",
					},
				},
			},

			Err: false,
		},

		{
			Name: "Default",
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Optional: true,
					Default:  "foo",
				},
			},

			State: nil,

			Config: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": &terraform.ResourceAttrDiff{
						Old: "",
						New: "foo",
					},
				},
			},

			Err: false,
		},

		{
			Name: "DefaultFunc, value",
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Optional: true,
					DefaultFunc: func() (interface{}, error) {
						return "foo", nil
					},
				},
			},

			State: nil,

			Config: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": &terraform.ResourceAttrDiff{
						Old: "",
						New: "foo",
					},
				},
			},

			Err: false,
		},

		{
			Name: "DefaultFunc, configuration set",
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Optional: true,
					DefaultFunc: func() (interface{}, error) {
						return "foo", nil
					},
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"availability_zone": "bar",
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": &terraform.ResourceAttrDiff{
						Old: "",
						New: "bar",
					},
				},
			},

			Err: false,
		},

		{
			Name: "String with StateFunc",
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Optional: true,
					Computed: true,
					StateFunc: func(a interface{}) string {
						return a.(string) + "!"
					},
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"availability_zone": "foo",
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": &terraform.ResourceAttrDiff{
						Old:      "",
						New:      "foo!",
						NewExtra: "foo",
					},
				},
			},

			Err: false,
		},

		{
			Name: "StateFunc not called with nil value",
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Optional: true,
					Computed: true,
					StateFunc: func(a interface{}) string {
						t.Fatalf("should not get here!")
						return ""
					},
				},
			},

			State: nil,

			Config: map[string]interface{}{},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "",
						NewComputed: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "Variable (just checking)",
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Optional: true,
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"availability_zone": "${var.foo}",
			},

			ConfigVariables: map[string]ast.Variable{
				"var.foo": interfaceToVariableSwallowError("bar"),
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": &terraform.ResourceAttrDiff{
						Old: "",
						New: "bar",
					},
				},
			},

			Err: false,
		},

		{
			Name: "Variable computed",
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Optional: true,
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"availability_zone": "${var.foo}",
			},

			ConfigVariables: map[string]ast.Variable{
				"var.foo": interfaceToVariableSwallowError(config.UnknownVariableValue),
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "${var.foo}",
						NewComputed: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "Int decode",
			Schema: map[string]*Schema{
				"port": &Schema{
					Type:     TypeInt,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"port": 27,
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"port": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "27",
						RequiresNew: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "bool decode",
			Schema: map[string]*Schema{
				"port": &Schema{
					Type:     TypeBool,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"port": false,
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"port": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "false",
						RequiresNew: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "Bool",
			Schema: map[string]*Schema{
				"delete": &Schema{
					Type:     TypeBool,
					Optional: true,
					Default:  false,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"delete": "false",
				},
			},

			Config: nil,

			Diff: nil,

			Err: false,
		},

		{
			Name: "List decode",
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeList,
					Required: true,
					Elem:     &Schema{Type: TypeInt},
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"ports": []interface{}{1, 2, 5},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": &terraform.ResourceAttrDiff{
						Old: "0",
						New: "3",
					},
					"ports.0": &terraform.ResourceAttrDiff{
						Old: "",
						New: "1",
					},
					"ports.1": &terraform.ResourceAttrDiff{
						Old: "",
						New: "2",
					},
					"ports.2": &terraform.ResourceAttrDiff{
						Old: "",
						New: "5",
					},
				},
			},

			Err: false,
		},

		{
			Name: "List decode with promotion",
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:          TypeList,
					Required:      true,
					Elem:          &Schema{Type: TypeInt},
					PromoteSingle: true,
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"ports": "5",
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": &terraform.ResourceAttrDiff{
						Old: "0",
						New: "1",
					},
					"ports.0": &terraform.ResourceAttrDiff{
						Old: "",
						New: "5",
					},
				},
			},

			Err: false,
		},

		{
			Name: "List decode with promotion with list",
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:          TypeList,
					Required:      true,
					Elem:          &Schema{Type: TypeInt},
					PromoteSingle: true,
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"ports": []interface{}{"5"},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": &terraform.ResourceAttrDiff{
						Old: "0",
						New: "1",
					},
					"ports.0": &terraform.ResourceAttrDiff{
						Old: "",
						New: "5",
					},
				},
			},

			Err: false,
		},

		{
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeList,
					Required: true,
					Elem:     &Schema{Type: TypeInt},
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"ports": []interface{}{1, "${var.foo}"},
			},

			ConfigVariables: map[string]ast.Variable{
				"var.foo": interfaceToVariableSwallowError([]interface{}{"2", "5"}),
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": &terraform.ResourceAttrDiff{
						Old: "0",
						New: "3",
					},
					"ports.0": &terraform.ResourceAttrDiff{
						Old: "",
						New: "1",
					},
					"ports.1": &terraform.ResourceAttrDiff{
						Old: "",
						New: "2",
					},
					"ports.2": &terraform.ResourceAttrDiff{
						Old: "",
						New: "5",
					},
				},
			},

			Err: false,
		},

		{
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeList,
					Required: true,
					Elem:     &Schema{Type: TypeInt},
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"ports": []interface{}{1, "${var.foo}"},
			},

			ConfigVariables: map[string]ast.Variable{
				"var.foo": interfaceToVariableSwallowError([]interface{}{
					config.UnknownVariableValue, "5"}),
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": &terraform.ResourceAttrDiff{
						Old:         "0",
						New:         "",
						NewComputed: true,
					},
				},
			},

			Err: false,
		},

		{
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeList,
					Required: true,
					Elem:     &Schema{Type: TypeInt},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"ports.#": "3",
					"ports.0": "1",
					"ports.1": "2",
					"ports.2": "5",
				},
			},

			Config: map[string]interface{}{
				"ports": []interface{}{1, 2, 5},
			},

			Diff: nil,

			Err: false,
		},

		{
			Name: "",
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeList,
					Required: true,
					Elem:     &Schema{Type: TypeInt},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"ports.#": "2",
					"ports.0": "1",
					"ports.1": "2",
				},
			},

			Config: map[string]interface{}{
				"ports": []interface{}{1, 2, 5},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": &terraform.ResourceAttrDiff{
						Old: "2",
						New: "3",
					},
					"ports.2": &terraform.ResourceAttrDiff{
						Old: "",
						New: "5",
					},
				},
			},

			Err: false,
		},

		{
			Name: "",
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeList,
					Required: true,
					Elem:     &Schema{Type: TypeInt},
					ForceNew: true,
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"ports": []interface{}{1, 2, 5},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": &terraform.ResourceAttrDiff{
						Old:         "0",
						New:         "3",
						RequiresNew: true,
					},
					"ports.0": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "1",
						RequiresNew: true,
					},
					"ports.1": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "2",
						RequiresNew: true,
					},
					"ports.2": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "5",
						RequiresNew: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "",
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeList,
					Optional: true,
					Computed: true,
					Elem:     &Schema{Type: TypeInt},
				},
			},

			State: nil,

			Config: map[string]interface{}{},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": &terraform.ResourceAttrDiff{
						Old:         "",
						NewComputed: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "List with computed set",
			Schema: map[string]*Schema{
				"config": &Schema{
					Type:     TypeList,
					Optional: true,
					ForceNew: true,
					MinItems: 1,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"name": {
								Type:     TypeString,
								Required: true,
							},

							"rules": {
								Type:     TypeSet,
								Computed: true,
								Elem:     &Schema{Type: TypeString},
								Set:      HashString,
							},
						},
					},
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"config": []interface{}{
					map[string]interface{}{
						"name": "hello",
					},
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"config.#": &terraform.ResourceAttrDiff{
						Old:         "0",
						New:         "1",
						RequiresNew: true,
					},

					"config.0.name": &terraform.ResourceAttrDiff{
						Old: "",
						New: "hello",
					},

					"config.0.rules.#": &terraform.ResourceAttrDiff{
						Old:         "",
						NewComputed: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "Set",
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeSet,
					Required: true,
					Elem:     &Schema{Type: TypeInt},
					Set: func(a interface{}) int {
						return a.(int)
					},
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"ports": []interface{}{5, 2, 1},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": &terraform.ResourceAttrDiff{
						Old: "0",
						New: "3",
					},
					"ports.1": &terraform.ResourceAttrDiff{
						Old: "",
						New: "1",
					},
					"ports.2": &terraform.ResourceAttrDiff{
						Old: "",
						New: "2",
					},
					"ports.5": &terraform.ResourceAttrDiff{
						Old: "",
						New: "5",
					},
				},
			},

			Err: false,
		},

		{
			Name: "Set",
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeSet,
					Computed: true,
					Required: true,
					Elem:     &Schema{Type: TypeInt},
					Set: func(a interface{}) int {
						return a.(int)
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"ports.#": "0",
				},
			},

			Config: nil,

			Diff: nil,

			Err: false,
		},

		{
			Name: "Set",
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeSet,
					Optional: true,
					Computed: true,
					Elem:     &Schema{Type: TypeInt},
					Set: func(a interface{}) int {
						return a.(int)
					},
				},
			},

			State: nil,

			Config: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": &terraform.ResourceAttrDiff{
						Old:         "",
						NewComputed: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "Set",
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeSet,
					Required: true,
					Elem:     &Schema{Type: TypeInt},
					Set: func(a interface{}) int {
						return a.(int)
					},
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"ports": []interface{}{"${var.foo}", 1},
			},

			ConfigVariables: map[string]ast.Variable{
				"var.foo": interfaceToVariableSwallowError([]interface{}{"2", "5"}),
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": &terraform.ResourceAttrDiff{
						Old: "0",
						New: "3",
					},
					"ports.1": &terraform.ResourceAttrDiff{
						Old: "",
						New: "1",
					},
					"ports.2": &terraform.ResourceAttrDiff{
						Old: "",
						New: "2",
					},
					"ports.5": &terraform.ResourceAttrDiff{
						Old: "",
						New: "5",
					},
				},
			},

			Err: false,
		},

		{
			Name: "Set",
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeSet,
					Required: true,
					Elem:     &Schema{Type: TypeInt},
					Set: func(a interface{}) int {
						return a.(int)
					},
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"ports": []interface{}{1, "${var.foo}"},
			},

			ConfigVariables: map[string]ast.Variable{
				"var.foo": interfaceToVariableSwallowError([]interface{}{
					config.UnknownVariableValue, "5"}),
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "",
						NewComputed: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "Set",
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeSet,
					Required: true,
					Elem:     &Schema{Type: TypeInt},
					Set: func(a interface{}) int {
						return a.(int)
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"ports.#": "2",
					"ports.1": "1",
					"ports.2": "2",
				},
			},

			Config: map[string]interface{}{
				"ports": []interface{}{5, 2, 1},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": &terraform.ResourceAttrDiff{
						Old: "2",
						New: "3",
					},
					"ports.1": &terraform.ResourceAttrDiff{
						Old: "1",
						New: "1",
					},
					"ports.2": &terraform.ResourceAttrDiff{
						Old: "2",
						New: "2",
					},
					"ports.5": &terraform.ResourceAttrDiff{
						Old: "",
						New: "5",
					},
				},
			},

			Err: false,
		},

		{
			Name: "Set",
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeSet,
					Required: true,
					Elem:     &Schema{Type: TypeInt},
					Set: func(a interface{}) int {
						return a.(int)
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"ports.#": "2",
					"ports.1": "1",
					"ports.2": "2",
				},
			},

			Config: map[string]interface{}{},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": &terraform.ResourceAttrDiff{
						Old: "2",
						New: "0",
					},
					"ports.1": &terraform.ResourceAttrDiff{
						Old:        "1",
						New:        "0",
						NewRemoved: true,
					},
					"ports.2": &terraform.ResourceAttrDiff{
						Old:        "2",
						New:        "0",
						NewRemoved: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "Set",
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeSet,
					Optional: true,
					Computed: true,
					Elem:     &Schema{Type: TypeInt},
					Set: func(a interface{}) int {
						return a.(int)
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"availability_zone": "bar",
					"ports.#":           "1",
					"ports.80":          "80",
				},
			},

			Config: map[string]interface{}{},

			Diff: nil,

			Err: false,
		},

		{
			Name: "Set",
			Schema: map[string]*Schema{
				"ingress": &Schema{
					Type:     TypeSet,
					Required: true,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"ports": &Schema{
								Type:     TypeList,
								Optional: true,
								Elem:     &Schema{Type: TypeInt},
							},
						},
					},
					Set: func(v interface{}) int {
						m := v.(map[string]interface{})
						ps := m["ports"].([]interface{})
						result := 0
						for _, p := range ps {
							result += p.(int)
						}
						return result
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"ingress.#":           "2",
					"ingress.80.ports.#":  "1",
					"ingress.80.ports.0":  "80",
					"ingress.443.ports.#": "1",
					"ingress.443.ports.0": "443",
				},
			},

			Config: map[string]interface{}{
				"ingress": []map[string]interface{}{
					map[string]interface{}{
						"ports": []interface{}{443},
					},
					map[string]interface{}{
						"ports": []interface{}{80},
					},
				},
			},

			Diff: nil,

			Err: false,
		},

		{
			Name: "List of structure decode",
			Schema: map[string]*Schema{
				"ingress": &Schema{
					Type:     TypeList,
					Required: true,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"from": &Schema{
								Type:     TypeInt,
								Required: true,
							},
						},
					},
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"ingress": []interface{}{
					map[string]interface{}{
						"from": 8080,
					},
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ingress.#": &terraform.ResourceAttrDiff{
						Old: "0",
						New: "1",
					},
					"ingress.0.from": &terraform.ResourceAttrDiff{
						Old: "",
						New: "8080",
					},
				},
			},

			Err: false,
		},

		{
			Name: "ComputedWhen",
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:         TypeString,
					Computed:     true,
					ComputedWhen: []string{"port"},
				},

				"port": &Schema{
					Type:     TypeInt,
					Optional: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"availability_zone": "foo",
					"port":              "80",
				},
			},

			Config: map[string]interface{}{
				"port": 80,
			},

			Diff: nil,

			Err: false,
		},

		{
			Name: "",
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:         TypeString,
					Computed:     true,
					ComputedWhen: []string{"port"},
				},

				"port": &Schema{
					Type:     TypeInt,
					Optional: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"port": "80",
				},
			},

			Config: map[string]interface{}{
				"port": 80,
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": &terraform.ResourceAttrDiff{
						NewComputed: true,
					},
				},
			},

			Err: false,
		},

		/* TODO
		{
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:         TypeString,
					Computed:     true,
					ComputedWhen: []string{"port"},
				},

				"port": &Schema{
					Type:     TypeInt,
					Optional: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"availability_zone": "foo",
					"port":              "80",
				},
			},

			Config: map[string]interface{}{
				"port": 8080,
			},

			Diff: &terraform.ResourceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": &terraform.ResourceAttrDiff{
						Old:         "foo",
						NewComputed: true,
					},
					"port": &terraform.ResourceAttrDiff{
						Old: "80",
						New: "8080",
					},
				},
			},

			Err: false,
		},
		*/

		{
			Name: "Maps",
			Schema: map[string]*Schema{
				"config_vars": &Schema{
					Type: TypeMap,
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"config_vars": []interface{}{
					map[string]interface{}{
						"bar": "baz",
					},
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"config_vars.%": &terraform.ResourceAttrDiff{
						Old: "0",
						New: "1",
					},

					"config_vars.bar": &terraform.ResourceAttrDiff{
						Old: "",
						New: "baz",
					},
				},
			},

			Err: false,
		},

		{
			Name: "Maps",
			Schema: map[string]*Schema{
				"config_vars": &Schema{
					Type: TypeMap,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"config_vars.foo": "bar",
				},
			},

			Config: map[string]interface{}{
				"config_vars": []interface{}{
					map[string]interface{}{
						"bar": "baz",
					},
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"config_vars.foo": &terraform.ResourceAttrDiff{
						Old:        "bar",
						NewRemoved: true,
					},
					"config_vars.bar": &terraform.ResourceAttrDiff{
						Old: "",
						New: "baz",
					},
				},
			},

			Err: false,
		},

		{
			Name: "Maps",
			Schema: map[string]*Schema{
				"vars": &Schema{
					Type:     TypeMap,
					Optional: true,
					Computed: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"vars.foo": "bar",
				},
			},

			Config: map[string]interface{}{
				"vars": []interface{}{
					map[string]interface{}{
						"bar": "baz",
					},
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"vars.foo": &terraform.ResourceAttrDiff{
						Old:        "bar",
						New:        "",
						NewRemoved: true,
					},
					"vars.bar": &terraform.ResourceAttrDiff{
						Old: "",
						New: "baz",
					},
				},
			},

			Err: false,
		},

		{
			Name: "Maps",
			Schema: map[string]*Schema{
				"vars": &Schema{
					Type:     TypeMap,
					Computed: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"vars.foo": "bar",
				},
			},

			Config: nil,

			Diff: nil,

			Err: false,
		},

		{
			Name: "Maps",
			Schema: map[string]*Schema{
				"config_vars": &Schema{
					Type: TypeList,
					Elem: &Schema{Type: TypeMap},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"config_vars.#":     "1",
					"config_vars.0.foo": "bar",
				},
			},

			Config: map[string]interface{}{
				"config_vars": []interface{}{
					map[string]interface{}{
						"bar": "baz",
					},
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"config_vars.0.foo": &terraform.ResourceAttrDiff{
						Old:        "bar",
						NewRemoved: true,
					},
					"config_vars.0.bar": &terraform.ResourceAttrDiff{
						Old: "",
						New: "baz",
					},
				},
			},

			Err: false,
		},

		{
			Name: "Maps",
			Schema: map[string]*Schema{
				"config_vars": &Schema{
					Type: TypeList,
					Elem: &Schema{Type: TypeMap},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"config_vars.#":     "1",
					"config_vars.0.foo": "bar",
					"config_vars.0.bar": "baz",
				},
			},

			Config: map[string]interface{}{},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"config_vars.#": &terraform.ResourceAttrDiff{
						Old: "1",
						New: "0",
					},
					"config_vars.0.%": &terraform.ResourceAttrDiff{
						Old: "2",
						New: "0",
					},
					"config_vars.0.foo": &terraform.ResourceAttrDiff{
						Old:        "bar",
						NewRemoved: true,
					},
					"config_vars.0.bar": &terraform.ResourceAttrDiff{
						Old:        "baz",
						NewRemoved: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "ForceNews",
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Optional: true,
					ForceNew: true,
				},

				"address": &Schema{
					Type:     TypeString,
					Optional: true,
					Computed: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"availability_zone": "bar",
					"address":           "foo",
				},
			},

			Config: map[string]interface{}{
				"availability_zone": "foo",
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": &terraform.ResourceAttrDiff{
						Old:         "bar",
						New:         "foo",
						RequiresNew: true,
					},

					"address": &terraform.ResourceAttrDiff{
						Old:         "foo",
						New:         "",
						NewComputed: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "Set",
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Optional: true,
					ForceNew: true,
				},

				"ports": &Schema{
					Type:     TypeSet,
					Optional: true,
					Computed: true,
					Elem:     &Schema{Type: TypeInt},
					Set: func(a interface{}) int {
						return a.(int)
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"availability_zone": "bar",
					"ports.#":           "1",
					"ports.80":          "80",
				},
			},

			Config: map[string]interface{}{
				"availability_zone": "foo",
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": &terraform.ResourceAttrDiff{
						Old:         "bar",
						New:         "foo",
						RequiresNew: true,
					},

					"ports.#": &terraform.ResourceAttrDiff{
						Old:         "1",
						New:         "",
						NewComputed: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "Set",
			Schema: map[string]*Schema{
				"instances": &Schema{
					Type:     TypeSet,
					Elem:     &Schema{Type: TypeString},
					Optional: true,
					Computed: true,
					Set: func(v interface{}) int {
						return len(v.(string))
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"instances.#": "0",
				},
			},

			Config: map[string]interface{}{
				"instances": []interface{}{"${var.foo}"},
			},

			ConfigVariables: map[string]ast.Variable{
				"var.foo": interfaceToVariableSwallowError(config.UnknownVariableValue),
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"instances.#": &terraform.ResourceAttrDiff{
						NewComputed: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "Set",
			Schema: map[string]*Schema{
				"route": &Schema{
					Type:     TypeSet,
					Optional: true,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"index": &Schema{
								Type:     TypeInt,
								Required: true,
							},

							"gateway": &Schema{
								Type:     TypeString,
								Optional: true,
							},
						},
					},
					Set: func(v interface{}) int {
						m := v.(map[string]interface{})
						return m["index"].(int)
					},
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"route": []map[string]interface{}{
					map[string]interface{}{
						"index":   "1",
						"gateway": "${var.foo}",
					},
				},
			},

			ConfigVariables: map[string]ast.Variable{
				"var.foo": interfaceToVariableSwallowError(config.UnknownVariableValue),
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"route.#": &terraform.ResourceAttrDiff{
						Old: "0",
						New: "1",
					},
					"route.~1.index": &terraform.ResourceAttrDiff{
						Old: "",
						New: "1",
					},
					"route.~1.gateway": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "${var.foo}",
						NewComputed: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "Set",
			Schema: map[string]*Schema{
				"route": &Schema{
					Type:     TypeSet,
					Optional: true,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"index": &Schema{
								Type:     TypeInt,
								Required: true,
							},

							"gateway": &Schema{
								Type:     TypeSet,
								Optional: true,
								Elem:     &Schema{Type: TypeInt},
								Set: func(a interface{}) int {
									return a.(int)
								},
							},
						},
					},
					Set: func(v interface{}) int {
						m := v.(map[string]interface{})
						return m["index"].(int)
					},
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"route": []map[string]interface{}{
					map[string]interface{}{
						"index": "1",
						"gateway": []interface{}{
							"${var.foo}",
						},
					},
				},
			},

			ConfigVariables: map[string]ast.Variable{
				"var.foo": interfaceToVariableSwallowError(config.UnknownVariableValue),
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"route.#": &terraform.ResourceAttrDiff{
						Old: "0",
						New: "1",
					},
					"route.~1.index": &terraform.ResourceAttrDiff{
						Old: "",
						New: "1",
					},
					"route.~1.gateway.#": &terraform.ResourceAttrDiff{
						NewComputed: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "Computed maps",
			Schema: map[string]*Schema{
				"vars": &Schema{
					Type:     TypeMap,
					Computed: true,
				},
			},

			State: nil,

			Config: nil,

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"vars.%": &terraform.ResourceAttrDiff{
						Old:         "",
						NewComputed: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "Computed maps",
			Schema: map[string]*Schema{
				"vars": &Schema{
					Type:     TypeMap,
					Computed: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"vars.%": "0",
				},
			},

			Config: map[string]interface{}{
				"vars": map[string]interface{}{
					"bar": "${var.foo}",
				},
			},

			ConfigVariables: map[string]ast.Variable{
				"var.foo": interfaceToVariableSwallowError(config.UnknownVariableValue),
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"vars.%": &terraform.ResourceAttrDiff{
						Old:         "",
						NewComputed: true,
					},
				},
			},

			Err: false,
		},

		{
			Name:   " - Empty",
			Schema: map[string]*Schema{},

			State: &terraform.InstanceState{},

			Config: map[string]interface{}{},

			Diff: nil,

			Err: false,
		},

		{
			Name: "Float",
			Schema: map[string]*Schema{
				"some_threshold": &Schema{
					Type: TypeFloat,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"some_threshold": "567.8",
				},
			},

			Config: map[string]interface{}{
				"some_threshold": 12.34,
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"some_threshold": &terraform.ResourceAttrDiff{
						Old: "567.8",
						New: "12.34",
					},
				},
			},

			Err: false,
		},

		{
			Name: "https://github.com/hashicorp/terraform/issues/824",
			Schema: map[string]*Schema{
				"block_device": &Schema{
					Type:     TypeSet,
					Optional: true,
					Computed: true,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"device_name": &Schema{
								Type:     TypeString,
								Required: true,
							},
							"delete_on_termination": &Schema{
								Type:     TypeBool,
								Optional: true,
								Default:  true,
							},
						},
					},
					Set: func(v interface{}) int {
						var buf bytes.Buffer
						m := v.(map[string]interface{})
						buf.WriteString(fmt.Sprintf("%s-", m["device_name"].(string)))
						buf.WriteString(fmt.Sprintf("%t-", m["delete_on_termination"].(bool)))
						return hashcode.String(buf.String())
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"block_device.#":                                "2",
					"block_device.616397234.delete_on_termination":  "true",
					"block_device.616397234.device_name":            "/dev/sda1",
					"block_device.2801811477.delete_on_termination": "true",
					"block_device.2801811477.device_name":           "/dev/sdx",
				},
			},

			Config: map[string]interface{}{
				"block_device": []map[string]interface{}{
					map[string]interface{}{
						"device_name": "/dev/sda1",
					},
					map[string]interface{}{
						"device_name": "/dev/sdx",
					},
				},
			},
			Diff: nil,
			Err:  false,
		},

		{
			Name: "Zero value in state shouldn't result in diff",
			Schema: map[string]*Schema{
				"port": &Schema{
					Type:     TypeBool,
					Optional: true,
					ForceNew: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"port": "false",
				},
			},

			Config: map[string]interface{}{},

			Diff: nil,

			Err: false,
		},

		{
			Name: "Same as prev, but for sets",
			Schema: map[string]*Schema{
				"route": &Schema{
					Type:     TypeSet,
					Optional: true,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"index": &Schema{
								Type:     TypeInt,
								Required: true,
							},

							"gateway": &Schema{
								Type:     TypeSet,
								Optional: true,
								Elem:     &Schema{Type: TypeInt},
								Set: func(a interface{}) int {
									return a.(int)
								},
							},
						},
					},
					Set: func(v interface{}) int {
						m := v.(map[string]interface{})
						return m["index"].(int)
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"route.#": "0",
				},
			},

			Config: map[string]interface{}{},

			Diff: nil,

			Err: false,
		},

		{
			Name: "A set computed element shouldn't cause a diff",
			Schema: map[string]*Schema{
				"active": &Schema{
					Type:     TypeBool,
					Computed: true,
					ForceNew: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"active": "true",
				},
			},

			Config: map[string]interface{}{},

			Diff: nil,

			Err: false,
		},

		{
			Name: "An empty set should show up in the diff",
			Schema: map[string]*Schema{
				"instances": &Schema{
					Type:     TypeSet,
					Elem:     &Schema{Type: TypeString},
					Optional: true,
					ForceNew: true,
					Set: func(v interface{}) int {
						return len(v.(string))
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"instances.#": "1",
					"instances.3": "foo",
				},
			},

			Config: map[string]interface{}{},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"instances.#": &terraform.ResourceAttrDiff{
						Old:         "1",
						New:         "0",
						RequiresNew: true,
					},
					"instances.3": &terraform.ResourceAttrDiff{
						Old:         "foo",
						New:         "",
						NewRemoved:  true,
						RequiresNew: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "Map with empty value",
			Schema: map[string]*Schema{
				"vars": &Schema{
					Type: TypeMap,
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"vars": map[string]interface{}{
					"foo": "",
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"vars.%": &terraform.ResourceAttrDiff{
						Old: "0",
						New: "1",
					},
					"vars.foo": &terraform.ResourceAttrDiff{
						Old: "",
						New: "",
					},
				},
			},

			Err: false,
		},

		{
			Name: "Unset bool, not in state",
			Schema: map[string]*Schema{
				"force": &Schema{
					Type:     TypeBool,
					Optional: true,
					ForceNew: true,
				},
			},

			State: nil,

			Config: map[string]interface{}{},

			Diff: nil,

			Err: false,
		},

		{
			Name: "Unset set, not in state",
			Schema: map[string]*Schema{
				"metadata_keys": &Schema{
					Type:     TypeSet,
					Optional: true,
					ForceNew: true,
					Elem:     &Schema{Type: TypeInt},
					Set:      func(interface{}) int { return 0 },
				},
			},

			State: nil,

			Config: map[string]interface{}{},

			Diff: nil,

			Err: false,
		},

		{
			Name: "Unset list in state, should not show up computed",
			Schema: map[string]*Schema{
				"metadata_keys": &Schema{
					Type:     TypeList,
					Optional: true,
					Computed: true,
					ForceNew: true,
					Elem:     &Schema{Type: TypeInt},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"metadata_keys.#": "0",
				},
			},

			Config: map[string]interface{}{},

			Diff: nil,

			Err: false,
		},

		{
			Name: "Set element computed substring",
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeSet,
					Required: true,
					Elem:     &Schema{Type: TypeInt},
					Set: func(a interface{}) int {
						return a.(int)
					},
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"ports": []interface{}{1, "${var.foo}32"},
			},

			ConfigVariables: map[string]ast.Variable{
				"var.foo": interfaceToVariableSwallowError(config.UnknownVariableValue),
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "",
						NewComputed: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "Computed map without config that's known to be empty does not generate diff",
			Schema: map[string]*Schema{
				"tags": &Schema{
					Type:     TypeMap,
					Computed: true,
				},
			},

			Config: nil,

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"tags.%": "0",
				},
			},

			Diff: nil,

			Err: false,
		},

		{
			Name: "Set with hyphen keys",
			Schema: map[string]*Schema{
				"route": &Schema{
					Type:     TypeSet,
					Optional: true,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"index": &Schema{
								Type:     TypeInt,
								Required: true,
							},

							"gateway-name": &Schema{
								Type:     TypeString,
								Optional: true,
							},
						},
					},
					Set: func(v interface{}) int {
						m := v.(map[string]interface{})
						return m["index"].(int)
					},
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"route": []map[string]interface{}{
					map[string]interface{}{
						"index":        "1",
						"gateway-name": "hello",
					},
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"route.#": &terraform.ResourceAttrDiff{
						Old: "0",
						New: "1",
					},
					"route.1.index": &terraform.ResourceAttrDiff{
						Old: "",
						New: "1",
					},
					"route.1.gateway-name": &terraform.ResourceAttrDiff{
						Old: "",
						New: "hello",
					},
				},
			},

			Err: false,
		},

		{
			Name: ": StateFunc in nested set (#1759)",
			Schema: map[string]*Schema{
				"service_account": &Schema{
					Type:     TypeList,
					Optional: true,
					ForceNew: true,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"scopes": &Schema{
								Type:     TypeSet,
								Required: true,
								ForceNew: true,
								Elem: &Schema{
									Type: TypeString,
									StateFunc: func(v interface{}) string {
										return v.(string) + "!"
									},
								},
								Set: func(v interface{}) int {
									i, err := strconv.Atoi(v.(string))
									if err != nil {
										t.Fatalf("err: %s", err)
									}
									return i
								},
							},
						},
					},
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"service_account": []map[string]interface{}{
					{
						"scopes": []interface{}{"123"},
					},
				},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"service_account.#": &terraform.ResourceAttrDiff{
						Old:         "0",
						New:         "1",
						RequiresNew: true,
					},
					"service_account.0.scopes.#": &terraform.ResourceAttrDiff{
						Old:         "0",
						New:         "1",
						RequiresNew: true,
					},
					"service_account.0.scopes.123": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "123!",
						NewExtra:    "123",
						RequiresNew: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "Removing set elements",
			Schema: map[string]*Schema{
				"instances": &Schema{
					Type:     TypeSet,
					Elem:     &Schema{Type: TypeString},
					Optional: true,
					ForceNew: true,
					Set: func(v interface{}) int {
						return len(v.(string))
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"instances.#": "2",
					"instances.3": "333",
					"instances.2": "22",
				},
			},

			Config: map[string]interface{}{
				"instances": []interface{}{"333", "4444"},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"instances.#": &terraform.ResourceAttrDiff{
						Old: "2",
						New: "2",
					},
					"instances.2": &terraform.ResourceAttrDiff{
						Old:         "22",
						New:         "",
						NewRemoved:  true,
						RequiresNew: true,
					},
					"instances.3": &terraform.ResourceAttrDiff{
						Old: "333",
						New: "333",
					},
					"instances.4": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "4444",
						RequiresNew: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "Bools can be set with 0/1 in config, still get true/false",
			Schema: map[string]*Schema{
				"one": &Schema{
					Type:     TypeBool,
					Optional: true,
				},
				"two": &Schema{
					Type:     TypeBool,
					Optional: true,
				},
				"three": &Schema{
					Type:     TypeBool,
					Optional: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"one":   "false",
					"two":   "true",
					"three": "true",
				},
			},

			Config: map[string]interface{}{
				"one": "1",
				"two": "0",
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"one": &terraform.ResourceAttrDiff{
						Old: "false",
						New: "true",
					},
					"two": &terraform.ResourceAttrDiff{
						Old: "true",
						New: "false",
					},
					"three": &terraform.ResourceAttrDiff{
						Old:        "true",
						New:        "false",
						NewRemoved: true,
					},
				},
			},

			Err: false,
		},

		{
			Name:   "tainted in state w/ no attr changes is still a replacement",
			Schema: map[string]*Schema{},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"id": "someid",
				},
				Tainted: true,
			},

			Config: map[string]interface{}{},

			Diff: &terraform.InstanceDiff{
				Attributes:     map[string]*terraform.ResourceAttrDiff{},
				DestroyTainted: true,
			},

			Err: false,
		},

		{
			Name: "Set ForceNew only marks the changing element as ForceNew",
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeSet,
					Required: true,
					ForceNew: true,
					Elem:     &Schema{Type: TypeInt},
					Set: func(a interface{}) int {
						return a.(int)
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"ports.#": "3",
					"ports.1": "1",
					"ports.2": "2",
					"ports.4": "4",
				},
			},

			Config: map[string]interface{}{
				"ports": []interface{}{5, 2, 1},
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": &terraform.ResourceAttrDiff{
						Old: "3",
						New: "3",
					},
					"ports.1": &terraform.ResourceAttrDiff{
						Old: "1",
						New: "1",
					},
					"ports.2": &terraform.ResourceAttrDiff{
						Old: "2",
						New: "2",
					},
					"ports.5": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "5",
						RequiresNew: true,
					},
					"ports.4": &terraform.ResourceAttrDiff{
						Old:         "4",
						New:         "0",
						NewRemoved:  true,
						RequiresNew: true,
					},
				},
			},
		},

		{
			Name: "removed optional items should trigger ForceNew",
			Schema: map[string]*Schema{
				"description": &Schema{
					Type:     TypeString,
					ForceNew: true,
					Optional: true,
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"description": "foo",
				},
			},

			Config: map[string]interface{}{},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"description": &terraform.ResourceAttrDiff{
						Old:         "foo",
						New:         "",
						RequiresNew: true,
						NewRemoved:  true,
					},
				},
			},

			Err: false,
		},

		// GH-7715
		{
			Name: "computed value for boolean field",
			Schema: map[string]*Schema{
				"foo": &Schema{
					Type:     TypeBool,
					ForceNew: true,
					Computed: true,
					Optional: true,
				},
			},

			State: &terraform.InstanceState{},

			Config: map[string]interface{}{
				"foo": "${var.foo}",
			},

			ConfigVariables: map[string]ast.Variable{
				"var.foo": interfaceToVariableSwallowError(
					config.UnknownVariableValue),
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"foo": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "false",
						NewComputed: true,
						RequiresNew: true,
					},
				},
			},

			Err: false,
		},

		{
			Name: "Set ForceNew marks count as ForceNew if computed",
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeSet,
					Required: true,
					ForceNew: true,
					Elem:     &Schema{Type: TypeInt},
					Set: func(a interface{}) int {
						return a.(int)
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"ports.#": "3",
					"ports.1": "1",
					"ports.2": "2",
					"ports.4": "4",
				},
			},

			Config: map[string]interface{}{
				"ports": []interface{}{"${var.foo}", 2, 1},
			},

			ConfigVariables: map[string]ast.Variable{
				"var.foo": interfaceToVariableSwallowError(config.UnknownVariableValue),
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"ports.#": &terraform.ResourceAttrDiff{
						Old:         "3",
						New:         "",
						NewComputed: true,
						RequiresNew: true,
					},
				},
			},
		},

		{
			Name: "List with computed schema and ForceNew",
			Schema: map[string]*Schema{
				"config": &Schema{
					Type:     TypeList,
					Optional: true,
					ForceNew: true,
					Elem: &Schema{
						Type: TypeString,
					},
				},
			},

			State: &terraform.InstanceState{
				Attributes: map[string]string{
					"config.#": "2",
					"config.0": "a",
					"config.1": "b",
				},
			},

			Config: map[string]interface{}{
				"config": []interface{}{"${var.a}", "${var.b}"},
			},

			ConfigVariables: map[string]ast.Variable{
				"var.a": interfaceToVariableSwallowError(
					config.UnknownVariableValue),
				"var.b": interfaceToVariableSwallowError(
					config.UnknownVariableValue),
			},

			Diff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"config.#": &terraform.ResourceAttrDiff{
						Old:         "2",
						New:         "",
						RequiresNew: true,
						NewComputed: true,
					},
				},
			},

			Err: false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.Name), func(t *testing.T) {
			c, err := config.NewRawConfig(tc.Config)
			if err != nil {
				t.Fatalf("err: %s", err)
			}

			if len(tc.ConfigVariables) > 0 {
				if err := c.Interpolate(tc.ConfigVariables); err != nil {
					t.Fatalf("err: %s", err)
				}
			}

			d, err := schemaMap(tc.Schema).Diff(
				tc.State, terraform.NewResourceConfig(c))
			if err != nil != tc.Err {
				t.Fatalf("err: %s", err)
			}

			if !reflect.DeepEqual(tc.Diff, d) {
				t.Fatalf("expected:\n%#v\n\ngot:\n%#v", tc.Diff, d)
			}
		})
	}
}

func TestSchemaMap_Input(t *testing.T) {
	cases := map[string]struct {
		Schema map[string]*Schema
		Config map[string]interface{}
		Input  map[string]string
		Result map[string]interface{}
		Err    bool
	}{
		/*
		 * String decode
		 */

		"no input on optional field with no config": {
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Optional: true,
				},
			},

			Input:  map[string]string{},
			Result: map[string]interface{}{},
			Err:    false,
		},

		"input ignored when config has a value": {
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Optional: true,
				},
			},

			Config: map[string]interface{}{
				"availability_zone": "bar",
			},

			Input: map[string]string{
				"availability_zone": "foo",
			},

			Result: map[string]interface{}{},

			Err: false,
		},

		"input ignored when schema has a default": {
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Default:  "foo",
					Optional: true,
				},
			},

			Input: map[string]string{
				"availability_zone": "bar",
			},

			Result: map[string]interface{}{},

			Err: false,
		},

		"input ignored when default function returns a value": {
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type: TypeString,
					DefaultFunc: func() (interface{}, error) {
						return "foo", nil
					},
					Optional: true,
				},
			},

			Input: map[string]string{
				"availability_zone": "bar",
			},

			Result: map[string]interface{}{},

			Err: false,
		},

		"input ignored when default function returns an empty string": {
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Default:  "",
					Optional: true,
				},
			},

			Input: map[string]string{
				"availability_zone": "bar",
			},

			Result: map[string]interface{}{},

			Err: false,
		},

		"input used when default function returns nil": {
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type: TypeString,
					DefaultFunc: func() (interface{}, error) {
						return nil, nil
					},
					Required: true,
				},
			},

			Input: map[string]string{
				"availability_zone": "bar",
			},

			Result: map[string]interface{}{
				"availability_zone": "bar",
			},

			Err: false,
		},

		"input not used when optional default function returns nil": {
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type: TypeString,
					DefaultFunc: func() (interface{}, error) {
						return nil, nil
					},
					Optional: true,
				},
			},

			Input:  map[string]string{},
			Result: map[string]interface{}{},
			Err:    false,
		},
	}

	for i, tc := range cases {
		if tc.Config == nil {
			tc.Config = make(map[string]interface{})
		}

		c, err := config.NewRawConfig(tc.Config)
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		input := new(terraform.MockUIInput)
		input.InputReturnMap = tc.Input

		rc := terraform.NewResourceConfig(c)
		rc.Config = make(map[string]interface{})

		actual, err := schemaMap(tc.Schema).Input(input, rc)
		if err != nil != tc.Err {
			t.Fatalf("#%v err: %s", i, err)
		}

		if !reflect.DeepEqual(tc.Result, actual.Config) {
			t.Fatalf("#%v: bad:\n\ngot: %#v\nexpected: %#v", i, actual.Config, tc.Result)
		}
	}
}

func TestSchemaMap_InputDefault(t *testing.T) {
	emptyConfig := make(map[string]interface{})
	c, err := config.NewRawConfig(emptyConfig)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	rc := terraform.NewResourceConfig(c)
	rc.Config = make(map[string]interface{})

	input := new(terraform.MockUIInput)
	input.InputFn = func(opts *terraform.InputOpts) (string, error) {
		t.Fatalf("InputFn should not be called on: %#v", opts)
		return "", nil
	}

	schema := map[string]*Schema{
		"availability_zone": &Schema{
			Type:     TypeString,
			Default:  "foo",
			Optional: true,
		},
	}
	actual, err := schemaMap(schema).Input(input, rc)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := map[string]interface{}{}

	if !reflect.DeepEqual(expected, actual.Config) {
		t.Fatalf("got: %#v\nexpected: %#v", actual.Config, expected)
	}
}

func TestSchemaMap_InputDeprecated(t *testing.T) {
	emptyConfig := make(map[string]interface{})
	c, err := config.NewRawConfig(emptyConfig)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	rc := terraform.NewResourceConfig(c)
	rc.Config = make(map[string]interface{})

	input := new(terraform.MockUIInput)
	input.InputFn = func(opts *terraform.InputOpts) (string, error) {
		t.Fatalf("InputFn should not be called on: %#v", opts)
		return "", nil
	}

	schema := map[string]*Schema{
		"availability_zone": &Schema{
			Type:       TypeString,
			Deprecated: "long gone",
			Optional:   true,
		},
	}
	actual, err := schemaMap(schema).Input(input, rc)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := map[string]interface{}{}

	if !reflect.DeepEqual(expected, actual.Config) {
		t.Fatalf("got: %#v\nexpected: %#v", actual.Config, expected)
	}
}

func TestSchemaMap_InternalValidate(t *testing.T) {
	cases := map[string]struct {
		In  map[string]*Schema
		Err bool
	}{
		"nothing": {
			nil,
			false,
		},

		"Both optional and required": {
			map[string]*Schema{
				"foo": &Schema{
					Type:     TypeInt,
					Optional: true,
					Required: true,
				},
			},
			true,
		},

		"No optional and no required": {
			map[string]*Schema{
				"foo": &Schema{
					Type: TypeInt,
				},
			},
			true,
		},

		"Missing Type": {
			map[string]*Schema{
				"foo": &Schema{
					Required: true,
				},
			},
			true,
		},

		"Required but computed": {
			map[string]*Schema{
				"foo": &Schema{
					Type:     TypeInt,
					Required: true,
					Computed: true,
				},
			},
			true,
		},

		"Looks good": {
			map[string]*Schema{
				"foo": &Schema{
					Type:     TypeString,
					Required: true,
				},
			},
			false,
		},

		"Computed but has default": {
			map[string]*Schema{
				"foo": &Schema{
					Type:     TypeInt,
					Optional: true,
					Computed: true,
					Default:  "foo",
				},
			},
			true,
		},

		"Required but has default": {
			map[string]*Schema{
				"foo": &Schema{
					Type:     TypeInt,
					Optional: true,
					Required: true,
					Default:  "foo",
				},
			},
			true,
		},

		"List element not set": {
			map[string]*Schema{
				"foo": &Schema{
					Type: TypeList,
				},
			},
			true,
		},

		"List default": {
			map[string]*Schema{
				"foo": &Schema{
					Type:    TypeList,
					Elem:    &Schema{Type: TypeInt},
					Default: "foo",
				},
			},
			true,
		},

		"List element computed": {
			map[string]*Schema{
				"foo": &Schema{
					Type:     TypeList,
					Optional: true,
					Elem: &Schema{
						Type:     TypeInt,
						Computed: true,
					},
				},
			},
			true,
		},

		"List element with Set set": {
			map[string]*Schema{
				"foo": &Schema{
					Type:     TypeList,
					Elem:     &Schema{Type: TypeInt},
					Set:      func(interface{}) int { return 0 },
					Optional: true,
				},
			},
			true,
		},

		"Set element with no Set set": {
			map[string]*Schema{
				"foo": &Schema{
					Type:     TypeSet,
					Elem:     &Schema{Type: TypeInt},
					Optional: true,
				},
			},
			false,
		},

		"Required but computedWhen": {
			map[string]*Schema{
				"foo": &Schema{
					Type:         TypeInt,
					Required:     true,
					ComputedWhen: []string{"foo"},
				},
			},
			true,
		},

		"Conflicting attributes cannot be required": {
			map[string]*Schema{
				"blacklist": &Schema{
					Type:     TypeBool,
					Required: true,
				},
				"whitelist": &Schema{
					Type:          TypeBool,
					Optional:      true,
					ConflictsWith: []string{"blacklist"},
				},
			},
			true,
		},

		"Attribute with conflicts cannot be required": {
			map[string]*Schema{
				"whitelist": &Schema{
					Type:          TypeBool,
					Required:      true,
					ConflictsWith: []string{"blacklist"},
				},
			},
			true,
		},

		"ConflictsWith cannot be used w/ ComputedWhen": {
			map[string]*Schema{
				"blacklist": &Schema{
					Type:         TypeBool,
					ComputedWhen: []string{"foor"},
				},
				"whitelist": &Schema{
					Type:          TypeBool,
					Required:      true,
					ConflictsWith: []string{"blacklist"},
				},
			},
			true,
		},

		"Sub-resource invalid": {
			map[string]*Schema{
				"foo": &Schema{
					Type:     TypeList,
					Optional: true,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"foo": new(Schema),
						},
					},
				},
			},
			true,
		},

		"Sub-resource valid": {
			map[string]*Schema{
				"foo": &Schema{
					Type:     TypeList,
					Optional: true,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"foo": &Schema{
								Type:     TypeInt,
								Optional: true,
							},
						},
					},
				},
			},
			false,
		},

		"ValidateFunc on non-primitive": {
			map[string]*Schema{
				"foo": &Schema{
					Type:     TypeSet,
					Required: true,
					ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
						return
					},
				},
			},
			true,
		},

		"computed-only field with validateFunc": {
			map[string]*Schema{
				"string": &Schema{
					Type:     TypeString,
					Computed: true,
					ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
						es = append(es, fmt.Errorf("this is not fine"))
						return
					},
				},
			},
			true,
		},

		"computed-only field with diffSuppressFunc": {
			map[string]*Schema{
				"string": &Schema{
					Type:     TypeString,
					Computed: true,
					DiffSuppressFunc: func(k, old, new string, d *ResourceData) bool {
						// Always suppress any diff
						return false
					},
				},
			},
			true,
		},

		"invalid field name format #1": {
			map[string]*Schema{
				"with space": &Schema{
					Type:     TypeString,
					Optional: true,
				},
			},
			true,
		},

		"invalid field name format #2": {
			map[string]*Schema{
				"WithCapitals": &Schema{
					Type:     TypeString,
					Optional: true,
				},
			},
			true,
		},

		"invalid field name format of a Deprecated field": {
			map[string]*Schema{
				"WithCapitals": &Schema{
					Type:       TypeString,
					Optional:   true,
					Deprecated: "Use with_underscores instead",
				},
			},
			false,
		},

		"invalid field name format of a Removed field": {
			map[string]*Schema{
				"WithCapitals": &Schema{
					Type:     TypeString,
					Optional: true,
					Removed:  "Use with_underscores instead",
				},
			},
			false,
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			err := schemaMap(tc.In).InternalValidate(nil)
			if err != nil != tc.Err {
				if tc.Err {
					t.Fatalf("%q: Expected error did not occur:\n\n%#v", tn, tc.In)
				}
				t.Fatalf("%q: Unexpected error occurred: %s\n\n%#v", tn, err, tc.In)
			}
		})
	}

}

func TestSchemaMap_DiffSuppress(t *testing.T) {
	cases := map[string]struct {
		Schema          map[string]*Schema
		State           *terraform.InstanceState
		Config          map[string]interface{}
		ConfigVariables map[string]ast.Variable
		ExpectedDiff    *terraform.InstanceDiff
		Err             bool
	}{
		"#0 - Suppress otherwise valid diff by returning true": {
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
					DiffSuppressFunc: func(k, old, new string, d *ResourceData) bool {
						// Always suppress any diff
						return true
					},
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"availability_zone": "foo",
			},

			ExpectedDiff: nil,

			Err: false,
		},

		"#1 - Don't suppress diff by returning false": {
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
					DiffSuppressFunc: func(k, old, new string, d *ResourceData) bool {
						// Always suppress any diff
						return false
					},
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"availability_zone": "foo",
			},

			ExpectedDiff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": {
						Old: "",
						New: "foo",
					},
				},
			},

			Err: false,
		},

		"Default with suppress makes no diff": {
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
					Default:  "foo",
					DiffSuppressFunc: func(k, old, new string, d *ResourceData) bool {
						return true
					},
				},
			},

			State: nil,

			Config: map[string]interface{}{},

			ExpectedDiff: nil,

			Err: false,
		},

		"Default with false suppress makes diff": {
			Schema: map[string]*Schema{
				"availability_zone": {
					Type:     TypeString,
					Optional: true,
					Default:  "foo",
					DiffSuppressFunc: func(k, old, new string, d *ResourceData) bool {
						return false
					},
				},
			},

			State: nil,

			Config: map[string]interface{}{},

			ExpectedDiff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"availability_zone": {
						Old: "",
						New: "foo",
					},
				},
			},

			Err: false,
		},

		"Complex structure with set of computed string should mark root set as computed": {
			Schema: map[string]*Schema{
				"outer": &Schema{
					Type:     TypeSet,
					Optional: true,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"outer_str": &Schema{
								Type:     TypeString,
								Optional: true,
							},
							"inner": &Schema{
								Type:     TypeSet,
								Optional: true,
								Elem: &Resource{
									Schema: map[string]*Schema{
										"inner_str": &Schema{
											Type:     TypeString,
											Optional: true,
										},
									},
								},
								Set: func(v interface{}) int {
									return 2
								},
							},
						},
					},
					Set: func(v interface{}) int {
						return 1
					},
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"outer": []map[string]interface{}{
					map[string]interface{}{
						"outer_str": "foo",
						"inner": []map[string]interface{}{
							map[string]interface{}{
								"inner_str": "${var.bar}",
							},
						},
					},
				},
			},

			ConfigVariables: map[string]ast.Variable{
				"var.bar": interfaceToVariableSwallowError(config.UnknownVariableValue),
			},

			ExpectedDiff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"outer.#": &terraform.ResourceAttrDiff{
						Old: "0",
						New: "1",
					},
					"outer.~1.outer_str": &terraform.ResourceAttrDiff{
						Old: "",
						New: "foo",
					},
					"outer.~1.inner.#": &terraform.ResourceAttrDiff{
						Old: "0",
						New: "1",
					},
					"outer.~1.inner.~2.inner_str": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "${var.bar}",
						NewComputed: true,
					},
				},
			},

			Err: false,
		},

		"Complex structure with complex list of computed string should mark root set as computed": {
			Schema: map[string]*Schema{
				"outer": &Schema{
					Type:     TypeSet,
					Optional: true,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"outer_str": &Schema{
								Type:     TypeString,
								Optional: true,
							},
							"inner": &Schema{
								Type:     TypeList,
								Optional: true,
								Elem: &Resource{
									Schema: map[string]*Schema{
										"inner_str": &Schema{
											Type:     TypeString,
											Optional: true,
										},
									},
								},
							},
						},
					},
					Set: func(v interface{}) int {
						return 1
					},
				},
			},

			State: nil,

			Config: map[string]interface{}{
				"outer": []map[string]interface{}{
					map[string]interface{}{
						"outer_str": "foo",
						"inner": []map[string]interface{}{
							map[string]interface{}{
								"inner_str": "${var.bar}",
							},
						},
					},
				},
			},

			ConfigVariables: map[string]ast.Variable{
				"var.bar": interfaceToVariableSwallowError(config.UnknownVariableValue),
			},

			ExpectedDiff: &terraform.InstanceDiff{
				Attributes: map[string]*terraform.ResourceAttrDiff{
					"outer.#": &terraform.ResourceAttrDiff{
						Old: "0",
						New: "1",
					},
					"outer.~1.outer_str": &terraform.ResourceAttrDiff{
						Old: "",
						New: "foo",
					},
					"outer.~1.inner.#": &terraform.ResourceAttrDiff{
						Old: "0",
						New: "1",
					},
					"outer.~1.inner.0.inner_str": &terraform.ResourceAttrDiff{
						Old:         "",
						New:         "${var.bar}",
						NewComputed: true,
					},
				},
			},

			Err: false,
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			c, err := config.NewRawConfig(tc.Config)
			if err != nil {
				t.Fatalf("#%q err: %s", tn, err)
			}

			if len(tc.ConfigVariables) > 0 {
				if err := c.Interpolate(tc.ConfigVariables); err != nil {
					t.Fatalf("#%q err: %s", tn, err)
				}
			}

			d, err := schemaMap(tc.Schema).Diff(
				tc.State, terraform.NewResourceConfig(c))
			if err != nil != tc.Err {
				t.Fatalf("#%q err: %s", tn, err)
			}

			if !reflect.DeepEqual(tc.ExpectedDiff, d) {
				t.Fatalf("#%q:\n\nexpected:\n%#v\n\ngot:\n%#v", tn, tc.ExpectedDiff, d)
			}
		})
	}
}

func TestSchemaMap_Validate(t *testing.T) {
	cases := map[string]struct {
		Schema   map[string]*Schema
		Config   map[string]interface{}
		Vars     map[string]string
		Err      bool
		Errors   []error
		Warnings []string
	}{
		"Good": {
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			Config: map[string]interface{}{
				"availability_zone": "foo",
			},
		},

		"Good, because the var is not set and that error will come elsewhere": {
			Schema: map[string]*Schema{
				"size": &Schema{
					Type:     TypeInt,
					Required: true,
				},
			},

			Config: map[string]interface{}{
				"size": "${var.foo}",
			},

			Vars: map[string]string{
				"var.foo": config.UnknownVariableValue,
			},
		},

		"Required field not set": {
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Required: true,
				},
			},

			Config: map[string]interface{}{},

			Err: true,
		},

		"Invalid basic type": {
			Schema: map[string]*Schema{
				"port": &Schema{
					Type:     TypeInt,
					Required: true,
				},
			},

			Config: map[string]interface{}{
				"port": "I am invalid",
			},

			Err: true,
		},

		"Invalid complex type": {
			Schema: map[string]*Schema{
				"user_data": &Schema{
					Type:     TypeString,
					Optional: true,
				},
			},

			Config: map[string]interface{}{
				"user_data": []interface{}{
					map[string]interface{}{
						"foo": "bar",
					},
				},
			},

			Err: true,
		},

		"Bad type, interpolated": {
			Schema: map[string]*Schema{
				"size": &Schema{
					Type:     TypeInt,
					Required: true,
				},
			},

			Config: map[string]interface{}{
				"size": "${var.foo}",
			},

			Vars: map[string]string{
				"var.foo": "nope",
			},

			Err: true,
		},

		"Required but has DefaultFunc": {
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Required: true,
					DefaultFunc: func() (interface{}, error) {
						return "foo", nil
					},
				},
			},

			Config: nil,
		},

		"Required but has DefaultFunc return nil": {
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Required: true,
					DefaultFunc: func() (interface{}, error) {
						return nil, nil
					},
				},
			},

			Config: nil,

			Err: true,
		},

		"List with promotion": {
			Schema: map[string]*Schema{
				"ingress": &Schema{
					Type:          TypeList,
					Elem:          &Schema{Type: TypeInt},
					PromoteSingle: true,
					Optional:      true,
				},
			},

			Config: map[string]interface{}{
				"ingress": "5",
			},

			Err: false,
		},

		"List with promotion set as list": {
			Schema: map[string]*Schema{
				"ingress": &Schema{
					Type:          TypeList,
					Elem:          &Schema{Type: TypeInt},
					PromoteSingle: true,
					Optional:      true,
				},
			},

			Config: map[string]interface{}{
				"ingress": []interface{}{"5"},
			},

			Err: false,
		},

		"Optional sub-resource": {
			Schema: map[string]*Schema{
				"ingress": &Schema{
					Type: TypeList,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"from": &Schema{
								Type:     TypeInt,
								Required: true,
							},
						},
					},
				},
			},

			Config: map[string]interface{}{},

			Err: false,
		},

		"Sub-resource is the wrong type": {
			Schema: map[string]*Schema{
				"ingress": &Schema{
					Type:     TypeList,
					Required: true,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"from": &Schema{
								Type:     TypeInt,
								Required: true,
							},
						},
					},
				},
			},

			Config: map[string]interface{}{
				"ingress": []interface{}{"foo"},
			},

			Err: true,
		},

		"Not a list": {
			Schema: map[string]*Schema{
				"ingress": &Schema{
					Type: TypeList,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"from": &Schema{
								Type:     TypeInt,
								Required: true,
							},
						},
					},
				},
			},

			Config: map[string]interface{}{
				"ingress": "foo",
			},

			Err: true,
		},

		"Required sub-resource field": {
			Schema: map[string]*Schema{
				"ingress": &Schema{
					Type: TypeList,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"from": &Schema{
								Type:     TypeInt,
								Required: true,
							},
						},
					},
				},
			},

			Config: map[string]interface{}{
				"ingress": []interface{}{
					map[string]interface{}{},
				},
			},

			Err: true,
		},

		"Good sub-resource": {
			Schema: map[string]*Schema{
				"ingress": &Schema{
					Type:     TypeList,
					Optional: true,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"from": &Schema{
								Type:     TypeInt,
								Required: true,
							},
						},
					},
				},
			},

			Config: map[string]interface{}{
				"ingress": []interface{}{
					map[string]interface{}{
						"from": 80,
					},
				},
			},

			Err: false,
		},

		"Good sub-resource, interpolated value": {
			Schema: map[string]*Schema{
				"ingress": &Schema{
					Type:     TypeList,
					Optional: true,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"from": &Schema{
								Type:     TypeInt,
								Required: true,
							},
						},
					},
				},
			},

			Config: map[string]interface{}{
				"ingress": []interface{}{
					`${map("from", "80")}`,
				},
			},

			Vars: map[string]string{},

			Err: false,
		},

		"Good sub-resource, computed value": {
			Schema: map[string]*Schema{
				"ingress": &Schema{
					Type:     TypeList,
					Optional: true,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"from": &Schema{
								Type:     TypeInt,
								Optional: true,
							},
						},
					},
				},
			},

			Config: map[string]interface{}{
				"ingress": []interface{}{
					`${map("from", var.port)}`,
				},
			},

			Vars: map[string]string{
				"var.port": config.UnknownVariableValue,
			},

			Err: false,
		},

		"Invalid/unknown field": {
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			Config: map[string]interface{}{
				"foo": "bar",
			},

			Err: true,
		},

		"Invalid/unknown field with computed value": {
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			Config: map[string]interface{}{
				"foo": "${var.foo}",
			},

			Vars: map[string]string{
				"var.foo": config.UnknownVariableValue,
			},

			Err: true,
		},

		"Computed field set": {
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Computed: true,
				},
			},

			Config: map[string]interface{}{
				"availability_zone": "bar",
			},

			Err: true,
		},

		"Not a set": {
			Schema: map[string]*Schema{
				"ports": &Schema{
					Type:     TypeSet,
					Required: true,
					Elem:     &Schema{Type: TypeInt},
					Set: func(a interface{}) int {
						return a.(int)
					},
				},
			},

			Config: map[string]interface{}{
				"ports": "foo",
			},

			Err: true,
		},

		"Maps": {
			Schema: map[string]*Schema{
				"user_data": &Schema{
					Type:     TypeMap,
					Optional: true,
				},
			},

			Config: map[string]interface{}{
				"user_data": "foo",
			},

			Err: true,
		},

		"Good map: data surrounded by extra slice": {
			Schema: map[string]*Schema{
				"user_data": &Schema{
					Type:     TypeMap,
					Optional: true,
				},
			},

			Config: map[string]interface{}{
				"user_data": []interface{}{
					map[string]interface{}{
						"foo": "bar",
					},
				},
			},
		},

		"Good map": {
			Schema: map[string]*Schema{
				"user_data": &Schema{
					Type:     TypeMap,
					Optional: true,
				},
			},

			Config: map[string]interface{}{
				"user_data": map[string]interface{}{
					"foo": "bar",
				},
			},
		},

		"Bad map: just a slice": {
			Schema: map[string]*Schema{
				"user_data": &Schema{
					Type:     TypeMap,
					Optional: true,
				},
			},

			Config: map[string]interface{}{
				"user_data": []interface{}{
					"foo",
				},
			},

			Err: true,
		},

		"Good set: config has slice with single interpolated value": {
			Schema: map[string]*Schema{
				"security_groups": &Schema{
					Type:     TypeSet,
					Optional: true,
					Computed: true,
					ForceNew: true,
					Elem:     &Schema{Type: TypeString},
					Set: func(v interface{}) int {
						return len(v.(string))
					},
				},
			},

			Config: map[string]interface{}{
				"security_groups": []interface{}{"${var.foo}"},
			},

			Err: false,
		},

		"Bad set: config has single interpolated value": {
			Schema: map[string]*Schema{
				"security_groups": &Schema{
					Type:     TypeSet,
					Optional: true,
					Computed: true,
					ForceNew: true,
					Elem:     &Schema{Type: TypeString},
				},
			},

			Config: map[string]interface{}{
				"security_groups": "${var.foo}",
			},

			Err: true,
		},

		"Bad, subresource should not allow unknown elements": {
			Schema: map[string]*Schema{
				"ingress": &Schema{
					Type:     TypeList,
					Optional: true,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"port": &Schema{
								Type:     TypeInt,
								Required: true,
							},
						},
					},
				},
			},

			Config: map[string]interface{}{
				"ingress": []interface{}{
					map[string]interface{}{
						"port":  80,
						"other": "yes",
					},
				},
			},

			Err: true,
		},

		"Bad, subresource should not allow invalid types": {
			Schema: map[string]*Schema{
				"ingress": &Schema{
					Type:     TypeList,
					Optional: true,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"port": &Schema{
								Type:     TypeInt,
								Required: true,
							},
						},
					},
				},
			},

			Config: map[string]interface{}{
				"ingress": []interface{}{
					map[string]interface{}{
						"port": "bad",
					},
				},
			},

			Err: true,
		},

		"Bad, should not allow lists to be assigned to string attributes": {
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Required: true,
				},
			},

			Config: map[string]interface{}{
				"availability_zone": []interface{}{"foo", "bar", "baz"},
			},

			Err: true,
		},

		"Bad, should not allow maps to be assigned to string attributes": {
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Required: true,
				},
			},

			Config: map[string]interface{}{
				"availability_zone": map[string]interface{}{"foo": "bar", "baz": "thing"},
			},

			Err: true,
		},

		"Deprecated attribute usage generates warning, but not error": {
			Schema: map[string]*Schema{
				"old_news": &Schema{
					Type:       TypeString,
					Optional:   true,
					Deprecated: "please use 'new_news' instead",
				},
			},

			Config: map[string]interface{}{
				"old_news": "extra extra!",
			},

			Err: false,

			Warnings: []string{
				"\"old_news\": [DEPRECATED] please use 'new_news' instead",
			},
		},

		"Deprecated generates no warnings if attr not used": {
			Schema: map[string]*Schema{
				"old_news": &Schema{
					Type:       TypeString,
					Optional:   true,
					Deprecated: "please use 'new_news' instead",
				},
			},

			Err: false,

			Warnings: nil,
		},

		"Removed attribute usage generates error": {
			Schema: map[string]*Schema{
				"long_gone": &Schema{
					Type:     TypeString,
					Optional: true,
					Removed:  "no longer supported by Cloud API",
				},
			},

			Config: map[string]interface{}{
				"long_gone": "still here!",
			},

			Err: true,
			Errors: []error{
				fmt.Errorf("\"long_gone\": [REMOVED] no longer supported by Cloud API"),
			},
		},

		"Removed generates no errors if attr not used": {
			Schema: map[string]*Schema{
				"long_gone": &Schema{
					Type:     TypeString,
					Optional: true,
					Removed:  "no longer supported by Cloud API",
				},
			},

			Err: false,
		},

		"Conflicting attributes generate error": {
			Schema: map[string]*Schema{
				"whitelist": &Schema{
					Type:     TypeString,
					Optional: true,
				},
				"blacklist": &Schema{
					Type:          TypeString,
					Optional:      true,
					ConflictsWith: []string{"whitelist"},
				},
			},

			Config: map[string]interface{}{
				"whitelist": "white-val",
				"blacklist": "black-val",
			},

			Err: true,
			Errors: []error{
				fmt.Errorf("\"blacklist\": conflicts with whitelist (\"white-val\")"),
			},
		},

		"Required attribute & undefined conflicting optional are good": {
			Schema: map[string]*Schema{
				"required_att": &Schema{
					Type:     TypeString,
					Required: true,
				},
				"optional_att": &Schema{
					Type:          TypeString,
					Optional:      true,
					ConflictsWith: []string{"required_att"},
				},
			},

			Config: map[string]interface{}{
				"required_att": "required-val",
			},

			Err: false,
		},

		"Required conflicting attribute & defined optional generate error": {
			Schema: map[string]*Schema{
				"required_att": &Schema{
					Type:     TypeString,
					Required: true,
				},
				"optional_att": &Schema{
					Type:          TypeString,
					Optional:      true,
					ConflictsWith: []string{"required_att"},
				},
			},

			Config: map[string]interface{}{
				"required_att": "required-val",
				"optional_att": "optional-val",
			},

			Err: true,
			Errors: []error{
				fmt.Errorf(`"optional_att": conflicts with required_att ("required-val")`),
			},
		},

		"Computed + Optional fields conflicting with each other": {
			Schema: map[string]*Schema{
				"foo_att": &Schema{
					Type:          TypeString,
					Optional:      true,
					Computed:      true,
					ConflictsWith: []string{"bar_att"},
				},
				"bar_att": &Schema{
					Type:          TypeString,
					Optional:      true,
					Computed:      true,
					ConflictsWith: []string{"foo_att"},
				},
			},

			Config: map[string]interface{}{
				"foo_att": "foo-val",
				"bar_att": "bar-val",
			},

			Err: true,
			Errors: []error{
				fmt.Errorf(`"foo_att": conflicts with bar_att ("bar-val")`),
				fmt.Errorf(`"bar_att": conflicts with foo_att ("foo-val")`),
			},
		},

		"Computed + Optional fields NOT conflicting with each other": {
			Schema: map[string]*Schema{
				"foo_att": &Schema{
					Type:          TypeString,
					Optional:      true,
					Computed:      true,
					ConflictsWith: []string{"bar_att"},
				},
				"bar_att": &Schema{
					Type:          TypeString,
					Optional:      true,
					Computed:      true,
					ConflictsWith: []string{"foo_att"},
				},
			},

			Config: map[string]interface{}{
				"foo_att": "foo-val",
			},

			Err: false,
		},

		"Computed + Optional fields that conflict with none set": {
			Schema: map[string]*Schema{
				"foo_att": &Schema{
					Type:          TypeString,
					Optional:      true,
					Computed:      true,
					ConflictsWith: []string{"bar_att"},
				},
				"bar_att": &Schema{
					Type:          TypeString,
					Optional:      true,
					Computed:      true,
					ConflictsWith: []string{"foo_att"},
				},
			},

			Config: map[string]interface{}{},

			Err: false,
		},

		"Good with ValidateFunc": {
			Schema: map[string]*Schema{
				"validate_me": &Schema{
					Type:     TypeString,
					Required: true,
					ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
						return
					},
				},
			},
			Config: map[string]interface{}{
				"validate_me": "valid",
			},
			Err: false,
		},

		"Bad with ValidateFunc": {
			Schema: map[string]*Schema{
				"validate_me": &Schema{
					Type:     TypeString,
					Required: true,
					ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
						es = append(es, fmt.Errorf("something is not right here"))
						return
					},
				},
			},
			Config: map[string]interface{}{
				"validate_me": "invalid",
			},
			Err: true,
			Errors: []error{
				fmt.Errorf(`something is not right here`),
			},
		},

		"ValidateFunc not called when type does not match": {
			Schema: map[string]*Schema{
				"number": &Schema{
					Type:     TypeInt,
					Required: true,
					ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
						t.Fatalf("Should not have gotten validate call")
						return
					},
				},
			},
			Config: map[string]interface{}{
				"number": "NaN",
			},
			Err: true,
		},

		"ValidateFunc gets decoded type": {
			Schema: map[string]*Schema{
				"maybe": &Schema{
					Type:     TypeBool,
					Required: true,
					ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
						if _, ok := v.(bool); !ok {
							t.Fatalf("Expected bool, got: %#v", v)
						}
						return
					},
				},
			},
			Config: map[string]interface{}{
				"maybe": "true",
			},
		},

		"ValidateFunc is not called with a computed value": {
			Schema: map[string]*Schema{
				"validate_me": &Schema{
					Type:     TypeString,
					Required: true,
					ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
						es = append(es, fmt.Errorf("something is not right here"))
						return
					},
				},
			},
			Config: map[string]interface{}{
				"validate_me": "${var.foo}",
			},
			Vars: map[string]string{
				"var.foo": config.UnknownVariableValue,
			},

			Err: false,
		},

		"special timeouts field": {
			Schema: map[string]*Schema{
				"availability_zone": &Schema{
					Type:     TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},
			},

			Config: map[string]interface{}{
				TimeoutsConfigKey: "bar",
			},

			Err: false,
		},

		"invalid bool field": {
			Schema: map[string]*Schema{
				"bool_field": {
					Type:     TypeBool,
					Optional: true,
				},
			},
			Config: map[string]interface{}{
				"bool_field": "abcdef",
			},
			Err: true,
		},
		"invalid integer field": {
			Schema: map[string]*Schema{
				"integer_field": {
					Type:     TypeInt,
					Optional: true,
				},
			},
			Config: map[string]interface{}{
				"integer_field": "abcdef",
			},
			Err: true,
		},
		"invalid float field": {
			Schema: map[string]*Schema{
				"float_field": {
					Type:     TypeFloat,
					Optional: true,
				},
			},
			Config: map[string]interface{}{
				"float_field": "abcdef",
			},
			Err: true,
		},

		// Invalid map values
		"invalid bool map value": {
			Schema: map[string]*Schema{
				"boolMap": &Schema{
					Type:     TypeMap,
					Elem:     TypeBool,
					Optional: true,
				},
			},
			Config: map[string]interface{}{
				"boolMap": map[string]interface{}{
					"boolField": "notbool",
				},
			},
			Err: true,
		},
		"invalid int map value": {
			Schema: map[string]*Schema{
				"intMap": &Schema{
					Type:     TypeMap,
					Elem:     TypeInt,
					Optional: true,
				},
			},
			Config: map[string]interface{}{
				"intMap": map[string]interface{}{
					"intField": "notInt",
				},
			},
			Err: true,
		},
		"invalid float map value": {
			Schema: map[string]*Schema{
				"floatMap": &Schema{
					Type:     TypeMap,
					Elem:     TypeFloat,
					Optional: true,
				},
			},
			Config: map[string]interface{}{
				"floatMap": map[string]interface{}{
					"floatField": "notFloat",
				},
			},
			Err: true,
		},

		"map with positive validate function": {
			Schema: map[string]*Schema{
				"floatInt": &Schema{
					Type:     TypeMap,
					Elem:     TypeInt,
					Optional: true,
					ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
						return
					},
				},
			},
			Config: map[string]interface{}{
				"floatInt": map[string]interface{}{
					"rightAnswer": "42",
					"tooMuch":     "43",
				},
			},
			Err: false,
		},
		"map with negative validate function": {
			Schema: map[string]*Schema{
				"floatInt": &Schema{
					Type:     TypeMap,
					Elem:     TypeInt,
					Optional: true,
					ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
						es = append(es, fmt.Errorf("this is not fine"))
						return
					},
				},
			},
			Config: map[string]interface{}{
				"floatInt": map[string]interface{}{
					"rightAnswer": "42",
					"tooMuch":     "43",
				},
			},
			Err: true,
		},

		// The Validation function should not see interpolation strings from
		// non-computed values.
		"set with partially computed list and map": {
			Schema: map[string]*Schema{
				"outer": &Schema{
					Type:     TypeSet,
					Optional: true,
					Computed: true,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"list": &Schema{
								Type:     TypeList,
								Optional: true,
								Elem: &Schema{
									Type: TypeString,
									ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
										if strings.HasPrefix(v.(string), "${") {
											es = append(es, fmt.Errorf("should not have interpolations"))
										}
										return
									},
								},
							},
						},
					},
				},
			},
			Config: map[string]interface{}{
				"outer": []map[string]interface{}{
					{
						"list": []interface{}{"${var.a}", "${var.b}", "c"},
					},
				},
			},
			Vars: map[string]string{
				"var.a": "A",
				"var.b": config.UnknownVariableValue,
			},
			Err: false,
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			c, err := config.NewRawConfig(tc.Config)
			if err != nil {
				t.Fatalf("err: %s", err)
			}
			if tc.Vars != nil {
				vars := make(map[string]ast.Variable)
				for k, v := range tc.Vars {
					vars[k] = ast.Variable{Value: v, Type: ast.TypeString}
				}

				if err := c.Interpolate(vars); err != nil {
					t.Fatalf("err: %s", err)
				}
			}

			ws, es := schemaMap(tc.Schema).Validate(terraform.NewResourceConfig(c))
			if len(es) > 0 != tc.Err {
				if len(es) == 0 {
					t.Errorf("%q: no errors", tn)
				}

				for _, e := range es {
					t.Errorf("%q: err: %s", tn, e)
				}

				t.FailNow()
			}

			if !reflect.DeepEqual(ws, tc.Warnings) {
				t.Fatalf("%q: warnings:\n\nexpected: %#v\ngot:%#v", tn, tc.Warnings, ws)
			}

			if tc.Errors != nil {
				sort.Sort(errorSort(es))
				sort.Sort(errorSort(tc.Errors))

				if !reflect.DeepEqual(es, tc.Errors) {
					t.Fatalf("%q: errors:\n\nexpected: %q\ngot: %q", tn, tc.Errors, es)
				}
			}
		})

	}
}

func TestSchemaSet_ValidateMaxItems(t *testing.T) {
	cases := map[string]struct {
		Schema          map[string]*Schema
		State           *terraform.InstanceState
		Config          map[string]interface{}
		ConfigVariables map[string]string
		Diff            *terraform.InstanceDiff
		Err             bool
		Errors          []error
	}{
		"#0": {
			Schema: map[string]*Schema{
				"aliases": &Schema{
					Type:     TypeSet,
					Optional: true,
					MaxItems: 1,
					Elem:     &Schema{Type: TypeString},
				},
			},
			State: nil,
			Config: map[string]interface{}{
				"aliases": []interface{}{"foo", "bar"},
			},
			Diff: nil,
			Err:  true,
			Errors: []error{
				fmt.Errorf("aliases: attribute supports 1 item maximum, config has 2 declared"),
			},
		},
		"#1": {
			Schema: map[string]*Schema{
				"aliases": &Schema{
					Type:     TypeSet,
					Optional: true,
					Elem:     &Schema{Type: TypeString},
				},
			},
			State: nil,
			Config: map[string]interface{}{
				"aliases": []interface{}{"foo", "bar"},
			},
			Diff:   nil,
			Err:    false,
			Errors: nil,
		},
		"#2": {
			Schema: map[string]*Schema{
				"aliases": &Schema{
					Type:     TypeSet,
					Optional: true,
					MaxItems: 1,
					Elem:     &Schema{Type: TypeString},
				},
			},
			State: nil,
			Config: map[string]interface{}{
				"aliases": []interface{}{"foo"},
			},
			Diff:   nil,
			Err:    false,
			Errors: nil,
		},
	}

	for tn, tc := range cases {
		c, err := config.NewRawConfig(tc.Config)
		if err != nil {
			t.Fatalf("%q: err: %s", tn, err)
		}
		_, es := schemaMap(tc.Schema).Validate(terraform.NewResourceConfig(c))

		if len(es) > 0 != tc.Err {
			if len(es) == 0 {
				t.Errorf("%q: no errors", tn)
			}

			for _, e := range es {
				t.Errorf("%q: err: %s", tn, e)
			}

			t.FailNow()
		}

		if tc.Errors != nil {
			if !reflect.DeepEqual(es, tc.Errors) {
				t.Fatalf("%q: expected: %q\ngot: %q", tn, tc.Errors, es)
			}
		}
	}
}

func TestSchemaSet_ValidateMinItems(t *testing.T) {
	cases := map[string]struct {
		Schema          map[string]*Schema
		State           *terraform.InstanceState
		Config          map[string]interface{}
		ConfigVariables map[string]string
		Diff            *terraform.InstanceDiff
		Err             bool
		Errors          []error
	}{
		"#0": {
			Schema: map[string]*Schema{
				"aliases": &Schema{
					Type:     TypeSet,
					Optional: true,
					MinItems: 2,
					Elem:     &Schema{Type: TypeString},
				},
			},
			State: nil,
			Config: map[string]interface{}{
				"aliases": []interface{}{"foo", "bar"},
			},
			Diff:   nil,
			Err:    false,
			Errors: nil,
		},
		"#1": {
			Schema: map[string]*Schema{
				"aliases": &Schema{
					Type:     TypeSet,
					Optional: true,
					Elem:     &Schema{Type: TypeString},
				},
			},
			State: nil,
			Config: map[string]interface{}{
				"aliases": []interface{}{"foo", "bar"},
			},
			Diff:   nil,
			Err:    false,
			Errors: nil,
		},
		"#2": {
			Schema: map[string]*Schema{
				"aliases": &Schema{
					Type:     TypeSet,
					Optional: true,
					MinItems: 2,
					Elem:     &Schema{Type: TypeString},
				},
			},
			State: nil,
			Config: map[string]interface{}{
				"aliases": []interface{}{"foo"},
			},
			Diff: nil,
			Err:  true,
			Errors: []error{
				fmt.Errorf("aliases: attribute supports 2 item as a minimum, config has 1 declared"),
			},
		},
	}

	for tn, tc := range cases {
		c, err := config.NewRawConfig(tc.Config)
		if err != nil {
			t.Fatalf("%q: err: %s", tn, err)
		}
		_, es := schemaMap(tc.Schema).Validate(terraform.NewResourceConfig(c))

		if len(es) > 0 != tc.Err {
			if len(es) == 0 {
				t.Errorf("%q: no errors", tn)
			}

			for _, e := range es {
				t.Errorf("%q: err: %s", tn, e)
			}

			t.FailNow()
		}

		if tc.Errors != nil {
			if !reflect.DeepEqual(es, tc.Errors) {
				t.Fatalf("%q: expected: %q\ngot: %q", tn, tc.Errors, es)
			}
		}
	}
}

// errorSort implements sort.Interface to sort errors by their error message
type errorSort []error

func (e errorSort) Len() int      { return len(e) }
func (e errorSort) Swap(i, j int) { e[i], e[j] = e[j], e[i] }
func (e errorSort) Less(i, j int) bool {
	return e[i].Error() < e[j].Error()
}

func TestSchemaMapDeepCopy(t *testing.T) {
	schema := map[string]*Schema{
		"foo": &Schema{
			Type: TypeString,
		},
	}
	source := schemaMap(schema)
	dest := source.DeepCopy()
	dest["foo"].ForceNew = true
	if reflect.DeepEqual(source, dest) {
		t.Fatalf("source and dest should not match")
	}
}
