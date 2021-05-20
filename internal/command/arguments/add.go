package arguments

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Add represents the command-line arguments for the Add command.
type Add struct {
	// Addr specifies which resource to generate configuration for.
	Addr addrs.AbsResourceInstance

	// FromResourceAddr specifies the address of an existing resource in state
	// which should be used to populate the template.
	FromResourceAddr *addrs.AbsResourceInstance

	// OutPath contains an optional path to store the generated configuration.
	OutPath string

	// Optional specifies whether or not to include optional attributes in the
	// generated configuration. Defaults to false.
	Optional bool

	// Provider specifies the provider for the target.
	Provider addrs.Provider

	// State from the common extended flags
	State *State

	// ViewType specifies which output format to use
	ViewType ViewType
}

func ParseAdd(args []string) (*Add, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	add := &Add{State: &State{}, ViewType: ViewHuman}

	var provider string
	var fromAddr string

	cmdFlags := extendedFlagSet("add", add.State, nil, nil)
	cmdFlags.StringVar(&fromAddr, "from-state", "", "fill attribute values from a resource already managed by terraform")
	cmdFlags.BoolVar(&add.Optional, "optional", false, "include optional attributes")
	cmdFlags.StringVar(&add.OutPath, "out", "", "out")
	cmdFlags.StringVar(&provider, "provider", "", "provider")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
		return add, diags
	}

	if provider != "" {
		absProvider, providerDiags := addrs.ParseProviderSourceString(provider)
		if providerDiags.HasErrors() {
			// The diagnostics returned from ParseProviderSourceString are
			// specific to the "source" attribute and not suitable for this use
			// case.
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				fmt.Sprintf("Invalid provider string: %s", provider),
				`The "provider" argument must be in the format "[hostname/][namespace/]name"`,
			))
			return add, diags
		}
		add.Provider = absProvider
	}

	args = cmdFlags.Args()

	if len(args) == 0 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Too few command line arguments",
			"Expected exactly one positional argument.",
		))
		return add, diags
	}

	if len(args) > 1 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Too many command line arguments",
			"Expected exactly one positional argument.",
		))
		return add, diags
	}

	// parse address from the argument
	addr, addrDiags := addrs.ParseAbsResourceInstanceStr(args[0])
	if addrDiags.HasErrors() {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			fmt.Sprintf("Error parsing resource address: %s", args[0]),
			"This command requires that the address argument references one specific resource instance.",
		))
		return add, diags
	}
	add.Addr = addr

	if fromAddr != "" {
		stateAddr, addrDiags := addrs.ParseAbsResourceInstanceStr(fromAddr)
		if addrDiags.HasErrors() {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				fmt.Sprintf("Error parsing resource address: %s", stateAddr),
				fmt.Sprintf("Error parsing -from-state resource address: %s", addrDiags.Err().Error()),
			))
			return add, diags
		}
		add.FromResourceAddr = &stateAddr

		if stateAddr.Resource.Resource.Type != addr.Resource.Resource.Type {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Resource type mismatch",
				"The target address and -from-state address must have the same resource type.",
			))
			return add, diags
		}
	}

	return add, diags
}
