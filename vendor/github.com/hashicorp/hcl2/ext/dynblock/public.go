package dynblock

import (
	"github.com/hashicorp/hcl2/hcl"
)

// Expand "dynamic" blocks in the given body, returning a new body that
// has those blocks expanded.
//
// The given EvalContext is used when evaluating "for_each" and "labels"
// attributes within dynamic blocks, allowing those expressions access to
// variables and functions beyond the iterator variable created by the
// iteration.
//
// Expand returns no diagnostics because no blocks are actually expanded
// until a call to Content or PartialContent on the returned body, which
// will then expand only the blocks selected by the schema.
//
// "dynamic" blocks are also expanded automatically within nested blocks
// in the given body, including within other dynamic blocks, thus allowing
// multi-dimensional iteration. However, it is not possible to
// dynamically-generate the "dynamic" blocks themselves except through nesting.
//
//     parent {
//       dynamic "child" {
//         for_each = child_objs
//         content {
//           dynamic "grandchild" {
//             for_each = child.value.children
//             labels   = [grandchild.key]
//             content {
//               parent_key = child.key
//               value      = grandchild.value
//             }
//           }
//         }
//       }
//     }
func Expand(body hcl.Body, ctx *hcl.EvalContext) hcl.Body {
	return &expandBody{
		original:   body,
		forEachCtx: ctx,
	}
}
