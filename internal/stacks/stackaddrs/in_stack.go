package stackaddrs

import "github.com/hashicorp/terraform/internal/collections"

// StackItemConfig is a type set containing all of the address types that make
// sense to consider as belonging statically to a [Stack].
type StackItemConfig[T any] interface {
	inStackConfigSigil()
	String() string
	collections.UniqueKeyer[T]
}

// StackItemDynamic is a type set containing all of the address types that make
// sense to consider as belonging dynamically to a [StackInstance].
type StackItemDynamic[T any] interface {
	inStackInstanceSigil()
	String() string
	collections.UniqueKeyer[T]
}

// InStackConfig is the generic form of addresses representing configuration
// objects belonging to particular nodes in the static tree of stack
// configurations.
type InStackConfig[T StackItemConfig[T]] struct {
	Stack Stack
	Item  T
}

func Config[T StackItemConfig[T]](stackAddr Stack, relAddr T) InStackConfig[T] {
	return InStackConfig[T]{
		Stack: stackAddr,
		Item:  relAddr,
	}
}

func (ist InStackConfig[T]) String() string {
	if ist.Stack.IsRoot() {
		return ist.Item.String()
	}
	return ist.Stack.String() + "." + ist.Item.String()
}

func (ist InStackConfig[T]) UniqueKey() collections.UniqueKey[InStackConfig[T]] {
	return inStackConfigKey[T]{
		stackKey: ist.Stack.UniqueKey(),
		itemKey:  ist.Item.UniqueKey(),
	}
}

type inStackConfigKey[T StackItemConfig[T]] struct {
	stackKey collections.UniqueKey[Stack]
	itemKey  collections.UniqueKey[T]
}

// IsUniqueKey implements collections.UniqueKey.
func (inStackConfigKey[T]) IsUniqueKey(InStackConfig[T]) {}

// InStackInstance is the generic form of addresses representing dynamic
// instances of objects that exist within an instance of a stack.
type InStackInstance[T StackItemDynamic[T]] struct {
	Stack StackInstance
	Item  T
}

func Absolute[T StackItemDynamic[T]](stackAddr StackInstance, relAddr T) InStackInstance[T] {
	return InStackInstance[T]{
		Stack: stackAddr,
		Item:  relAddr,
	}
}

func (ist InStackInstance[T]) String() string {
	if ist.Stack.IsRoot() {
		return ist.Item.String()
	}
	return ist.Stack.String() + "." + ist.Item.String()
}

func (ist InStackInstance[T]) UniqueKey() collections.UniqueKey[InStackInstance[T]] {
	return inStackInstanceKey[T]{
		stackKey: ist.Stack.UniqueKey(),
		itemKey:  ist.Item.UniqueKey(),
	}
}

type inStackInstanceKey[T StackItemDynamic[T]] struct {
	stackKey collections.UniqueKey[StackInstance]
	itemKey  collections.UniqueKey[T]
}

// IsUniqueKey implements collections.UniqueKey.
func (inStackInstanceKey[T]) IsUniqueKey(InStackInstance[T]) {}

// ConfigForAbs returns the "in stack config" equivalent of the given
// "in stack instance" (absolute) address by just discarding any
// instance keys from the stack instance steps.
func ConfigForAbs[T interface {
	StackItemDynamic[T]
	StackItemConfig[T]
}](absAddr InStackInstance[T]) InStackConfig[T] {
	return Config(absAddr.Stack.ConfigAddr(), absAddr.Item)
}
