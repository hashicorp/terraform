package change

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/plans"
)

func Primitive(before, after interface{}, ctype cty.Type) Renderer {
	return &primitiveRenderer{
		before: before,
		after:  after,
		ctype:  ctype,
	}
}

type primitiveRenderer struct {
	NoWarningsRenderer

	before interface{}
	after  interface{}
	ctype  cty.Type
}

func (renderer primitiveRenderer) Render(change Change, indent int, opts RenderOpts) string {
	beforeValue := renderPrimitiveValue(renderer.before, renderer.ctype)
	afterValue := renderPrimitiveValue(renderer.after, renderer.ctype)

	switch change.action {
	case plans.Create:
		return fmt.Sprintf("%s%s", afterValue, change.forcesReplacement())
	case plans.Delete:
		return fmt.Sprintf("%s%s%s", beforeValue, change.nullSuffix(opts.overrideNullSuffix), change.forcesReplacement())
	case plans.NoOp:
		return fmt.Sprintf("%s%s", beforeValue, change.forcesReplacement())
	default:
		return fmt.Sprintf("%s [yellow]->[reset] %s%s", beforeValue, afterValue, change.forcesReplacement())
	}
}

func renderPrimitiveValue(value interface{}, t cty.Type) string {
	switch value.(type) {
	case nil:
		return "[dark_gray]null[reset]"
	}

	switch {
	case t == cty.String:
		return fmt.Sprintf("%q", value.(string))
	case t == cty.Bool:
		if value.(bool) {
			return "true"
		}
		return "false"
	case t == cty.Number:
		return fmt.Sprintf("%g", value)
	default:
		panic("unrecognized primitive type: " + t.FriendlyName())
	}
}
