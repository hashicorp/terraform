package cloud

import (
	"context"
	"encoding/json"
	"fmt"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
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
		return nil, fmt.Errorf("Could not read state version outputs: %w", err)
	}

	result := make(map[string]*states.OutputValue)

	for _, output := range so.Items {
		value, err := s.cloudOutputToCtyValue(ctx, output)
		if err != nil {
			return nil, fmt.Errorf("Could not interpret output value as simple json: %w", err)
		}

		result[output.Name] = &states.OutputValue{
			Value:     *value,
			Sensitive: output.Sensitive,
		}
	}

	return result, nil
}

func (s *State) cloudOutputToCtyValue(ctx context.Context, output *tfe.StateVersionOutput) (*cty.Value, error) {
	value := output.Value
	// If an output is sensitive, the API requires that we fetch this output individually to get the value
	// Terraform will decline to reveal the value under some circumstances, but we must provide it to callers
	if output.Sensitive {
		svo, err := s.Client.client.StateVersionOutputs.Read(ctx, output.ID)
		if err != nil {
			return nil, fmt.Errorf("Could not read sensitive output %s: %w", output.ID, err)
		}

		value = svo.Value
	}

	buf, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("Could not interpret output value: %w", err)
	}

	v := ctyjson.SimpleJSONValue{}
	err = v.UnmarshalJSON(buf)

	if err != nil {
		return nil, fmt.Errorf("Could not interpret output value as simple json: %w", err)
	}

	return &v.Value, err
}
