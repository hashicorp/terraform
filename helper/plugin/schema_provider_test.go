package plugin

import (
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/config/hcl2shim"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/plugin/proto"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/msgpack"
)

// the TestProvider functions have been adapted from the helper/schema fixtures

func TestProviderGetSchema(t *testing.T) {
	p := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"bar": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"foo": &schema.Resource{
				SchemaVersion: 1,
				Schema: map[string]*schema.Schema{
					"bar": {
						Type:     schema.TypeString,
						Required: true,
					},
				},
			},
		},
		DataSourcesMap: map[string]*schema.Resource{
			"baz": &schema.Resource{
				SchemaVersion: 2,
				Schema: map[string]*schema.Schema{
					"bur": {
						Type:     schema.TypeString,
						Required: true,
					},
				},
			},
		},
	}

	want := providers.GetSchemaResponse{
		Provider: providers.Schema{
			Version: 0,
			Block:   schema.InternalMap(p.Schema).CoreConfigSchema(),
		},
		ResourceTypes: map[string]providers.Schema{
			"foo": {
				Version: 1,
				Block:   p.ResourcesMap["foo"].CoreConfigSchema(),
			},
		},
		DataSources: map[string]providers.Schema{
			"baz": {
				Version: 2,
				Block:   p.DataSourcesMap["baz"].CoreConfigSchema(),
			},
		},
	}

	provider := &GRPCProviderServer{
		provider: p,
	}

	resp, err := provider.GetSchema(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}
	diags := plugin.ProtoToDiagnostics(resp.Diagnostics)
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	schemaResp := providers.GetSchemaResponse{
		Provider: plugin.ProtoToProviderSchema(resp.Provider),
		ResourceTypes: map[string]providers.Schema{
			"foo": plugin.ProtoToProviderSchema(resp.ResourceSchemas["foo"]),
		},
		DataSources: map[string]providers.Schema{
			"baz": plugin.ProtoToProviderSchema(resp.DataSourceSchemas["baz"]),
		},
	}

	if !cmp.Equal(schemaResp, want, equateEmpty, typeComparer) {
		t.Error("wrong result:\n", cmp.Diff(schemaResp, want, equateEmpty, typeComparer))
	}
}

func TestProviderValidate(t *testing.T) {
	cases := []struct {
		Name string
		P    *schema.Provider
		Err  bool
		Warn bool
	}{
		{
			Name: "warning",
			P: &schema.Provider{
				Schema: map[string]*schema.Schema{
					"foo": &schema.Schema{
						Type:     schema.TypeString,
						Optional: true,
						ValidateFunc: func(_ interface{}, _ string) ([]string, []error) {
							return []string{"warning"}, nil
						},
					},
				},
			},
			Warn: true,
		},
		{
			Name: "error",
			P: &schema.Provider{
				Schema: map[string]*schema.Schema{
					"foo": &schema.Schema{
						Type:     schema.TypeString,
						Optional: true,
						ValidateFunc: func(_ interface{}, _ string) ([]string, []error) {
							return nil, []error{errors.New("error")}
						},
					},
				},
			},
			Err: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			provider := &GRPCProviderServer{
				provider: tc.P,
			}

			cfgSchema := schema.InternalMap(tc.P.Schema).CoreConfigSchema()
			val := hcl2shim.HCL2ValueFromConfigValue(map[string]interface{}{"foo": "bar"})
			val, err := cfgSchema.CoerceValue(val)
			if err != nil {
				t.Fatal(err)
			}

			mp, err := msgpack.Marshal(val, cfgSchema.ImpliedType())
			if err != nil {
				t.Fatal(err)
			}

			req := &proto.ValidateProviderConfig_Request{
				Config: &proto.DynamicValue{Msgpack: mp},
			}

			resp, err := provider.ValidateProviderConfig(nil, req)
			if err != nil {
				t.Fatal(err)
			}

			diags := plugin.ProtoToDiagnostics(resp.Diagnostics)

			var warn tfdiags.Diagnostic
			for _, d := range diags {
				if d.Severity() == tfdiags.Warning {
					warn = d
				}
			}

			switch {
			case tc.Err:
				if !diags.HasErrors() {
					t.Fatal("expected error")
				}
			case !tc.Err:
				if diags.HasErrors() {
					t.Fatal(diags.Err())
				}

			case tc.Warn:
				if warn == nil {
					t.Fatal("expected warning")
				}
			case !tc.Warn:
				if warn != nil {
					t.Fatal("unexpected warning", warn)
				}
			}
		})
	}
}

func TestProviderValidateResource(t *testing.T) {
	cases := []struct {
		Name   string
		P      *schema.Provider
		Type   string
		Config map[string]interface{}
		Err    bool
		Warn   bool
	}{
		{
			Name: "error",
			P: &schema.Provider{
				ResourcesMap: map[string]*schema.Resource{
					"foo": &schema.Resource{
						Schema: map[string]*schema.Schema{
							"attr": &schema.Schema{
								Type:     schema.TypeString,
								Optional: true,
								ValidateFunc: func(_ interface{}, _ string) ([]string, []error) {
									return nil, []error{errors.New("warn")}
								},
							},
						},
					},
				},
			},
			Type: "foo",
			Err:  true,
		},
		{
			Name: "ok",
			P: &schema.Provider{
				ResourcesMap: map[string]*schema.Resource{
					"foo": &schema.Resource{
						Schema: map[string]*schema.Schema{
							"attr": &schema.Schema{
								Type:     schema.TypeString,
								Optional: true,
							},
						},
					},
				},
			},
			Config: map[string]interface{}{"attr": "bar"},
			Type:   "foo",
		},
		{
			Name: "warn",
			P: &schema.Provider{
				ResourcesMap: map[string]*schema.Resource{
					"foo": &schema.Resource{
						Schema: map[string]*schema.Schema{
							"attr": &schema.Schema{
								Type:     schema.TypeString,
								Optional: true,
								ValidateFunc: func(_ interface{}, _ string) ([]string, []error) {
									return []string{"warn"}, nil
								},
							},
						},
					},
				},
			},
			Type:   "foo",
			Config: map[string]interface{}{"attr": "bar"},
			Err:    false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			provider := &GRPCProviderServer{
				provider: tc.P,
			}

			cfgSchema := tc.P.ResourcesMap[tc.Type].CoreConfigSchema()
			val := hcl2shim.HCL2ValueFromConfigValue(tc.Config)
			val, err := cfgSchema.CoerceValue(val)
			if err != nil {
				t.Fatal(err)
			}

			mp, err := msgpack.Marshal(val, cfgSchema.ImpliedType())
			if err != nil {
				t.Fatal(err)
			}

			req := &proto.ValidateResourceTypeConfig_Request{
				TypeName: tc.Type,
				Config:   &proto.DynamicValue{Msgpack: mp},
			}

			resp, err := provider.ValidateResourceTypeConfig(nil, req)
			if err != nil {
				t.Fatal(err)
			}

			diags := plugin.ProtoToDiagnostics(resp.Diagnostics)

			var warn tfdiags.Diagnostic
			for _, d := range diags {
				if d.Severity() == tfdiags.Warning {
					warn = d
				}
			}

			switch {
			case tc.Err:
				if !diags.HasErrors() {
					t.Fatal("expected error")
				}
			case !tc.Err:
				if diags.HasErrors() {
					t.Fatal(diags.Err())
				}

			case tc.Warn:
				if warn == nil {
					t.Fatal("expected warning")
				}
			case !tc.Warn:
				if warn != nil {
					t.Fatal("unexpected warning", warn)
				}
			}
		})
	}
}

func TestProviderImportState_default(t *testing.T) {

	p := &GRPCProviderServer{
		provider: &schema.Provider{
			ResourcesMap: map[string]*schema.Resource{
				"foo": &schema.Resource{
					Importer: &schema.ResourceImporter{},
				},
			},
		},
	}

	req := &proto.ImportResourceState_Request{
		TypeName: "foo",
		Id:       "bar",
	}
	resp, err := p.ImportResourceState(nil, req)
	if err != nil {
		t.Fatal(err)
	}
	diags := plugin.ProtoToDiagnostics(resp.Diagnostics)
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	if len(resp.ImportedResources) != 1 {
		t.Fatalf("expected 1 import, git %#v", resp.ImportedResources)
	}
}

func TestProviderImportState_setsId(t *testing.T) {
	var val string
	stateFunc := func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
		val = d.Id()
		return []*schema.ResourceData{d}, nil
	}

	p := &GRPCProviderServer{
		provider: &schema.Provider{
			ResourcesMap: map[string]*schema.Resource{
				"foo": &schema.Resource{
					Importer: &schema.ResourceImporter{
						State: stateFunc,
					},
				},
			},
		},
	}

	req := &proto.ImportResourceState_Request{
		TypeName: "foo",
		Id:       "bar",
	}
	resp, err := p.ImportResourceState(nil, req)
	if err != nil {
		t.Fatal(err)
	}
	diags := plugin.ProtoToDiagnostics(resp.Diagnostics)
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	if len(resp.ImportedResources) != 1 {
		t.Fatalf("expected 1 import, git %#v", resp.ImportedResources)
	}

	if val != "bar" {
		t.Fatal("should set id")
	}
}

func TestProviderImportState_setsType(t *testing.T) {
	var tVal string
	stateFunc := func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
		d.SetId("foo")
		tVal = d.State().Ephemeral.Type
		return []*schema.ResourceData{d}, nil
	}

	p := &GRPCProviderServer{
		provider: &schema.Provider{
			ResourcesMap: map[string]*schema.Resource{
				"foo": &schema.Resource{
					Importer: &schema.ResourceImporter{
						State: stateFunc,
					},
				},
			},
		},
	}

	req := &proto.ImportResourceState_Request{
		TypeName: "foo",
		Id:       "bar",
	}
	resp, err := p.ImportResourceState(nil, req)
	if err != nil {
		t.Fatal(err)
	}
	diags := plugin.ProtoToDiagnostics(resp.Diagnostics)
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	if tVal != "foo" {
		t.Fatal("should set type")
	}
}

func TestProviderStop(t *testing.T) {
	var p schema.Provider

	if p.Stopped() {
		t.Fatal("should not be stopped")
	}

	// Verify stopch blocks
	ch := p.StopContext().Done()
	select {
	case <-ch:
		t.Fatal("should not be stopped")
	case <-time.After(10 * time.Millisecond):
	}

	provider := &GRPCProviderServer{
		provider: &p,
	}

	// Stop it
	if _, err := provider.Stop(nil, &proto.Stop_Request{}); err != nil {
		t.Fatal(err)
	}

	// Verify
	if !p.Stopped() {
		t.Fatal("should be stopped")
	}

	select {
	case <-ch:
	case <-time.After(10 * time.Millisecond):
		t.Fatal("should be stopped")
	}
}

func TestProviderStop_stopFirst(t *testing.T) {
	var p schema.Provider

	provider := &GRPCProviderServer{
		provider: &p,
	}

	// Stop it
	_, err := provider.Stop(nil, &proto.Stop_Request{})
	if err != nil {
		t.Fatal(err)
	}

	// Verify
	if !p.Stopped() {
		t.Fatal("should be stopped")
	}

	select {
	case <-p.StopContext().Done():
	case <-time.After(10 * time.Millisecond):
		t.Fatal("should be stopped")
	}
}

// add the implicit "id" attribute for test resources
func testResource(block *configschema.Block) *configschema.Block {
	if block.Attributes == nil {
		block.Attributes = make(map[string]*configschema.Attribute)
	}

	if block.BlockTypes == nil {
		block.BlockTypes = make(map[string]*configschema.NestedBlock)
	}

	if block.Attributes["id"] == nil {
		block.Attributes["id"] = &configschema.Attribute{
			Type:     cty.String,
			Optional: true,
			Computed: true,
		}
	}
	return block
}
