package jsonformat

import (
	"sort"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"
	"github.com/hashicorp/terraform/internal/command/jsonformat/differ"
	"github.com/hashicorp/terraform/internal/command/jsonplan"
	"github.com/hashicorp/terraform/internal/plans"
)

func precomputeDiffs(plan Plan) diffs {
	diffs := diffs{
		outputs: make(map[string]computed.Diff),
	}

	for _, drift := range plan.ResourceDrift {
		schema := plan.ProviderSchemas[drift.ProviderName].ResourceSchemas[drift.Type]
		diffs.drift = append(diffs.drift, diff{
			change: drift,
			diff:   differ.FromJsonChange(drift.Change).ComputeDiffForBlock(schema.Block),
		})
	}

	for _, change := range plan.ResourceChanges {
		schema := plan.ProviderSchemas[change.ProviderName].ResourceSchemas[change.Type]
		diffs.changes = append(diffs.changes, diff{
			change: change,
			diff:   differ.FromJsonChange(change.Change).ComputeDiffForBlock(schema.Block),
		})
	}

	for key, output := range plan.OutputChanges {
		diffs.outputs[key] = differ.FromJsonChange(output).ComputeDiffForOutput()
	}

	less := func(drs []diff) func(i, j int) bool {
		return func(i, j int) bool {
			iA := drs[i].change.Address
			jA := drs[j].change.Address
			if iA == jA {
				return drs[i].change.Deposed < drs[j].change.Deposed
			}
			return iA < jA
		}
	}

	sort.Slice(diffs.drift, less(diffs.drift))
	sort.Slice(diffs.changes, less(diffs.changes))

	return diffs
}

type diffs struct {
	drift   []diff
	changes []diff
	outputs map[string]computed.Diff
}

func (d diffs) Empty() bool {
	for _, change := range d.changes {
		if change.diff.Action != plans.NoOp || change.Moved() {
			return false
		}
	}

	for _, output := range d.outputs {
		if output.Action != plans.NoOp {
			return false
		}
	}

	return true
}

type diff struct {
	change jsonplan.ResourceChange
	diff   computed.Diff
}

func (d diff) Moved() bool {
	return len(d.change.PreviousAddress) > 0 && d.change.PreviousAddress != d.change.Address
}
