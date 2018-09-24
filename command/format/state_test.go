package format

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/states"
	"github.com/mitchellh/colorstring"
	"github.com/zclconf/go-cty/cty"
)

var disabledColorize = &colorstring.Colorize{
	Colors:  colorstring.DefaultColors,
	Disable: true,
}

func TestState(t *testing.T) {
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
			Type: "test_thing",
			Name: "baz",
		}.Instance(addrs.IntKey(0)),
		&states.ResourceInstanceObjectSrc{
			Status:        states.ObjectReady,
			SchemaVersion: 1,
			AttrsJSON:     []byte(`{"woozles":"confuzles"}`),
		},
		addrs.ProviderConfig{
			Type: "test",
		}.Absolute(addrs.RootModuleInstance),
	)

	tests := []struct {
		State *StateOpts
		Want  string
	}{
		{
			&StateOpts{
				State: &states.State{},
				Color: disabledColorize,
			},
			"The state file is empty. No resources are represented.",
		},
		{
			&StateOpts{
				State: state,
				Color: disabledColorize,
			},
			"module.test_module.test_resource.foo",
		},
	}

	for _, tt := range tests {
		got := State(tt.State)
		if got != tt.Want {
			t.Errorf(
				"wrong result\ninput: %v\ngot: %s\nwant: %s",
				tt.State.State, got, tt.Want,
			)
		}
	}
}

func mustParseModuleInstanceStr(s string) addrs.ModuleInstance {
	addr, err := addrs.ParseModuleInstanceStr(s)
	if err != nil {
		fmt.Printf(err.Err().Error())
		panic(err)
	}
	return addr
}
