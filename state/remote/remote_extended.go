package remote

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform/state"
)

// See realisation in hashicorp/terraform/backend/remote-state/s3/client.go
type ExtendedClient interface {
	PutRecoveryLog([]byte) error
	PutLostResourceLog([]byte) error
	DeleteRecoveryLog() error
	GetRecoveryLog() (*Payload, error)
}

// Realisation of RecoveryLogWriter interface rof remote state.
func (s *State) WriteRecoveryLog(data []byte) error {
	extendedClient, ok := s.Client.(ExtendedClient)
	if ok {
		return extendedClient.PutRecoveryLog(data)
	} else {
		fmt.Printf("Backend Client does not support 'PutRecoveryLog' functional.")
	}
	return nil
}

func (s *State) WriteLostResourceLog(data []byte) error {
	extendedClient, ok := s.Client.(ExtendedClient)
	if ok {
		extendedClient.PutLostResourceLog(data)
	}
	return nil
}

func (s *State) DeleteRecoveryLog() error {
	extendedClient, ok := s.Client.(ExtendedClient)
	if ok {
		return extendedClient.DeleteRecoveryLog()
	}
	return nil
}

// Realisation of RecoveryLogReader interface rof remote state.
func (s *State) ReadRecoveryLog() (map[string]state.Instance, error) {
	instances := map[string]state.Instance{}
	extendedClient, ok := s.Client.(ExtendedClient)
	if ok {
		payload, err := extendedClient.GetRecoveryLog()
		if err != nil {
			return nil, err
		}
		if payload == nil || payload.Data == nil {
			return map[string]state.Instance{}, nil
		}
		err = json.Unmarshal(payload.Data, &instances)
		return instances, err
	}
	return instances, nil
}
