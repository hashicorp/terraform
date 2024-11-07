// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package testing

import (
	"sync"

	"github.com/zclconf/go-cty/cty"
)

// ResourceStore is a simple data store, that can let the mock provider defined
// in this package store and return interesting values for resources and data
// sources.
type ResourceStore struct {
	mutex sync.RWMutex

	Resources map[string]cty.Value
}

func NewResourceStore() *ResourceStore {
	return &ResourceStore{
		Resources: map[string]cty.Value{},
	}
}

func (rs *ResourceStore) Get(id string) (cty.Value, bool) {
	rs.mutex.RLock()
	defer rs.mutex.RUnlock()

	value, exists := rs.Resources[id]
	return value, exists
}

func (rs *ResourceStore) Set(id string, value cty.Value) {
	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	rs.Resources[id] = value
}

func (rs *ResourceStore) Delete(id string) {
	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	delete(rs.Resources, id)
}

// ResourceStoreBuilder is an implementation of the builder pattern for building
// a ResourceStore with prepopulated values.
type ResourceStoreBuilder struct {
	store *ResourceStore
}

func NewResourceStoreBuilder() *ResourceStoreBuilder {
	return &ResourceStoreBuilder{
		store: NewResourceStore(),
	}
}

func (b *ResourceStoreBuilder) AddResource(id string, value cty.Value) *ResourceStoreBuilder {
	if b.store == nil {
		panic("cannot add resources after calling Build()")
	}

	b.store.Set(id, value)
	return b
}

func (b *ResourceStoreBuilder) Build() *ResourceStore {
	if b.store == nil {
		panic("cannot call Build() more than once")
	}

	store := b.store
	b.store = nil
	return store
}
