package terraform

// HookAction is an enum of actions that can be taken as a result of a hook
// callback. This allows you to modify the behavior of Terraform at runtime.
type HookAction byte

const (
	// HookActionContinue continues with processing as usual.
	HookActionContinue HookAction = iota

	// HookActionHalt halts immediately: no more hooks are processed
	// and the action that Terraform was about to take is cancelled.
	HookActionHalt
)

// Hook is the interface that must be implemented to hook into various
// parts of Terraform, allowing you to inspect or change behavior at runtime.
//
// There are MANY hook points into Terraform. If you only want to implement
// some hook points, but not all (which is the likely case), then embed the
// NilHook into your struct, which implements all of the interface but does
// nothing. Then, override only the functions you want to implement.
type Hook interface {
	// PreRefresh is called before a resource is refreshed.
	PreRefresh(*ResourceState) (HookAction, error)

	// PostRefresh is called after a resource is refreshed.
	PostRefresh(*ResourceState) (HookAction, error)
}

// NilHook is a Hook implementation that does nothing. It exists only to
// simplify implementing hooks. You can embed this into your Hook implementation
// and only implement the functions you are interested in.
type NilHook struct{}

func (*NilHook) PreRefresh(*ResourceState) (HookAction, error) {
	return HookActionContinue, nil
}

func (*NilHook) PostRefresh(*ResourceState) (HookAction, error) {
	return HookActionContinue, nil
}
