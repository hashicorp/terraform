package backend

import (
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/states/statemgr"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// Nil is a no-op implementation of Backend.
//
// This is useful to embed within another struct to implement all of the
// backend interface for testing.
type Nil struct{}

func (Nil) ConfigSchema() *configschema.Block {
	return &configschema.Block{}
}

func (Nil) PrepareConfig(v cty.Value) (cty.Value, tfdiags.Diagnostics) {
	return v, nil
}

func (Nil) Configure(cty.Value) tfdiags.Diagnostics {
	return nil
}

func (Nil) StateMgr(string) (statemgr.Full, error) {
	// We must return a non-nil manager to adhere to the interface, so
	// we'll return an in-memory-only one.
	return statemgr.NewFullFake(statemgr.NewTransientInMemory(nil), nil), nil
}

func (Nil) DeleteWorkspace(string) error {
	return nil
}

func (Nil) Workspaces() ([]string, error) {
	return []string{DefaultStateName}, nil
}
