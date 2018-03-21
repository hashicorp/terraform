package backend

import (
	"github.com/hashicorp/terraform/config/configschema"
	"github.com/hashicorp/terraform/state"
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

func (Nil) ValidateConfig(cty.Value) tfdiags.Diagnostics {
	return nil
}

func (Nil) Configure(cty.Value) tfdiags.Diagnostics {
	return nil
}

func (Nil) State(string) (state.State, error) {
	// We have to return a non-nil state to adhere to the interface
	return &state.InmemState{}, nil
}

func (Nil) DeleteState(string) error {
	return nil
}

func (Nil) States() ([]string, error) {
	return []string{DefaultStateName}, nil
}
