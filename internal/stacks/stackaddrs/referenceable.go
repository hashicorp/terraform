// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackaddrs

// Referenceable is a type set containing all address types that can be
// the target of an expression-based reference within a particular stack.
type Referenceable interface {
	referenceableSigil()
	String() string
}

var _ Referenceable = Component{}
var _ Referenceable = StackCall{}
var _ Referenceable = InputVariable{}
var _ Referenceable = LocalValue{}
var _ Referenceable = ProviderConfigRef{}
var _ Referenceable = TestOnlyGlobal{}
