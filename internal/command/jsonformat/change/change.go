package change

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/plans"
)

// Change captures a change to a single block, element or attribute.
//
// It essentially merges common functionality across all types of changes,
// namely the replace logic and the action / change type. Any remaining
// behaviour can be offloaded to the renderer which will be unique for the
// various change types (eg. maps, objects, lists, blocks, primitives, etc.).
type Change struct {
	// renderer captures the uncommon functionality across the different kinds
	// of changes. Each type of change (lists, blocks, sets, etc.) will have a
	// unique renderer.
	renderer Renderer

	// action is the action described by this change (such as create, delete,
	// update, etc.).
	action plans.Action

	// replace tells the Change that it should add the `# forces replacement`
	// suffix.
	//
	// Every single change could potentially add this suffix, so we embed it in
	// the change as common functionality instead of in the specific renderers.
	replace bool
}

// New creates a new Change object with the provided renderer, action and
// replace context.
func New(renderer Renderer, action plans.Action, replace bool) Change {
	return Change{
		renderer: renderer,
		action:   action,
		replace:  replace,
	}
}

// Render prints the Change into a human-readable string referencing the
// specified RenderOpts.
//
// If the returned string is a single line, then indent should be ignored.
//
// If the return string is multiple lines, then indent should be used to offset
// the beginning of all lines but the first by the specified amount.
func (change Change) Render(indent int, opts RenderOpts) string {
	return change.renderer.Render(change, indent, opts)
}

// Warnings returns a list of strings that should be rendered as warnings before
// a given change is rendered.
//
// As with the Render function, the indent should only be applied on multiline
// warnings and on the second and following lines.
func (change Change) Warnings(indent int) []string {
	return change.renderer.Warnings(change, indent)
}

// Action returns the plans.Action that this change describes.
func (change Change) Action() plans.Action {
	return change.action
}

// nullSuffix returns the `-> null` suffix if the change is a delete action, and
// it has not been overridden.
func (change Change) nullSuffix(override bool) string {
	if !override && change.action == plans.Delete {
		return " [dark_gray]-> null[reset]"
	}
	return ""
}

// forcesReplacement returns the `# forces replacement` suffix if this change is
// driving the entire resource to be replaced.
func (change Change) forcesReplacement() string {
	if change.replace {
		return " [red]# forces replacement[reset]"
	}
	return ""
}

// indent returns whitespace that is the required length for the specified
// indent.
func (change Change) indent(indent int) string {
	return strings.Repeat("    ", indent)
}

// emptySymbol returns an empty string that is the same length as an action
// symbol (eg. '  +', '+/-', ...). It is used to offset additional lines in
// change renderer outputs alongside the indent function.
func (change Change) emptySymbol() string {
	return "   "
}

// unchanged prints out a description saying how many of 'keyword' have been
// hidden because they are unchanged or noop actions.
func (change Change) unchanged(keyword string, count int) string {
	if count == 1 {
		return fmt.Sprintf("[dark_gray]# (%d unchanged %s hidden)[reset]", count, keyword)
	}
	return fmt.Sprintf("[dark_gray]# (%d unchanged %ss hidden)[reset]", count, keyword)
}

// ensureValidAttributeName checks if `name` contains any HCL syntax and returns
// it surrounded by quotation marks if it does.
func (change Change) ensureValidAttributeName(name string) string {
	if !hclsyntax.ValidIdentifier(name) {
		return fmt.Sprintf("%q", name)
	}
	return name
}
