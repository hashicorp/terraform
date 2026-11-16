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

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/policy/proto"
)

// Object is one resource object in a policy session (contract §3.4): the
// subject of an evaluation, a related resource, or a match of the
// getresources callback.
type Object struct {
	// Generation is the policy phase; see policy.Client.NextGeneration.
	Generation uint64
	// Address is the resource instance address, relative to the component
	// in Stacks.
	Address string
	// DeposedKey is the deposed key of a deposed object, and "" otherwise.
	DeposedKey string
	// OldObject is set for the object that a replace deletes.
	OldObject bool

	Type     string
	Metadata *proto.PolicyEvaluateResourceRequest_ResourceMetadata
	// Attrs and PriorAttrs are the object's values, keeping their marks, or
	// cty.NilVal when absent: Attrs of a DELETE, and PriorAttrs of a CREATE
	// or of a state value without a change.
	Attrs, PriorAttrs cty.Value
	// Schema is the schema of the resource type, whose sensitive attributes
	// the encoding of the values redacts.
	Schema *configschema.Block
}

// Identifier returns the identifier of the object. The address goes last
// because it is the only part that can contain ":".
func (o *Object) Identifier() string {
	kind := "current"
	switch {
	case o.DeposedKey != "":
		kind = "deposed " + o.DeposedKey
	case o.OldObject:
		kind = "old"
	}
	sum := sha256.Sum256(fmt.Appendf(nil, "resource_object:%d:%s:%s", o.Generation, kind, o.Address))
	return base64.StdEncoding.EncodeToString(sum[:])
}

// Resource is a match of the getresources callback. Value keeps its marks.
// Object is the object that Value is the attrs of, if the plugin accepts
// identifiers for matches.
type Resource struct {
	Value  cty.Value
	Object *Object
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
