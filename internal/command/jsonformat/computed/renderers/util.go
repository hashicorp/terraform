package renderers

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/command/format"

	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"
	"github.com/hashicorp/terraform/internal/plans"
)

// NoWarningsRenderer defines a Warnings function that returns an empty list of
// warnings. This can be used by other renderers to ensure we don't see lots of
// repeats of this empty function.
type NoWarningsRenderer struct{}

// WarningsHuman returns an empty slice, as the name NoWarningsRenderer suggests.
func (render NoWarningsRenderer) WarningsHuman(_ computed.Diff, _ int, _ computed.RenderHumanOpts) []string {
	return nil
}

// nullSuffix returns the `-> null` suffix if the change is a delete action, and
// it has not been overridden.
func nullSuffix(action plans.Action, opts computed.RenderHumanOpts) string {
	if !opts.OverrideNullSuffix && action == plans.Delete {
		return opts.Colorize.Color(" [dark_gray]-> null[reset]")
	}
	return ""
}

// forcesReplacement returns the `# forces replacement` suffix if this change is
// driving the entire resource to be replaced.
func forcesReplacement(replace bool, opts computed.RenderHumanOpts) string {
	if replace || opts.OverrideForcesReplacement {
		return opts.Colorize.Color(" [red]# forces replacement[reset]")
	}
	return ""
}

// indent returns whitespace that is the required length for the specified
// indent.
func formatIndent(indent int) string {
	return strings.Repeat("    ", indent)
}

// unchanged prints out a description saying how many of 'keyword' have been
// hidden because they are unchanged or noop actions.
func unchanged(keyword string, count int, opts computed.RenderHumanOpts) string {
	if count == 1 {
		return opts.Colorize.Color(fmt.Sprintf("[dark_gray]# (%d unchanged %s hidden)[reset]", count, keyword))
	}
	return opts.Colorize.Color(fmt.Sprintf("[dark_gray]# (%d unchanged %ss hidden)[reset]", count, keyword))
}

// ensureValidAttributeName checks if `name` contains any HCL syntax and returns
// it surrounded by quotation marks if it does.
func ensureValidAttributeName(name string) string {
	if !hclsyntax.ValidIdentifier(name) {
		return fmt.Sprintf("%q", name)
	}
	return name
}

func colorizeDiffAction(action plans.Action, opts computed.RenderHumanOpts) string {
	return opts.Colorize.Color(format.DiffActionSymbol(action))
}
