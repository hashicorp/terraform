package debug

import (
	"context"
	"sync"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/tfdiags"
)

// Debugger is the main type representing the debug engine.
//
// Most of the methods of this type are intended for use by a debug adapter
// frontend to control execution in response to commands from the debugger
// user.
type Debugger struct {
	config *configs.Config

	// mu is a mutex that must be held in order to reconfigure the debugger,
	// such as adding and removing breakpoints. We don't use it for
	// interactions with the language runtime, because the runtime's
	// RuntimeContext implementation is expected to take care of whatever
	// synchronization is needed for the concurrency it causes for calls
	// into the Interface API.
	mu sync.Mutex
}

// NewDebugger constructs and returns a new debugger, ready to configure
// and use.
//
// Pass the root Config object for the configuration that the debug session
// will be working with. The debugger will use the configuration to translate
// between source locations, as used by debugger adapter frontends, and
// debuggable object addresses.
func NewDebugger(config *configs.Config) *Debugger {
	return &Debugger{
		config: config,
	}
}

// Interface returns an object that the language runtime should use to
// mark relevant events so that the debugger can, when appropriate, block
// further process and allow the debugger-user to inspect the runtime state.
func (d *Debugger) Interface() Interface {
	return mainInterface{d}
}

// SourceRange returns a pointer to the source range for the definition of
// the given object, if one is available, or the zero value of
// tfdiags.SourceRange if the object is only implied and not explicitly
// configured in source code.
func (d *Debugger) SourceRange(addr addrs.Debuggable) tfdiags.SourceRange {
	switch addr := addr.(type) {
	case addrs.AbsInputVariableInstance:
		mc := d.config.DescendentForInstance(addr.Module)
		if mc == nil {
			return tfdiags.SourceRange{}
		}
		vc := mc.Module.Variables[addr.Variable.Name]
		if vc == nil {
			return tfdiags.SourceRange{}
		}
		return tfdiags.SourceRangeFromHCL(vc.DeclRange)
	case addrs.AbsLocalValue:
		mc := d.config.DescendentForInstance(addr.Module)
		if mc == nil {
			return tfdiags.SourceRange{}
		}
		lc := mc.Module.Locals[addr.LocalValue.Name]
		if lc == nil {
			return tfdiags.SourceRange{}
		}
		return tfdiags.SourceRangeFromHCL(lc.DeclRange)
	case addrs.AbsOutputValue:
		mc := d.config.DescendentForInstance(addr.Module)
		if mc == nil {
			return tfdiags.SourceRange{}
		}
		oc := mc.Module.Outputs[addr.OutputValue.Name]
		if oc == nil {
			return tfdiags.SourceRange{}
		}
		return tfdiags.SourceRangeFromHCL(oc.DeclRange)
	case addrs.AbsResourceInstance:
		// Resource instances don't exist as configuration constructs
		// of their own, so we attribute them to the call that created them.
		mc := d.config.DescendentForInstance(addr.Module)
		if mc == nil {
			return tfdiags.SourceRange{}
		}
		rc := mc.Module.ResourceByAddr(addr.Resource.Resource)
		if rc == nil {
			return tfdiags.SourceRange{}
		}
		return tfdiags.SourceRangeFromHCL(rc.DeclRange)
	case addrs.AbsResource:
		// For resources that have either count or for_each set, we'll
		// indicate that argument as the range for the resource itself,
		// to distinguish the expansion step from the individual instance
		// evaluations. Otherwise, we'll use the resource block itself.
		mc := d.config.DescendentForInstance(addr.Module)
		if mc == nil {
			return tfdiags.SourceRange{}
		}
		rc := mc.Module.ResourceByAddr(addr.Resource)
		if rc == nil {
			return tfdiags.SourceRange{}
		}
		switch {
		case rc.ForEach != nil:
			return tfdiags.SourceRangeFromHCL(rc.ForEach.Range())
		case rc.Count != nil:
			return tfdiags.SourceRangeFromHCL(rc.Count.Range())
		default:
			return tfdiags.SourceRangeFromHCL(rc.DeclRange)
		}
	case addrs.ModuleInstance:
		// Module instances are attributed to the call that
		// created them, since instances don't exist as a source
		// construct independently of the call.
		if addr.IsRoot() {
			// The root has no call, so it's not breakpointable and
			// has no source location.
			return tfdiags.SourceRange{}
		}
		call := addr.Call()
		return d.SourceRange(call)
	case addrs.AbsModuleCall:
		mc := d.config.DescendentForInstance(addr.Caller)
		if mc == nil {
			return tfdiags.SourceRange{}
		}
		cc := mc.Module.ModuleCalls[addr.Call.Name]
		if cc == nil {
			return tfdiags.SourceRange{}
		}

		// For calls that have either count or for_each set, we'll
		// indicate that argument as the range for the resource itself,
		// to distinguish the expansion step from the individual instance
		// evaluations. Otherwise, we'll use the module call block itself.
		switch {
		case cc.ForEach != nil:
			return tfdiags.SourceRangeFromHCL(cc.ForEach.Range())
		case cc.Count != nil:
			return tfdiags.SourceRangeFromHCL(cc.Count.Range())
		default:
			return tfdiags.SourceRangeFromHCL(cc.DeclRange)
		}
	case addrs.AbsProviderConfig:
		// TODO: Look to see if there's a matching config block.
		return tfdiags.SourceRange{}
	default:
		return tfdiags.SourceRange{}
	}
}

// beginDebuggable handles the BeginDebuggable event marked by the language
// runtime.
func (d *Debugger) beginDebuggable(ctx context.Context, addr addrs.Debuggable, runtime RuntimeContext) {
}

// endDebuggable handles the EndDebuggable event marked by the language
// runtime.
func (d *Debugger) endDebuggable(ctx context.Context, addr addrs.Debuggable, runtime RuntimeContext) {
}

// failure handles the Failure event marked by the language runtime.
func (d *Debugger) failure(ctx context.Context, addr addrs.Debuggable, runtime RuntimeContext, diags tfdiags.Diagnostics) {
}
