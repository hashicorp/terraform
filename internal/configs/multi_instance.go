package configs

import "github.com/hashicorp/hcl/v2"

// MultiInstance is an interface implemented by types that represent
// configuration constructs that can expand to refer to more than
// one "instance" of whatever they represent.
type MultiInstance interface {
	multiInstance() // only types in this package may implement this interface

	// CountExpr returns the Count expression for an object that has
	// count-based repetition enabled, or nil if it's not enabled.
	//
	// If CountExpr returns a non-nil expression then ForEachExpr must
	// return a nil expression.
	CountExpr() hcl.Expression

	// ForEachExpr returns the ForEach expression for an object that has
	// for_each-based repetition enabled, or nil if it's not enabled.
	//
	// If ForEachExpr returns a non-nil expression then CountExpr must
	// return a nil expression.
	ForEachExpr() hcl.Expression
}
