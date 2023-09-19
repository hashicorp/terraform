package dynrpcserver

import (
	"context"
	"sync"

	tf1 "github.com/hashicorp/terraform/internal/rpcapi/terraform1"
)

type Stacks struct {
	impl tf1.StacksServer
	mu   sync.RWMutex
}

var _ tf1.StacksServer = (*Stacks)(nil)

func NewStacksStub() *Stacks {
	return &Stacks{}
}

func (s *Stacks) ApplyStackChanges(a0 *tf1.ApplyStackChanges_Request, a1 tf1.Stacks_ApplyStackChangesServer) error {
	impl, err := s.realRPCServer()
	if err != nil {
		return err
	}
	return impl.ApplyStackChanges(a0, a1)
}

func (s *Stacks) CloseStackConfiguration(a0 context.Context, a1 *tf1.CloseStackConfiguration_Request) (*tf1.CloseStackConfiguration_Response, error) {
	impl, err := s.realRPCServer()
	if err != nil {
		return nil, err
	}
	return impl.CloseStackConfiguration(a0, a1)
}

func (s *Stacks) FindStackConfigurationComponents(a0 context.Context, a1 *tf1.FindStackConfigurationComponents_Request) (*tf1.FindStackConfigurationComponents_Response, error) {
	impl, err := s.realRPCServer()
	if err != nil {
		return nil, err
	}
	return impl.FindStackConfigurationComponents(a0, a1)
}

func (s *Stacks) FindStackConfigurationProviders(a0 context.Context, a1 *tf1.FindStackConfigurationProviders_Request) (*tf1.FindStackConfigurationProviders_Response, error) {
	impl, err := s.realRPCServer()
	if err != nil {
		return nil, err
	}
	return impl.FindStackConfigurationProviders(a0, a1)
}

func (s *Stacks) OpenStackConfiguration(a0 context.Context, a1 *tf1.OpenStackConfiguration_Request) (*tf1.OpenStackConfiguration_Response, error) {
	impl, err := s.realRPCServer()
	if err != nil {
		return nil, err
	}
	return impl.OpenStackConfiguration(a0, a1)
}

func (s *Stacks) PlanStackChanges(a0 *tf1.PlanStackChanges_Request, a1 tf1.Stacks_PlanStackChangesServer) error {
	impl, err := s.realRPCServer()
	if err != nil {
		return err
	}
	return impl.PlanStackChanges(a0, a1)
}

func (s *Stacks) ActivateRPCServer(impl tf1.StacksServer) {
	s.mu.Lock()
	s.impl = impl
	s.mu.Unlock()
}

func (s *Stacks) realRPCServer() (tf1.StacksServer, error) {
	s.mu.RLock()
	impl := s.impl
	s.mu.RUnlock()
	if impl == nil {
		return nil, unavailableErr
	}
	return impl, nil
}
