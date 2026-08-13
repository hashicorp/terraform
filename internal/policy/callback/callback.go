// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package callback

import (
	"context"
	"sync"

	"github.com/zclconf/go-cty/cty"
)

type Direction string

const (
	DirectionInbound  Direction = "inbound"
	DirectionOutbound Direction = "outbound"
)

// RelationshipBlock represents a relationship between two resource types, with a list of related attribute pairs.
// A relationship can also nest other relationships.
type RelationshipBlock struct {
	SubjectType string
	RelatedType string
	// Inbound is true if the source attribute points to the target attribute.
	// A block is outbound by default, so this value will be false in that case.
	Direction      Direction
	AttributePairs []RelatedAttributePair

	// QueryAttributes is a map of literal values to query objects related to the subject.
	// If an object of the related type is matched by these attributes, the reference
	// comparison is skipped.
	QueryAttributes cty.Value

	// A relationship block itself can have a nested block
	Nested *RelationshipBlock
}

// RelatedAttributePair represents a pair of related attributes between two resources.
type RelatedAttributePair struct {
	SubjectAttribute string
	RelatedAttribute string
}

type Functions struct {
	GetResources func(ctx context.Context, resource string, attrs cty.Value) ([]cty.Value, bool, error)
	// RelatedResources returns candidate resources that match the defined relationship block
	RelatedResources func(ctx context.Context, blk *RelationshipBlock) (RelatedResource, error)
	GetDataSource    func(ctx context.Context, datasource string, attrs cty.Value) (cty.Value, bool, error)
}

type RelatedResource struct {
	Related []RelatedResource
	Value   cty.Value
	Partial bool
}

type ResourceFunction struct {
	ResourceType string
	Functions
}

// Registry is an interface for managing callback functions for resources and
// data sources during policy evaluation.
type Registry interface {
	Get(id uint32) (ResourceFunction, bool)
	Register(resource string, fns Functions) uint32
	Unregister(id uint32)
}

var _ Registry = (*InternalRegistry)(nil)

// InternalRegistry stores a mapping of evaluation IDs to callback functions,
// allowing resources to register functions that will be called during their
// policy evaluation.
type InternalRegistry struct {
	lock     sync.RWMutex
	provider map[uint32]ResourceFunction
	counter  uint32
}

func NewRegistry() *InternalRegistry {
	return &InternalRegistry{
		provider: make(map[uint32]ResourceFunction),
	}
}

func (s *InternalRegistry) Register(resource string, fns Functions) uint32 {
	s.lock.Lock()
	defer s.lock.Unlock()
	id := s.counter
	s.counter++
	s.provider[id] = ResourceFunction{Functions: fns, ResourceType: resource}
	return id
}

func (s *InternalRegistry) Unregister(id uint32) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.provider, id)
}

func (s *InternalRegistry) Get(id uint32) (ResourceFunction, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	fns, ok := s.provider[id]
	return fns, ok
}
