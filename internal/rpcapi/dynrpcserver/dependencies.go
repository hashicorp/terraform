package dynrpcserver

import (
	"context"
	"sync"

	tf1 "github.com/hashicorp/terraform/internal/rpcapi/terraform1"
)

type Dependencies struct {
	impl tf1.DependenciesServer
	mu   sync.RWMutex
}

var _ tf1.DependenciesServer = (*Dependencies)(nil)

func NewDependenciesStub() *Dependencies {
	return &Dependencies{}
}

func (s *Dependencies) CloseSourceBundle(a0 context.Context, a1 *tf1.CloseSourceBundle_Request) (*tf1.CloseSourceBundle_Response, error) {
	impl, err := s.realRPCServer()
	if err != nil {
		return nil, err
	}
	return impl.CloseSourceBundle(a0, a1)
}

func (s *Dependencies) OpenSourceBundle(a0 context.Context, a1 *tf1.OpenSourceBundle_Request) (*tf1.OpenSourceBundle_Response, error) {
	impl, err := s.realRPCServer()
	if err != nil {
		return nil, err
	}
	return impl.OpenSourceBundle(a0, a1)
}

func (s *Dependencies) ActivateRPCServer(impl tf1.DependenciesServer) {
	s.mu.Lock()
	s.impl = impl
	s.mu.Unlock()
}

func (s *Dependencies) realRPCServer() (tf1.DependenciesServer, error) {
	s.mu.RLock()
	impl := s.impl
	s.mu.RUnlock()
	if impl == nil {
		return nil, unavailableErr
	}
	return impl, nil
}
