package arguments

import (
	"fmt"

	tfaddr "github.com/hashicorp/terraform-registry-address"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// MetadataSchema contains the command-line arguments for the metadata
// schema command.
type MetadataSchema struct {
	JSON bool

	// Var, State are the common extended flags
	Vars  *Vars
	State *State // probably should not handle this with extended flags - most state flags don't apply to this command (maybe????)

	Filters SchemaFilters
}

type SchemaFilters struct {
	ProviderFilter    *tfaddr.Provider
	ProvisionerFilter string
	RType             SchemaType // type as in "resource", "action", etc
	TypeName          string     // "aws_instance", "local-exec"
	Prune             bool
	ConfigOnly        bool
	StateOnly         bool
}

// inspired by addrs.ResourceMode, extended to include things that aren't strictly Resources
type SchemaType int

const (
	NoType SchemaType = iota
	DataSource
	Resource
	EphemeralResource
	ListResource
	Function
	Action
	Provider
	Provisioner
	StateStore
	Invalid
)

func ParseMetadataSchemas(args []string) (*MetadataSchema, tfdiags.Diagnostics) {
	schemas := &MetadataSchema{
		Vars:    &Vars{},
		State:   &State{},
		Filters: SchemaFilters{},
	}
	var diags tfdiags.Diagnostics

	// raw args to be parsed
	var providerFilter, rType string

	// flags directly assigned to the schemas struct
	cmdFlags := extendedFlagSet("providers schema", schemas.State, nil, schemas.Vars)
	cmdFlags.BoolVar(&schemas.JSON, "json", false, "produce JSON output")
	cmdFlags.BoolVar(&schemas.Filters.ConfigOnly, "config-only", false, "only return providers in config")
	cmdFlags.BoolVar(&schemas.Filters.StateOnly, "state-only", false, "only return providers in state")
	cmdFlags.StringVar(&schemas.Filters.TypeName, "name", "", "type name")
	cmdFlags.BoolVar(&schemas.Filters.Prune, "prune", false, "limit response to schemas used in the config and/or state")
	cmdFlags.StringVar(&schemas.Filters.ProvisionerFilter, "provisioner", "", "return schemas from this provisioner")

	// more parsing required
	cmdFlags.StringVar(&rType, "type", "", "resource type") // needs better descriptor for resource++
	cmdFlags.StringVar(&providerFilter, "provider", "", "return schemas from this provider")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	args = cmdFlags.Args()
	if len(args) > 0 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Too many command line arguments",
			"Expected no positional arguments.",
		))
	}

	if !schemas.JSON {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"The -json flag is required",
			"The `terraform metadata schema` command requires the `-json` flag.",
		))
	}

	if schemas.Filters.ConfigOnly && schemas.Filters.StateOnly {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Conflicting flags",
			"-config-only and -state-only are mutually exclusive flags.",
		))
	}

	if providerFilter != "" {
		switch providerFilter {
		case "terraform", "builtin/terraform", "hashicorp/terraform": // i'm sure one of the parsers can actually handle this
			tfp := tfaddr.Provider(addrs.NewBuiltInProvider("terraform"))
			schemas.Filters.ProviderFilter = &tfp
		default:
			provider, pDiags := addrs.ParseProviderSourceString(providerFilter)
			if diags.HasErrors() {
				diags.Append(pDiags)
			}
			schemas.Filters.ProviderFilter = &provider
		}
	}

	switch rType {
	case "":
		schemas.Filters.RType = NoType
	case "datasource", "data-source":
		schemas.Filters.RType = DataSource
	case "resource":
		schemas.Filters.RType = Resource
	case "ephemeral", "ephemeral-resource":
		schemas.Filters.RType = EphemeralResource
	case "list", "list-resource":
		schemas.Filters.RType = ListResource
	case "action":
		schemas.Filters.RType = Action
	case "function":
		schemas.Filters.RType = Function
	case "provider", "provider-config":
		schemas.Filters.RType = Provider
	case "provisioner":
		schemas.Filters.RType = Provisioner
	case "state-store":
		schemas.Filters.RType = StateStore
	default:
		schemas.Filters.RType = Invalid
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid arguments",
			fmt.Sprintf("%q is not a valid resource type filter.", rType),
		))
	}

	return schemas, diags
}
