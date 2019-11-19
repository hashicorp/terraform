package tfdiags

import "fmt"

// ConsolidateWarnings checks if there is an unreasonable amount of warnings
// with the same summary in the receiver and, if so, returns a new diagnostics
// with some of those warnings consolidated into a single warning in order
// to reduce the verbosity of the output.
//
// This mechanism is here primarily for diagnostics printed out at the CLI. In
// other contexts it is likely better to just return the warnings directly,
// particularly if they are going to be interpreted by software rather than
// by a human reader.
//
// The returned slice always has a separate backing array from the reciever,
// but some diagnostic values themselves might be shared.
//
// The definition of "unreasonable" may change in future releases.
func (diags Diagnostics) ConsolidateWarnings() Diagnostics {
	// We'll start grouping when there are more than this number of warnings
	// with the same summary.
	const unreasonableThreshold = 2

	if len(diags) == 0 {
		return nil
	}

	newDiags := make(Diagnostics, 0, len(diags))

	// We'll track how many times we've seen each warning summary so we can
	// decide when to start consolidating. Once we _have_ started consolidating,
	// we'll also track the object representing the consolidated warning
	// so we can continue appending to it.
	warningStats := make(map[string]int)
	warningGroups := make(map[string]*warningGroup)

	for _, diag := range diags {
		severity := diag.Severity()
		if severity != Warning || diag.Source().Subject == nil {
			// Only warnings can get special treatment, and we only
			// consolidate warnings that have source locations because
			// our primary goal here is to deal with the situation where
			// some configuration language feature is producing a warning
			// each time it's used across a potentially-large config.
			newDiags = newDiags.Append(diag)
			continue
		}

		desc := diag.Description()
		summary := desc.Summary
		if g, ok := warningGroups[summary]; ok {
			// We're already grouping this one, so we'll just continue it.
			g.Append(diag)
			continue
		}

		warningStats[summary]++
		if warningStats[summary] == unreasonableThreshold {
			// Initially creating the group doesn't really change anything
			// visibly in the result, since a group with only one warning
			// is just a passthrough anyway, but once we do this any additional
			// warnings with the same summary will get appended to this group.
			g := &warningGroup{}
			newDiags = newDiags.Append(g)
			warningGroups[summary] = g
			g.Append(diag)
			continue
		}

		// If this warning is not consolidating yet then we'll just append
		// it directly.
		newDiags = newDiags.Append(diag)
	}

	return newDiags
}

// A warningGroup is one or more warning diagnostics grouped together for
// UI consolidation purposes.
//
// A warningGroup with only one diagnostic in it is just a passthrough for
// that one diagnostic. If it has more than one then it will behave mostly
// like the first one but its detail message will include an additional
// sentence mentioning the consolidation. A warningGroup with no diagnostics
// at all is invalid and will panic when used.
type warningGroup struct {
	Warnings Diagnostics
}

var _ Diagnostic = (*warningGroup)(nil)

func (wg *warningGroup) Severity() Severity {
	return wg.Warnings[0].Severity()
}

func (wg *warningGroup) Description() Description {
	desc := wg.Warnings[0].Description()
	if len(wg.Warnings) < 2 {
		return desc
	}
	extraCount := len(wg.Warnings) - 1
	var msg string
	switch extraCount {
	case 1:
		msg = "(and one more similar warning elsewhere)"
	default:
		msg = fmt.Sprintf("(and %d more similar warnings elsewhere)", extraCount)
	}
	if desc.Detail != "" {
		desc.Detail = desc.Detail + "\n\n" + msg
	} else {
		desc.Detail = msg
	}
	return desc
}

func (wg *warningGroup) Source() Source {
	return wg.Warnings[0].Source()
}

func (wg *warningGroup) FromExpr() *FromExpr {
	return wg.Warnings[0].FromExpr()
}

func (wg *warningGroup) Append(diag Diagnostic) {
	if diag.Severity() != Warning {
		panic("can't append a non-warning diagnostic to a warningGroup")
	}
	wg.Warnings = append(wg.Warnings, diag)
}
