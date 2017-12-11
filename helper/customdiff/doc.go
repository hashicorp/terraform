// Package customdiff provides a set of reusable and composable functions
// to enable more "declarative" use of the CustomizeDiff mechanism available
// for resources in package helper/schema.
//
// The intent of these helpers is to make the intent of a set of diff
// customizations easier to see, rather than lost in a sea of Go function
// boilerplate. They should _not_ be used in situations where they _obscure_
// intent, e.g. by over-using the composition functions where a single
// function containing normal Go control flow statements would be more
// straightforward.
package customdiff
