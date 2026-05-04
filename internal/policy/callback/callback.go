// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package callback

import (
	"sync"
	"sync/atomic"

	"github.com/zclconf/go-cty/cty"
)

type Functions struct {
	GetResources  func(resource string, attrs cty.Value) ([]cty.Value, error)
	GetDataSource func(datasource string, attrs cty.Value) (cty.Value, error)
}

// InternalRegistry stores a mapping of evaluation IDs to callback functions,
// allowing resources to register functions that will be called during their
// policy evaluation.
type InternalRegistry struct {
	lock     sync.RWMutex
	provider map[uint32]Functions
	counter  uint32
}

func NewRegistry() *InternalRegistry {
	return &InternalRegistry{
		provider: make(map[uint32]Functions),
	}
}

func (s *InternalRegistry) Register(id uint32, fns Functions) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.provider[id] = fns
}

func (s *InternalRegistry) Unregister(id uint32) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.provider, id)
}

func (s *InternalRegistry) Get(id uint32) (Functions, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	fns, ok := s.provider[id]
	return fns, ok
}

func (s *InternalRegistry) NextID() uint32 {
	return atomic.AddUint32(&s.counter, 1)
}
