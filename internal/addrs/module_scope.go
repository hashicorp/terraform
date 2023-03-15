package addrs

// ModuleEvalScope represents the different kinds of module scope that
// Terraform Core can evaluate expressions within.
//
// Its only two implementations are [ModuleInstance] and
// [PartialExpandedModule], where the latter represents the evaluation context
// for the "partial evaluation" mode which produces placeholder values for
// not-yet-expanded modules.
//
// A nil ModuleEvalScope represents no evaluation scope at all, whereas a
// typed ModuleEvalScope represents either an exact expanded module or a
// partial-expanded module.
type ModuleEvalScope interface {
	moduleEvalScopeSigil()
}

func (ModuleInstance) moduleEvalScopeSigil() {}

func (PartialExpandedModule) moduleEvalScopeSigil() {}
