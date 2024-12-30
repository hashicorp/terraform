// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackaddrs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// InConfigComponent represents addresses of objects that belong to the modules
// associated with a particular component.
//
// Although the type parameter is rather unconstrained, it doesn't make sense to
// use this for types other than those from package addrs that represent
// configuration constructs, like [addrs.ConfigResource], etc.
type InConfigComponent[T InComponentable] struct {
	Component ConfigComponent
	Item      T
}

// ConfigResource represents a resource configuration from inside a
// particular component.
type ConfigResource = InConfigComponent[addrs.ConfigResource]

// ConfigModule represents a module from inside a particular component.
//
// Note that the string representation of the address of the root module of
// a component is identical to the string representation of the component
// address alone.
type ConfigModule = InConfigComponent[addrs.Module]

func (c InConfigComponent[T]) String() string {
	itemStr := c.Item.String()
	componentStr := c.Component.String()
	if itemStr == "" {
		return componentStr
	}
	return componentStr + "." + itemStr
}

// UniqueKey implements collections.UniqueKeyer.
func (c InConfigComponent[T]) UniqueKey() collections.UniqueKey[InConfigComponent[T]] {
	return inConfigComponentKey[T]{
		componentKey: c.Component.UniqueKey(),
		itemKey:      c.Item.UniqueKey(),
	}
}

type inConfigComponentKey[T InComponentable] struct {
	componentKey collections.UniqueKey[ConfigComponent]
	itemKey      addrs.UniqueKey
}

// IsUniqueKey implements collections.UniqueKey.
func (inConfigComponentKey[T]) IsUniqueKey(InConfigComponent[T]) {}

// InAbsComponentInstance represents addresses of objects that belong to the module
// instances associated with a particular component instance.
//
// Although the type parameter is rather unconstrained, it doesn't make sense to
// use this for types other than those from package addrs that represent
// objects that can belong to Terraform modules, like
// [addrs.AbsResourceInstance], etc.
type InAbsComponentInstance[T InComponentable] struct {
	Component AbsComponentInstance
	Item      T
}

// AbsResource represents a not-yet-expanded resource from inside a particular
// component instance.
type AbsResource = InAbsComponentInstance[addrs.AbsResource]

var _ collections.UniqueKeyer[AbsResource] = AbsResource{}

// AbsResourceInstance represents an instance of a resource from inside a
// particular component instance.
type AbsResourceInstance = InAbsComponentInstance[addrs.AbsResourceInstance]

// AbsResourceInstanceObject represents an object associated with an instance
// of a resource from inside a particular component instance.
type AbsResourceInstanceObject = InAbsComponentInstance[addrs.AbsResourceInstanceObject]

// AbsModuleInstance represents an instance of a module from inside a
// particular component instance.
//
// Note that the string representation of the address of the root module of
// a component instance is identical to the string representation of the
// component instance address alone.
type AbsModuleInstance = InAbsComponentInstance[addrs.ModuleInstance]

func (c InAbsComponentInstance[T]) String() string {
	itemStr := c.Item.String()
	componentStr := c.Component.String()
	if itemStr == "" {
		return componentStr
	}
	return componentStr + "." + itemStr
}

// UniqueKey implements collections.UniqueKeyer.
func (c InAbsComponentInstance[T]) UniqueKey() collections.UniqueKey[InAbsComponentInstance[T]] {
	return inAbsComponentInstanceKey[T]{
		componentKey: c.Component.UniqueKey(),
		itemKey:      c.Item.UniqueKey(),
	}
}

type inAbsComponentInstanceKey[T InComponentable] struct {
	componentKey collections.UniqueKey[AbsComponentInstance]
	itemKey      addrs.UniqueKey
}

// IsUniqueKey implements collections.UniqueKey.
func (inAbsComponentInstanceKey[T]) IsUniqueKey(InAbsComponentInstance[T]) {}

// InComponentable just embeds the interfaces that we require for the type
// parameters of both the [InConfigComponent] and [InAbsComponent] types.
type InComponentable interface {
	addrs.UniqueKeyer
	fmt.Stringer
}

func ParseAbsResourceInstanceObject(traversal hcl.Traversal) (AbsResourceInstanceObject, tfdiags.Diagnostics) {
	stack, remain, diags := parseAbsComponentInstance(traversal)
	if diags.HasErrors() {
		return AbsResourceInstanceObject{}, diags
	}

	resource, diags := addrs.ParseAbsResourceInstance(remain)
	if diags.HasErrors() {
		return AbsResourceInstanceObject{}, diags
	}

	return AbsResourceInstanceObject{
		Component: stack,
		Item: addrs.AbsResourceInstanceObject{
			ResourceInstance: resource,
		},
	}, diags
}

func ParseAbsResourceInstanceObjectStr(s string) (AbsResourceInstanceObject, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	traversal, hclDiags := hclsyntax.ParseTraversalAbs([]byte(s), "", hcl.InitialPos)
	diags = diags.Append(hclDiags)
	if diags.HasErrors() {
		return AbsResourceInstanceObject{}, diags
	}

	ret, moreDiags := ParseAbsResourceInstanceObject(traversal)
	diags = diags.Append(moreDiags)
	return ret, diags
}
