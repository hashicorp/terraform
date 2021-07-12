package refactoring

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type MoveStatement struct {
	From, To  *addrs.MoveEndpointInModule
	DeclRange tfdiags.SourceRange
}

// FindMoveStatements recurses through the modules of the given configuration
// and returns a flat set of all "moved" blocks defined within, in a
// deterministic but undefined order.
func FindMoveStatements(rootCfg *configs.Config) []MoveStatement {
	return findMoveStatements(rootCfg, nil)
}

func findMoveStatements(cfg *configs.Config, into []MoveStatement) []MoveStatement {
	modAddr := cfg.Path
	for _, mc := range cfg.Module.Moved {
		fromAddr, toAddr := addrs.UnifyMoveEndpoints(modAddr, mc.From, mc.To)
		if fromAddr == nil || toAddr == nil {
			// Invalid combination should get caught by our separate
			// validation rules elsewhere.
			continue
		}

		into = append(into, MoveStatement{
			From:      fromAddr,
			To:        toAddr,
			DeclRange: tfdiags.SourceRangeFromHCL(mc.DeclRange),
		})
	}

	for _, childCfg := range cfg.Children {
		into = findMoveStatements(childCfg, into)
	}

	return into
}
