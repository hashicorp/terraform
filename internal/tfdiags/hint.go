package tfdiags

// HintMessage is a special diagnostic type used exclusively for the
// "terraform validate -hint" feature, which we use to report suggestions that
// a module author might find useful when debugging or when trying to make
// a module clearer for future debugging, but that don't necessarily need to be
// resolved in all situations.
//
// It's invalid to return Hint diagnostics in any codepath except Terraform Core
// validation when the hint mode is enabled. Other codepaths may not
// necessarily be able to return hint diagnostics correctly, including
// potentially returning them with an incorrect severity or even crashing due
// to the severity not being recognized. In such cases, the bug is that we
// returned a Hint in an inappropriate context, not that the context in
// question didn't handle the hint diagnostics.
type HintMessage struct {
	Summary string
	Detail  string

	SourceRange  SourceRange
	ContextRange *SourceRange
}

var _ Diagnostic = (*HintMessage)(nil)

func (h *HintMessage) Severity() Severity {
	return Hint
}

func (h *HintMessage) Description() Description {
	return Description{
		Summary: h.Summary,
		Detail:  h.Detail,
	}
}

func (h *HintMessage) Source() Source {
	ret := Source{}
	// All hints must at least have a source range, because suggesting
	// problems with source code is what hints are for.
	ret.Subject = &h.SourceRange

	// Some hints might also have a separate context range, if it seems
	// useful to include more context in a diagnostic message in addition
	// to just the directly-highlighted part. Otherwise, we'll just focus
	// on the highlighted part.
	if h.ContextRange != nil {
		ret.Context = h.ContextRange
	}

	return ret
}

func (h *HintMessage) FromExpr() *FromExpr {
	// Hint mode does its work with static analysis and partial
	// information, so hints can't generate true expression evaluation
	// information.
	return nil
}
