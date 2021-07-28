package refactoring

import (
	"fmt"

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
			// Invalid combination should've been caught during original
			// configuration decoding, in the configs package.
			panic(fmt.Sprintf("incompatible move endpoints in %s", mc.DeclRange))
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

func (s *MoveStatement) ObjectKind() addrs.MoveEndpointKind {
	// addrs.UnifyMoveEndpoints guarantees that both of our addresses have
	// the same kind, so we can just arbitrary use From and assume To will
	// match it.
	return s.From.ObjectKind()
}
