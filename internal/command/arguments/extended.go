package arguments

import (
	"flag"
	"fmt"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// DefaultParallelism is the limit Terraform places on total parallel
// operations as it walks the dependency graph.
const DefaultParallelism = 10

// State describes arguments which are used to define how Terraform interacts
// with state.
type State struct {
	// Lock controls whether or not the state manager is used to lock state
	// during operations.
	Lock bool

	// LockTimeout allows setting a time limit on acquiring the state lock.
	// The default is 0, meaning no limit.
	LockTimeout time.Duration

	// StatePath specifies a non-default location for the state file. The
	// default value is blank, which is interpeted as "terraform.tfstate".
	StatePath string

	// StateOutPath specifies a different path to write the final state file.
	// The default value is blank, which results in state being written back to
	// StatePath.
	StateOutPath string

	// BackupPath specifies the path where a backup copy of the state file will
	// be stored before the new state is written. The default value is blank,
	// which is interpreted as StateOutPath +
	// ".backup".
	BackupPath string
}

// Operation describes arguments which are used to configure how a Terraform
// operation such as a plan or apply executes.
type Operation struct {
	// PlanMode selects one of the mutually-exclusive planning modes that
	// decides the overall goal of a plan operation. This field is relevant
	// only for an operation that produces a plan.
	PlanMode plans.Mode

	// Parallelism is the limit Terraform places on total parallel operations
	// as it walks the dependency graph.
	Parallelism int

	// Refresh controls whether or not the operation should refresh existing
	// state before proceeding. Default is true.
	Refresh bool

	// Targets allow limiting an operation to a set of resource addresses and
	// their dependencies.
	Targets []addrs.Targetable

	// ForceReplace addresses cause Terraform to force a particular set of
	// resource instances to generate "replace" actions in any plan where they
	// would normally have generated "no-op" or "update" actions.
	//
	// This is currently limited to specific instances because typical uses
	// of replace are associated with only specific remote objects that the
	// user has somehow learned to be malfunctioning, in which case it
	// would be unusual and potentially dangerous to replace everything under
	// a module all at once. We could potentially loosen this later if we
	// learn a use-case for broader matching.
	ForceReplace []addrs.AbsResourceInstance

	// These private fields are used only temporarily during decoding. Use
	// method Parse to populate the exported fields from these, validating
	// the raw values in the process.
	targetsRaw      []string
	forceReplaceRaw []string
	destroyRaw      bool
	refreshOnlyRaw  bool
}

// Parse must be called on Operation after initial flag parse. This processes
// the raw target flags into addrs.Targetable values, returning diagnostics if
// invalid.
func (o *Operation) Parse() tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	o.Targets = nil

	for _, tr := range o.targetsRaw {
		traversal, syntaxDiags := hclsyntax.ParseTraversalAbs([]byte(tr), "", hcl.Pos{Line: 1, Column: 1})
		if syntaxDiags.HasErrors() {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				fmt.Sprintf("Invalid target %q", tr),
				syntaxDiags[0].Detail,
			))
			continue
		}

		target, targetDiags := addrs.ParseTarget(traversal)
		if targetDiags.HasErrors() {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				fmt.Sprintf("Invalid target %q", tr),
				targetDiags[0].Description().Detail,
			))
			continue
		}

		o.Targets = append(o.Targets, target.Subject)
	}

	for _, raw := range o.forceReplaceRaw {
		traversal, syntaxDiags := hclsyntax.ParseTraversalAbs([]byte(raw), "", hcl.Pos{Line: 1, Column: 1})
		if syntaxDiags.HasErrors() {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				fmt.Sprintf("Invalid force-replace address %q", raw),
				syntaxDiags[0].Detail,
			))
			continue
		}

		addr, addrDiags := addrs.ParseAbsResourceInstance(traversal)
		if addrDiags.HasErrors() {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				fmt.Sprintf("Invalid force-replace address %q", raw),
				addrDiags[0].Description().Detail,
			))
			continue
		}

		if addr.Resource.Resource.Mode != addrs.ManagedResourceMode {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				fmt.Sprintf("Invalid force-replace address %q", raw),
				"Only managed resources can be used with the -replace=... option.",
			))
			continue
		}

		o.ForceReplace = append(o.ForceReplace, addr)
	}

	// If you add a new possible value for o.PlanMode here, consider also
	// adding a specialized error message for it in ParseApplyDestroy.
	switch {
	case o.destroyRaw && o.refreshOnlyRaw:
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Incompatible plan mode options",
			"The -destroy and -refresh-only options are mutually-exclusive.",
		))
	case o.destroyRaw:
		o.PlanMode = plans.DestroyMode
	case o.refreshOnlyRaw:
		o.PlanMode = plans.RefreshOnlyMode
		if !o.Refresh {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Incompatible refresh options",
				"It doesn't make sense to use -refresh-only at the same time as -refresh=false, because Terraform would have nothing to do.",
			))
		}
	default:
		o.PlanMode = plans.NormalMode
	}

	return diags
}

// Vars describes arguments which specify non-default variable values. This
// interfce is unfortunately obscure, because the order of the CLI arguments
// determines the final value of the gathered variables. In future it might be
// desirable for the arguments package to handle the gathering of variables
// directly, returning a map of variable values.
type Vars struct {
	vars     *flagNameValueSlice
	varFiles *flagNameValueSlice
}

func (v *Vars) All() []FlagNameValue {
	if v.vars == nil {
		return nil
	}
	return v.vars.AllItems()
}

func (v *Vars) Empty() bool {
	if v.vars == nil {
		return true
	}
	return v.vars.Empty()
}

// extendedFlagSet creates a FlagSet with common backend, operation, and vars
// flags used in many commands. Target structs for each subset of flags must be
// provided in order to support those flags.
func extendedFlagSet(name string, state *State, operation *Operation, vars *Vars) *flag.FlagSet {
	f := defaultFlagSet(name)

	if state == nil && operation == nil && vars == nil {
		panic("use defaultFlagSet")
	}

	if state != nil {
		f.BoolVar(&state.Lock, "lock", true, "lock")
		f.DurationVar(&state.LockTimeout, "lock-timeout", 0, "lock-timeout")
		f.StringVar(&state.StatePath, "state", "", "state-path")
		f.StringVar(&state.StateOutPath, "state-out", "", "state-path")
		f.StringVar(&state.BackupPath, "backup", "", "backup-path")
	}

	if operation != nil {
		f.IntVar(&operation.Parallelism, "parallelism", DefaultParallelism, "parallelism")
		f.BoolVar(&operation.Refresh, "refresh", true, "refresh")
		f.BoolVar(&operation.destroyRaw, "destroy", false, "destroy")
		f.BoolVar(&operation.refreshOnlyRaw, "refresh-only", false, "refresh-only")
		f.Var((*flagStringSlice)(&operation.targetsRaw), "target", "target")
		f.Var((*flagStringSlice)(&operation.forceReplaceRaw), "replace", "replace")
	}

	// Gather all -var and -var-file arguments into one heterogenous structure
	// to preserve the overall order.
	if vars != nil {
		varsFlags := newFlagNameValueSlice("-var")
		varFilesFlags := varsFlags.Alias("-var-file")
		vars.vars = &varsFlags
		vars.varFiles = &varFilesFlags
		f.Var(vars.vars, "var", "var")
		f.Var(vars.varFiles, "var-file", "var-file")
	}

	return f
}
