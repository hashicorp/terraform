package cloud

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

// State is similar to remote State and delegates to it, except in the case of output values,
// which use a separate methodology that ensures the caller is authorized to read cloud
// workspace outputs.
type State struct {
	Client *remoteClient

	delegate remote.State
}

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

// GetRootOutputValues fetches output values from Terraform Cloud
func (s *State) GetRootOutputValues() (map[string]*states.OutputValue, error) {
	ctx := context.Background()

	so, err := s.Client.client.StateVersionOutputs.ReadCurrent(ctx, s.Client.workspace.ID)

	if err != nil {
		return nil, fmt.Errorf("could not read state version outputs: %w", err)
	}

	result := make(map[string]*states.OutputValue)

	for _, output := range so.Items {
		if output.Sensitive {
			// Since this is a sensitive value, the output must be requested explicitly in order to
			// read its value, which is assumed to be present by callers
			sensitiveOutput, err := s.Client.client.StateVersionOutputs.Read(ctx, output.ID)
			if err != nil {
				return nil, fmt.Errorf("could not read state version output %s: %w", output.ID, err)
			}
			output.Value = sensitiveOutput.Value
		}

		bufType, err := json.Marshal(output.DetailedType)
		if err != nil {
			return nil, fmt.Errorf("could not marshal output %s type: %w", output.ID, err)
		}

		var ctype cty.Type
		err = ctype.UnmarshalJSON(bufType)
		if err != nil {
			return nil, fmt.Errorf("could not interpret output %s type: %w", output.ID, err)
		}

		cval, err := gocty.ToCtyValue(output.Value, ctype)
		if err != nil {
			return nil, fmt.Errorf("could not interpret value %v as type %s for output %s: %w", cval, ctype.FriendlyName(), output.ID, err)
		}

		result[output.Name] = &states.OutputValue{
			Value:     cval,
			Sensitive: output.Sensitive,
		}
	}

	return result, nil
}
