package terraform

import (
	"reflect"
	"sync"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/config/configschema"
)

func TestBuiltinEvalContextProviderInput(t *testing.T) {
	var lock sync.Mutex
	cache := make(map[string]map[string]interface{})

	ctx1 := testBuiltinEvalContext(t)
	ctx1.PathValue = []string{"root"}
	ctx1.ProviderInputConfig = cache
	ctx1.ProviderLock = &lock

	ctx2 := testBuiltinEvalContext(t)
	ctx2.PathValue = []string{"root", "child"}
	ctx2.ProviderInputConfig = cache
	ctx2.ProviderLock = &lock

	expected1 := map[string]interface{}{"value": "foo"}
	ctx1.SetProviderInput("foo", expected1)

	expected2 := map[string]interface{}{"value": "bar"}
	ctx2.SetProviderInput("foo", expected2)

	actual1 := ctx1.ProviderInput("foo")
	actual2 := ctx2.ProviderInput("foo")

	if !reflect.DeepEqual(actual1, expected1) {
		t.Fatalf("bad: %#v %#v", actual1, expected1)
	}
	if !reflect.DeepEqual(actual2, expected2) {
		t.Fatalf("bad: %#v %#v", actual2, expected2)
	}
}

func TestBuildingEvalContextInitProvider(t *testing.T) {
	var lock sync.Mutex

	testP := &MockResourceProvider{
		ResourcesReturn: []ResourceType{
			{
				Name:            "test_thing",
				SchemaAvailable: true,
			},
		},
		DataSourcesReturn: []DataSource{
			{
				Name:            "test_thing",
				SchemaAvailable: true,
			},
		},
		GetSchemaReturn: &ProviderSchema{
			Provider: &configschema.Block{},
			ResourceTypes: map[string]*configschema.Block{
				"test_thing": &configschema.Block{},
			},
			DataSources: map[string]*configschema.Block{
				"test_thing": &configschema.Block{},
			},
		},
	}

	ctx := testBuiltinEvalContext(t)
	ctx.ProviderLock = &lock
	ctx.ProviderCache = make(map[string]ResourceProvider)
	ctx.Components = &basicComponentFactory{
		providers: map[string]ResourceProviderFactory{
			"test": ResourceProviderFactoryFixed(testP),
		},
	}

	_, err := ctx.InitProvider("test", "test")
	if err != nil {
		t.Fatalf("error initializing provider test: %s", err)
	}
	_, err = ctx.InitProvider("test", "test.foo")
	if err != nil {
		t.Fatalf("error initializing provider test.foo: %s", err)
	}

	{
		got := testP.GetSchemaRequest
		want := &ProviderSchemaRequest{
			DataSources:   []string{"test_thing"},
			ResourceTypes: []string{"test_thing"},
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("wrong schema request\ngot: %swant: %s", spew.Sdump(got), spew.Sdump(want))
		}
	}

	{
		schema := ctx.ProviderSchema("test")
		if got, want := schema, testP.GetSchemaReturn; !reflect.DeepEqual(got, want) {
			t.Errorf("wrong schema\ngot: %swant: %s", spew.Sdump(got), spew.Sdump(want))
		}
	}

	{
		schema := ctx.ProviderSchema("test.foo")
		if got, want := schema, testP.GetSchemaReturn; !reflect.DeepEqual(got, want) {
			t.Errorf("wrong schema\ngot: %swant: %s", spew.Sdump(got), spew.Sdump(want))
		}
	}
}

func testBuiltinEvalContext(t *testing.T) *BuiltinEvalContext {
	return &BuiltinEvalContext{}
}
