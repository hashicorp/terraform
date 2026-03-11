// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package statekeys

import (
	"github.com/hashicorp/terraform/internal/collections"
)

type KeySet collections.Set[Key]

// NewKeySet returns an initialized set of [Key] that's ready to use and
// treats two keys as unique if they have the same string representation.
func NewKeySet() collections.Set[Key] {
	return collections.NewSetFunc[Key](stateKeyUniqueKey)
}

// NewKeyMap returns an initialized map from [Key] to V that's ready to use and
// treats two keys as unique if they have the same string representation.
func NewKeyMap[V any]() collections.Map[Key, V] {
	return collections.NewMapFunc[Key, V](stateKeyUniqueKey)
}

// stateKeyCollectionsKey is an internal adapter so that [statekeys.Key] values
// can be used as [collections.Set] elements and [collections.Map] keys.
type stateKeyCollectionsKey string

// IsUniqueKey implements collections.UniqueKey.
func (stateKeyCollectionsKey) IsUniqueKey(Key) {
}

func stateKeyUniqueKey(k Key) collections.UniqueKey[Key] {
	return stateKeyCollectionsKey(String(k))
}
