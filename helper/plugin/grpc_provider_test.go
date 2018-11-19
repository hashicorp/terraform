package plugin

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform/helper/schema"
	proto "github.com/hashicorp/terraform/internal/tfplugin5"
	"github.com/hashicorp/terraform/terraform"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/msgpack"
)

// The GRPCProviderServer will directly implement the go protobuf server
var _ proto.ProviderServer = (*GRPCProviderServer)(nil)

var (
	typeComparer  = cmp.Comparer(cty.Type.Equals)
	valueComparer = cmp.Comparer(cty.Value.RawEquals)
	equateEmpty   = cmpopts.EquateEmpty()
)

func TestUpgradeState_jsonState(t *testing.T) {
	r := &schema.Resource{
		SchemaVersion: 2,
		Schema: map[string]*schema.Schema{
			"two": {
				Type:     schema.TypeInt,
				Optional: true,
			},
		},
	}

	r.StateUpgraders = []schema.StateUpgrader{
		{
			Version: 0,
			Type: cty.Object(map[string]cty.Type{
				"id":   cty.String,
				"zero": cty.Number,
			}),
			Upgrade: func(m map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
				_, ok := m["zero"].(float64)
				if !ok {
					return nil, fmt.Errorf("zero not found in %#v", m)
				}
				m["one"] = float64(1)
				delete(m, "zero")
				return m, nil
			},
		},
		{
			Version: 1,
			Type: cty.Object(map[string]cty.Type{
				"id":  cty.String,
				"one": cty.Number,
			}),
			Upgrade: func(m map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
				_, ok := m["one"].(float64)
				if !ok {
					return nil, fmt.Errorf("one not found in %#v", m)
				}
				m["two"] = float64(2)
				delete(m, "one")
				return m, nil
			},
		},
	}

	server := &GRPCProviderServer{
		provider: &schema.Provider{
			ResourcesMap: map[string]*schema.Resource{
				"test": r,
			},
		},
	}

	req := &proto.UpgradeResourceState_Request{
		TypeName: "test",
		Version:  0,
		RawState: &proto.RawState{
			Json: []byte(`{"id":"bar","zero":0}`),
		},
	}

	resp, err := server.UpgradeResourceState(nil, req)
	if err != nil {
		t.Fatal(err)
	}

	if len(resp.Diagnostics) > 0 {
		for _, d := range resp.Diagnostics {
			t.Errorf("%#v", d)
		}
		t.Fatal("error")
	}

	val, err := msgpack.Unmarshal(resp.UpgradedState.Msgpack, r.CoreConfigSchema().ImpliedType())
	if err != nil {
		t.Fatal(err)
	}

	expected := cty.ObjectVal(map[string]cty.Value{
		"id":  cty.StringVal("bar"),
		"two": cty.NumberIntVal(2),
	})

	if !cmp.Equal(expected, val, valueComparer, equateEmpty) {
		t.Fatal(cmp.Diff(expected, val, valueComparer, equateEmpty))
	}
}

func TestUpgradeState_flatmapState(t *testing.T) {
	r := &schema.Resource{
		SchemaVersion: 4,
		Schema: map[string]*schema.Schema{
			"four": {
				Type:     schema.TypeInt,
				Required: true,
			},
		},
		// this MigrateState will take the state to version 2
		MigrateState: func(v int, is *terraform.InstanceState, _ interface{}) (*terraform.InstanceState, error) {
			switch v {
			case 0:
				_, ok := is.Attributes["zero"]
				if !ok {
					return nil, fmt.Errorf("zero not found in %#v", is.Attributes)
				}
				is.Attributes["one"] = "1"
				delete(is.Attributes, "zero")
				fallthrough
			case 1:
				_, ok := is.Attributes["one"]
				if !ok {
					return nil, fmt.Errorf("one not found in %#v", is.Attributes)
				}
				is.Attributes["two"] = "2"
				delete(is.Attributes, "one")
			default:
				return nil, fmt.Errorf("invalid schema version %d", v)
			}
			return is, nil
		},
	}

	r.StateUpgraders = []schema.StateUpgrader{
		{
			Version: 2,
			Type: cty.Object(map[string]cty.Type{
				"id":  cty.String,
				"two": cty.Number,
			}),
			Upgrade: func(m map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
				_, ok := m["two"].(float64)
				if !ok {
					return nil, fmt.Errorf("two not found in %#v", m)
				}
				m["three"] = float64(3)
				delete(m, "two")
				return m, nil
			},
		},
		{
			Version: 3,
			Type: cty.Object(map[string]cty.Type{
				"id":    cty.String,
				"three": cty.Number,
			}),
			Upgrade: func(m map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
				_, ok := m["three"].(float64)
				if !ok {
					return nil, fmt.Errorf("three not found in %#v", m)
				}
				m["four"] = float64(4)
				delete(m, "three")
				return m, nil
			},
		},
	}

	server := &GRPCProviderServer{
		provider: &schema.Provider{
			ResourcesMap: map[string]*schema.Resource{
				"test": r,
			},
		},
	}

	testReqs := []*proto.UpgradeResourceState_Request{
		{
			TypeName: "test",
			Version:  0,
			RawState: &proto.RawState{
				Flatmap: map[string]string{
					"id":   "bar",
					"zero": "0",
				},
			},
		},
		{
			TypeName: "test",
			Version:  1,
			RawState: &proto.RawState{
				Flatmap: map[string]string{
					"id":  "bar",
					"one": "1",
				},
			},
		},
		// two and  up could be stored in flatmap or json states
		{
			TypeName: "test",
			Version:  2,
			RawState: &proto.RawState{
				Flatmap: map[string]string{
					"id":  "bar",
					"two": "2",
				},
			},
		},
		{
			TypeName: "test",
			Version:  2,
			RawState: &proto.RawState{
				Json: []byte(`{"id":"bar","two":2}`),
			},
		},
		{
			TypeName: "test",
			Version:  3,
			RawState: &proto.RawState{
				Flatmap: map[string]string{
					"id":    "bar",
					"three": "3",
				},
			},
		},
		{
			TypeName: "test",
			Version:  3,
			RawState: &proto.RawState{
				Json: []byte(`{"id":"bar","three":3}`),
			},
		},
		{
			TypeName: "test",
			Version:  4,
			RawState: &proto.RawState{
				Flatmap: map[string]string{
					"id":   "bar",
					"four": "4",
				},
			},
		},
		{
			TypeName: "test",
			Version:  4,
			RawState: &proto.RawState{
				Json: []byte(`{"id":"bar","four":4}`),
			},
		},
	}

	for i, req := range testReqs {
		t.Run(fmt.Sprintf("%d-%d", i, req.Version), func(t *testing.T) {
			resp, err := server.UpgradeResourceState(nil, req)
			if err != nil {
				t.Fatal(err)
			}

			if len(resp.Diagnostics) > 0 {
				for _, d := range resp.Diagnostics {
					t.Errorf("%#v", d)
				}
				t.Fatal("error")
			}

			val, err := msgpack.Unmarshal(resp.UpgradedState.Msgpack, r.CoreConfigSchema().ImpliedType())
			if err != nil {
				t.Fatal(err)
			}

			expected := cty.ObjectVal(map[string]cty.Value{
				"id":   cty.StringVal("bar"),
				"four": cty.NumberIntVal(4),
			})

			if !cmp.Equal(expected, val, valueComparer, equateEmpty) {
				t.Fatal(cmp.Diff(expected, val, valueComparer, equateEmpty))
			}
		})
	}
}

func TestPlanResourceChange(t *testing.T) {
	r := &schema.Resource{
		SchemaVersion: 4,
		Schema: map[string]*schema.Schema{
			"foo": {
				Type:     schema.TypeInt,
				Optional: true,
			},
		},
	}

	server := &GRPCProviderServer{
		provider: &schema.Provider{
			ResourcesMap: map[string]*schema.Resource{
				"test": r,
			},
		},
	}

	schema := r.CoreConfigSchema()
	priorState, err := msgpack.Marshal(cty.NullVal(schema.ImpliedType()), schema.ImpliedType())
	if err != nil {
		t.Fatal(err)
	}

	// A propsed state with only the ID unknown will produce a nil diff, and
	// should return the propsed state value.
	proposedVal, err := schema.CoerceValue(cty.ObjectVal(map[string]cty.Value{
		"id": cty.UnknownVal(cty.String),
	}))
	if err != nil {
		t.Fatal(err)
	}
	proposedState, err := msgpack.Marshal(proposedVal, schema.ImpliedType())
	if err != nil {
		t.Fatal(err)
	}

	testReq := &proto.PlanResourceChange_Request{
		TypeName: "test",
		PriorState: &proto.DynamicValue{
			Msgpack: priorState,
		},
		ProposedNewState: &proto.DynamicValue{
			Msgpack: proposedState,
		},
	}

	resp, err := server.PlanResourceChange(context.Background(), testReq)
	if err != nil {
		t.Fatal(err)
	}

	plannedStateVal, err := msgpack.Unmarshal(resp.PlannedState.Msgpack, schema.ImpliedType())
	if err != nil {
		t.Fatal(err)
	}

	if !cmp.Equal(proposedVal, plannedStateVal, valueComparer) {
		t.Fatal(cmp.Diff(proposedVal, plannedStateVal, valueComparer))
	}
}

func TestApplyResourceChange(t *testing.T) {
	r := &schema.Resource{
		SchemaVersion: 4,
		Schema: map[string]*schema.Schema{
			"foo": {
				Type:     schema.TypeInt,
				Optional: true,
			},
		},
		Create: func(rd *schema.ResourceData, _ interface{}) error {
			rd.SetId("bar")
			return nil
		},
	}

	server := &GRPCProviderServer{
		provider: &schema.Provider{
			ResourcesMap: map[string]*schema.Resource{
				"test": r,
			},
		},
	}

	schema := r.CoreConfigSchema()
	priorState, err := msgpack.Marshal(cty.NullVal(schema.ImpliedType()), schema.ImpliedType())
	if err != nil {
		t.Fatal(err)
	}

	// A proposed state with only the ID unknown will produce a nil diff, and
	// should return the proposed state value.
	plannedVal, err := schema.CoerceValue(cty.ObjectVal(map[string]cty.Value{
		"id": cty.UnknownVal(cty.String),
	}))
	if err != nil {
		t.Fatal(err)
	}
	plannedState, err := msgpack.Marshal(plannedVal, schema.ImpliedType())
	if err != nil {
		t.Fatal(err)
	}

	testReq := &proto.ApplyResourceChange_Request{
		TypeName: "test",
		PriorState: &proto.DynamicValue{
			Msgpack: priorState,
		},
		PlannedState: &proto.DynamicValue{
			Msgpack: plannedState,
		},
	}

	resp, err := server.ApplyResourceChange(context.Background(), testReq)
	if err != nil {
		t.Fatal(err)
	}

	newStateVal, err := msgpack.Unmarshal(resp.NewState.Msgpack, schema.ImpliedType())
	if err != nil {
		t.Fatal(err)
	}

	id := newStateVal.GetAttr("id").AsString()
	if id != "bar" {
		t.Fatalf("incorrect final state: %#v\n", newStateVal)
	}
}

func TestPrepareProviderConfig(t *testing.T) {
	for _, tc := range []struct {
		Name         string
		Schema       map[string]*schema.Schema
		ConfigVal    cty.Value
		ExpectError  string
		ExpectConfig cty.Value
	}{
		{
			Name: "test prepare",
			Schema: map[string]*schema.Schema{
				"foo": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
			},
			ConfigVal: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("bar"),
			}),
			ExpectConfig: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("bar"),
			}),
		},
		{
			Name: "test default",
			Schema: map[string]*schema.Schema{
				"foo": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
					Default:  "default",
				},
			},
			ConfigVal: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.NullVal(cty.String),
			}),
			ExpectConfig: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("default"),
			}),
		},
		{
			Name: "test defaultfunc",
			Schema: map[string]*schema.Schema{
				"foo": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
					DefaultFunc: func() (interface{}, error) {
						return "defaultfunc", nil
					},
				},
			},
			ConfigVal: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.NullVal(cty.String),
			}),
			ExpectConfig: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("defaultfunc"),
			}),
		},
		{
			Name: "test default required",
			Schema: map[string]*schema.Schema{
				"foo": &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
					DefaultFunc: func() (interface{}, error) {
						return "defaultfunc", nil
					},
				},
			},
			ConfigVal: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.NullVal(cty.String),
			}),
			ExpectConfig: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("defaultfunc"),
			}),
		},
		{
			Name: "test incorrect type",
			Schema: map[string]*schema.Schema{
				"foo": &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
				},
			},
			ConfigVal: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.NumberIntVal(3),
			}),
			ExpectConfig: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("3"),
			}),
		},
		{
			Name: "test incorrect default type",
			Schema: map[string]*schema.Schema{
				"foo": &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
					Default:  true,
				},
			},
			ConfigVal: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.NullVal(cty.String),
			}),
			ExpectConfig: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("true"),
			}),
		},
		{
			Name: "test deprecated default",
			Schema: map[string]*schema.Schema{
				"foo": &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
					Default:  "do not use",
					Removed:  "don't use this",
				},
			},
			ConfigVal: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.NullVal(cty.String),
			}),
			ExpectConfig: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.NullVal(cty.String),
			}),
		},
	} {
		t.Run(tc.Name, func(t *testing.T) {
			server := &GRPCProviderServer{
				provider: &schema.Provider{
					Schema: tc.Schema,
				},
			}

			block := schema.InternalMap(tc.Schema).CoreConfigSchema()

			rawConfig, err := msgpack.Marshal(tc.ConfigVal, block.ImpliedType())
			if err != nil {
				t.Fatal(err)
			}

			testReq := &proto.PrepareProviderConfig_Request{
				Config: &proto.DynamicValue{
					Msgpack: rawConfig,
				},
			}

			resp, err := server.PrepareProviderConfig(nil, testReq)
			if err != nil {
				t.Fatal(err)
			}

			if tc.ExpectError == "" && len(resp.Diagnostics) > 0 {
				for _, d := range resp.Diagnostics {
					if !strings.Contains(d.Summary, tc.ExpectError) {
						t.Fatalf("Unexpected error: %s/%s", d.Summary, d.Detail)
					}
				}
			}

			val, err := msgpack.Unmarshal(resp.PreparedConfig.Msgpack, block.ImpliedType())
			if err != nil {
				t.Fatal(err)
			}

			if tc.ExpectConfig.GoString() != val.GoString() {
				t.Fatalf("\nexpected: %#v\ngot: %#v", tc.ExpectConfig, val)
			}
		})
	}
}

func TestGetSchemaTimeouts(t *testing.T) {
	r := &schema.Resource{
		SchemaVersion: 4,
		Timeouts: &schema.ResourceTimeout{
			Create:  schema.DefaultTimeout(time.Second),
			Read:    schema.DefaultTimeout(2 * time.Second),
			Update:  schema.DefaultTimeout(3 * time.Second),
			Default: schema.DefaultTimeout(10 * time.Second),
		},
		Schema: map[string]*schema.Schema{
			"foo": {
				Type:     schema.TypeInt,
				Optional: true,
			},
		},
	}

	// verify that the timeouts appear in the schema as defined
	block := r.CoreConfigSchema()
	timeoutsBlock := block.BlockTypes["timeouts"]
	if timeoutsBlock == nil {
		t.Fatal("missing timeouts in schema")
	}

	if timeoutsBlock.Attributes["create"] == nil {
		t.Fatal("missing create timeout in schema")
	}
	if timeoutsBlock.Attributes["read"] == nil {
		t.Fatal("missing read timeout in schema")
	}
	if timeoutsBlock.Attributes["update"] == nil {
		t.Fatal("missing update timeout in schema")
	}
	if d := timeoutsBlock.Attributes["delete"]; d != nil {
		t.Fatalf("unexpected delete timeout in schema: %#v", d)
	}
	if timeoutsBlock.Attributes["default"] == nil {
		t.Fatal("missing default timeout in schema")
	}
}

func TestNormalizeFlatmapContainers(t *testing.T) {
	for i, tc := range []struct {
		attrs  map[string]string
		expect map[string]string
	}{
		{
			attrs:  map[string]string{"id": "1", "multi.2.set.#": "1", "multi.1.set.#": "0", "single.#": "0"},
			expect: map[string]string{"id": "1"},
		},
		{
			attrs:  map[string]string{"id": "1", "multi.2.set.#": "2", "multi.2.set.1.foo": "bar", "multi.1.set.#": "0", "single.#": "0"},
			expect: map[string]string{"id": "1", "multi.2.set.#": "1", "multi.2.set.1.foo": "bar"},
		},
		{
			attrs:  map[string]string{"id": "78629a0f5f3f164f", "multi.#": "1"},
			expect: map[string]string{"id": "78629a0f5f3f164f"},
		},
		{
			attrs:  map[string]string{"multi.529860700.set.#": "1", "multi.#": "1", "id": "78629a0f5f3f164f"},
			expect: map[string]string{"id": "78629a0f5f3f164f"},
		},
		{
			attrs:  map[string]string{"set.2.required": "bar", "set.2.list.#": "1", "set.2.list.0": "x", "set.1.list.#": "0"},
			expect: map[string]string{"set.2.list.#": "1", "set.2.list.0": "x", "set.2.required": "bar", "set.#": "1"},
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			got := normalizeFlatmapContainers(tc.attrs)
			if !reflect.DeepEqual(tc.expect, got) {
				t.Fatalf("expected:\n%#v\ngot:\n%#v\n", tc.expect, got)
			}
		})
	}
}
