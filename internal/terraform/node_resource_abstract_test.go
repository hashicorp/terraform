package terraform

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/zclconf/go-cty/cty"
)

func TestNodeAbstractResourceProvider(t *testing.T) {
	tests := []struct {
		Addr   addrs.ConfigResource
		Config *configs.Resource
		Want   addrs.Provider
	}{
		{
			Addr: addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "null_resource",
				Name: "baz",
			}.InModule(addrs.RootModule),
			Want: addrs.Provider{
				Hostname:  addrs.DefaultProviderRegistryHost,
				Namespace: "hashicorp",
				Type:      "null",
			},
		},
		{
			Addr: addrs.Resource{
				Mode: addrs.DataResourceMode,
				Type: "terraform_remote_state",
				Name: "baz",
			}.InModule(addrs.RootModule),
			Want: addrs.Provider{
				// As a special case, the type prefix "terraform_" maps to
				// the builtin provider, not the default one.
				Hostname:  addrs.BuiltInProviderHost,
				Namespace: addrs.BuiltInProviderNamespace,
				Type:      "terraform",
			},
		},
		{
			Addr: addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "null_resource",
				Name: "baz",
			}.InModule(addrs.RootModule),
			Config: &configs.Resource{
				// Just enough configs.Resource for the Provider method. Not
				// actually valid for general use.
				Provider: addrs.Provider{
					Hostname:  addrs.DefaultProviderRegistryHost,
					Namespace: "awesomecorp",
					Type:      "happycloud",
				},
			},
			// The config overrides the default behavior.
			Want: addrs.Provider{
				Hostname:  addrs.DefaultProviderRegistryHost,
				Namespace: "awesomecorp",
				Type:      "happycloud",
			},
		},
		{
			Addr: addrs.Resource{
				Mode: addrs.DataResourceMode,
				Type: "terraform_remote_state",
				Name: "baz",
			}.InModule(addrs.RootModule),
			Config: &configs.Resource{
				// Just enough configs.Resource for the Provider method. Not
				// actually valid for general use.
				Provider: addrs.Provider{
					Hostname:  addrs.DefaultProviderRegistryHost,
					Namespace: "awesomecorp",
					Type:      "happycloud",
				},
			},
			// The config overrides the default behavior.
			Want: addrs.Provider{
				Hostname:  addrs.DefaultProviderRegistryHost,
				Namespace: "awesomecorp",
				Type:      "happycloud",
			},
		},
	}

	for _, test := range tests {
		var name string
		if test.Config != nil {
			name = fmt.Sprintf("%s with configured %s", test.Addr, test.Config.Provider)
		} else {
			name = fmt.Sprintf("%s with no configuration", test.Addr)
		}
		t.Run(name, func(t *testing.T) {
			node := &NodeAbstractResource{
				// Just enough NodeAbstractResource for the Provider function.
				// (This would not be valid for some other functions.)
				Addr:   test.Addr,
				Config: test.Config,
			}
			got := node.Provider()
			if got != test.Want {
				t.Errorf("wrong result\naddr:  %s\nconfig: %#v\ngot:   %s\nwant:  %s", test.Addr, test.Config, got, test.Want)
			}
		})
	}
}

func TestNodeAbstractResource_ReadResourceInstanceState(t *testing.T) {
	mockProvider := mockProviderWithResourceTypeSchema("aws_instance", &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"id": {
				Type:     cty.String,
				Optional: true,
			},
		},
	})

	tests := map[string]struct {
		State              *states.State
		Node               *NodeAbstractResource
		ExpectedInstanceId string
	}{
		"ReadState gets primary instance state": {
			State: states.BuildState(func(s *states.SyncState) {
				providerAddr := addrs.AbsProviderConfig{
					Provider: addrs.NewDefaultProvider("aws"),
					Module:   addrs.RootModule,
				}
				oneAddr := addrs.Resource{
					Mode: addrs.ManagedResourceMode,
					Type: "aws_instance",
					Name: "bar",
				}.Absolute(addrs.RootModuleInstance)
				s.SetResourceProvider(oneAddr, providerAddr)
				s.SetResourceInstanceCurrent(oneAddr.Instance(addrs.NoKey), &states.ResourceInstanceObjectSrc{
					Status:    states.ObjectReady,
					AttrsJSON: []byte(`{"id":"i-abc123"}`),
				}, providerAddr)
			}),
			Node: &NodeAbstractResource{
				Addr:             mustConfigResourceAddr("aws_instance.bar"),
				ResolvedProvider: mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
			},
			ExpectedInstanceId: "i-abc123",
		},
	}

	for k, test := range tests {
		t.Run(k, func(t *testing.T) {
			ctx := new(MockEvalContext)
			ctx.StateState = test.State.SyncWrapper()
			ctx.PathPath = addrs.RootModuleInstance
			ctx.ProviderSchemaSchema = mockProvider.ProviderSchema()
			ctx.ProviderProvider = providers.Interface(mockProvider)

			got, readDiags := test.Node.readResourceInstanceState(ctx, test.Node.Addr.Resource.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance))
			if readDiags.HasErrors() {
				t.Fatalf("[%s] Got err: %#v", k, readDiags.Err())
			}

			expected := test.ExpectedInstanceId

			if !(got != nil && got.Value.GetAttr("id") == cty.StringVal(expected)) {
				t.Fatalf("[%s] Expected output with ID %#v, got: %#v", k, expected, got)
			}
		})
	}
}

func TestNodeAbstractResource_ReadResourceInstanceStateDeposed(t *testing.T) {
	mockProvider := mockProviderWithResourceTypeSchema("aws_instance", &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"id": {
				Type:     cty.String,
				Optional: true,
			},
		},
	})

	tests := map[string]struct {
		State              *states.State
		Node               *NodeAbstractResource
		ExpectedInstanceId string
	}{
		"ReadStateDeposed gets deposed instance": {
			State: states.BuildState(func(s *states.SyncState) {
				providerAddr := addrs.AbsProviderConfig{
					Provider: addrs.NewDefaultProvider("aws"),
					Module:   addrs.RootModule,
				}
				oneAddr := addrs.Resource{
					Mode: addrs.ManagedResourceMode,
					Type: "aws_instance",
					Name: "bar",
				}.Absolute(addrs.RootModuleInstance)
				s.SetResourceProvider(oneAddr, providerAddr)
				s.SetResourceInstanceDeposed(oneAddr.Instance(addrs.NoKey), states.DeposedKey("00000001"), &states.ResourceInstanceObjectSrc{
					Status:    states.ObjectReady,
					AttrsJSON: []byte(`{"id":"i-abc123"}`),
				}, providerAddr)
			}),
			Node: &NodeAbstractResource{
				Addr:             mustConfigResourceAddr("aws_instance.bar"),
				ResolvedProvider: mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
			},
			ExpectedInstanceId: "i-abc123",
		},
	}
	for k, test := range tests {
		t.Run(k, func(t *testing.T) {
			ctx := new(MockEvalContext)
			ctx.StateState = test.State.SyncWrapper()
			ctx.PathPath = addrs.RootModuleInstance
			ctx.ProviderSchemaSchema = mockProvider.ProviderSchema()
			ctx.ProviderProvider = providers.Interface(mockProvider)

			key := states.DeposedKey("00000001") // shim from legacy state assigns 0th deposed index this key

			got, readDiags := test.Node.readResourceInstanceStateDeposed(ctx, test.Node.Addr.Resource.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance), key)
			if readDiags.HasErrors() {
				t.Fatalf("[%s] Got err: %#v", k, readDiags.Err())
			}

			expected := test.ExpectedInstanceId

			if !(got != nil && got.Value.GetAttr("id") == cty.StringVal(expected)) {
				t.Fatalf("[%s] Expected output with ID %#v, got: %#v", k, expected, got)
			}
		})
	}
}
