package format

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/zclconf/go-cty/cty"
)

func TestState(t *testing.T) {
	tests := []struct {
		State *StateOpts
		Want  string
	}{
		{
			&StateOpts{
				State:   &states.State{},
				Color:   disabledColorize,
				Schemas: &terraform.Schemas{},
			},
			"The state file is empty. No resources are represented.",
		},
		{
			&StateOpts{
				State:   basicState(t),
				Color:   disabledColorize,
				Schemas: testSchemas(),
			},
			basicStateOutput,
		},
		{
			&StateOpts{
				State:   nestedState(t),
				Color:   disabledColorize,
				Schemas: testSchemas(),
			},
			nestedStateOutput,
		},
		{
			&StateOpts{
				State:   deposedState(t),
				Color:   disabledColorize,
				Schemas: testSchemas(),
			},
			deposedNestedStateOutput,
		},
		{
			&StateOpts{
				State:   onlyDeposedState(t),
				Color:   disabledColorize,
				Schemas: testSchemas(),
			},
			onlyDeposedOutput,
		},
		{
			&StateOpts{
				State:   stateWithMoreOutputs(t),
				Color:   disabledColorize,
				Schemas: testSchemas(),
			},
			stateWithMoreOutputsOutput,
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			got := State(tt.State)
			if got != tt.Want {
				t.Errorf(
					"wrong result\ninput: %v\ngot: \n%q\nwant: \n%q",
					tt.State.State, got, tt.Want,
				)
			}
		})
	}
}

func testProvider() *terraform.MockProvider {
	p := new(terraform.MockProvider)
	p.ReadResourceFn = func(req providers.ReadResourceRequest) providers.ReadResourceResponse {
		return providers.ReadResourceResponse{NewState: req.PriorState}
	}

	p.GetProviderSchemaResponse = testProviderSchema()

	return p
}

func testProviderSchema() *providers.GetProviderSchemaResponse {
	return &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{
			Block: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"region": {Type: cty.String, Optional: true},
				},
			},
		},
		ResourceTypes: map[string]providers.Schema{
			"test_resource": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id":      {Type: cty.String, Computed: true},
						"foo":     {Type: cty.String, Optional: true},
						"woozles": {Type: cty.String, Optional: true},
					},
					BlockTypes: map[string]*configschema.NestedBlock{
						"nested": {
							Nesting: configschema.NestingList,
							Block: configschema.Block{
								Attributes: map[string]*configschema.Attribute{
									"compute": {Type: cty.String, Optional: true},
									"value":   {Type: cty.String, Optional: true},
								},
							},
						},
					},
				},
			},
		},
		DataSources: map[string]providers.Schema{
			"test_data_source": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"compute": {Type: cty.String, Optional: true},
						"value":   {Type: cty.String, Computed: true},
					},
				},
			},
		},
	}
}

func testSchemas() *terraform.Schemas {
	provider := testProvider()
	return &terraform.Schemas{
		Providers: map[addrs.Provider]*terraform.ProviderSchema{
			addrs.NewDefaultProvider("test"): provider.ProviderSchema(),
		},
	}
}

const basicStateOutput = `# data.test_data_source.data:
data "test_data_source" "data" {
    compute = "sure"
}

# test_resource.baz[0]:
resource "test_resource" "baz" {
    woozles = "confuzles"
}


Outputs:

bar = "bar value"`

const nestedStateOutput = `# test_resource.baz[0]:
resource "test_resource" "baz" {
    woozles = "confuzles"

    nested {
        value = "42"
    }
}`

const deposedNestedStateOutput = `# test_resource.baz[0]:
resource "test_resource" "baz" {
    woozles = "confuzles"

    nested {
        value = "42"
    }
}

# test_resource.baz[0]: (deposed object 1234)
resource "test_resource" "baz" {
    woozles = "confuzles"

    nested {
        value = "42"
    }
}`

const onlyDeposedOutput = `# test_resource.baz[0]:
# test_resource.baz[0]: (deposed object 1234)
resource "test_resource" "baz" {
    woozles = "confuzles"

    nested {
        value = "42"
    }
}

# test_resource.baz[0]: (deposed object 5678)
resource "test_resource" "baz" {
    woozles = "confuzles"

    nested {
        value = "42"
    }
}`

const stateWithMoreOutputsOutput = `# test_resource.baz[0]:
resource "test_resource" "baz" {
    woozles = "confuzles"
}


Outputs:

bool_var = true
int_var = 42
map_var = {
    "first"  = "foo"
    "second" = "bar"
}
sensitive_var = (sensitive value)
string_var = "string value"`

func basicState(t *testing.T) *states.State {
	state := states.NewState()

	rootModule := state.RootModule()
	if rootModule == nil {
		t.Errorf("root module is nil; want valid object")
	}

	rootModule.SetLocalValue("foo", cty.StringVal("foo value"))
	rootModule.SetOutputValue("bar", cty.StringVal("bar value"), false)
	rootModule.SetResourceInstanceCurrent(
		addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_resource",
			Name: "baz",
		}.Instance(addrs.IntKey(0)),
		&states.ResourceInstanceObjectSrc{
			Status:        states.ObjectReady,
			SchemaVersion: 1,
			AttrsJSON:     []byte(`{"woozles":"confuzles"}`),
		},
		addrs.AbsProviderConfig{
			Provider: addrs.NewDefaultProvider("test"),
			Module:   addrs.RootModule,
		},
	)
	rootModule.SetResourceInstanceCurrent(
		addrs.Resource{
			Mode: addrs.DataResourceMode,
			Type: "test_data_source",
			Name: "data",
		}.Instance(addrs.NoKey),
		&states.ResourceInstanceObjectSrc{
			Status:        states.ObjectReady,
			SchemaVersion: 1,
			AttrsJSON:     []byte(`{"compute":"sure"}`),
		},
		addrs.AbsProviderConfig{
			Provider: addrs.NewDefaultProvider("test"),
			Module:   addrs.RootModule,
		},
	)
	return state
}

func stateWithMoreOutputs(t *testing.T) *states.State {
	state := states.NewState()

	rootModule := state.RootModule()
	if rootModule == nil {
		t.Errorf("root module is nil; want valid object")
	}

	rootModule.SetOutputValue("string_var", cty.StringVal("string value"), false)
	rootModule.SetOutputValue("int_var", cty.NumberIntVal(42), false)
	rootModule.SetOutputValue("bool_var", cty.BoolVal(true), false)
	rootModule.SetOutputValue("sensitive_var", cty.StringVal("secret!!!"), true)
	rootModule.SetOutputValue("map_var", cty.MapVal(map[string]cty.Value{
		"first":  cty.StringVal("foo"),
		"second": cty.StringVal("bar"),
	}), false)

	rootModule.SetResourceInstanceCurrent(
		addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_resource",
			Name: "baz",
		}.Instance(addrs.IntKey(0)),
		&states.ResourceInstanceObjectSrc{
			Status:        states.ObjectReady,
			SchemaVersion: 1,
			AttrsJSON:     []byte(`{"woozles":"confuzles"}`),
		},
		addrs.AbsProviderConfig{
			Provider: addrs.NewDefaultProvider("test"),
			Module:   addrs.RootModule,
		},
	)
	return state
}

func nestedState(t *testing.T) *states.State {
	state := states.NewState()

	rootModule := state.RootModule()
	if rootModule == nil {
		t.Errorf("root module is nil; want valid object")
	}

	rootModule.SetResourceInstanceCurrent(
		addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_resource",
			Name: "baz",
		}.Instance(addrs.IntKey(0)),
		&states.ResourceInstanceObjectSrc{
			Status:        states.ObjectReady,
			SchemaVersion: 1,
			AttrsJSON:     []byte(`{"woozles":"confuzles","nested": [{"value": "42"}]}`),
		},
		addrs.AbsProviderConfig{
			Provider: addrs.NewDefaultProvider("test"),
			Module:   addrs.RootModule,
		},
	)
	return state
}

func deposedState(t *testing.T) *states.State {
	state := nestedState(t)
	rootModule := state.RootModule()
	rootModule.SetResourceInstanceDeposed(
		addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_resource",
			Name: "baz",
		}.Instance(addrs.IntKey(0)),
		states.DeposedKey("1234"),
		&states.ResourceInstanceObjectSrc{
			Status:        states.ObjectReady,
			SchemaVersion: 1,
			AttrsJSON:     []byte(`{"woozles":"confuzles","nested": [{"value": "42"}]}`),
		},
		addrs.AbsProviderConfig{
			Provider: addrs.NewDefaultProvider("test"),
			Module:   addrs.RootModule,
		},
	)
	return state
}

// replicate a corrupt resource where only a deposed exists
func onlyDeposedState(t *testing.T) *states.State {
	state := states.NewState()

	rootModule := state.RootModule()
	if rootModule == nil {
		t.Errorf("root module is nil; want valid object")
	}

	rootModule.SetResourceInstanceDeposed(
		addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_resource",
			Name: "baz",
		}.Instance(addrs.IntKey(0)),
		states.DeposedKey("1234"),
		&states.ResourceInstanceObjectSrc{
			Status:        states.ObjectReady,
			SchemaVersion: 1,
			AttrsJSON:     []byte(`{"woozles":"confuzles","nested": [{"value": "42"}]}`),
		},
		addrs.AbsProviderConfig{
			Provider: addrs.NewDefaultProvider("test"),
			Module:   addrs.RootModule,
		},
	)
	rootModule.SetResourceInstanceDeposed(
		addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_resource",
			Name: "baz",
		}.Instance(addrs.IntKey(0)),
		states.DeposedKey("5678"),
		&states.ResourceInstanceObjectSrc{
			Status:        states.ObjectReady,
			SchemaVersion: 1,
			AttrsJSON:     []byte(`{"woozles":"confuzles","nested": [{"value": "42"}]}`),
		},
		addrs.AbsProviderConfig{
			Provider: addrs.NewDefaultProvider("test"),
			Module:   addrs.RootModule,
		},
	)
	return state
}
