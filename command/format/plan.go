package format

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/colorstring"
)

// Plan is a representation of a plan optimized for display to
// an end-user, as opposed to terraform.Plan which is for internal use.
//
// DisplayPlan excludes implementation details that may otherwise appear
// in the main plan, such as destroy actions on data sources (which are
// there only to clean up the state).
type Plan struct {
	Resources []*InstanceDiff
}

// InstanceDiff is a representation of an instance diff optimized
// for display, in conjunction with DisplayPlan.
type InstanceDiff struct {
	Addr   *terraform.ResourceAddress
	Action terraform.DiffChangeType

	// Attributes describes changes to the attributes of the instance.
	//
	// For destroy diffs this is always nil.
	Attributes []*AttributeDiff

	Tainted bool
	Deposed bool
}

// AttributeDiff is a representation of an attribute diff optimized
// for display, in conjunction with DisplayInstanceDiff.
type AttributeDiff struct {
	// Path is a dot-delimited traversal through possibly many levels of list and map structure,
	// intended for display purposes only.
	Path string

	Action terraform.DiffChangeType

	OldValue string
	NewValue string

	NewComputed bool
	Sensitive   bool
	ForcesNew   bool
}

// PlanStats gives summary counts for a Plan.
type PlanStats struct {
	ToAdd, ToChange, ToDestroy int
}

// NewPlan produces a display-oriented Plan from a terraform.Plan.
func NewPlan(plan *terraform.Plan) *Plan {
	ret := &Plan{}
	if plan == nil || plan.Diff == nil || plan.Diff.Empty() {
		// Nothing to do!
		return ret
	}

	for _, m := range plan.Diff.Modules {
		var modulePath []string
		if !m.IsRoot() {
			// trim off the leading "root" path segment, since it's implied
			// when we use a path in a resource address.
			modulePath = m.Path[1:]
		}

		for k, r := range m.Resources {
			if r.Empty() {
				continue
			}

			addr, err := terraform.ParseResourceAddressForInstanceDiff(modulePath, k)
			if err != nil {
				// should never happen; indicates invalid diff
				panic("invalid resource address in diff")
			}

			dataSource := addr.Mode == config.DataResourceMode

			// We create "destroy" actions for data resources so we can clean
			// up their entries in state, but this is an implementation detail
			// that users shouldn't see.
			if dataSource && r.ChangeType() == terraform.DiffDestroy {
				continue
			}

			did := &InstanceDiff{
				Addr:    addr,
				Action:  r.ChangeType(),
				Tainted: r.DestroyTainted,
				Deposed: r.DestroyDeposed,
			}

			if dataSource && did.Action == terraform.DiffCreate {
				// Use "refresh" as the action for display, since core
				// currently uses Create for this.
				did.Action = terraform.DiffRefresh
			}

			ret.Resources = append(ret.Resources, did)

			if did.Action == terraform.DiffDestroy {
				// Don't show any outputs for destroy actions
				continue
			}

			for k, a := range r.Attributes {
				var action terraform.DiffChangeType
				switch {
				case a.NewRemoved:
					action = terraform.DiffDestroy
				case did.Action == terraform.DiffCreate:
					action = terraform.DiffCreate
				default:
					action = terraform.DiffUpdate
				}

				did.Attributes = append(did.Attributes, &AttributeDiff{
					Path:   k,
					Action: action,

					OldValue: a.Old,
					NewValue: a.New,

					Sensitive:   a.Sensitive,
					ForcesNew:   a.RequiresNew,
					NewComputed: a.NewComputed,
				})
			}

			// Sort the attributes by their paths for display
			sort.Slice(did.Attributes, func(i, j int) bool {
				iPath := did.Attributes[i].Path
				jPath := did.Attributes[j].Path

				// as a special case, "id" is always first
				switch {
				case iPath != jPath && (iPath == "id" || jPath == "id"):
					return iPath == "id"
				default:
					return iPath < jPath
				}
			})

		}
	}

	// Sort the instance diffs by their addresses for display.
	sort.Slice(ret.Resources, func(i, j int) bool {
		iAddr := ret.Resources[i].Addr
		jAddr := ret.Resources[j].Addr
		return iAddr.Less(jAddr)
	})

	return ret
}

// Format produces and returns a text representation of the receiving plan
// intended for display in a terminal.
//
// If color is not nil, it is used to colorize the output.
func (p *Plan) Format(color *colorstring.Colorize) string {
	if p.Empty() {
		return "This plan does nothing."
	}

	if color == nil {
		color = &colorstring.Colorize{
			Colors: colorstring.DefaultColors,
			Reset:  false,
		}
	}

	// Find the longest path length of all the paths that are changing,
	// so we can align them all.
	keyLen := 0
	for _, r := range p.Resources {
		for _, attr := range r.Attributes {
			key := attr.Path

			if len(key) > keyLen {
				keyLen = len(key)
			}
		}
	}

	buf := new(bytes.Buffer)
	for _, r := range p.Resources {
		formatPlanInstanceDiff(buf, r, keyLen, color)
	}

	return strings.TrimSpace(buf.String())
}

// Stats returns statistics about the plan
func (p *Plan) Stats() PlanStats {
	var ret PlanStats
	for _, r := range p.Resources {
		switch r.Action {
		case terraform.DiffCreate:
			ret.ToAdd++
		case terraform.DiffUpdate:
			ret.ToChange++
		case terraform.DiffDestroyCreate:
			ret.ToAdd++
			ret.ToDestroy++
		case terraform.DiffDestroy:
			ret.ToDestroy++
		}
	}
	return ret
}

// ActionCounts returns the number of diffs for each action type
func (p *Plan) ActionCounts() map[terraform.DiffChangeType]int {
	ret := map[terraform.DiffChangeType]int{}
	for _, r := range p.Resources {
		ret[r.Action]++
	}
	return ret
}

// Empty returns true if there is at least one resource diff in the receiving plan.
func (p *Plan) Empty() bool {
	return len(p.Resources) == 0
}

// DiffActionSymbol returns a string that, once passed through a
// colorstring.Colorize, will produce a result that can be written
// to a terminal to produce a symbol made of three printable
// characters, possibly interspersed with VT100 color codes.
func DiffActionSymbol(action terraform.DiffChangeType) string {
	switch action {
	case terraform.DiffDestroyCreate:
		return "[red]-[reset]/[green]+[reset]"
	case terraform.DiffCreate:
		return "  [green]+[reset]"
	case terraform.DiffDestroy:
		return "  [red]-[reset]"
	case terraform.DiffRefresh:
		return " [cyan]<=[reset]"
	default:
		return "  [yellow]~[reset]"
	}
}

// formatPlanInstanceDiff writes the text representation of the given instance diff
// to the given buffer, using the given colorizer.
func formatPlanInstanceDiff(buf *bytes.Buffer, r *InstanceDiff, keyLen int, colorizer *colorstring.Colorize) {
	addrStr := r.Addr.String()

	// Determine the color for the text (green for adding, yellow
	// for change, red for delete), and symbol, and output the
	// resource header.
	color := "yellow"
	symbol := DiffActionSymbol(r.Action)
	oldValues := true
	switch r.Action {
	case terraform.DiffDestroyCreate:
		color = "yellow"
	case terraform.DiffCreate:
		color = "green"
		oldValues = false
	case terraform.DiffDestroy:
		color = "red"
	case terraform.DiffRefresh:
		color = "cyan"
		oldValues = false
	}

	var extraStr string
	if r.Tainted {
		extraStr = extraStr + " (tainted)"
	}
	if r.Deposed {
		extraStr = extraStr + " (deposed)"
	}
	if r.Action == terraform.DiffDestroyCreate {
		extraStr = extraStr + colorizer.Color(" [red][bold](new resource required)")
	}

	buf.WriteString(
		colorizer.Color(fmt.Sprintf(
			"[%s]%s [%s]%s%s\n",
			color, symbol, color, addrStr, extraStr,
		)),
	)

	for _, attr := range r.Attributes {

		v := attr.NewValue
		var dispV string
		switch {
		case v == "" && attr.NewComputed:
			dispV = "<computed>"
		case attr.Sensitive:
			dispV = "<sensitive>"
		default:
			dispV = fmt.Sprintf("%q", v)
		}

		updateMsg := ""
		switch {
		case attr.ForcesNew && r.Action == terraform.DiffDestroyCreate:
			updateMsg = colorizer.Color(" [red](forces new resource)")
		case attr.Sensitive && oldValues:
			updateMsg = colorizer.Color(" [yellow](attribute changed)")
		}

		if oldValues {
			u := attr.OldValue
			var dispU string
			switch {
			case attr.Sensitive:
				dispU = "<sensitive>"
			default:
				dispU = fmt.Sprintf("%q", u)
			}
			buf.WriteString(fmt.Sprintf(
				"      %s:%s %s => %s%s\n",
				attr.Path,
				strings.Repeat(" ", keyLen-len(attr.Path)),
				dispU, dispV,
				updateMsg,
			))
		} else {
			buf.WriteString(fmt.Sprintf(
				"      %s:%s %s%s\n",
				attr.Path,
				strings.Repeat(" ", keyLen-len(attr.Path)),
				dispV,
				updateMsg,
			))
		}
	}

	// Write the reset color so we don't bleed color into later text
	buf.WriteString(colorizer.Color("[reset]\n"))
}
