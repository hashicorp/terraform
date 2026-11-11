// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package callback

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/zclconf/go-cty/cty"
)

// The kinds of a ValueSource.
const (
	// KindPlanned is a change's After.
	KindPlanned = "planned"
	// KindPrior is a change's Before.
	KindPrior = "prior"
	// KindState is a current object read from state.
	KindState = "state"
)

// ValueSource names a resource value in one policy phase.
type ValueSource struct {
	// Generation is the policy phase; see policy.Client.NextGeneration.
	Generation uint64
	Kind       string
	DeposedKey string
	// Address is the resource instance address, qualified with the component
	// instance in Stacks.
	Address string
}

// ID returns the value ID of the source. The address goes last because it is
// the only part that can contain ":".
func (s ValueSource) ID() string {
	sum := sha256.Sum256(fmt.Appendf(nil, "resource_value:%d:%s:%s:%s", s.Generation, s.Kind, s.DeposedKey, s.Address))
	return base64.StdEncoding.EncodeToString(sum[:])
}

// Resource is a match of the getresources callback. Value keeps its marks.
type Resource struct {
	Value  cty.Value
	Source ValueSource
}

type Functions struct {
	GetResources  func(ctx context.Context, resource string, attrs cty.Value) ([]Resource, bool, error)
	GetDataSource func(ctx context.Context, datasource string, attrs cty.Value) (cty.Value, bool, error)
}

// Registry is an interface for managing callback functions for resources and
// data sources during policy evaluation.
type Registry interface {
	Get(id uint32) (Functions, bool)
	NextID() uint32
	Register(id uint32, fns Functions)
	Unregister(id uint32)
}

var _ Registry = (*InternalRegistry)(nil)

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
