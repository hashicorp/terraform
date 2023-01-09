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

func (renderer primitiveRenderer) Render(result Change, indent int, opts RenderOpts) string {
	var beforeValue, afterValue string

	if renderer.before != nil {
		beforeValue = *renderer.before
	} else {
		beforeValue = "[dark_gray]null[reset]"
	}

	if renderer.after != nil {
		afterValue = *renderer.after
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
