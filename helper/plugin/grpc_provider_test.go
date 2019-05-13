package plugin

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform/helper/schema"
	proto "github.com/hashicorp/terraform/internal/tfplugin5"
	"github.com/hashicorp/terraform/plugin/convert"
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

func TestUpgradeState_removedAttr(t *testing.T) {
	r1 := &schema.Resource{
		Schema: map[string]*schema.Schema{
			"two": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}

	r2 := &schema.Resource{
		Schema: map[string]*schema.Schema{
			"multi": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"set": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"required": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	r3 := &schema.Resource{
		Schema: map[string]*schema.Schema{
			"config_mode_attr": {
				Type:              schema.TypeList,
				ConfigMode:        schema.SchemaConfigModeAttr,
				SkipCoreTypeCheck: true,
				Optional:          true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"foo": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
		},
	}

	p := &schema.Provider{
		ResourcesMap: map[string]*schema.Resource{
			"r1": r1,
			"r2": r2,
			"r3": r3,
		},
	}

	server := &GRPCProviderServer{
		provider: p,
	}

	for _, tc := range []struct {
		name     string
		raw      string
		expected cty.Value
	}{
		{
			name: "r1",
			raw:  `{"id":"bar","removed":"removed","two":"2"}`,
			expected: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("bar"),
				"two": cty.StringVal("2"),
			}),
		},
		{
			name: "r2",
			raw:  `{"id":"bar","multi":[{"set":[{"required":"ok","removed":"removed"}]}]}`,
			expected: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("bar"),
				"multi": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"set": cty.SetVal([]cty.Value{
							cty.ObjectVal(map[string]cty.Value{
								"required": cty.StringVal("ok"),
							}),
						}),
					}),
				}),
			}),
		},
		{
			name: "r3",
			raw:  `{"id":"bar","config_mode_attr":[{"foo":"ok","removed":"removed"}]}`,
			expected: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("bar"),
				"config_mode_attr": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.StringVal("ok"),
					}),
				}),
			}),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := &proto.UpgradeResourceState_Request{
				TypeName: tc.name,
				Version:  0,
				RawState: &proto.RawState{
					Json: []byte(tc.raw),
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
			val, err := msgpack.Unmarshal(resp.UpgradedState.Msgpack, p.ResourcesMap[tc.name].CoreConfigSchema().ImpliedType())
			if err != nil {
				t.Fatal(err)
			}
			if !tc.expected.RawEquals(val) {
				t.Fatalf("\nexpected: %#v\ngot:      %#v\n", tc.expected, val)
			}
		})
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
					Optional: true,
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
			Name: "test incorrect default bool type",
			Schema: map[string]*schema.Schema{
				"foo": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
					Default:  "",
				},
			},
			ConfigVal: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.NullVal(cty.Bool),
			}),
			ExpectConfig: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.False,
			}),
		},
		{
			Name: "test deprecated default",
			Schema: map[string]*schema.Schema{
				"foo": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
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

			if tc.ExpectError != "" && len(resp.Diagnostics) > 0 {
				for _, d := range resp.Diagnostics {
					if !strings.Contains(d.Summary, tc.ExpectError) {
						t.Fatalf("Unexpected error: %s/%s", d.Summary, d.Detail)
					}
				}
				return
			}

			// we should have no errors past this point
			for _, d := range resp.Diagnostics {
				if d.Severity == proto.Diagnostic_ERROR {
					t.Fatal(resp.Diagnostics)
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

func TestNormalizeNullValues(t *testing.T) {
	for i, tc := range []struct {
		Src, Dst, Expect cty.Value
		Apply            bool
	}{
		{
			// The known set value is copied over the null set value
			Src: cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.NullVal(cty.String),
					}),
				}),
			}),
			Dst: cty.ObjectVal(map[string]cty.Value{
				"set": cty.NullVal(cty.Set(cty.Object(map[string]cty.Type{
					"foo": cty.String,
				}))),
			}),
			Expect: cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.NullVal(cty.String),
					}),
				}),
			}),
			Apply: true,
		},
		{
			// A zero set value is kept
			Src: cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetValEmpty(cty.String),
			}),
			Dst: cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetValEmpty(cty.String),
			}),
			Expect: cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetValEmpty(cty.String),
			}),
		},
		{
			// The known set value is copied over the null set value
			Src: cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.NullVal(cty.String),
					}),
				}),
			}),
			Dst: cty.ObjectVal(map[string]cty.Value{
				"set": cty.NullVal(cty.Set(cty.Object(map[string]cty.Type{
					"foo": cty.String,
				}))),
			}),
			// If we're only in a plan, we can't compare sets at all
			Expect: cty.ObjectVal(map[string]cty.Value{
				"set": cty.NullVal(cty.Set(cty.Object(map[string]cty.Type{
					"foo": cty.String,
				}))),
			}),
		},
		{
			// The empty map is copied over the null map
			Src: cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapValEmpty(cty.String),
			}),
			Dst: cty.ObjectVal(map[string]cty.Value{
				"map": cty.NullVal(cty.Map(cty.String)),
			}),
			Expect: cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapValEmpty(cty.String),
			}),
			Apply: true,
		},
		{
			// A zero value primitive is copied over a null primitive
			Src: cty.ObjectVal(map[string]cty.Value{
				"string": cty.StringVal(""),
			}),
			Dst: cty.ObjectVal(map[string]cty.Value{
				"string": cty.NullVal(cty.String),
			}),
			Expect: cty.ObjectVal(map[string]cty.Value{
				"string": cty.StringVal(""),
			}),
			Apply: true,
		},
		{
			// Plan primitives are kept
			Src: cty.ObjectVal(map[string]cty.Value{
				"string": cty.StringVal(""),
			}),
			Dst: cty.ObjectVal(map[string]cty.Value{
				"string": cty.NullVal(cty.String),
			}),
			Expect: cty.ObjectVal(map[string]cty.Value{
				"string": cty.NullVal(cty.String),
			}),
		},
		{
			// The null map is retained, because the src was unknown
			Src: cty.ObjectVal(map[string]cty.Value{
				"map": cty.UnknownVal(cty.Map(cty.String)),
			}),
			Dst: cty.ObjectVal(map[string]cty.Value{
				"map": cty.NullVal(cty.Map(cty.String)),
			}),
			Expect: cty.ObjectVal(map[string]cty.Value{
				"map": cty.NullVal(cty.Map(cty.String)),
			}),
			Apply: true,
		},
		{
			// the nul set is retained, because the src set contains an unknown value
			Src: cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.UnknownVal(cty.String),
					}),
				}),
			}),
			Dst: cty.ObjectVal(map[string]cty.Value{
				"set": cty.NullVal(cty.Set(cty.Object(map[string]cty.Type{
					"foo": cty.String,
				}))),
			}),
			Expect: cty.ObjectVal(map[string]cty.Value{
				"set": cty.NullVal(cty.Set(cty.Object(map[string]cty.Type{
					"foo": cty.String,
				}))),
			}),
			Apply: true,
		},
		{
			// Retain don't re-add unexpected planned values in a map
			Src: cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("a"),
					"b": cty.StringVal(""),
				}),
			}),
			Dst: cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("a"),
				}),
			}),
			Expect: cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("a"),
				}),
			}),
		},
		{
			// Remove extra values after apply
			Src: cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("a"),
					"b": cty.StringVal("b"),
				}),
			}),
			Dst: cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("a"),
				}),
			}),
			Expect: cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("a"),
				}),
			}),
			Apply: true,
		},
		{
			Src: cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("a"),
			}),
			Dst: cty.EmptyObjectVal,
			Expect: cty.ObjectVal(map[string]cty.Value{
				"a": cty.NullVal(cty.String),
			}),
		},

		// a list in an object in a list, going from null to empty
		{
			Src: cty.ObjectVal(map[string]cty.Value{
				"network_interface": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"network_ip":    cty.UnknownVal(cty.String),
						"access_config": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{"public_ptr_domain_name": cty.String, "nat_ip": cty.String}))),
						"address":       cty.NullVal(cty.String),
						"name":          cty.StringVal("nic0"),
					})}),
			}),
			Dst: cty.ObjectVal(map[string]cty.Value{
				"network_interface": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"network_ip":    cty.StringVal("10.128.0.64"),
						"access_config": cty.ListValEmpty(cty.Object(map[string]cty.Type{"public_ptr_domain_name": cty.String, "nat_ip": cty.String})),
						"address":       cty.StringVal("address"),
						"name":          cty.StringVal("nic0"),
					}),
				}),
			}),
			Expect: cty.ObjectVal(map[string]cty.Value{
				"network_interface": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"network_ip":    cty.StringVal("10.128.0.64"),
						"access_config": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{"public_ptr_domain_name": cty.String, "nat_ip": cty.String}))),
						"address":       cty.StringVal("address"),
						"name":          cty.StringVal("nic0"),
					}),
				}),
			}),
			Apply: true,
		},

		// a list in an object in a list, going from empty to null
		{
			Src: cty.ObjectVal(map[string]cty.Value{
				"network_interface": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"network_ip":    cty.UnknownVal(cty.String),
						"access_config": cty.ListValEmpty(cty.Object(map[string]cty.Type{"public_ptr_domain_name": cty.String, "nat_ip": cty.String})),
						"address":       cty.NullVal(cty.String),
						"name":          cty.StringVal("nic0"),
					})}),
			}),
			Dst: cty.ObjectVal(map[string]cty.Value{
				"network_interface": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"network_ip":    cty.StringVal("10.128.0.64"),
						"access_config": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{"public_ptr_domain_name": cty.String, "nat_ip": cty.String}))),
						"address":       cty.StringVal("address"),
						"name":          cty.StringVal("nic0"),
					}),
				}),
			}),
			Expect: cty.ObjectVal(map[string]cty.Value{
				"network_interface": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"network_ip":    cty.StringVal("10.128.0.64"),
						"access_config": cty.ListValEmpty(cty.Object(map[string]cty.Type{"public_ptr_domain_name": cty.String, "nat_ip": cty.String})),
						"address":       cty.StringVal("address"),
						"name":          cty.StringVal("nic0"),
					}),
				}),
			}),
			Apply: true,
		},
		// the empty list should be transferred, but the new unknown should not be overridden
		{
			Src: cty.ObjectVal(map[string]cty.Value{
				"network_interface": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"network_ip":    cty.StringVal("10.128.0.64"),
						"access_config": cty.ListValEmpty(cty.Object(map[string]cty.Type{"public_ptr_domain_name": cty.String, "nat_ip": cty.String})),
						"address":       cty.NullVal(cty.String),
						"name":          cty.StringVal("nic0"),
					})}),
			}),
			Dst: cty.ObjectVal(map[string]cty.Value{
				"network_interface": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"network_ip":    cty.UnknownVal(cty.String),
						"access_config": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{"public_ptr_domain_name": cty.String, "nat_ip": cty.String}))),
						"address":       cty.StringVal("address"),
						"name":          cty.StringVal("nic0"),
					}),
				}),
			}),
			Expect: cty.ObjectVal(map[string]cty.Value{
				"network_interface": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"network_ip":    cty.UnknownVal(cty.String),
						"access_config": cty.ListValEmpty(cty.Object(map[string]cty.Type{"public_ptr_domain_name": cty.String, "nat_ip": cty.String})),
						"address":       cty.StringVal("address"),
						"name":          cty.StringVal("nic0"),
					}),
				}),
			}),
		},
		{
			// fix unknowns added to a map
			Src: cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("a"),
					"b": cty.StringVal(""),
				}),
			}),
			Dst: cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("a"),
					"b": cty.UnknownVal(cty.String),
				}),
			}),
			Expect: cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("a"),
					"b": cty.StringVal(""),
				}),
			}),
		},
		{
			// fix unknowns lost from a list
			Src: cty.ObjectVal(map[string]cty.Value{
				"top": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"list": cty.ListVal([]cty.Value{
							cty.ObjectVal(map[string]cty.Value{
								"values": cty.ListVal([]cty.Value{cty.UnknownVal(cty.String)}),
							}),
						}),
					}),
				}),
			}),
			Dst: cty.ObjectVal(map[string]cty.Value{
				"top": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"list": cty.ListVal([]cty.Value{
							cty.ObjectVal(map[string]cty.Value{
								"values": cty.NullVal(cty.List(cty.String)),
							}),
						}),
					}),
				}),
			}),
			Expect: cty.ObjectVal(map[string]cty.Value{
				"top": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"list": cty.ListVal([]cty.Value{
							cty.ObjectVal(map[string]cty.Value{
								"values": cty.ListVal([]cty.Value{cty.UnknownVal(cty.String)}),
							}),
						}),
					}),
				}),
			}),
		},
		{
			Src: cty.ObjectVal(map[string]cty.Value{
				"set": cty.NullVal(cty.Set(cty.Object(map[string]cty.Type{
					"list": cty.List(cty.String),
				}))),
			}),
			Dst: cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetValEmpty(cty.Object(map[string]cty.Type{
					"list": cty.List(cty.String),
				})),
			}),
			Expect: cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetValEmpty(cty.Object(map[string]cty.Type{
					"list": cty.List(cty.String),
				})),
			}),
		},
		{
			Src: cty.ObjectVal(map[string]cty.Value{
				"set": cty.NullVal(cty.Set(cty.Object(map[string]cty.Type{
					"list": cty.List(cty.String),
				}))),
			}),
			Dst: cty.ObjectVal(map[string]cty.Value{
				"set": cty.SetValEmpty(cty.Object(map[string]cty.Type{
					"list": cty.List(cty.String),
				})),
			}),
			Expect: cty.ObjectVal(map[string]cty.Value{
				"set": cty.NullVal(cty.Set(cty.Object(map[string]cty.Type{
					"list": cty.List(cty.String),
				}))),
			}),
			Apply: true,
		},
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			got := normalizeNullValues(tc.Dst, tc.Src, tc.Apply)
			if !got.RawEquals(tc.Expect) {
				t.Fatalf("\nexpected: %#v\ngot:      %#v\n", tc.Expect, got)
			}
		})
	}
}

func TestValidateNulls(t *testing.T) {
	for i, tc := range []struct {
		Cfg cty.Value
		Err bool
	}{
		{
			Cfg: cty.ObjectVal(map[string]cty.Value{
				"list": cty.ListVal([]cty.Value{
					cty.StringVal("string"),
					cty.NullVal(cty.String),
				}),
			}),
			Err: true,
		},
		{
			Cfg: cty.ObjectVal(map[string]cty.Value{
				"map": cty.MapVal(map[string]cty.Value{
					"string": cty.StringVal("string"),
					"null":   cty.NullVal(cty.String),
				}),
			}),
			Err: false,
		},
		{
			Cfg: cty.ObjectVal(map[string]cty.Value{
				"object": cty.ObjectVal(map[string]cty.Value{
					"list": cty.ListVal([]cty.Value{
						cty.StringVal("string"),
						cty.NullVal(cty.String),
					}),
				}),
			}),
			Err: true,
		},
		{
			Cfg: cty.ObjectVal(map[string]cty.Value{
				"object": cty.ObjectVal(map[string]cty.Value{
					"list": cty.ListVal([]cty.Value{
						cty.StringVal("string"),
						cty.NullVal(cty.String),
					}),
					"list2": cty.ListVal([]cty.Value{
						cty.StringVal("string"),
						cty.NullVal(cty.String),
					}),
				}),
			}),
			Err: true,
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			d := validateConfigNulls(tc.Cfg, nil)
			diags := convert.ProtoToDiagnostics(d)
			switch {
			case tc.Err:
				if !diags.HasErrors() {
					t.Fatal("expected error")
				}
			default:
				if diags.HasErrors() {
					t.Fatalf("unexpected error: %q", diags.Err())
				}
			}
		})
	}
}
