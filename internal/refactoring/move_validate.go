package refactoring

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ValidateMoves tests whether all of the given move statements comply with
// both the single-statement validation rules and the "big picture" rules
// that constrain statements in relation to one another.
//
// The validation rules are primarily in terms of the configuration, but
// ValidateMoves also takes the expander that resulted from creating a plan
// so that it can see which instances are defined for each module and resource,
// to precisely validate move statements involving specific-instance addresses.
//
// Because validation depends on the planning result but move execution must
// happen _before_ planning, we have the unusual situation where sibling
// function ApplyMoves must run before ValidateMoves and must therefore
// tolerate and ignore any invalid statements. The plan walk will then
// construct in incorrect plan (because it'll be starting from the wrong
// prior state) but ValidateMoves will block actually showing that invalid
// plan to the user.
func ValidateMoves(stmts []MoveStatement, rootCfg *configs.Config, expander *instances.Expander) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	g := buildMoveStatementGraph(stmts)

	if len(g.Cycles()) != 0 {
		// TODO: proper error messages for this
		diags = diags.Append(fmt.Errorf("move statement cycles"))
	}

	// TODO: Various other validation rules

	return diags
}
