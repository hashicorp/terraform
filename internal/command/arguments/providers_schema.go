// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"flag"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ProvidersSchema represents the command-line arguments for the providers
// schema command.
type ProvidersSchema struct {
	JSON bool

	// Vars are the variable-related flags (-var, -var-file).
	Vars *Vars

	// Provider is the normalized fully-qualified provider address parsed from a
	// -provider selector. It is meaningful only when ProviderSet is true.
	Provider    addrs.Provider
	ProviderSet bool

	// Kind is the canonical schema category parsed from a -kind selector. It is
	// meaningful only when KindSet is true.
	Kind    Kind
	KindSet bool

	// Type is the exact, case-sensitive object type parsed from a -type
	// selector. It is meaningful only when TypeSet is true.
	Type    string
	TypeSet bool
}

// selectorFlag is a flag.Value that records each non-empty value supplied for a
// selector flag (-provider, -kind, -type).
//
// Go's stdlib flag package silently keeps only the last value for a repeated
// flag, so a plain StringVar cannot detect repeats. This type counts non-empty
// Set calls so the parser can both treat an empty value (e.g. -provider=) as
// omitted and reject a repeated non-empty flag (see
// proposals/provider-subcommand-filtering/design_decisions.md #5).
type selectorFlag struct {
	values []string
}

var _ flag.Value = (*selectorFlag)(nil)

func (s *selectorFlag) String() string { return "" }

func (s *selectorFlag) Set(raw string) error {
	if raw == "" {
		// An empty selector collapses to omitted. It is intentionally not
		// recorded so it never trips the repeated-flag check below.
		return nil
	}
	s.values = append(s.values, raw)
	return nil
}

// resolve reduces the collected values for a selector flag to a single value.
// It reports whether a value was supplied and appends a diagnostic if the
// non-empty flag was repeated.
func (s *selectorFlag) resolve(name string) (string, bool, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	switch len(s.values) {
	case 0:
		return "", false, diags
	case 1:
		return s.values[0], true, diags
	default:
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			fmt.Sprintf("Duplicate %s flag", name),
			fmt.Sprintf("The %s flag may be set at most once with a non-empty value.", name),
		))
		return "", false, diags
	}
}

// parseProviderSelector parses and normalizes a raw -provider selector value
// into a fully-qualified provider address using the standard provider source
// parser.
//
// Bare names ("aws") and shorthand ("hashicorp/aws") are normalized to the
// default registry/namespace FQN (registry.terraform.io/hashicorp/aws); a full
// source string is returned unchanged. Aliases like "aws.us_east_1", malformed
// sources, and sources with too many parts are rejected by the parser's own
// diagnostic, which names the offending value (see
// proposals/provider-subcommand-filtering/design_decisions.md #3). No manual
// string heuristics are added here.
//
// Callers are responsible for treating an empty value as "omitted" before
// calling this; it is only invoked for a non-empty selector.
func parseProviderSelector(raw string) (addrs.Provider, tfdiags.Diagnostics) {
	return addrs.ParseProviderSourceString(raw)
}

// ParseProvidersSchema processes CLI arguments, returning a ProvidersSchema
// value and errors. If errors are encountered, a ProvidersSchema value is
// still returned representing the best effort interpretation of the arguments.
func ParseProvidersSchema(args []string) (*ProvidersSchema, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	providersSchema := &ProvidersSchema{
		Vars: &Vars{},
	}

	cmdFlags := extendedFlagSet("providers schema", nil, nil, providersSchema.Vars)
	cmdFlags.BoolVar(&providersSchema.JSON, "json", false, "produce JSON output")

	var providerFlag selectorFlag
	cmdFlags.Var(&providerFlag, "provider", "filter to a single provider")

	var kindFlag selectorFlag
	cmdFlags.Var(&kindFlag, "kind", "filter to a single schema category")

	var typeFlag selectorFlag
	cmdFlags.Var(&typeFlag, "type", "filter to a single object type")

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

	if !providersSchema.JSON {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"The -json flag is required",
			"The `terraform providers schema` command requires the `-json` flag.",
		))
	}

	// -provider: parse and normalize the selector syntax at parse time (no
	// schemas needed). Existence among the loaded providers is checked later in
	// the command's Run, after schemas load.
	if raw, ok, providerDiags := providerFlag.resolve("-provider"); ok {
		diags = diags.Append(providerDiags)
		provider, parseDiags := parseProviderSelector(raw)
		diags = diags.Append(parseDiags)
		if !parseDiags.HasErrors() {
			providersSchema.Provider = provider
			providersSchema.ProviderSet = true
		}
	} else {
		diags = diags.Append(providerDiags)
	}

	// -kind: validate against the canonical labels at parse time (a closed
	// enum, so this never needs the loaded schemas).
	if raw, ok, kindDiags := kindFlag.resolve("-kind"); ok {
		diags = diags.Append(kindDiags)
		if kind, valid := ParseProviderSchemaKind(raw); valid {
			providersSchema.Kind = kind
			providersSchema.KindSet = true
		} else {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid -kind value",
				fmt.Sprintf(
					"The -kind value %q is not a recognized schema category. Valid values are: %s.",
					raw, strings.Join(ProviderSchemaKinds(), ", "),
				),
			))
		}
	} else {
		diags = diags.Append(kindDiags)
	}

	// -type: an open-ended, exact, case-sensitive object type. It is stored
	// verbatim; matching happens during post-schema filtering.
	if raw, ok, typeDiags := typeFlag.resolve("-type"); ok {
		diags = diags.Append(typeDiags)
		providersSchema.Type = raw
		providersSchema.TypeSet = true
	} else {
		diags = diags.Append(typeDiags)
	}

	// -type cannot be combined with -kind=provider: the provider configuration
	// schema is not keyed by object type (see
	// proposals/provider-subcommand-filtering/design_decisions.md #7, #8).
	if providersSchema.KindSet && providersSchema.Kind == KindProvider && providersSchema.TypeSet {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid combination of -kind and -type",
			"The -type flag cannot be used with -kind=provider, because the provider configuration schema is not keyed by object type.",
		))
	}

	return providersSchema, diags
}
