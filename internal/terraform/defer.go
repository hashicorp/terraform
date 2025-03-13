// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

// GraphNodeDeferrable is an interface that can be implemented by graph nodes
// that can be deferred.
type GraphNodeDeferrable interface {
	SetDeferred(bool)
	IsUserDeferred() bool
}

type Deferred struct {
	deferred bool
}

func (n *Deferred) SetDeferred(deferred bool) {
	n.deferred = deferred
}

func (n *Deferred) IsUserDeferred() bool {
	return n.deferred
}
