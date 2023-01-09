package change

import (
	"strings"

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
