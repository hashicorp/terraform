package stackaddrs

// StackItemConfig is a type set containing all of the address types that make
// sense to consider as belonging statically to a [Stack].
type StackItemConfig interface {
	inStackConfigSigil()
}

// StackItemDynamic is a type set containing all of the address types that make
// sense to consider as belonging dynamically to a [StackInstance].
type StackItemDynamic interface {
	inStackInstanceSigil()
}

// InStackConfig is the generic form of addresses representing configuration
// objects belonging to particular nodes in the static tree of stack
// configurations.
type InStackConfig[T StackItemConfig] struct {
	Stack Stack
	Item  T
}

func Config[T StackItemConfig](stackAddr Stack, relAddr T) InStackConfig[T] {
	return InStackConfig[T]{
		Stack: stackAddr,
		Item:  relAddr,
	}
}

// InStackInstance is the generic form of addresses representing dynamic
// instances of objects that exist within an instance of a stack.
type InStackInstance[T StackItemDynamic] struct {
	Stack StackInstance
	Item  T
}

func Absolute[T StackItemDynamic](stackAddr StackInstance, relAddr T) InStackInstance[T] {
	return InStackInstance[T]{
		Stack: stackAddr,
		Item:  relAddr,
	}
}

// ConfigForAbs returns the "in stack config" equivalent of the given
// "in stack instance" (absolute) address by just discarding any
// instance keys from the stack instance steps.
func ConfigForAbs[T interface {
	StackItemDynamic
	StackItemConfig
}](absAddr InStackInstance[T]) InStackConfig[T] {
	return Config(absAddr.Stack.ConfigAddr(), absAddr.Item)
}
