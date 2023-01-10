package renderers

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"
	"github.com/hashicorp/terraform/internal/plans"
)

// NoWarningsRenderer defines a Warnings function that returns an empty list of
// warnings. This can be used by other renderers to ensure we don't see lots of
// repeats of this empty function.
type NoWarningsRenderer struct{}

// WarningsHuman returns an empty slice, as the name NoWarningsRenderer suggests.
func (render NoWarningsRenderer) WarningsHuman(diff computed.Diff, indent int) []string {
	return nil
}

// nullSuffix returns the `-> null` suffix if the change is a delete action, and
// it has not been overridden.
func nullSuffix(override bool, action plans.Action) string {
	if !override && action == plans.Delete {
		return " [dark_gray]-> null[reset]"
	}
	return ""
}

// forcesReplacement returns the `# forces replacement` suffix if this change is
// driving the entire resource to be replaced.
func forcesReplacement(replace bool, override bool) string {
	if replace || override {
		return " [red]# forces replacement[reset]"
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
func unchanged(keyword string, count int) string {
	if count == 1 {
		return fmt.Sprintf("[dark_gray]# (%d unchanged %s hidden)[reset]", count, keyword)
	}
	return fmt.Sprintf("[dark_gray]# (%d unchanged %ss hidden)[reset]", count, keyword)
}

// ensureValidAttributeName checks if `name` contains any HCL syntax and returns
// it surrounded by quotation marks if it does.
func ensureValidAttributeName(name string) string {
	if !hclsyntax.ValidIdentifier(name) {
		return fmt.Sprintf("%q", name)
	}
	return name
}
