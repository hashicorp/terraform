package command

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/mitchellh/cli"
	"github.com/zclconf/go-cty/cty"
)

// simple test cases with a simple resource schema
func TestAdd_basic(t *testing.T) {
	td := tempDir(t)
	testCopyDir(t, testFixturePath("add/basic"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	p := testProvider()
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id":    {Type: cty.String, Optional: true, Computed: true},
						"ami":   {Type: cty.String, Optional: true, Description: "the ami to use"},
						"value": {Type: cty.String, Required: true, Description: "a value of a thing"},
					},
				},
			},
		},
	}

	overrides := &testingOverrides{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"):                                providers.FactoryFixed(p),
			addrs.NewProvider("registry.terraform.io", "happycorp", "test"): providers.FactoryFixed(p),
		},
	}

	t.Run("basic", func(t *testing.T) {
		view, done := testView(t)
		c := &AddCommand{
			Meta: Meta{
				testingOverrides: overrides,
				View:             view,
			},
		}
		args := []string{"test_instance.new"}
		code := c.Run(args)
		output := done(t)
		if code != 0 {
			fmt.Println(output.Stderr())
			t.Fatalf("wrong exit status. Got %d, want 0", code)
		}
		expected := `resource "test_instance" "new" {
  value = null # REQUIRED string
}
`

		if !cmp.Equal(output.Stdout(), expected) {
			t.Fatalf("wrong output:\n%s", cmp.Diff(expected, output.Stdout()))
		}
	})

	t.Run("basic to file", func(t *testing.T) {
		view, done := testView(t)
		c := &AddCommand{
			Meta: Meta{
				testingOverrides: overrides,
				View:             view,
			},
		}
		outPath := "add.tf"
		args := []string{fmt.Sprintf("-out=%s", outPath), "test_instance.new"}
		code := c.Run(args)
		output := done(t)
		if code != 0 {
			fmt.Println(output.Stderr())
			t.Fatalf("wrong exit status. Got %d, want 0", code)
		}
		expected := `resource "test_instance" "new" {
  value = null # REQUIRED string
}
`
		result, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatalf("error reading result file %s: %s", outPath, err.Error())
		}
		// While the entire directory will get removed once the whole test suite
		// is done, we remove this lest it gets in the way of another (not yet
		// written) test.
		os.Remove(outPath)

		if !cmp.Equal(expected, string(result)) {
			t.Fatalf("wrong output:\n%s", cmp.Diff(expected, string(result)))
		}
	})

	t.Run("optionals", func(t *testing.T) {
		view, done := testView(t)
		c := &AddCommand{
			Meta: Meta{
				testingOverrides: overrides,
				View:             view,
			},
		}
		args := []string{"-optional", "test_instance.new"}
		code := c.Run(args)
		if code != 0 {
			t.Fatalf("wrong exit status. Got %d, want 0", code)
		}
		output := done(t)
		expected := `resource "test_instance" "new" {
  ami   = null # OPTIONAL string
  id    = null # OPTIONAL string
  value = null # REQUIRED string
}
`

		if !cmp.Equal(output.Stdout(), expected) {
			t.Fatalf("wrong output:\n%s", cmp.Diff(expected, output.Stdout()))
		}
	})

	t.Run("alternate provider for resource", func(t *testing.T) {
		view, done := testView(t)
		c := &AddCommand{
			Meta: Meta{
				testingOverrides: overrides,
				View:             view,
			},
		}
		args := []string{"-provider=provider[\"registry.terraform.io/happycorp/test\"].alias", "test_instance.new"}
		code := c.Run(args)
		output := done(t)
		if code != 0 {
			t.Fatalf("wrong exit status. Got %d, want 0", code)
		}

		// The provider happycorp/test has a localname "othertest" in the provider configuration.
		expected := `resource "test_instance" "new" {
  provider = othertest.alias
  value    = null # REQUIRED string
}
`

		if !cmp.Equal(output.Stdout(), expected) {
			t.Fatalf("wrong output:\n%s", cmp.Diff(expected, output.Stdout()))
		}
	})

	t.Run("resource exists error", func(t *testing.T) {
		view, done := testView(t)
		c := &AddCommand{
			Meta: Meta{
				testingOverrides: overrides,
				View:             view,
			},
		}
		args := []string{"test_instance.exists"}
		code := c.Run(args)
		if code != 1 {
			t.Fatalf("wrong exit status. Got %d, want 0", code)
		}

		output := done(t)
		if !strings.Contains(output.Stderr(), "The resource test_instance.exists is already in this configuration") {
			t.Fatalf("missing expected error message: %s", output.Stderr())
		}
	})

	t.Run("provider not in configuration", func(t *testing.T) {
		view, done := testView(t)
		c := &AddCommand{
			Meta: Meta{
				testingOverrides: overrides,
				View:             view,
			},
		}
		args := []string{"toast_instance.new"}
		code := c.Run(args)
		if code != 1 {
			t.Fatalf("wrong exit status. Got %d, want 0", code)
		}

		output := done(t)
		if !strings.Contains(output.Stderr(), "No schema found for provider registry.terraform.io/hashicorp/toast.") {
			t.Fatalf("missing expected error message: %s", output.Stderr())
		}
	})

	t.Run("no schema for resource", func(t *testing.T) {
		view, done := testView(t)
		c := &AddCommand{
			Meta: Meta{
				testingOverrides: overrides,
				View:             view,
			},
		}
		args := []string{"test_pet.meow"}
		code := c.Run(args)
		if code != 1 {
			t.Fatalf("wrong exit status. Got %d, want 0", code)
		}

		output := done(t)
		if !strings.Contains(output.Stderr(), "No resource schema found for test_pet.") {
			t.Fatalf("missing expected error message: %s", output.Stderr())
		}
	})
}

func TestAdd(t *testing.T) {
	td := tempDir(t)
	testCopyDir(t, testFixturePath("add/module"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// a simple hashicorp/test provider, and a more complex happycorp/test provider
	p := testProvider()
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id": {Type: cty.String, Required: true},
					},
				},
			},
		},
	}

	happycorp := testProvider()
	happycorp.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id":    {Type: cty.String, Optional: true, Computed: true},
						"ami":   {Type: cty.String, Optional: true, Description: "the ami to use"},
						"value": {Type: cty.String, Required: true, Description: "a value of a thing"},
						"disks": {
							NestedType: &configschema.Object{
								Nesting: configschema.NestingList,
								Attributes: map[string]*configschema.Attribute{
									"size":        {Type: cty.String, Optional: true},
									"mount_point": {Type: cty.String, Required: true},
								},
							},
							Optional: true,
						},
					},
					BlockTypes: map[string]*configschema.NestedBlock{
						"network_interface": {
							Nesting:  configschema.NestingList,
							MinItems: 1,
							Block: configschema.Block{
								Attributes: map[string]*configschema.Attribute{
									"device_index": {Type: cty.String, Optional: true},
									"description":  {Type: cty.String, Optional: true},
								},
							},
						},
					},
				},
			},
		},
	}
	providerSource, psClose := newMockProviderSource(t, map[string][]string{
		"registry.terraform.io/happycorp/test": {"1.0.0"},
		"registry.terraform.io/hashicorp/test": {"1.0.0"},
	})
	defer psClose()

	overrides := &testingOverrides{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewProvider("registry.terraform.io", "happycorp", "test"): providers.FactoryFixed(happycorp),
			addrs.NewDefaultProvider("test"):                                providers.FactoryFixed(p),
		},
	}

	// the test fixture uses a module, so we need to run init.
	m := Meta{
		testingOverrides: overrides,
		ProviderSource:   providerSource,
		Ui:               new(cli.MockUi),
	}

	init := &InitCommand{
		Meta: m,
	}

	code := init.Run([]string{})
	if code != 0 {
		t.Fatal("init failed")
	}

	t.Run("optional", func(t *testing.T) {
		view, done := testView(t)
		c := &AddCommand{
			Meta: Meta{
				testingOverrides: overrides,
				View:             view,
			},
		}
		args := []string{"-optional", "test_instance.new"}
		code := c.Run(args)
		output := done(t)
		if code != 0 {
			t.Fatalf("wrong exit status. Got %d, want 0", code)
		}

		expected := `resource "test_instance" "new" {
  ami = null           # OPTIONAL string
  disks = [{           # OPTIONAL list of object
    mount_point = null # REQUIRED string
    size        = null # OPTIONAL string
  }]
  id    = null          # OPTIONAL string
  value = null          # REQUIRED string
  network_interface {   # REQUIRED block
    description  = null # OPTIONAL string
    device_index = null # OPTIONAL string
  }
}
`

		if !cmp.Equal(output.Stdout(), expected) {
			t.Fatalf("wrong output:\n%s", cmp.Diff(expected, output.Stdout()))
		}

	})

	t.Run("chooses correct provider for root module", func(t *testing.T) {
		// in the root module of this test fixture, "test" is the local name for "happycorp/test"
		view, done := testView(t)
		c := &AddCommand{
			Meta: Meta{
				testingOverrides: overrides,
				View:             view,
			},
		}
		args := []string{"test_instance.new"}
		code := c.Run(args)
		output := done(t)
		if code != 0 {
			t.Fatalf("wrong exit status. Got %d, want 0", code)
		}

		expected := `resource "test_instance" "new" {
  value = null        # REQUIRED string
  network_interface { # REQUIRED block
  }
}
`

		if !cmp.Equal(output.Stdout(), expected) {
			t.Fatalf("wrong output:\n%s", cmp.Diff(expected, output.Stdout()))
		}
	})

	t.Run("chooses correct provider for child module", func(t *testing.T) {
		// in the child module of this test fixture, "test" is a default "hashicorp/test" provider
		view, done := testView(t)
		c := &AddCommand{
			Meta: Meta{
				testingOverrides: overrides,
				View:             view,
			},
		}
		args := []string{"module.child.test_instance.new"}
		code := c.Run(args)
		output := done(t)
		if code != 0 {
			t.Fatalf("wrong exit status. Got %d, want 0", code)
		}

		expected := `resource "test_instance" "new" {
  id = null # REQUIRED string
}
`

		if !cmp.Equal(output.Stdout(), expected) {
			t.Fatalf("wrong output:\n%s", cmp.Diff(expected, output.Stdout()))
		}
	})

	t.Run("chooses correct provider for an unknown module", func(t *testing.T) {
		// it's weird but ok to use a new/unknown module name; terraform will
		// fall back on default providers (unless a -provider argument is
		// supplied)
		view, done := testView(t)
		c := &AddCommand{
			Meta: Meta{
				testingOverrides: overrides,
				View:             view,
			},
		}
		args := []string{"module.madeup.test_instance.new"}
		code := c.Run(args)
		output := done(t)
		if code != 0 {
			t.Fatalf("wrong exit status. Got %d, want 0", code)
		}

		expected := `resource "test_instance" "new" {
  id = null # REQUIRED string
}
`

		if !cmp.Equal(output.Stdout(), expected) {
			t.Fatalf("wrong output:\n%s", cmp.Diff(expected, output.Stdout()))
		}
	})
}

func TestAdd_from_state(t *testing.T) {
	td := tempDir(t)
	testCopyDir(t, testFixturePath("add/basic"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// write some state
	testState := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "new",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON:    []byte("{\"id\":\"bar\",\"ami\":\"ami-123456\",\"disks\":[{\"mount_point\":\"diska\",\"size\":null}],\"value\":\"bloop\"}"),
				Status:       states.ObjectReady,
				Dependencies: []addrs.ConfigResource{},
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
	})
	f, err := os.Create("terraform.tfstate")
	if err != nil {
		t.Fatalf("failed to create temporary state file: %s", err)
	}
	defer f.Close()
	err = writeStateForTesting(testState, f)
	if err != nil {
		t.Fatalf("failed to write state file: %s", err)
	}

	p := testProvider()
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id":    {Type: cty.String, Optional: true, Computed: true},
						"ami":   {Type: cty.String, Optional: true, Description: "the ami to use"},
						"value": {Type: cty.String, Required: true, Description: "a value of a thing"},
						"disks": {
							NestedType: &configschema.Object{
								Nesting: configschema.NestingList,
								Attributes: map[string]*configschema.Attribute{
									"size":        {Type: cty.String, Optional: true},
									"mount_point": {Type: cty.String, Required: true},
								},
							},
							Optional: true,
						},
					},
					BlockTypes: map[string]*configschema.NestedBlock{
						"network_interface": {
							Nesting:  configschema.NestingList,
							MinItems: 1,
							Block: configschema.Block{
								Attributes: map[string]*configschema.Attribute{
									"device_index": {Type: cty.String, Optional: true},
									"description":  {Type: cty.String, Optional: true},
								},
							},
						},
					},
				},
			},
		},
	}
	overrides := &testingOverrides{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"):                                providers.FactoryFixed(p),
			addrs.NewProvider("registry.terraform.io", "happycorp", "test"): providers.FactoryFixed(p),
		},
	}
	view, done := testView(t)
	c := &AddCommand{
		Meta: Meta{
			testingOverrides: overrides,
			View:             view,
		},
	}

	args := []string{"-from-state", "test_instance.new"}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		fmt.Println(output.Stderr())
		t.Fatalf("wrong exit status. Got %d, want 0", code)
	}

	expected := `resource "test_instance" "new" {
  ami = "ami-123456"
  disks = [
    {
      mount_point = "diska"
      size        = null
    },
  ]
  id    = "bar"
  value = "bloop"
}
`

	if !cmp.Equal(output.Stdout(), expected) {
		t.Fatalf("wrong output:\n%s", cmp.Diff(expected, output.Stdout()))
	}

}
