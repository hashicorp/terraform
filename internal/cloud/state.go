package cloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/go-tfe"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"

	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statemgr"
)

// State is similar to remote State and delegates to it, except in the case of output values,
// which use a separate methodology that ensures the caller is authorized to read cloud
// workspace outputs.
type State struct {
	Client *remoteClient

	delegate remote.State
}

var ErrStateVersionUnauthorizedUpgradeState = errors.New(strings.TrimSpace(`
You are not authorized to read the full state version containing outputs.
State versions created by terraform v1.3.0 and newer do not require this level
of authorization and therefore this error can usually be fixed by upgrading the
remote state version.
`))

// Proof that cloud State is a statemgr.Persistent interface
var _ statemgr.Persistent = (*State)(nil)

func NewState(client *remoteClient) *State {
	return &State{
		Client:   client,
		delegate: remote.State{Client: client},
	}
}

// State delegates calls to read State to the remote State
func (s *State) State() *states.State {
	return s.delegate.State()
}

// Lock delegates calls to lock state to the remote State
func (s *State) Lock(info *statemgr.LockInfo) (string, error) {
	return s.delegate.Lock(info)
}

// Unlock delegates calls to unlock state to the remote State
func (s *State) Unlock(id string) error {
	return s.delegate.Unlock(id)
}

// RefreshState delegates calls to refresh State to the remote State
func (s *State) RefreshState() error {
	return s.delegate.RefreshState()
}

// RefreshState delegates calls to refresh State to the remote State
func (s *State) PersistState() error {
	return s.delegate.PersistState()
}

// WriteState delegates calls to write State to the remote State
func (s *State) WriteState(state *states.State) error {
	return s.delegate.WriteState(state)
}

func (s *State) fallbackReadOutputsFromFullState() (map[string]*states.OutputValue, error) {
	log.Printf("[DEBUG] falling back to reading full state")

	if err := s.RefreshState(); err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	state := s.State()
	if state == nil {
		// We know that there is supposed to be state (and this is not simply a new workspace
		// without state) because the fallback is only invoked when outputs are present but
		// detailed types are not available.
		return nil, ErrStateVersionUnauthorizedUpgradeState
	}

	return state.RootModule().OutputValues, nil
}

// GetRootOutputValues fetches output values from Terraform Cloud
func (s *State) GetRootOutputValues() (map[string]*states.OutputValue, error) {
	ctx := context.Background()

	so, err := s.Client.client.StateVersionOutputs.ReadCurrent(ctx, s.Client.workspace.ID)

	if err != nil {
		return nil, fmt.Errorf("could not read state version outputs: %w", err)
	}

	result := make(map[string]*states.OutputValue)

	for _, output := range so.Items {
		if output.DetailedType == nil {
			// If there is no detailed type information available, this state was probably created
			// with a version of terraform < 1.3.0. In this case, we'll eject completely from this
			// function and fall back to the old behavior of reading the entire state file, which
			// requires a higher level of authorization.
			return s.fallbackReadOutputsFromFullState()
		}

		if output.Sensitive {
			// Since this is a sensitive value, the output must be requested explicitly in order to
			// read its value, which is assumed to be present by callers
			sensitiveOutput, err := s.Client.client.StateVersionOutputs.Read(ctx, output.ID)
			if err != nil {
				return nil, fmt.Errorf("could not read state version output %s: %w", output.ID, err)
			}
			output.Value = sensitiveOutput.Value
		}

		cval, err := tfeOutputToCtyValue(*output)
		if err != nil {
			return nil, fmt.Errorf("could not decode output %s (ID %s)", output.Name, output.ID)
		}

		result[output.Name] = &states.OutputValue{
			Value:     cval,
			Sensitive: output.Sensitive,
		}
	}

	return result, nil
}

// tfeOutputToCtyValue decodes a combination of TFE output value and detailed-type to create a
// cty value that is suitable for use in terraform.
func tfeOutputToCtyValue(output tfe.StateVersionOutput) (cty.Value, error) {
	var result cty.Value
	bufType, err := json.Marshal(output.DetailedType)
	if err != nil {
		return result, fmt.Errorf("could not marshal output %s type: %w", output.ID, err)
	}

	var ctype cty.Type
	err = ctype.UnmarshalJSON(bufType)
	if err != nil {
		return result, fmt.Errorf("could not interpret output %s type: %w", output.ID, err)
	}

	result, err = gocty.ToCtyValue(output.Value, ctype)
	if err != nil {
		return result, fmt.Errorf("could not interpret value %v as type %s for output %s: %w", result, ctype.FriendlyName(), output.ID, err)
	}

	return result, nil
}
