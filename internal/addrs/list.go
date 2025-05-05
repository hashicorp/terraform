// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/collections"
)

// List is an address for a list block within a query configuration
type List struct {
	collections.UniqueKeyer[List]
	referenceable
	Type string
	Name string
}

func (r List) String() string {
	return fmt.Sprintf("%s.%s", r.Type, r.Name)
}

type ListResource struct {
	List     List
	Resource Resource
}

func (l List) UniqueKey() UniqueKey {
	return l // A List is its own UniqueKey
}

func (r List) uniqueKeySigil() {}

func (r List) Absolute() {}

// ListInstance is an address for a specific instance of a list.
// When a list is defined in configuration with "count" or "for_each" it
// produces zero or more instances, which can be addressed using this type.
type ListInstance struct {
	referenceable
	List List
	Key  InstanceKey
}

// Instance produces the address for a specific instance of the receiver
// that is idenfied by the given key.
func (r List) Instance(key InstanceKey) ListInstance {
	return ListInstance{
		List: r,
		Key:  key,
	}
}

// func (r List) Equal(o List) bool {
// 	return r.Mode == o.Mode && r.Name == o.Name && r.Type == o.Type
// }

// func (r List) Less(o List) bool {
// 	switch {
// 	case r.Mode != o.Mode:
// 		return r.Mode == DataResourceMode

// 	case r.Type != o.Type:
// 		return r.Type < o.Type

// 	case r.Name != o.Name:
// 		return r.Name < o.Name

// 	default:
// 		return false
// 	}
// }

// func (r List) UniqueKey() UniqueKey {
// 	return r // A List is its own UniqueKey
// }

// func (r List) uniqueKeySigil() {}

// // Instance produces the address for a specific instance of the receiver
// // that is idenfied by the given key.
// func (r List) Instance(key InstanceKey) ListInstance {
// 	return ListInstance{
// 		List: r,
// 		Key:  key,
// 	}
// }

// // Absolute returns an AbsResource from the receiver and the given module
// // instance address.
// func (r List) Absolute(module ModuleInstance) AbsResource {
// 	return AbsResource{
// 		Module: module,
// 		List:   r,
// 	}
// }

// // InModule returns a ConfigResource from the receiver and the given module
// // address.
// func (r List) InModule(module Module) ConfigResource {
// 	return ConfigResource{
// 		Module: module,
// 		List:   r,
// 	}
// }

// // ImpliedProvider returns the implied provider type name, for e.g. the "aws" in
// // "aws_instance"
// func (r List) ImpliedProvider() string {
// 	typeName := r.Type
// 	if under := strings.Index(typeName, "_"); under != -1 {
// 		typeName = typeName[:under]
// 	}

// 	return typeName
// }

// // ListInstance is an address for a specific instance of a resource.
// // When a resource is defined in configuration with "count" or "for_each" it
// // produces zero or more instances, which can be addressed using this type.
// type ListInstance struct {
// 	referenceable
// 	List List
// 	Key  InstanceKey
// }

// func (r ListInstance) ContainingResource() List {
// 	return r.List
// }

// func (r ListInstance) String() string {
// 	if r.Key == NoKey {
// 		return r.List.String()
// 	}
// 	return r.List.String() + r.Key.String()
// }

// func (r ListInstance) Equal(o ListInstance) bool {
// 	return r.Key == o.Key && r.List.Equal(o.List)
// }

// func (r ListInstance) Less(o ListInstance) bool {
// 	if !r.List.Equal(o.List) {
// 		return r.List.Less(o.List)
// 	}

// 	if r.Key != o.Key {
// 		return InstanceKeyLess(r.Key, o.Key)
// 	}

// 	return false
// }

// func (r ListInstance) UniqueKey() UniqueKey {
// 	return r // A ListInstance is its own UniqueKey
// }

// func (r ListInstance) uniqueKeySigil() {}

// // Absolute returns an AbsResourceInstance from the receiver and the given module
// // instance address.
// func (r ListInstance) Absolute(module ModuleInstance) AbsResourceInstance {
// 	return AbsResourceInstance{
// 		Module: module,
// 		List:   r,
// 	}
// }

// // AbsResource is an absolute address for a resource under a given module path.
// type AbsResource struct {
// 	targetable
// 	Module ModuleInstance
// 	List   List
// }

// // List returns the address of a particular resource within the receiver.
// func (m ModuleInstance) List(mode ResourceMode, typeName string, name string) AbsResource {
// 	return AbsResource{
// 		Module: m,
// 		List: List{
// 			Mode: mode,
// 			Type: typeName,
// 			Name: name,
// 		},
// 	}
// }

// // Instance produces the address for a specific instance of the receiver
// // that is idenfied by the given key.
// func (r AbsResource) Instance(key InstanceKey) AbsResourceInstance {
// 	return AbsResourceInstance{
// 		Module: r.Module,
// 		List:   r.List.Instance(key),
// 	}
// }

// // Config returns the unexpanded ConfigResource for this AbsResource.
// func (r AbsResource) Config() ConfigResource {
// 	return ConfigResource{
// 		Module: r.Module.Module(),
// 		List:   r.List,
// 	}
// }

// // TargetContains implements Targetable by returning true if the given other
// // address is either equal to the receiver or is an instance of the
// // receiver.
// func (r AbsResource) TargetContains(other Targetable) bool {
// 	switch to := other.(type) {

// 	case AbsResource:
// 		// We'll use our stringification as a cheat-ish way to test for equality.
// 		return to.String() == r.String()

// 	case ConfigResource:
// 		// if an absolute resource from parsing a target address contains a
// 		// ConfigResource, the string representation will match
// 		return to.String() == r.String()

// 	case AbsResourceInstance:
// 		return r.TargetContains(to.ContainingResource())

// 	default:
// 		return false

// 	}
// }

// func (r AbsResource) AddrType() TargetableAddrType {
// 	return AbsResourceAddrType
// }

// func (r AbsResource) String() string {
// 	if len(r.Module) == 0 {
// 		return r.List.String()
// 	}
// 	return fmt.Sprintf("%s.%s", r.Module.String(), r.List.String())
// }

// // AffectedAbsResource returns the AbsResource.
// func (r AbsResource) AffectedAbsResource() AbsResource {
// 	return r
// }

// func (r AbsResource) Equal(o AbsResource) bool {
// 	return r.Module.Equal(o.Module) && r.List.Equal(o.List)
// }

// func (r AbsResource) Less(o AbsResource) bool {
// 	if !r.Module.Equal(o.Module) {
// 		return r.Module.Less(o.Module)
// 	}

// 	if !r.List.Equal(o.List) {
// 		return r.List.Less(o.List)
// 	}

// 	return false
// }

// func (r AbsResource) absMoveableSigil() {
// 	// AbsResource is moveable
// }

// type absResourceKey string

// func (r absResourceKey) uniqueKeySigil() {}

// func (r AbsResource) UniqueKey() UniqueKey {
// 	return absResourceKey(r.String())
// }

// // AbsResourceInstance is an absolute address for a resource instance under a
// // given module path.
// type AbsResourceInstance struct {
// 	targetable
// 	Module ModuleInstance
// 	List   ListInstance
// }

// // ListInstance returns the address of a particular resource instance within the receiver.
// func (m ModuleInstance) ListInstance(mode ResourceMode, typeName string, name string, key InstanceKey) AbsResourceInstance {
// 	return AbsResourceInstance{
// 		Module: m,
// 		List: ListInstance{
// 			List: List{
// 				Mode: mode,
// 				Type: typeName,
// 				Name: name,
// 			},
// 			Key: key,
// 		},
// 	}
// }

// // ContainingResource returns the address of the resource that contains the
// // receving resource instance. In other words, it discards the key portion
// // of the address to produce an AbsResource value.
// func (r AbsResourceInstance) ContainingResource() AbsResource {
// 	return AbsResource{
// 		Module: r.Module,
// 		List:   r.List.ContainingResource(),
// 	}
// }

// // ConfigResource returns the address of the configuration block that declared
// // this instance.
// func (r AbsResourceInstance) ConfigResource() ConfigResource {
// 	return ConfigResource{
// 		Module: r.Module.Module(),
// 		List:   r.List.List,
// 	}
// }

// // CurrentObject returns the address of the resource instance's "current"
// // object, which is the one used for expression evaluation etc.
// func (r AbsResourceInstance) CurrentObject() AbsResourceInstanceObject {
// 	return AbsResourceInstanceObject{
// 		ListInstance: r,
// 		DeposedKey:   NotDeposed,
// 	}
// }

// // DeposedObject returns the address of a "deposed" object for the receiving
// // resource instance, which appears only if a create-before-destroy replacement
// // succeeds the create step but fails the destroy step, making the original
// // object live on as a desposed object.
// //
// // If the given [DeposedKey] is [NotDeposed] then this is equivalent to
// // [AbsResourceInstance.CurrentObject].
// func (r AbsResourceInstance) DeposedObject(key DeposedKey) AbsResourceInstanceObject {
// 	return AbsResourceInstanceObject{
// 		ListInstance: r,
// 		DeposedKey:   key,
// 	}
// }

// // TargetContains implements Targetable by returning true if the given other
// // address is equal to the receiver.
// func (r AbsResourceInstance) TargetContains(other Targetable) bool {
// 	switch to := other.(type) {

// 	// while we currently don't start with an AbsResourceInstance as a target
// 	// address, check all resource types for consistency.
// 	case AbsResourceInstance:
// 		// We'll use our stringification as a cheat-ish way to test for equality.
// 		return to.String() == r.String()
// 	case ConfigResource:
// 		return to.String() == r.String()
// 	case AbsResource:
// 		return to.String() == r.String()

// 	default:
// 		return false

// 	}
// }
