package change

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/plans"
)

func Primitive(before, after *string) Renderer {
	return primitiveRenderer{
		before: before,
		after:  after,
	}
}

type primitiveRenderer struct {
	NoWarningsRenderer

	before *string
	after  *string
}

func (render primitiveRenderer) Render(result Change, indent int, opts RenderOpts) string {
	var beforeValue, afterValue string

	if render.before != nil {
		beforeValue = *render.before
	} else {
		beforeValue = "[dark_gray]null[reset]"
	}

	if render.after != nil {
		afterValue = *render.after
	} else {
		afterValue = "[dark_gray]null[reset]"
	}

	switch result.action {
	case plans.Create:
		return fmt.Sprintf("%s%s", afterValue, result.forcesReplacement())
	case plans.Delete:
		return fmt.Sprintf("%s%s%s", beforeValue, result.nullSuffix(opts.overrideNullSuffix), result.forcesReplacement())
	case plans.NoOp:
		return fmt.Sprintf("%s%s", beforeValue, result.forcesReplacement())
	default:
		return fmt.Sprintf("%s [yellow]->[reset] %s%s", beforeValue, afterValue, result.forcesReplacement())
	}
}
