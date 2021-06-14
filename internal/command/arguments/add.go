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

	// FromState specifies that the configuration should be populated with
	// values from state.
	FromState bool

	// OutPath contains an optional path to store the generated configuration.
	OutPath string

	// Optional specifies whether or not to include optional attributes in the
	// generated configuration. Defaults to false.
	Optional bool

	// Provider specifies the provider for the target.
	Provider *addrs.AbsProviderConfig

	// State from the common extended flags.
	State *State

	// ViewType specifies which output format to use. ViewHuman is currently the
	// only supported view type.
	ViewType ViewType
}

func ParseAdd(args []string) (*Add, tfdiags.Diagnostics) {
	add := &Add{State: &State{}, ViewType: ViewHuman}

	var diags tfdiags.Diagnostics
	var provider string

	cmdFlags := extendedFlagSet("add", add.State, nil, nil)
	cmdFlags.BoolVar(&add.FromState, "from-state", false, "fill attribute values from a resource already managed by terraform")
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

	args = cmdFlags.Args()
	if len(args) != 1 {
		//var adj string
		adj := "few"
		if len(args) > 1 {
			adj = "many"
		}
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			fmt.Sprintf("Too %s command line arguments", adj),
			"Expected exactly one positional argument, giving the address of the resource to generate configuration for.",
		))
		return add, diags
	}

	// parse address from the argument
	addr, addrDiags := addrs.ParseAbsResourceInstanceStr(args[0])
	if addrDiags.HasErrors() {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			fmt.Sprintf("Error parsing resource address: %s", args[0]),
			"This command requires that the address argument specifies one resource instance.",
		))
		return add, diags
	}
	add.Addr = addr

	if provider != "" {
		if add.FromState {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Incompatible command-line options",
				"Cannot use both -from-state and -provider. The provider will be determined from the resource's state.",
			))
			return add, diags
		}

		absProvider, providerDiags := addrs.ParseAbsProviderConfigStr(provider)
		if providerDiags.HasErrors() {
			// The diagnostics returned from ParseAbsProviderConfigStr are
			// not always clear, so we wrap them in a single customized diagnostic.
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				fmt.Sprintf("Invalid provider string: %s", provider),
				providerDiags.Err().Error(),
			))
			return add, diags
		}
		add.Provider = &absProvider
	}

	return add, diags
}
