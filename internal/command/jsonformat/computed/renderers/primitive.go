package renderers

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/plans"
)

var _ computed.DiffRenderer = (*primitiveRenderer)(nil)

func Primitive(before, after interface{}, ctype cty.Type) computed.DiffRenderer {
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

func (renderer primitiveRenderer) RenderHuman(diff computed.Diff, indent int, opts computed.RenderHumanOpts) string {
	beforeValue := renderPrimitiveValue(renderer.before, renderer.ctype)
	afterValue := renderPrimitiveValue(renderer.after, renderer.ctype)

	switch diff.Action {
	case plans.Create:
		return fmt.Sprintf("%s%s", afterValue, forcesReplacement(diff.Replace))
	case plans.Delete:
		return fmt.Sprintf("%s%s%s", beforeValue, nullSuffix(opts.OverrideNullSuffix, diff.Action), forcesReplacement(diff.Replace))
	case plans.NoOp:
		return fmt.Sprintf("%s%s", beforeValue, forcesReplacement(diff.Replace))
	default:
		return fmt.Sprintf("%s [yellow]->[reset] %s%s", beforeValue, afterValue, forcesReplacement(diff.Replace))
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
