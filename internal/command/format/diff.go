package format

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/mitchellh/colorstring"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/objchange"
	"github.com/hashicorp/terraform/internal/states"
)

// DiffLanguage controls the description of the resource change reasons.
type DiffLanguage rune

//go:generate go run golang.org/x/tools/cmd/stringer -type=DiffLanguage diff.go

const (
	// DiffLanguageProposedChange indicates that the change is one which is
	// planned to be applied.
	DiffLanguageProposedChange DiffLanguage = 'P'

	// DiffLanguageDetectedDrift indicates that the change is detected drift
	// from the configuration.
	DiffLanguageDetectedDrift DiffLanguage = 'D'
)

// ResourceChange returns a string representation of a change to a particular
// resource, for inclusion in user-facing plan output.
//
// The resource schema must be provided along with the change so that the
// formatted change can reflect the configuration structure for the associated
// resource.
//
// If "color" is non-nil, it will be used to color the result. Otherwise,
// no color codes will be included.
func ResourceChange(
	change *plans.ResourceInstanceChange,
	schema *configschema.Block,
	color *colorstring.Colorize,
	language DiffLanguage,
) string {
	addr := change.Addr
	var buf bytes.Buffer

	if color == nil {
		color = &colorstring.Colorize{
			Colors:  colorstring.DefaultColors,
			Disable: true,
			Reset:   false,
		}
	}

	dispAddr := addr.String()
	if change.DeposedKey != states.NotDeposed {
		dispAddr = fmt.Sprintf("%s (deposed object %s)", dispAddr, change.DeposedKey)
	}

	switch change.Action {
	case plans.Create:
		buf.WriteString(fmt.Sprintf(color.Color("[bold]  # %s[reset] will be created"), dispAddr))
	case plans.Read:
		buf.WriteString(fmt.Sprintf(color.Color("[bold]  # %s[reset] will be read during apply"), dispAddr))
		switch change.ActionReason {
		case plans.ResourceInstanceReadBecauseConfigUnknown:
			buf.WriteString("\n  # (config refers to values not yet known)")
		case plans.ResourceInstanceReadBecauseDependencyPending:
			buf.WriteString("\n  # (depends on a resource or a module with changes pending)")
		case plans.ResourceInstanceReadBecauseSmokeTest:
			buf.WriteString("\n  # (provides data for the smoke testing phase)")
		}
	case plans.Update:
		switch language {
		case DiffLanguageProposedChange:
			buf.WriteString(fmt.Sprintf(color.Color("[bold]  # %s[reset] will be updated in-place"), dispAddr))
		case DiffLanguageDetectedDrift:
			buf.WriteString(fmt.Sprintf(color.Color("[bold]  # %s[reset] has changed"), dispAddr))
		default:
			buf.WriteString(fmt.Sprintf(color.Color("[bold]  # %s[reset] update (unknown reason %s)"), dispAddr, language))
		}
	case plans.CreateThenDelete, plans.DeleteThenCreate:
		switch change.ActionReason {
		case plans.ResourceInstanceReplaceBecauseTainted:
			buf.WriteString(fmt.Sprintf(color.Color("[bold]  # %s[reset] is tainted, so must be [bold][red]replaced"), dispAddr))
		case plans.ResourceInstanceReplaceByRequest:
			buf.WriteString(fmt.Sprintf(color.Color("[bold]  # %s[reset] will be [bold][red]replaced[reset], as requested"), dispAddr))
		case plans.ResourceInstanceReplaceByTriggers:
			buf.WriteString(fmt.Sprintf(color.Color("[bold]  # %s[reset] will be [bold][red]replaced[reset] due to changes in replace_triggered_by"), dispAddr))
		default:
			buf.WriteString(fmt.Sprintf(color.Color("[bold]  # %s[reset] must be [bold][red]replaced"), dispAddr))
		}
	case plans.Delete:
		switch language {
		case DiffLanguageProposedChange:
			buf.WriteString(fmt.Sprintf(color.Color("[bold]  # %s[reset] will be [bold][red]destroyed"), dispAddr))
		case DiffLanguageDetectedDrift:
			buf.WriteString(fmt.Sprintf(color.Color("[bold]  # %s[reset] has been deleted"), dispAddr))
		default:
			buf.WriteString(fmt.Sprintf(color.Color("[bold]  # %s[reset] delete (unknown reason %s)"), dispAddr, language))
		}
		// We can sometimes give some additional detail about why we're
		// proposing to delete. We show this as additional notes, rather than
		// as additional wording in the main action statement, in an attempt
		// to make the "will be destroyed" message prominent and consistent
		// in all cases, for easier scanning of this often-risky action.
		switch change.ActionReason {
		case plans.ResourceInstanceDeleteBecauseNoResourceConfig:
			buf.WriteString(fmt.Sprintf("\n  # (because %s is not in configuration)", addr.Resource.Resource))
		case plans.ResourceInstanceDeleteBecauseNoMoveTarget:
			buf.WriteString(fmt.Sprintf("\n  # (because %s was moved to %s, which is not in configuration)", change.PrevRunAddr, addr.Resource.Resource))
		case plans.ResourceInstanceDeleteBecauseNoModule:
			// FIXME: Ideally we'd truncate addr.Module to reflect the earliest
			// step that doesn't exist, so it's clearer which call this refers
			// to, but we don't have enough information out here in the UI layer
			// to decide that; only the "expander" in Terraform Core knows
			// which module instance keys are actually declared.
			buf.WriteString(fmt.Sprintf("\n  # (because %s is not in configuration)", addr.Module))
		case plans.ResourceInstanceDeleteBecauseWrongRepetition:
			// We have some different variations of this one
			switch addr.Resource.Key.(type) {
			case nil:
				buf.WriteString("\n  # (because resource uses count or for_each)")
			case addrs.IntKey:
				buf.WriteString("\n  # (because resource does not use count)")
			case addrs.StringKey:
				buf.WriteString("\n  # (because resource does not use for_each)")
			}
		case plans.ResourceInstanceDeleteBecauseCountIndex:
			buf.WriteString(fmt.Sprintf("\n  # (because index %s is out of range for count)", addr.Resource.Key))
		case plans.ResourceInstanceDeleteBecauseEachKey:
			buf.WriteString(fmt.Sprintf("\n  # (because key %s is not in for_each map)", addr.Resource.Key))
		}
		if change.DeposedKey != states.NotDeposed {
			// Some extra context about this unusual situation.
			buf.WriteString(color.Color("\n  # (left over from a partially-failed replacement of this instance)"))
		}
	case plans.NoOp:
		if change.Moved() {
			buf.WriteString(fmt.Sprintf(color.Color("[bold]  # %s[reset] has moved to [bold]%s[reset]"), change.PrevRunAddr.String(), dispAddr))
			break
		}
		fallthrough
	default:
		// should never happen, since the above is exhaustive
		buf.WriteString(fmt.Sprintf("%s has an action the plan renderer doesn't support (this is a bug)", dispAddr))
	}
	buf.WriteString(color.Color("[reset]\n"))

	if change.Moved() && change.Action != plans.NoOp {
		buf.WriteString(fmt.Sprintf(color.Color("  # [reset](moved from %s)\n"), change.PrevRunAddr.String()))
	}

	if change.Moved() && change.Action == plans.NoOp {
		buf.WriteString("    ")
	} else {
		buf.WriteString(color.Color(DiffActionSymbol(change.Action)) + " ")
	}

	switch addr.Resource.Resource.Mode {
	case addrs.ManagedResourceMode:
		buf.WriteString(fmt.Sprintf(
			"resource %q %q",
			addr.Resource.Resource.Type,
			addr.Resource.Resource.Name,
		))
	case addrs.DataResourceMode:
		buf.WriteString(fmt.Sprintf(
			"data %q %q",
			addr.Resource.Resource.Type,
			addr.Resource.Resource.Name,
		))
	default:
		// should never happen, since the above is exhaustive
		buf.WriteString(addr.String())
	}

	buf.WriteString(" {")

	p := blockBodyDiffPrinter{
		buf:             &buf,
		color:           color,
		action:          change.Action,
		requiredReplace: change.RequiredReplace,
	}

	// Most commonly-used resources have nested blocks that result in us
	// going at least three traversals deep while we recurse here, so we'll
	// start with that much capacity and then grow as needed for deeper
	// structures.
	path := make(cty.Path, 0, 3)

	result := p.writeBlockBodyDiff(schema, change.Before, change.After, 6, path)
	if result.bodyWritten {
		buf.WriteString("\n")
		buf.WriteString(strings.Repeat(" ", 4))
	}
	buf.WriteString("}\n")

	return buf.String()
}

// OutputChanges returns a string representation of a set of changes to output
// values for inclusion in user-facing plan output.
//
// If "color" is non-nil, it will be used to color the result. Otherwise,
// no color codes will be included.
func OutputChanges(
	changes []*plans.OutputChangeSrc,
	color *colorstring.Colorize,
) string {
	var buf bytes.Buffer
	p := blockBodyDiffPrinter{
		buf:    &buf,
		color:  color,
		action: plans.Update, // not actually used in this case, because we're not printing a containing block
	}

	// We're going to reuse the codepath we used for printing resource block
	// diffs, by pretending that the set of defined outputs are the attributes
	// of some resource. It's a little forced to do this, but it gives us all
	// the same formatting heuristics as we normally use for resource
	// attributes.
	oldVals := make(map[string]cty.Value, len(changes))
	newVals := make(map[string]cty.Value, len(changes))
	synthSchema := &configschema.Block{
		Attributes: make(map[string]*configschema.Attribute, len(changes)),
	}
	for _, changeSrc := range changes {
		name := changeSrc.Addr.OutputValue.Name
		change, err := changeSrc.Decode()
		if err != nil {
			// It'd be weird to get a decoding error here because that would
			// suggest that Terraform itself just produced an invalid plan, and
			// we don't have any good way to ignore it in this codepath, so
			// we'll just log it and ignore it.
			log.Printf("[ERROR] format.OutputChanges: Failed to decode planned change for output %q: %s", name, err)
			continue
		}
		synthSchema.Attributes[name] = &configschema.Attribute{
			Type:      cty.DynamicPseudoType, // output types are decided dynamically based on the given value
			Optional:  true,
			Sensitive: change.Sensitive,
		}
		oldVals[name] = change.Before
		newVals[name] = change.After
	}

	p.writeBlockBodyDiff(synthSchema, cty.ObjectVal(oldVals), cty.ObjectVal(newVals), 2, nil)

	return buf.String()
}

type blockBodyDiffPrinter struct {
	buf             *bytes.Buffer
	color           *colorstring.Colorize
	action          plans.Action
	requiredReplace cty.PathSet
	// verbose is set to true when using the "diff" printer to format state
	verbose bool
}

type blockBodyDiffResult struct {
	bodyWritten       bool
	skippedAttributes int
	skippedBlocks     int
}

const (
	forcesNewResourceCaption = " [red]# forces replacement[reset]"
	sensitiveCaption         = "(sensitive value)"
)

// writeBlockBodyDiff writes attribute or block differences
// and returns true if any differences were found and written
func (p *blockBodyDiffPrinter) writeBlockBodyDiff(schema *configschema.Block, old, new cty.Value, indent int, path cty.Path) blockBodyDiffResult {
	path = ctyEnsurePathCapacity(path, 1)
	result := blockBodyDiffResult{}

	// write the attributes diff
	blankBeforeBlocks := p.writeAttrsDiff(schema.Attributes, old, new, indent, path, &result)
	p.writeSkippedAttr(result.skippedAttributes, indent+2)

	{
		blockTypeNames := make([]string, 0, len(schema.BlockTypes))
		for name := range schema.BlockTypes {
			blockTypeNames = append(blockTypeNames, name)
		}
		sort.Strings(blockTypeNames)

		for _, name := range blockTypeNames {
			blockS := schema.BlockTypes[name]
			oldVal := ctyGetAttrMaybeNull(old, name)
			newVal := ctyGetAttrMaybeNull(new, name)

			result.bodyWritten = true
			skippedBlocks := p.writeNestedBlockDiffs(name, blockS, oldVal, newVal, blankBeforeBlocks, indent, path)
			if skippedBlocks > 0 {
				result.skippedBlocks += skippedBlocks
			}

			// Always include a blank for any subsequent block types.
			blankBeforeBlocks = true
		}
		if result.skippedBlocks > 0 {
			noun := "blocks"
			if result.skippedBlocks == 1 {
				noun = "block"
			}
			p.buf.WriteString("\n\n")
			p.buf.WriteString(strings.Repeat(" ", indent+2))
			p.buf.WriteString(fmt.Sprintf(p.color.Color("[dark_gray]# (%d unchanged %s hidden)[reset]"), result.skippedBlocks, noun))
		}
	}

	return result
}

func (p *blockBodyDiffPrinter) writeAttrsDiff(
	attrsS map[string]*configschema.Attribute,
	old, new cty.Value,
	indent int,
	path cty.Path,
	result *blockBodyDiffResult) bool {

	attrNames := make([]string, 0, len(attrsS))
	displayAttrNames := make(map[string]string, len(attrsS))
	attrNameLen := 0
	for name := range attrsS {
		oldVal := ctyGetAttrMaybeNull(old, name)
		newVal := ctyGetAttrMaybeNull(new, name)
		if oldVal.IsNull() && newVal.IsNull() {
			// Skip attributes where both old and new values are null
			// (we do this early here so that we'll do our value alignment
			// based on the longest attribute name that has a change, rather
			// than the longest attribute name in the full set.)
			continue
		}

		attrNames = append(attrNames, name)
		displayAttrNames[name] = displayAttributeName(name)
		if len(displayAttrNames[name]) > attrNameLen {
			attrNameLen = len(displayAttrNames[name])
		}
	}
	sort.Strings(attrNames)
	if len(attrNames) == 0 {
		return false
	}

	for _, name := range attrNames {
		attrS := attrsS[name]
		oldVal := ctyGetAttrMaybeNull(old, name)
		newVal := ctyGetAttrMaybeNull(new, name)

		result.bodyWritten = true
		skipped := p.writeAttrDiff(displayAttrNames[name], attrS, oldVal, newVal, attrNameLen, indent, path)
		if skipped {
			result.skippedAttributes++
		}
	}

	return true
}

// getPlanActionAndShow returns the action value
// and a boolean for showJustNew. In this function we
// modify the old and new values to remove any possible marks
func getPlanActionAndShow(old cty.Value, new cty.Value) (plans.Action, bool) {
	var action plans.Action
	showJustNew := false
	switch {
	case old.IsNull():
		action = plans.Create
		showJustNew = true
	case new.IsNull():
		action = plans.Delete
	case ctyEqualWithUnknown(old, new):
		action = plans.NoOp
		showJustNew = true
	default:
		action = plans.Update
	}
	return action, showJustNew
}

func (p *blockBodyDiffPrinter) writeAttrDiff(name string, attrS *configschema.Attribute, old, new cty.Value, nameLen, indent int, path cty.Path) bool {
	path = append(path, cty.GetAttrStep{Name: name})
	action, showJustNew := getPlanActionAndShow(old, new)

	if action == plans.NoOp && !p.verbose && !identifyingAttribute(name, attrS) {
		return true
	}

	if attrS.NestedType != nil {
		p.writeNestedAttrDiff(name, attrS, old, new, nameLen, indent, path, action, showJustNew)
		return false
	}

	p.buf.WriteString("\n")

	p.writeSensitivityWarning(old, new, indent, action, false)

	p.buf.WriteString(strings.Repeat(" ", indent))
	p.writeActionSymbol(action)

	p.buf.WriteString(p.color.Color("[bold]"))
	p.buf.WriteString(name)
	p.buf.WriteString(p.color.Color("[reset]"))
	p.buf.WriteString(strings.Repeat(" ", nameLen-len(name)))
	p.buf.WriteString(" = ")

	if attrS.Sensitive {
		p.buf.WriteString(sensitiveCaption)
		if p.pathForcesNewResource(path) {
			p.buf.WriteString(p.color.Color(forcesNewResourceCaption))
		}
	} else {
		switch {
		case showJustNew:
			p.writeValue(new, action, indent+2)
			if p.pathForcesNewResource(path) {
				p.buf.WriteString(p.color.Color(forcesNewResourceCaption))
			}
		default:
			// We show new even if it is null to emphasize the fact
			// that it is being unset, since otherwise it is easy to
			// misunderstand that the value is still set to the old value.
			p.writeValueDiff(old, new, indent+2, path)
		}
	}

	return false
}

// writeNestedAttrDiff is responsible for formatting Attributes with NestedTypes
// in the diff.
func (p *blockBodyDiffPrinter) writeNestedAttrDiff(
	name string, attrWithNestedS *configschema.Attribute, old, new cty.Value,
	nameLen, indent int, path cty.Path, action plans.Action, showJustNew bool) {

	objS := attrWithNestedS.NestedType

	p.buf.WriteString("\n")
	p.writeSensitivityWarning(old, new, indent, action, false)
	p.buf.WriteString(strings.Repeat(" ", indent))
	p.writeActionSymbol(action)

	p.buf.WriteString(p.color.Color("[bold]"))
	p.buf.WriteString(name)
	p.buf.WriteString(p.color.Color("[reset]"))
	p.buf.WriteString(strings.Repeat(" ", nameLen-len(name)))

	// Then schema of the attribute itself can be marked sensitive, or the values assigned
	sensitive := attrWithNestedS.Sensitive || old.HasMark(marks.Sensitive) || new.HasMark(marks.Sensitive)
	if sensitive {
		p.buf.WriteString(" = ")
		p.buf.WriteString(sensitiveCaption)

		if p.pathForcesNewResource(path) {
			p.buf.WriteString(p.color.Color(forcesNewResourceCaption))
		}
		return
	}

	result := &blockBodyDiffResult{}
	switch objS.Nesting {
	case configschema.NestingSingle:
		p.buf.WriteString(" = {")
		if action != plans.NoOp && (p.pathForcesNewResource(path) || p.pathForcesNewResource(path[:len(path)-1])) {
			p.buf.WriteString(p.color.Color(forcesNewResourceCaption))
		}
		p.writeAttrsDiff(objS.Attributes, old, new, indent+4, path, result)
		p.writeSkippedAttr(result.skippedAttributes, indent+6)
		p.buf.WriteString("\n")
		p.buf.WriteString(strings.Repeat(" ", indent+2))
		p.buf.WriteString("}")

		if !new.IsKnown() {
			p.buf.WriteString(" -> (known after apply)")
		} else if new.IsNull() {
			p.buf.WriteString(p.color.Color("[dark_gray] -> null[reset]"))
		}

	case configschema.NestingList:
		p.buf.WriteString(" = [")
		if action != plans.NoOp && (p.pathForcesNewResource(path) || p.pathForcesNewResource(path[:len(path)-1])) {
			p.buf.WriteString(p.color.Color(forcesNewResourceCaption))
		}
		p.buf.WriteString("\n")

		oldItems := ctyCollectionValues(old)
		newItems := ctyCollectionValues(new)
		// Here we intentionally preserve the index-based correspondance
		// between old and new, rather than trying to detect insertions
		// and removals in the list, because this more accurately reflects
		// how Terraform Core and providers will understand the change,
		// particularly when the nested block contains computed attributes
		// that will themselves maintain correspondance by index.

		// commonLen is number of elements that exist in both lists, which
		// will be presented as updates (~). Any additional items in one
		// of the lists will be presented as either creates (+) or deletes (-)
		// depending on which list they belong to. maxLen is the number of
		// elements in that longer list.
		var commonLen int
		var maxLen int
		// unchanged is the number of unchanged elements
		var unchanged int

		switch {
		case len(oldItems) < len(newItems):
			commonLen = len(oldItems)
			maxLen = len(newItems)
		default:
			commonLen = len(newItems)
			maxLen = len(oldItems)
		}
		for i := 0; i < maxLen; i++ {
			path := append(path, cty.IndexStep{Key: cty.NumberIntVal(int64(i))})

			var action plans.Action
			var oldItem, newItem cty.Value
			switch {
			case i < commonLen:
				oldItem = oldItems[i]
				newItem = newItems[i]
				if oldItem.RawEquals(newItem) {
					action = plans.NoOp
					unchanged++
				} else {
					action = plans.Update
				}
			case i < len(oldItems):
				oldItem = oldItems[i]
				newItem = cty.NullVal(oldItem.Type())
				action = plans.Delete
			case i < len(newItems):
				newItem = newItems[i]
				oldItem = cty.NullVal(newItem.Type())
				action = plans.Create
			default:
				action = plans.NoOp
			}

			if action != plans.NoOp {
				p.buf.WriteString(strings.Repeat(" ", indent+4))
				p.writeActionSymbol(action)
				p.buf.WriteString("{")

				result := &blockBodyDiffResult{}
				p.writeAttrsDiff(objS.Attributes, oldItem, newItem, indent+8, path, result)
				if action == plans.Update {
					p.writeSkippedAttr(result.skippedAttributes, indent+10)
				}
				p.buf.WriteString("\n")

				p.buf.WriteString(strings.Repeat(" ", indent+6))
				p.buf.WriteString("},\n")
			}
		}
		p.writeSkippedElems(unchanged, indent+6)
		p.buf.WriteString(strings.Repeat(" ", indent+2))
		p.buf.WriteString("]")

		if !new.IsKnown() {
			p.buf.WriteString(" -> (known after apply)")
		} else if new.IsNull() {
			p.buf.WriteString(p.color.Color("[dark_gray] -> null[reset]"))
		}

	case configschema.NestingSet:
		oldItems := ctyCollectionValues(old)
		newItems := ctyCollectionValues(new)

		var all cty.Value
		if len(oldItems)+len(newItems) > 0 {
			allItems := make([]cty.Value, 0, len(oldItems)+len(newItems))
			allItems = append(allItems, oldItems...)
			allItems = append(allItems, newItems...)

			all = cty.SetVal(allItems)
		} else {
			all = cty.SetValEmpty(old.Type().ElementType())
		}

		p.buf.WriteString(" = [")

		var unchanged int

		for it := all.ElementIterator(); it.Next(); {
			_, val := it.Element()
			var action plans.Action
			var oldValue, newValue cty.Value
			switch {
			case !val.IsKnown():
				action = plans.Update
				newValue = val
			case !new.IsKnown():
				action = plans.Delete
				// the value must have come from the old set
				oldValue = val
				// Mark the new val as null, but the entire set will be
				// displayed as "(unknown after apply)"
				newValue = cty.NullVal(val.Type())
			case old.IsNull() || !old.HasElement(val).True():
				action = plans.Create
				oldValue = cty.NullVal(val.Type())
				newValue = val
			case new.IsNull() || !new.HasElement(val).True():
				action = plans.Delete
				oldValue = val
				newValue = cty.NullVal(val.Type())
			default:
				action = plans.NoOp
				oldValue = val
				newValue = val
			}

			if action == plans.NoOp {
				unchanged++
				continue
			}

			p.buf.WriteString("\n")
			p.buf.WriteString(strings.Repeat(" ", indent+4))
			p.writeActionSymbol(action)
			p.buf.WriteString("{")

			if p.pathForcesNewResource(path) || p.pathForcesNewResource(path[:len(path)-1]) {
				p.buf.WriteString(p.color.Color(forcesNewResourceCaption))
			}

			path := append(path, cty.IndexStep{Key: val})
			p.writeAttrsDiff(objS.Attributes, oldValue, newValue, indent+8, path, result)

			p.buf.WriteString("\n")
			p.buf.WriteString(strings.Repeat(" ", indent+6))
			p.buf.WriteString("},")
		}
		p.buf.WriteString("\n")
		p.writeSkippedElems(unchanged, indent+6)
		p.buf.WriteString(strings.Repeat(" ", indent+2))
		p.buf.WriteString("]")

		if !new.IsKnown() {
			p.buf.WriteString(" -> (known after apply)")
		} else if new.IsNull() {
			p.buf.WriteString(p.color.Color("[dark_gray] -> null[reset]"))
		}

	case configschema.NestingMap:
		// For the sake of handling nested blocks, we'll treat a null map
		// the same as an empty map since the config language doesn't
		// distinguish these anyway.
		old = ctyNullBlockMapAsEmpty(old)
		new = ctyNullBlockMapAsEmpty(new)

		oldItems := old.AsValueMap()

		newItems := map[string]cty.Value{}

		if new.IsKnown() {
			newItems = new.AsValueMap()
		}

		allKeys := make(map[string]bool)
		for k := range oldItems {
			allKeys[k] = true
		}
		for k := range newItems {
			allKeys[k] = true
		}
		allKeysOrder := make([]string, 0, len(allKeys))
		for k := range allKeys {
			allKeysOrder = append(allKeysOrder, k)
		}
		sort.Strings(allKeysOrder)

		p.buf.WriteString(" = {\n")

		// unchanged tracks the number of unchanged elements
		unchanged := 0
		for _, k := range allKeysOrder {
			var action plans.Action
			oldValue := oldItems[k]

			newValue := newItems[k]
			switch {
			case oldValue == cty.NilVal:
				oldValue = cty.NullVal(newValue.Type())
				action = plans.Create
			case newValue == cty.NilVal:
				newValue = cty.NullVal(oldValue.Type())
				action = plans.Delete
			case !newValue.RawEquals(oldValue):
				action = plans.Update
			default:
				action = plans.NoOp
				unchanged++
			}

			if action != plans.NoOp {
				p.buf.WriteString(strings.Repeat(" ", indent+4))
				p.writeActionSymbol(action)
				fmt.Fprintf(p.buf, "%q = {", k)
				if p.pathForcesNewResource(path) || p.pathForcesNewResource(path[:len(path)-1]) {
					p.buf.WriteString(p.color.Color(forcesNewResourceCaption))
				}

				path := append(path, cty.IndexStep{Key: cty.StringVal(k)})
				p.writeAttrsDiff(objS.Attributes, oldValue, newValue, indent+8, path, result)
				p.writeSkippedAttr(result.skippedAttributes, indent+10)
				p.buf.WriteString("\n")
				p.buf.WriteString(strings.Repeat(" ", indent+6))
				p.buf.WriteString("},\n")
			}
		}

		p.writeSkippedElems(unchanged, indent+6)
		p.buf.WriteString(strings.Repeat(" ", indent+2))
		p.buf.WriteString("}")
		if !new.IsKnown() {
			p.buf.WriteString(" -> (known after apply)")
		} else if new.IsNull() {
			p.buf.WriteString(p.color.Color("[dark_gray] -> null[reset]"))
		}
	}
}

func (p *blockBodyDiffPrinter) writeNestedBlockDiffs(name string, blockS *configschema.NestedBlock, old, new cty.Value, blankBefore bool, indent int, path cty.Path) int {
	skippedBlocks := 0
	path = append(path, cty.GetAttrStep{Name: name})
	if old.IsNull() && new.IsNull() {
		// Nothing to do if both old and new is null
		return skippedBlocks
	}

	// If either the old or the new value is marked,
	// Display a special diff because it is irrelevant
	// to list all obfuscated attributes as (sensitive value)
	if old.HasMark(marks.Sensitive) || new.HasMark(marks.Sensitive) {
		p.writeSensitiveNestedBlockDiff(name, old, new, indent, blankBefore, path)
		return 0
	}

	// Where old/new are collections representing a nesting mode other than
	// NestingSingle, we assume the collection value can never be unknown
	// since we always produce the container for the nested objects, even if
	// the objects within are computed.

	switch blockS.Nesting {
	case configschema.NestingSingle, configschema.NestingGroup:
		var action plans.Action
		eqV := new.Equals(old)
		switch {
		case old.IsNull():
			action = plans.Create
		case new.IsNull():
			action = plans.Delete
		case !new.IsWhollyKnown() || !old.IsWhollyKnown():
			// "old" should actually always be known due to our contract
			// that old values must never be unknown, but we'll allow it
			// anyway to be robust.
			action = plans.Update
		case !eqV.IsKnown() || !eqV.True():
			action = plans.Update
		}

		skipped := p.writeNestedBlockDiff(name, nil, &blockS.Block, action, old, new, indent, blankBefore, path)
		if skipped {
			return 1
		}
	case configschema.NestingList:
		// For the sake of handling nested blocks, we'll treat a null list
		// the same as an empty list since the config language doesn't
		// distinguish these anyway.
		old = ctyNullBlockListAsEmpty(old)
		new = ctyNullBlockListAsEmpty(new)

		oldItems := ctyCollectionValues(old)
		newItems := ctyCollectionValues(new)

		// Here we intentionally preserve the index-based correspondance
		// between old and new, rather than trying to detect insertions
		// and removals in the list, because this more accurately reflects
		// how Terraform Core and providers will understand the change,
		// particularly when the nested block contains computed attributes
		// that will themselves maintain correspondance by index.

		// commonLen is number of elements that exist in both lists, which
		// will be presented as updates (~). Any additional items in one
		// of the lists will be presented as either creates (+) or deletes (-)
		// depending on which list they belong to.
		var commonLen int
		switch {
		case len(oldItems) < len(newItems):
			commonLen = len(oldItems)
		default:
			commonLen = len(newItems)
		}

		blankBeforeInner := blankBefore
		for i := 0; i < commonLen; i++ {
			path := append(path, cty.IndexStep{Key: cty.NumberIntVal(int64(i))})
			oldItem := oldItems[i]
			newItem := newItems[i]
			action := plans.Update
			if oldItem.RawEquals(newItem) {
				action = plans.NoOp
			}
			skipped := p.writeNestedBlockDiff(name, nil, &blockS.Block, action, oldItem, newItem, indent, blankBeforeInner, path)
			if skipped {
				skippedBlocks++
			} else {
				blankBeforeInner = false
			}
		}
		for i := commonLen; i < len(oldItems); i++ {
			path := append(path, cty.IndexStep{Key: cty.NumberIntVal(int64(i))})
			oldItem := oldItems[i]
			newItem := cty.NullVal(oldItem.Type())
			skipped := p.writeNestedBlockDiff(name, nil, &blockS.Block, plans.Delete, oldItem, newItem, indent, blankBeforeInner, path)
			if skipped {
				skippedBlocks++
			} else {
				blankBeforeInner = false
			}
		}
		for i := commonLen; i < len(newItems); i++ {
			path := append(path, cty.IndexStep{Key: cty.NumberIntVal(int64(i))})
			newItem := newItems[i]
			oldItem := cty.NullVal(newItem.Type())
			skipped := p.writeNestedBlockDiff(name, nil, &blockS.Block, plans.Create, oldItem, newItem, indent, blankBeforeInner, path)
			if skipped {
				skippedBlocks++
			} else {
				blankBeforeInner = false
			}
		}
	case configschema.NestingSet:
		// For the sake of handling nested blocks, we'll treat a null set
		// the same as an empty set since the config language doesn't
		// distinguish these anyway.
		old = ctyNullBlockSetAsEmpty(old)
		new = ctyNullBlockSetAsEmpty(new)

		oldItems := ctyCollectionValues(old)
		newItems := ctyCollectionValues(new)

		if (len(oldItems) + len(newItems)) == 0 {
			// Nothing to do if both sets are empty
			return 0
		}

		allItems := make([]cty.Value, 0, len(oldItems)+len(newItems))
		allItems = append(allItems, oldItems...)
		allItems = append(allItems, newItems...)
		all := cty.SetVal(allItems)

		blankBeforeInner := blankBefore
		for it := all.ElementIterator(); it.Next(); {
			_, val := it.Element()
			var action plans.Action
			var oldValue, newValue cty.Value
			switch {
			case !val.IsKnown():
				action = plans.Update
				newValue = val
			case !old.HasElement(val).True():
				action = plans.Create
				oldValue = cty.NullVal(val.Type())
				newValue = val
			case !new.HasElement(val).True():
				action = plans.Delete
				oldValue = val
				newValue = cty.NullVal(val.Type())
			default:
				action = plans.NoOp
				oldValue = val
				newValue = val
			}
			path := append(path, cty.IndexStep{Key: val})
			skipped := p.writeNestedBlockDiff(name, nil, &blockS.Block, action, oldValue, newValue, indent, blankBeforeInner, path)
			if skipped {
				skippedBlocks++
			} else {
				blankBeforeInner = false
			}
		}

	case configschema.NestingMap:
		// For the sake of handling nested blocks, we'll treat a null map
		// the same as an empty map since the config language doesn't
		// distinguish these anyway.
		old = ctyNullBlockMapAsEmpty(old)
		new = ctyNullBlockMapAsEmpty(new)

		oldItems := old.AsValueMap()
		newItems := new.AsValueMap()
		if (len(oldItems) + len(newItems)) == 0 {
			// Nothing to do if both maps are empty
			return 0
		}

		allKeys := make(map[string]bool)
		for k := range oldItems {
			allKeys[k] = true
		}
		for k := range newItems {
			allKeys[k] = true
		}
		allKeysOrder := make([]string, 0, len(allKeys))
		for k := range allKeys {
			allKeysOrder = append(allKeysOrder, k)
		}
		sort.Strings(allKeysOrder)

		blankBeforeInner := blankBefore
		for _, k := range allKeysOrder {
			var action plans.Action
			oldValue := oldItems[k]
			newValue := newItems[k]
			switch {
			case oldValue == cty.NilVal:
				oldValue = cty.NullVal(newValue.Type())
				action = plans.Create
			case newValue == cty.NilVal:
				newValue = cty.NullVal(oldValue.Type())
				action = plans.Delete
			case !newValue.RawEquals(oldValue):
				action = plans.Update
			default:
				action = plans.NoOp
			}

			path := append(path, cty.IndexStep{Key: cty.StringVal(k)})
			skipped := p.writeNestedBlockDiff(name, &k, &blockS.Block, action, oldValue, newValue, indent, blankBeforeInner, path)
			if skipped {
				skippedBlocks++
			} else {
				blankBeforeInner = false
			}
		}
	}
	return skippedBlocks
}

func (p *blockBodyDiffPrinter) writeSensitiveNestedBlockDiff(name string, old, new cty.Value, indent int, blankBefore bool, path cty.Path) {
	var action plans.Action
	switch {
	case old.IsNull():
		action = plans.Create
	case new.IsNull():
		action = plans.Delete
	case !new.IsWhollyKnown() || !old.IsWhollyKnown():
		// "old" should actually always be known due to our contract
		// that old values must never be unknown, but we'll allow it
		// anyway to be robust.
		action = plans.Update
	case !ctyEqualValueAndMarks(old, new):
		action = plans.Update
	}

	if blankBefore {
		p.buf.WriteRune('\n')
	}

	// New line before warning printing
	p.buf.WriteRune('\n')
	p.writeSensitivityWarning(old, new, indent, action, true)
	p.buf.WriteString(strings.Repeat(" ", indent))
	p.writeActionSymbol(action)
	fmt.Fprintf(p.buf, "%s {", name)
	if action != plans.NoOp && p.pathForcesNewResource(path) {
		p.buf.WriteString(p.color.Color(forcesNewResourceCaption))
	}
	p.buf.WriteRune('\n')
	p.buf.WriteString(strings.Repeat(" ", indent+4))
	p.buf.WriteString("# At least one attribute in this block is (or was) sensitive,\n")
	p.buf.WriteString(strings.Repeat(" ", indent+4))
	p.buf.WriteString("# so its contents will not be displayed.")
	p.buf.WriteRune('\n')
	p.buf.WriteString(strings.Repeat(" ", indent+2))
	p.buf.WriteString("}")
}

func (p *blockBodyDiffPrinter) writeNestedBlockDiff(name string, label *string, blockS *configschema.Block, action plans.Action, old, new cty.Value, indent int, blankBefore bool, path cty.Path) bool {
	if action == plans.NoOp && !p.verbose {
		return true
	}

	if blankBefore {
		p.buf.WriteRune('\n')
	}

	p.buf.WriteString("\n")
	p.buf.WriteString(strings.Repeat(" ", indent))
	p.writeActionSymbol(action)

	if label != nil {
		fmt.Fprintf(p.buf, "%s %q {", name, *label)
	} else {
		fmt.Fprintf(p.buf, "%s {", name)
	}

	if action != plans.NoOp && (p.pathForcesNewResource(path) || p.pathForcesNewResource(path[:len(path)-1])) {
		p.buf.WriteString(p.color.Color(forcesNewResourceCaption))
	}

	result := p.writeBlockBodyDiff(blockS, old, new, indent+4, path)
	if result.bodyWritten {
		p.buf.WriteString("\n")
		p.buf.WriteString(strings.Repeat(" ", indent+2))
	}
	p.buf.WriteString("}")

	return false
}

func (p *blockBodyDiffPrinter) writeValue(val cty.Value, action plans.Action, indent int) {
	// Could check specifically for the sensitivity marker
	if val.HasMark(marks.Sensitive) {
		p.buf.WriteString(sensitiveCaption)
		return
	}

	if !val.IsKnown() {
		p.buf.WriteString("(known after apply)")
		return
	}
	if val.IsNull() {
		p.buf.WriteString(p.color.Color("[dark_gray]null[reset]"))
		return
	}

	ty := val.Type()

	switch {
	case ty.IsPrimitiveType():
		switch ty {
		case cty.String:
			{
				// Special behavior for JSON strings containing array or object
				src := []byte(val.AsString())
				ty, err := ctyjson.ImpliedType(src)
				// check for the special case of "null", which decodes to nil,
				// and just allow it to be printed out directly
				if err == nil && !ty.IsPrimitiveType() && strings.TrimSpace(val.AsString()) != "null" {
					jv, err := ctyjson.Unmarshal(src, ty)
					if err == nil {
						p.buf.WriteString("jsonencode(")
						if jv.LengthInt() == 0 {
							p.writeValue(jv, action, 0)
						} else {
							p.buf.WriteByte('\n')
							p.buf.WriteString(strings.Repeat(" ", indent+4))
							p.writeValue(jv, action, indent+4)
							p.buf.WriteByte('\n')
							p.buf.WriteString(strings.Repeat(" ", indent))
						}
						p.buf.WriteByte(')')
						break // don't *also* do the normal behavior below
					}
				}
			}

			if strings.Contains(val.AsString(), "\n") {
				// It's a multi-line string, so we want to use the multi-line
				// rendering so it'll be readable. Rather than re-implement
				// that here, we'll just re-use the multi-line string diff
				// printer with no changes, which ends up producing the
				// result we want here.
				// The path argument is nil because we don't track path
				// information into strings and we know that a string can't
				// have any indices or attributes that might need to be marked
				// as (requires replacement), which is what that argument is for.
				p.writeValueDiff(val, val, indent, nil)
				break
			}

			fmt.Fprintf(p.buf, "%q", val.AsString())
		case cty.Bool:
			if val.True() {
				p.buf.WriteString("true")
			} else {
				p.buf.WriteString("false")
			}
		case cty.Number:
			bf := val.AsBigFloat()
			p.buf.WriteString(bf.Text('f', -1))
		default:
			// should never happen, since the above is exhaustive
			fmt.Fprintf(p.buf, "%#v", val)
		}
	case ty.IsListType() || ty.IsSetType() || ty.IsTupleType():
		p.buf.WriteString("[")

		it := val.ElementIterator()
		for it.Next() {
			_, val := it.Element()

			p.buf.WriteString("\n")
			p.buf.WriteString(strings.Repeat(" ", indent+2))
			p.writeActionSymbol(action)
			p.writeValue(val, action, indent+4)
			p.buf.WriteString(",")
		}

		if val.LengthInt() > 0 {
			p.buf.WriteString("\n")
			p.buf.WriteString(strings.Repeat(" ", indent))
		}
		p.buf.WriteString("]")
	case ty.IsMapType():
		p.buf.WriteString("{")

		keyLen := 0
		for it := val.ElementIterator(); it.Next(); {
			key, _ := it.Element()
			if keyStr := key.AsString(); len(keyStr) > keyLen {
				keyLen = len(keyStr)
			}
		}

		for it := val.ElementIterator(); it.Next(); {
			key, val := it.Element()

			p.buf.WriteString("\n")
			p.buf.WriteString(strings.Repeat(" ", indent+2))
			p.writeActionSymbol(action)
			p.writeValue(key, action, indent+4)
			p.buf.WriteString(strings.Repeat(" ", keyLen-len(key.AsString())))
			p.buf.WriteString(" = ")
			p.writeValue(val, action, indent+4)
		}

		if val.LengthInt() > 0 {
			p.buf.WriteString("\n")
			p.buf.WriteString(strings.Repeat(" ", indent))
		}
		p.buf.WriteString("}")
	case ty.IsObjectType():
		p.buf.WriteString("{")

		atys := ty.AttributeTypes()
		attrNames := make([]string, 0, len(atys))
		displayAttrNames := make(map[string]string, len(atys))
		nameLen := 0
		for attrName := range atys {
			attrNames = append(attrNames, attrName)
			displayAttrNames[attrName] = displayAttributeName(attrName)
			if len(displayAttrNames[attrName]) > nameLen {
				nameLen = len(displayAttrNames[attrName])
			}
		}
		sort.Strings(attrNames)

		for _, attrName := range attrNames {
			val := val.GetAttr(attrName)
			displayAttrName := displayAttrNames[attrName]

			p.buf.WriteString("\n")
			p.buf.WriteString(strings.Repeat(" ", indent+2))
			p.writeActionSymbol(action)
			p.buf.WriteString(displayAttrName)
			p.buf.WriteString(strings.Repeat(" ", nameLen-len(displayAttrName)))
			p.buf.WriteString(" = ")
			p.writeValue(val, action, indent+4)
		}

		if len(attrNames) > 0 {
			p.buf.WriteString("\n")
			p.buf.WriteString(strings.Repeat(" ", indent))
		}
		p.buf.WriteString("}")
	}
}

func (p *blockBodyDiffPrinter) writeValueDiff(old, new cty.Value, indent int, path cty.Path) {
	ty := old.Type()
	typesEqual := ctyTypesEqual(ty, new.Type())

	// We have some specialized diff implementations for certain complex
	// values where it's useful to see a visualization of the diff of
	// the nested elements rather than just showing the entire old and
	// new values verbatim.
	// However, these specialized implementations can apply only if both
	// values are known and non-null.
	if old.IsKnown() && new.IsKnown() && !old.IsNull() && !new.IsNull() && typesEqual {
		if old.HasMark(marks.Sensitive) || new.HasMark(marks.Sensitive) {
			p.buf.WriteString(sensitiveCaption)
			if p.pathForcesNewResource(path) {
				p.buf.WriteString(p.color.Color(forcesNewResourceCaption))
			}
			return
		}

		switch {
		case ty == cty.String:
			// We have special behavior for both multi-line strings in general
			// and for strings that can parse as JSON. For the JSON handling
			// to apply, both old and new must be valid JSON.
			// For single-line strings that don't parse as JSON we just fall
			// out of this switch block and do the default old -> new rendering.
			oldS := old.AsString()
			newS := new.AsString()

			{
				// Special behavior for JSON strings containing object or
				// list values.
				oldBytes := []byte(oldS)
				newBytes := []byte(newS)
				oldType, oldErr := ctyjson.ImpliedType(oldBytes)
				newType, newErr := ctyjson.ImpliedType(newBytes)
				if oldErr == nil && newErr == nil && !(oldType.IsPrimitiveType() && newType.IsPrimitiveType()) {
					oldJV, oldErr := ctyjson.Unmarshal(oldBytes, oldType)
					newJV, newErr := ctyjson.Unmarshal(newBytes, newType)
					if oldErr == nil && newErr == nil {
						if !oldJV.RawEquals(newJV) { // two JSON values may differ only in insignificant whitespace
							p.buf.WriteString("jsonencode(")
							p.buf.WriteByte('\n')
							p.buf.WriteString(strings.Repeat(" ", indent+2))
							p.writeActionSymbol(plans.Update)
							p.writeValueDiff(oldJV, newJV, indent+4, path)
							p.buf.WriteByte('\n')
							p.buf.WriteString(strings.Repeat(" ", indent))
							p.buf.WriteByte(')')
						} else {
							// if they differ only in insignificant whitespace
							// then we'll note that but still expand out the
							// effective value.
							if p.pathForcesNewResource(path) {
								p.buf.WriteString(p.color.Color("jsonencode( [red]# whitespace changes force replacement[reset]"))
							} else {
								p.buf.WriteString(p.color.Color("jsonencode( [dim]# whitespace changes[reset]"))
							}
							p.buf.WriteByte('\n')
							p.buf.WriteString(strings.Repeat(" ", indent+4))
							p.writeValue(oldJV, plans.NoOp, indent+4)
							p.buf.WriteByte('\n')
							p.buf.WriteString(strings.Repeat(" ", indent))
							p.buf.WriteByte(')')
						}
						return
					}
				}
			}

			if !strings.Contains(oldS, "\n") && !strings.Contains(newS, "\n") {
				break
			}

			p.buf.WriteString("<<-EOT")
			if p.pathForcesNewResource(path) {
				p.buf.WriteString(p.color.Color(forcesNewResourceCaption))
			}
			p.buf.WriteString("\n")

			var oldLines, newLines []cty.Value
			{
				r := strings.NewReader(oldS)
				sc := bufio.NewScanner(r)
				for sc.Scan() {
					oldLines = append(oldLines, cty.StringVal(sc.Text()))
				}
			}
			{
				r := strings.NewReader(newS)
				sc := bufio.NewScanner(r)
				for sc.Scan() {
					newLines = append(newLines, cty.StringVal(sc.Text()))
				}
			}

			// Optimization for strings which are exactly equal: just print
			// directly without calculating the sequence diff. This makes a
			// significant difference when this code path is reached via a
			// writeValue call with a large multi-line string.
			if oldS == newS {
				for _, line := range newLines {
					p.buf.WriteString(strings.Repeat(" ", indent+4))
					p.buf.WriteString(line.AsString())
					p.buf.WriteString("\n")
				}
			} else {
				diffLines := ctySequenceDiff(oldLines, newLines)
				for _, diffLine := range diffLines {
					p.buf.WriteString(strings.Repeat(" ", indent+2))
					p.writeActionSymbol(diffLine.Action)

					switch diffLine.Action {
					case plans.NoOp, plans.Delete:
						p.buf.WriteString(diffLine.Before.AsString())
					case plans.Create:
						p.buf.WriteString(diffLine.After.AsString())
					default:
						// Should never happen since the above covers all
						// actions that ctySequenceDiff can return for strings
						p.buf.WriteString(diffLine.After.AsString())

					}
					p.buf.WriteString("\n")
				}
			}

			p.buf.WriteString(strings.Repeat(" ", indent)) // +4 here because there's no symbol
			p.buf.WriteString("EOT")

			return

		case ty.IsSetType():
			p.buf.WriteString("[")
			if p.pathForcesNewResource(path) {
				p.buf.WriteString(p.color.Color(forcesNewResourceCaption))
			}
			p.buf.WriteString("\n")

			var addedVals, removedVals, allVals []cty.Value
			for it := old.ElementIterator(); it.Next(); {
				_, val := it.Element()
				allVals = append(allVals, val)
				if new.HasElement(val).False() {
					removedVals = append(removedVals, val)
				}
			}
			for it := new.ElementIterator(); it.Next(); {
				_, val := it.Element()
				allVals = append(allVals, val)
				if val.IsKnown() && old.HasElement(val).False() {
					addedVals = append(addedVals, val)
				}
			}

			var all, added, removed cty.Value
			if len(allVals) > 0 {
				all = cty.SetVal(allVals)
			} else {
				all = cty.SetValEmpty(ty.ElementType())
			}
			if len(addedVals) > 0 {
				added = cty.SetVal(addedVals)
			} else {
				added = cty.SetValEmpty(ty.ElementType())
			}
			if len(removedVals) > 0 {
				removed = cty.SetVal(removedVals)
			} else {
				removed = cty.SetValEmpty(ty.ElementType())
			}

			suppressedElements := 0
			for it := all.ElementIterator(); it.Next(); {
				_, val := it.Element()

				var action plans.Action
				switch {
				case !val.IsKnown():
					action = plans.Update
				case added.HasElement(val).True():
					action = plans.Create
				case removed.HasElement(val).True():
					action = plans.Delete
				default:
					action = plans.NoOp
				}

				if action == plans.NoOp && !p.verbose {
					suppressedElements++
					continue
				}

				p.buf.WriteString(strings.Repeat(" ", indent+2))
				p.writeActionSymbol(action)
				p.writeValue(val, action, indent+4)
				p.buf.WriteString(",\n")
			}

			if suppressedElements > 0 {
				p.writeActionSymbol(plans.NoOp)
				p.buf.WriteString(strings.Repeat(" ", indent+2))
				noun := "elements"
				if suppressedElements == 1 {
					noun = "element"
				}
				p.buf.WriteString(fmt.Sprintf(p.color.Color("[dark_gray]# (%d unchanged %s hidden)[reset]"), suppressedElements, noun))
				p.buf.WriteString("\n")
			}

			p.buf.WriteString(strings.Repeat(" ", indent))
			p.buf.WriteString("]")
			return
		case ty.IsListType() || ty.IsTupleType():
			p.buf.WriteString("[")
			if p.pathForcesNewResource(path) {
				p.buf.WriteString(p.color.Color(forcesNewResourceCaption))
			}
			p.buf.WriteString("\n")

			elemDiffs := ctySequenceDiff(old.AsValueSlice(), new.AsValueSlice())

			// Maintain a stack of suppressed lines in the diff for later
			// display or elision
			var suppressedElements []*plans.Change
			var changeShown bool

			for i := 0; i < len(elemDiffs); i++ {
				if !p.verbose {
					for i < len(elemDiffs) && elemDiffs[i].Action == plans.NoOp {
						suppressedElements = append(suppressedElements, elemDiffs[i])
						i++
					}
				}

				// If we have some suppressed elements on the stack…
				if len(suppressedElements) > 0 {
					// If we've just rendered a change, display the first
					// element in the stack as context
					if changeShown {
						elemDiff := suppressedElements[0]
						p.buf.WriteString(strings.Repeat(" ", indent+4))
						p.writeValue(elemDiff.After, elemDiff.Action, indent+4)
						p.buf.WriteString(",\n")
						suppressedElements = suppressedElements[1:]
					}

					hidden := len(suppressedElements)

					// If we're not yet at the end of the list, capture the
					// last element on the stack as context for the upcoming
					// change to be rendered
					var nextContextDiff *plans.Change
					if hidden > 0 && i < len(elemDiffs) {
						hidden--
						nextContextDiff = suppressedElements[hidden]
					}

					// If there are still hidden elements, show an elision
					// statement counting them
					if hidden > 0 {
						p.writeActionSymbol(plans.NoOp)
						p.buf.WriteString(strings.Repeat(" ", indent+2))
						noun := "elements"
						if hidden == 1 {
							noun = "element"
						}
						p.buf.WriteString(fmt.Sprintf(p.color.Color("[dark_gray]# (%d unchanged %s hidden)[reset]"), hidden, noun))
						p.buf.WriteString("\n")
					}

					// Display the next context diff if it was captured above
					if nextContextDiff != nil {
						p.buf.WriteString(strings.Repeat(" ", indent+4))
						p.writeValue(nextContextDiff.After, nextContextDiff.Action, indent+4)
						p.buf.WriteString(",\n")
					}

					// Suppressed elements have now been handled so clear them again
					suppressedElements = nil
				}

				if i >= len(elemDiffs) {
					break
				}

				elemDiff := elemDiffs[i]
				p.buf.WriteString(strings.Repeat(" ", indent+2))
				p.writeActionSymbol(elemDiff.Action)
				switch elemDiff.Action {
				case plans.NoOp, plans.Delete:
					p.writeValue(elemDiff.Before, elemDiff.Action, indent+4)
				case plans.Update:
					p.writeValueDiff(elemDiff.Before, elemDiff.After, indent+4, path)
				case plans.Create:
					p.writeValue(elemDiff.After, elemDiff.Action, indent+4)
				default:
					// Should never happen since the above covers all
					// actions that ctySequenceDiff can return.
					p.writeValue(elemDiff.After, elemDiff.Action, indent+4)
				}

				p.buf.WriteString(",\n")
				changeShown = true
			}

			p.buf.WriteString(strings.Repeat(" ", indent))
			p.buf.WriteString("]")

			return

		case ty.IsMapType():
			p.buf.WriteString("{")
			if p.pathForcesNewResource(path) {
				p.buf.WriteString(p.color.Color(forcesNewResourceCaption))
			}
			p.buf.WriteString("\n")

			var allKeys []string
			keyLen := 0
			for it := old.ElementIterator(); it.Next(); {
				k, _ := it.Element()
				keyStr := k.AsString()
				allKeys = append(allKeys, keyStr)
				if len(keyStr) > keyLen {
					keyLen = len(keyStr)
				}
			}
			for it := new.ElementIterator(); it.Next(); {
				k, _ := it.Element()
				keyStr := k.AsString()
				allKeys = append(allKeys, keyStr)
				if len(keyStr) > keyLen {
					keyLen = len(keyStr)
				}
			}

			sort.Strings(allKeys)

			suppressedElements := 0
			lastK := ""
			for i, k := range allKeys {
				if i > 0 && lastK == k {
					continue // skip duplicates (list is sorted)
				}
				lastK = k

				kV := cty.StringVal(k)
				var action plans.Action
				if old.HasIndex(kV).False() {
					action = plans.Create
				} else if new.HasIndex(kV).False() {
					action = plans.Delete
				}

				if old.HasIndex(kV).True() && new.HasIndex(kV).True() {
					if ctyEqualValueAndMarks(old.Index(kV), new.Index(kV)) {
						action = plans.NoOp
					} else {
						action = plans.Update
					}
				}

				if action == plans.NoOp && !p.verbose {
					suppressedElements++
					continue
				}

				path := append(path, cty.IndexStep{Key: kV})

				oldV := old.Index(kV)
				newV := new.Index(kV)
				p.writeSensitivityWarning(oldV, newV, indent+2, action, false)

				p.buf.WriteString(strings.Repeat(" ", indent+2))
				p.writeActionSymbol(action)
				p.writeValue(cty.StringVal(k), action, indent+4)
				p.buf.WriteString(strings.Repeat(" ", keyLen-len(k)))
				p.buf.WriteString(" = ")
				switch action {
				case plans.Create, plans.NoOp:
					v := new.Index(kV)
					if v.HasMark(marks.Sensitive) {
						p.buf.WriteString(sensitiveCaption)
					} else {
						p.writeValue(v, action, indent+4)
					}
				case plans.Delete:
					oldV := old.Index(kV)
					newV := cty.NullVal(oldV.Type())
					p.writeValueDiff(oldV, newV, indent+4, path)
				default:
					if oldV.HasMark(marks.Sensitive) || newV.HasMark(marks.Sensitive) {
						p.buf.WriteString(sensitiveCaption)
					} else {
						p.writeValueDiff(oldV, newV, indent+4, path)
					}
				}

				p.buf.WriteByte('\n')
			}

			if suppressedElements > 0 {
				p.writeActionSymbol(plans.NoOp)
				p.buf.WriteString(strings.Repeat(" ", indent+2))
				noun := "elements"
				if suppressedElements == 1 {
					noun = "element"
				}
				p.buf.WriteString(fmt.Sprintf(p.color.Color("[dark_gray]# (%d unchanged %s hidden)[reset]"), suppressedElements, noun))
				p.buf.WriteString("\n")
			}

			p.buf.WriteString(strings.Repeat(" ", indent))
			p.buf.WriteString("}")

			return
		case ty.IsObjectType():
			p.buf.WriteString("{")
			p.buf.WriteString("\n")

			forcesNewResource := p.pathForcesNewResource(path)

			var allKeys []string
			displayKeys := make(map[string]string)
			keyLen := 0
			for it := old.ElementIterator(); it.Next(); {
				k, _ := it.Element()
				keyStr := k.AsString()
				allKeys = append(allKeys, keyStr)
				displayKeys[keyStr] = displayAttributeName(keyStr)
				if len(displayKeys[keyStr]) > keyLen {
					keyLen = len(displayKeys[keyStr])
				}
			}
			for it := new.ElementIterator(); it.Next(); {
				k, _ := it.Element()
				keyStr := k.AsString()
				allKeys = append(allKeys, keyStr)
				displayKeys[keyStr] = displayAttributeName(keyStr)
				if len(displayKeys[keyStr]) > keyLen {
					keyLen = len(displayKeys[keyStr])
				}
			}

			sort.Strings(allKeys)

			suppressedElements := 0
			lastK := ""
			for i, k := range allKeys {
				if i > 0 && lastK == k {
					continue // skip duplicates (list is sorted)
				}
				lastK = k

				kV := k
				var action plans.Action
				if !old.Type().HasAttribute(kV) {
					action = plans.Create
				} else if !new.Type().HasAttribute(kV) {
					action = plans.Delete
				} else if ctyEqualValueAndMarks(old.GetAttr(kV), new.GetAttr(kV)) {
					action = plans.NoOp
				} else {
					action = plans.Update
				}

				// TODO: If in future we have a schema associated with this
				// object, we should pass the attribute's schema to
				// identifyingAttribute here.
				if action == plans.NoOp && !p.verbose && !identifyingAttribute(k, nil) {
					suppressedElements++
					continue
				}

				path := append(path, cty.GetAttrStep{Name: kV})

				p.buf.WriteString(strings.Repeat(" ", indent+2))
				p.writeActionSymbol(action)
				p.buf.WriteString(displayKeys[k])
				p.buf.WriteString(strings.Repeat(" ", keyLen-len(displayKeys[k])))
				p.buf.WriteString(" = ")

				switch action {
				case plans.Create, plans.NoOp:
					v := new.GetAttr(kV)
					p.writeValue(v, action, indent+4)
				case plans.Delete:
					oldV := old.GetAttr(kV)
					newV := cty.NullVal(oldV.Type())
					p.writeValueDiff(oldV, newV, indent+4, path)
				default:
					oldV := old.GetAttr(kV)
					newV := new.GetAttr(kV)
					p.writeValueDiff(oldV, newV, indent+4, path)
				}

				p.buf.WriteString("\n")
			}

			if suppressedElements > 0 {
				p.writeActionSymbol(plans.NoOp)
				p.buf.WriteString(strings.Repeat(" ", indent+2))
				noun := "elements"
				if suppressedElements == 1 {
					noun = "element"
				}
				p.buf.WriteString(fmt.Sprintf(p.color.Color("[dark_gray]# (%d unchanged %s hidden)[reset]"), suppressedElements, noun))
				p.buf.WriteString("\n")
			}

			p.buf.WriteString(strings.Repeat(" ", indent))
			p.buf.WriteString("}")

			if forcesNewResource {
				p.buf.WriteString(p.color.Color(forcesNewResourceCaption))
			}
			return
		}
	}

	// In all other cases, we just show the new and old values as-is
	p.writeValue(old, plans.Delete, indent)
	if new.IsNull() {
		p.buf.WriteString(p.color.Color(" [dark_gray]->[reset] "))
	} else {
		p.buf.WriteString(p.color.Color(" [yellow]->[reset] "))
	}

	p.writeValue(new, plans.Create, indent)
	if p.pathForcesNewResource(path) {
		p.buf.WriteString(p.color.Color(forcesNewResourceCaption))
	}
}

// writeActionSymbol writes a symbol to represent the given action, followed
// by a space.
//
// It only supports the actions that can be represented with a single character:
// Create, Delete, Update and NoAction.
func (p *blockBodyDiffPrinter) writeActionSymbol(action plans.Action) {
	switch action {
	case plans.Create:
		p.buf.WriteString(p.color.Color("[green]+[reset] "))
	case plans.Delete:
		p.buf.WriteString(p.color.Color("[red]-[reset] "))
	case plans.Update:
		p.buf.WriteString(p.color.Color("[yellow]~[reset] "))
	case plans.NoOp:
		p.buf.WriteString("  ")
	default:
		// Should never happen
		p.buf.WriteString(p.color.Color("? "))
	}
}

func (p *blockBodyDiffPrinter) writeSensitivityWarning(old, new cty.Value, indent int, action plans.Action, isBlock bool) {
	// Dont' show this warning for create or delete
	if action == plans.Create || action == plans.Delete {
		return
	}

	// Customize the warning based on if it is an attribute or block
	diffType := "attribute value"
	if isBlock {
		diffType = "block"
	}

	// If only attribute sensitivity is changing, clarify that the value is unchanged
	var valueUnchangedSuffix string
	if !isBlock {
		oldUnmarked, _ := old.UnmarkDeep()
		newUnmarked, _ := new.UnmarkDeep()
		if oldUnmarked.RawEquals(newUnmarked) {
			valueUnchangedSuffix = " The value is unchanged."
		}
	}

	if new.HasMark(marks.Sensitive) && !old.HasMark(marks.Sensitive) {
		p.buf.WriteString(strings.Repeat(" ", indent))
		p.buf.WriteString(fmt.Sprintf(p.color.Color("# [yellow]Warning:[reset] this %s will be marked as sensitive and will not\n"), diffType))
		p.buf.WriteString(strings.Repeat(" ", indent))
		p.buf.WriteString(fmt.Sprintf("# display in UI output after applying this change.%s\n", valueUnchangedSuffix))
	}

	// Note if changing this attribute will change its sensitivity
	if old.HasMark(marks.Sensitive) && !new.HasMark(marks.Sensitive) {
		p.buf.WriteString(strings.Repeat(" ", indent))
		p.buf.WriteString(fmt.Sprintf(p.color.Color("# [yellow]Warning:[reset] this %s will no longer be marked as sensitive\n"), diffType))
		p.buf.WriteString(strings.Repeat(" ", indent))
		p.buf.WriteString(fmt.Sprintf("# after applying this change.%s\n", valueUnchangedSuffix))
	}
}

func (p *blockBodyDiffPrinter) pathForcesNewResource(path cty.Path) bool {
	if !p.action.IsReplace() || p.requiredReplace.Empty() {
		// "requiredReplace" only applies when the instance is being replaced,
		// and we should only inspect that set if it is not empty
		return false
	}
	return p.requiredReplace.Has(path)
}

func ctyEmptyString(value cty.Value) bool {
	if !value.IsNull() && value.IsKnown() {
		valueType := value.Type()
		if valueType == cty.String && value.AsString() == "" {
			return true
		}
	}
	return false
}

func ctyGetAttrMaybeNull(val cty.Value, name string) cty.Value {
	attrType := val.Type().AttributeType(name)

	if val.IsNull() {
		return cty.NullVal(attrType)
	}

	// We treat "" as null here
	// as existing SDK doesn't support null yet.
	// This allows us to avoid spurious diffs
	// until we introduce null to the SDK.
	attrValue := val.GetAttr(name)
	// If the value is marked, the ctyEmptyString function will fail
	if !val.ContainsMarked() && ctyEmptyString(attrValue) {
		return cty.NullVal(attrType)
	}

	return attrValue
}

func ctyCollectionValues(val cty.Value) []cty.Value {
	if !val.IsKnown() || val.IsNull() {
		return nil
	}

	ret := make([]cty.Value, 0, val.LengthInt())
	for it := val.ElementIterator(); it.Next(); {
		_, value := it.Element()
		ret = append(ret, value)
	}
	return ret
}

// ctySequenceDiff returns differences between given sequences of cty.Value(s)
// in the form of Create, Delete, or Update actions (for objects).
func ctySequenceDiff(old, new []cty.Value) []*plans.Change {
	var ret []*plans.Change
	lcs := objchange.LongestCommonSubsequence(old, new)
	var oldI, newI, lcsI int
	for oldI < len(old) || newI < len(new) || lcsI < len(lcs) {
		// We first process items in the old and new sequences which are not
		// equal to the current common sequence item.  Old items are marked as
		// deletions, and new items are marked as additions.
		//
		// There is an exception for deleted & created object items, which we
		// try to render as updates where that makes sense.
		for oldI < len(old) && (lcsI >= len(lcs) || !old[oldI].RawEquals(lcs[lcsI])) {
			// Render this as an object update if all of these are true:
			//
			// - the current old item is an object;
			// - there's a current new item which is also an object;
			// - either there are no common items left, or the current new item
			//   doesn't equal the current common item.
			//
			// Why do we need the the last clause? If we have current items in all
			// three sequences, and the current new item is equal to a common item,
			// then we should just need to advance the old item list and we'll
			// eventually find a common item matching both old and new.
			//
			// This combination of conditions allows us to render an object update
			// diff instead of a combination of delete old & create new.
			isObjectDiff := old[oldI].Type().IsObjectType() && newI < len(new) && new[newI].Type().IsObjectType() && (lcsI >= len(lcs) || !new[newI].RawEquals(lcs[lcsI]))
			if isObjectDiff {
				ret = append(ret, &plans.Change{
					Action: plans.Update,
					Before: old[oldI],
					After:  new[newI],
				})
				oldI++
				newI++ // we also consume the next "new" in this case
				continue
			}

			// Otherwise, this item is not part of the common sequence, so
			// render as a deletion.
			ret = append(ret, &plans.Change{
				Action: plans.Delete,
				Before: old[oldI],
				After:  cty.NullVal(old[oldI].Type()),
			})
			oldI++
		}
		for newI < len(new) && (lcsI >= len(lcs) || !new[newI].RawEquals(lcs[lcsI])) {
			ret = append(ret, &plans.Change{
				Action: plans.Create,
				Before: cty.NullVal(new[newI].Type()),
				After:  new[newI],
			})
			newI++
		}

		// When we've exhausted the old & new sequences of items which are not
		// in the common subsequence, we render a common item and continue.
		if lcsI < len(lcs) {
			ret = append(ret, &plans.Change{
				Action: plans.NoOp,
				Before: lcs[lcsI],
				After:  lcs[lcsI],
			})

			// All of our indexes advance together now, since the line
			// is common to all three sequences.
			lcsI++
			oldI++
			newI++
		}
	}
	return ret
}

// ctyEqualValueAndMarks checks equality of two possibly-marked values,
// considering partially-unknown values and equal values with different marks
// as inequal
func ctyEqualWithUnknown(old, new cty.Value) bool {
	if !old.IsWhollyKnown() || !new.IsWhollyKnown() {
		return false
	}
	return ctyEqualValueAndMarks(old, new)
}

// ctyEqualValueAndMarks checks equality of two possibly-marked values,
// considering equal values with different marks as inequal
func ctyEqualValueAndMarks(old, new cty.Value) bool {
	oldUnmarked, oldMarks := old.UnmarkDeep()
	newUnmarked, newMarks := new.UnmarkDeep()
	sameValue := oldUnmarked.Equals(newUnmarked)
	return sameValue.IsKnown() && sameValue.True() && oldMarks.Equal(newMarks)
}

// ctyTypesEqual checks equality of two types more loosely
// by avoiding checks of object/tuple elements
// as we render differences on element-by-element basis anyway
func ctyTypesEqual(oldT, newT cty.Type) bool {
	if oldT.IsObjectType() && newT.IsObjectType() {
		return true
	}
	if oldT.IsTupleType() && newT.IsTupleType() {
		return true
	}
	return oldT.Equals(newT)
}

func ctyEnsurePathCapacity(path cty.Path, minExtra int) cty.Path {
	if cap(path)-len(path) >= minExtra {
		return path
	}
	newCap := cap(path) * 2
	if newCap < (len(path) + minExtra) {
		newCap = len(path) + minExtra
	}
	newPath := make(cty.Path, len(path), newCap)
	copy(newPath, path)
	return newPath
}

// ctyNullBlockListAsEmpty either returns the given value verbatim if it is non-nil
// or returns an empty value of a suitable type to serve as a placeholder for it.
//
// In particular, this function handles the special situation where a "list" is
// actually represented as a tuple type where nested blocks contain
// dynamically-typed values.
func ctyNullBlockListAsEmpty(in cty.Value) cty.Value {
	if !in.IsNull() {
		return in
	}
	if ty := in.Type(); ty.IsListType() {
		return cty.ListValEmpty(ty.ElementType())
	}
	return cty.EmptyTupleVal // must need a tuple, then
}

// ctyNullBlockMapAsEmpty either returns the given value verbatim if it is non-nil
// or returns an empty value of a suitable type to serve as a placeholder for it.
//
// In particular, this function handles the special situation where a "map" is
// actually represented as an object type where nested blocks contain
// dynamically-typed values.
func ctyNullBlockMapAsEmpty(in cty.Value) cty.Value {
	if !in.IsNull() {
		return in
	}
	if ty := in.Type(); ty.IsMapType() {
		return cty.MapValEmpty(ty.ElementType())
	}
	return cty.EmptyObjectVal // must need an object, then
}

// ctyNullBlockSetAsEmpty either returns the given value verbatim if it is non-nil
// or returns an empty value of a suitable type to serve as a placeholder for it.
func ctyNullBlockSetAsEmpty(in cty.Value) cty.Value {
	if !in.IsNull() {
		return in
	}
	// Dynamically-typed attributes are not supported inside blocks backed by
	// sets, so our result here is always a set.
	return cty.SetValEmpty(in.Type().ElementType())
}

// DiffActionSymbol returns a string that, once passed through a
// colorstring.Colorize, will produce a result that can be written
// to a terminal to produce a symbol made of three printable
// characters, possibly interspersed with VT100 color codes.
func DiffActionSymbol(action plans.Action) string {
	switch action {
	case plans.DeleteThenCreate:
		return "[red]-[reset]/[green]+[reset]"
	case plans.CreateThenDelete:
		return "[green]+[reset]/[red]-[reset]"
	case plans.Create:
		return "  [green]+[reset]"
	case plans.Delete:
		return "  [red]-[reset]"
	case plans.Read:
		return " [cyan]<=[reset]"
	case plans.Update:
		return "  [yellow]~[reset]"
	default:
		return "  ?"
	}
}

// Extremely coarse heuristic for determining whether or not a given attribute
// name is important for identifying a resource. In the future, this may be
// replaced by a flag in the schema, but for now this is likely to be good
// enough.
func identifyingAttribute(name string, attrSchema *configschema.Attribute) bool {
	return name == "id" || name == "tags" || name == "name"
}

func (p *blockBodyDiffPrinter) writeSkippedAttr(skipped, indent int) {
	if skipped > 0 {
		noun := "attributes"
		if skipped == 1 {
			noun = "attribute"
		}
		p.buf.WriteString("\n")
		p.buf.WriteString(strings.Repeat(" ", indent))
		p.buf.WriteString(fmt.Sprintf(p.color.Color("[dark_gray]# (%d unchanged %s hidden)[reset]"), skipped, noun))
	}
}

func (p *blockBodyDiffPrinter) writeSkippedElems(skipped, indent int) {
	if skipped > 0 {
		noun := "elements"
		if skipped == 1 {
			noun = "element"
		}
		p.buf.WriteString(strings.Repeat(" ", indent))
		p.buf.WriteString(fmt.Sprintf(p.color.Color("[dark_gray]# (%d unchanged %s hidden)[reset]"), skipped, noun))
		p.buf.WriteString("\n")
	}
}

func displayAttributeName(name string) string {
	if !hclsyntax.ValidIdentifier(name) {
		return fmt.Sprintf("%q", name)
	}
	return name
}
