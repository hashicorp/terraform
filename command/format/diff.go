package format

import (
	"bufio"
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/configschema"
	"github.com/hashicorp/terraform/diffs"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/colorstring"
	"github.com/zclconf/go-cty/cty"
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
	addr *terraform.ResourceAddress,
	change *diffs.Change,
	schema *configschema.Block,
	color *colorstring.Colorize,
) string {
	var buf bytes.Buffer

	if color == nil {
		color = &colorstring.Colorize{
			Colors:  colorstring.DefaultColors,
			Disable: true,
			Reset:   false,
		}
	}

	buf.WriteString(color.Color("[reset]"))

	switch change.Action {
	case diffs.Create:
		buf.WriteString(color.Color("[green]  +[reset] "))
	case diffs.Read:
		buf.WriteString(color.Color("[cyan] <=[reset] "))
	case diffs.Update:
		buf.WriteString(color.Color("[yellow]  ~[reset] "))
	case diffs.Replace:
		buf.WriteString(color.Color("[red]-[reset]/[green]+[reset] "))
	case diffs.Delete:
		buf.WriteString(color.Color("[red]  -[reset] "))
	default:
		// should never happen, since the above is exhaustive
		buf.WriteString(color.Color("??? "))
	}

	switch addr.Mode {
	case config.ManagedResourceMode:
		buf.WriteString(color.Color(fmt.Sprintf(
			"resource [bold]%q[reset] [bold]%q[reset]",
			addr.Type,
			addr.Name,
		)))
		if addr.Index != -1 {
			buf.WriteString(fmt.Sprintf(" [%d]", addr.Index))
		}
	case config.DataResourceMode:
		buf.WriteString(color.Color(fmt.Sprintf(
			"data [bold]%q[reset] [bold]%q[reset] ",
			addr.Type,
			addr.Name,
		)))
		if addr.Index != -1 {
			buf.WriteString(fmt.Sprintf(" [%d]", addr.Index))
		}
	default:
		// should never happen, since the above is exhaustive
		buf.WriteString(addr.String())
	}

	buf.WriteString(" {")
	if change.Action == diffs.Replace {
		buf.WriteString(color.Color(" [bold][red]# new resource required[reset]"))
	}
	buf.WriteString("\n")

	p := blockBodyDiffPrinter{
		buf:           &buf,
		color:         color,
		action:        change.Action,
		forcedReplace: change.ForcedReplace,
	}

	// Most commonly-used resources have nested blocks that result in us
	// going at least three traversals deep while we recurse here, so we'll
	// start with that much capacity and then grow as needed for deeper
	// structures.
	path := make(cty.Path, 0, 3)

	p.writeBlockBodyDiff(schema, change.Old, change.New, 6, path)

	buf.WriteString("    }\n")

	return buf.String()
}

type ctyValueDiff struct {
	Action diffs.Action
	Value  cty.Value
}

type blockBodyDiffPrinter struct {
	buf           *bytes.Buffer
	color         *colorstring.Colorize
	action        diffs.Action
	forcedReplace diffs.PathSet
}

const forcesNewResourceCaption = " [red]# (forces new resource)[reset]"

func (p *blockBodyDiffPrinter) writeBlockBodyDiff(schema *configschema.Block, old, new cty.Value, indent int, path cty.Path) {
	path = ctyEnsurePathCapacity(path, 1)

	blankBeforeBlocks := false
	{
		attrNames := make([]string, 0, len(schema.Attributes))
		attrNameLen := 0
		for name := range schema.Attributes {
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
			if len(name) > attrNameLen {
				attrNameLen = len(name)
			}
		}
		sort.Strings(attrNames)
		if len(attrNames) > 0 {
			blankBeforeBlocks = true
		}

		for _, name := range attrNames {
			attrS := schema.Attributes[name]
			oldVal := ctyGetAttrMaybeNull(old, name)
			newVal := ctyGetAttrMaybeNull(new, name)

			p.writeAttrDiff(name, attrS, oldVal, newVal, attrNameLen, indent, path)
		}
	}

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

			p.writeNestedBlockDiffs(name, blockS, oldVal, newVal, blankBeforeBlocks, indent, path)

			// Always include a blank for any subsequent block types.
			blankBeforeBlocks = true
		}
	}
}

func (p *blockBodyDiffPrinter) writeAttrDiff(name string, attrS *configschema.Attribute, old, new cty.Value, nameLen, indent int, path cty.Path) {
	path = append(path, cty.GetAttrStep{Name: name})
	p.buf.WriteString(strings.Repeat(" ", indent))
	showJustNew := false
	var action diffs.Action
	switch {
	case old.IsNull():
		action = diffs.Create
		showJustNew = true
	case new.IsNull():
		action = diffs.Delete
	case ctyEqualWithUnknown(old, new):
		action = diffs.NoAction
		showJustNew = true
	default:
		action = diffs.Update
	}

	p.writeActionSymbol(action)

	p.buf.WriteString(p.color.Color("[bold]"))
	p.buf.WriteString(name)
	p.buf.WriteString(p.color.Color("[reset]"))
	p.buf.WriteString(strings.Repeat(" ", nameLen-len(name)))
	p.buf.WriteString(" = ")

	if attrS.Sensitive {
		p.buf.WriteString("(sensitive value)")
	} else {
		switch {
		case showJustNew:
			p.writeValue(new, action, indent+2)
		default:
			// We show new even if it is null to emphasize the fact
			// that it is being unset, since otherwise it is easy to
			// misunderstand that the value is still set to the old value.
			p.writeValueDiff(old, new, indent+2, path)
		}
	}

	p.buf.WriteString("\n")

}

func (p *blockBodyDiffPrinter) writeNestedBlockDiffs(name string, blockS *configschema.NestedBlock, old, new cty.Value, blankBefore bool, indent int, path cty.Path) {
	path = append(path, cty.GetAttrStep{Name: name})
	if old.IsNull() && new.IsNull() {
		// Nothing to do if both old and new is null
		return
	}

	// Where old/new are collections representing a nesting mode other than
	// NestingSingle, we assume the collection value can never be unknown
	// since we always produce the container for the nested objects, even if
	// the objects within are computed.

	switch blockS.Nesting {
	case configschema.NestingSingle:
		var action diffs.Action
		switch {
		case old.IsNull():
			action = diffs.Create
		case new.IsNull():
			action = diffs.Delete
		case !new.IsKnown() || !old.IsKnown():
			// "old" should actually always be known due to our contract
			// that old values must never be unknown, but we'll allow it
			// anyway to be robust.
			action = diffs.Update
		case !(new.Equals(old).True()):
			action = diffs.Update
		}

		if blankBefore {
			p.buf.WriteRune('\n')
		}
		p.writeNestedBlockDiff(name, nil, &blockS.Block, action, old, new, indent, path)
	case configschema.NestingList:
		// For the sake of handling nested blocks, we'll treat a null list
		// the same as an empty list since the config language doesn't
		// distinguish these anyway.
		if old.IsNull() {
			old = cty.ListValEmpty(old.Type().ElementType())
		}
		if new.IsNull() {
			new = cty.ListValEmpty(new.Type().ElementType())
		}

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

		if blankBefore && (len(oldItems) > 0 || len(newItems) > 0) {
			p.buf.WriteRune('\n')
		}
		for i := 0; i < commonLen; i++ {
			path := append(path, cty.IndexStep{Key: cty.NumberIntVal(int64(i))})
			oldItem := oldItems[i]
			newItem := newItems[i]
			p.writeNestedBlockDiff(name, nil, &blockS.Block, diffs.Update, oldItem, newItem, indent, path)
		}
		for i := commonLen; i < len(oldItems); i++ {
			path := append(path, cty.IndexStep{Key: cty.NumberIntVal(int64(i))})
			oldItem := oldItems[i]
			newItem := cty.NullVal(oldItem.Type())
			p.writeNestedBlockDiff(name, nil, &blockS.Block, diffs.Delete, oldItem, newItem, indent, path)
		}
		for i := commonLen; i < len(newItems); i++ {
			path := append(path, cty.IndexStep{Key: cty.NumberIntVal(int64(i))})
			newItem := newItems[i]
			oldItem := cty.NullVal(newItem.Type())
			p.writeNestedBlockDiff(name, nil, &blockS.Block, diffs.Create, oldItem, newItem, indent, path)
		}
	case configschema.NestingSet:
		// For the sake of handling nested blocks, we'll treat a null set
		// the same as an empty set since the config language doesn't
		// distinguish these anyway.
		if old.IsNull() {
			old = cty.SetValEmpty(old.Type().ElementType())
		}
		if new.IsNull() {
			new = cty.SetValEmpty(new.Type().ElementType())
		}

		oldItems := ctyCollectionValues(old)
		newItems := ctyCollectionValues(new)

		if (len(oldItems) + len(newItems)) == 0 {
			// Nothing to do if both sets are empty
			return
		}

		allItems := make([]cty.Value, 0, len(oldItems)+len(newItems))
		allItems = append(allItems, oldItems...)
		allItems = append(allItems, newItems...)
		all := cty.SetVal(allItems)

		if blankBefore {
			p.buf.WriteRune('\n')
		}

		for it := all.ElementIterator(); it.Next(); {
			_, val := it.Element()
			var action diffs.Action
			var oldValue, newValue cty.Value
			switch {
			case !old.HasElement(val).True():
				action = diffs.Create
				oldValue = cty.NullVal(val.Type())
				newValue = val
			case !new.HasElement(val).True():
				action = diffs.Delete
				oldValue = val
				newValue = cty.NullVal(val.Type())
			default:
				action = diffs.NoAction
				oldValue = val
				newValue = val
			}
			path := append(path, cty.IndexStep{Key: val})
			p.writeNestedBlockDiff(name, nil, &blockS.Block, action, oldValue, newValue, indent, path)
		}

	case configschema.NestingMap:
		// TODO: Implement this, once helper/schema is actually able to
		// produce schemas containing nested map block types.
	}
}

func (p *blockBodyDiffPrinter) writeNestedBlockDiff(name string, label *string, blockS *configschema.Block, action diffs.Action, old, new cty.Value, indent int, path cty.Path) {
	p.buf.WriteString(strings.Repeat(" ", indent))
	p.writeActionSymbol(action)

	if label != nil {
		fmt.Fprintf(p.buf, "%s %q {", name, label)
	} else {
		fmt.Fprintf(p.buf, "%s {", name)
	}

	if action != diffs.NoAction && (p.pathForcesNewResource(path) || p.pathForcesNewResource(path[:len(path)-1])) {
		p.buf.WriteString(p.color.Color(forcesNewResourceCaption))
	}

	p.buf.WriteString("\n")

	p.writeBlockBodyDiff(blockS, old, new, indent+4, path)

	p.buf.WriteString(strings.Repeat(" ", indent+2))
	p.buf.WriteString("}\n")
}

func (p *blockBodyDiffPrinter) writeValue(val cty.Value, action diffs.Action, indent int) {
	if !val.IsKnown() {
		p.buf.WriteString("(known after apply)")
		return
	}
	if val.IsNull() {
		p.buf.WriteString("null")
		return
	}

	ty := val.Type()

	switch {
	case ty.IsPrimitiveType():
		switch ty {
		case cty.String:
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
		p.buf.WriteString("[\n")

		it := val.ElementIterator()
		for it.Next() {
			_, val := it.Element()
			p.buf.WriteString(strings.Repeat(" ", indent+2))
			p.writeActionSymbol(action)
			p.writeValue(val, action, indent+4)
			p.buf.WriteString(",\n")
		}

		p.buf.WriteString(strings.Repeat(" ", indent))
		p.buf.WriteString("]")
	case ty.IsMapType():
		p.buf.WriteString("{\n")

		it := val.ElementIterator()
		for it.Next() {
			key, val := it.Element()
			p.buf.WriteString(strings.Repeat(" ", indent+2))
			p.writeActionSymbol(action)
			p.writeValue(key, action, indent+4)
			p.buf.WriteString(" = ")
			p.writeValue(val, action, indent+4)
			p.buf.WriteString("\n")
		}

		p.buf.WriteString(strings.Repeat(" ", indent))
		p.buf.WriteString("}")
	case ty.IsObjectType():
		p.buf.WriteString("{\n")

		atys := ty.AttributeTypes()
		attrNames := make([]string, 0, len(atys))
		nameLen := 0
		for attrName := range atys {
			attrNames = append(attrNames, attrName)
			if len(attrName) > nameLen {
				nameLen = len(attrName)
			}
		}
		sort.Strings(attrNames)

		for _, attrName := range attrNames {
			val := val.GetAttr(attrName)
			p.buf.WriteString(strings.Repeat(" ", indent+2))
			p.writeActionSymbol(action)
			p.buf.WriteString(attrName)
			p.buf.WriteString(strings.Repeat(" ", nameLen-len(attrName)))
			p.buf.WriteString(" = ")
			p.writeValue(val, action, indent+4)
			p.buf.WriteString("\n")
		}

		p.buf.WriteString(strings.Repeat(" ", indent))
		p.buf.WriteString("}")
	}
}

func (p *blockBodyDiffPrinter) writeValueDiff(old, new cty.Value, indent int, path cty.Path) {
	ty := old.Type()

	// We have some specialized diff implementations for certain complex
	// values where it's useful to see a visualization of the diff of
	// the nested elements rather than just showing the entire old and
	// new values verbatim.
	// However, these specialized implementations can apply only if both
	// values are known and non-null.
	if old.IsKnown() && new.IsKnown() && !old.IsNull() && !new.IsNull() {
		switch {
		// TODO: list diffs using longest-common-subsequence matching algorithm
		// TODO: map diffs showing changes on a per-key basis
		// TODO: multi-line string diffs showing lines added/removed using longest-common-subsequence

		case ty == cty.String:
			// We only have special behavior for multi-line strings here
			oldS := old.AsString()
			newS := new.AsString()
			if strings.Index(oldS, "\n") < 0 && strings.Index(newS, "\n") < 0 {
				break
			}

			p.buf.WriteString("<<~EOT")
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

			diffLines := ctySequenceDiff(oldLines, newLines)
			for _, diffLine := range diffLines {
				line := diffLine.Value.AsString()
				switch diffLine.Action {
				case diffs.Create:
					p.buf.WriteString(strings.Repeat(" ", indent+2))
					p.buf.WriteString(p.color.Color("[green]+[reset] "))
					p.buf.WriteString(line)
					p.buf.WriteString("\n")
				case diffs.Delete:
					p.buf.WriteString(strings.Repeat(" ", indent+2))
					p.buf.WriteString(p.color.Color("[red]-[reset] "))
					p.buf.WriteString(line)
					p.buf.WriteString("\n")
				case diffs.NoAction:
					p.buf.WriteString(strings.Repeat(" ", indent+2))
					p.buf.WriteString(p.color.Color("  "))
					p.buf.WriteString(line)
					p.buf.WriteString("\n")
				default:
					// Should never happen since the above covers all
					// actions that ctySequenceDiff can return.
					p.buf.WriteString(strings.Repeat(" ", indent+2))
					p.buf.WriteString(p.color.Color("? "))
					p.buf.WriteString(line)
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
				if old.HasElement(val).False() {
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

			for it := all.ElementIterator(); it.Next(); {
				_, val := it.Element()

				p.buf.WriteString(strings.Repeat(" ", indent+2))

				var action diffs.Action
				switch {
				case added.HasElement(val).True():
					action = diffs.Create
				case removed.HasElement(val).True():
					action = diffs.Delete
				default:
					action = diffs.NoAction
				}

				p.writeActionSymbol(action)
				p.writeValue(val, action, indent+4)
				p.buf.WriteString(",\n")
			}

			p.buf.WriteString(strings.Repeat(" ", indent))
			p.buf.WriteString("]")
			return
		}
	}

	// In all other cases, we just show the new and old values as-is
	p.writeValue(old, diffs.Delete, indent)
	p.buf.WriteString(p.color.Color(" [yellow]->[reset] "))
	p.writeValue(new, diffs.Create, indent)
	if p.pathForcesNewResource(path) {
		p.buf.WriteString(p.color.Color(forcesNewResourceCaption))
	}
}

// writeActionSymbol writes a symbol to represent the given action, followed
// by a space.
//
// It only supports the actions that can be represented with a single character:
// Create, Delete, Update and NoAction.
func (p *blockBodyDiffPrinter) writeActionSymbol(action diffs.Action) {
	switch action {
	case diffs.Create:
		p.buf.WriteString(p.color.Color("[green]+[reset] "))
	case diffs.Delete:
		p.buf.WriteString(p.color.Color("[red]-[reset] "))
	case diffs.Update:
		p.buf.WriteString(p.color.Color("[yellow]~[reset] "))
	case diffs.NoAction:
		p.buf.WriteString("  ")
	default:
		// Should never happen
		p.buf.WriteString(p.color.Color("? "))
	}
}

func (p *blockBodyDiffPrinter) pathForcesNewResource(path cty.Path) bool {
	if p.action != diffs.Replace {
		// "forcedReplace" only applies when the instance is being replaced
		return false
	}
	return p.forcedReplace.Has(path)
}

func ctyGetAttrMaybeNull(val cty.Value, name string) cty.Value {
	if val.IsNull() {
		ty := val.Type().AttributeType(name)
		return cty.NullVal(ty)
	}

	return val.GetAttr(name)
}

func ctyCollectionValues(val cty.Value) []cty.Value {
	ret := make([]cty.Value, 0, val.LengthInt())
	for it := val.ElementIterator(); it.Next(); {
		_, value := it.Element()
		ret = append(ret, value)
	}
	return ret
}

func ctySequenceDiff(old, new []cty.Value) []ctyValueDiff {
	var ret []ctyValueDiff
	lcs := diffs.LongestCommonSubsequence(old, new)
	var oldI, newI, lcsI int
	for oldI < len(old) || newI < len(new) || lcsI < len(lcs) {
		for oldI < len(old) && (lcsI >= len(lcs) || !old[oldI].RawEquals(lcs[lcsI])) {
			ret = append(ret, ctyValueDiff{
				Action: diffs.Delete,
				Value:  old[oldI],
			})
			oldI++
		}
		for newI < len(new) && (lcsI >= len(lcs) || !new[newI].RawEquals(lcs[lcsI])) {
			ret = append(ret, ctyValueDiff{
				Action: diffs.Create,
				Value:  new[newI],
			})
			newI++
		}
		if lcsI < len(lcs) {
			ret = append(ret, ctyValueDiff{
				Action: diffs.NoAction,
				Value:  new[newI],
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

// ctyObjectSequenceDiff is a variant of ctySequenceDiff that only works for
// values of object types. Whereas ctySequenceDiff can only return Create
// and Delete actions, this function can additionally return Update actions
// heuristically based on similarity of objects in the lists, which must
// be greater than or equal to the caller-specified threshold.
//
// See ctyObjectSimilarity for details on what "similarity" means here.
func ctyObjectSequenceDiff(old, new []cty.Value, threshold float64) []*diffs.Change {
	var ret []*diffs.Change
	lcs := diffs.LongestCommonSubsequence(old, new)
	var oldI, newI, lcsI int
	for oldI < len(old) || newI < len(new) || lcsI < len(lcs) {
		for oldI < len(old) && (lcsI >= len(lcs) || !old[oldI].RawEquals(lcs[lcsI])) {
			if newI < len(new) {
				// See if the next "new" is similar enough to our "old" that
				// we'll treat this as an Update rather than a Delete/Create.
				similarity := ctyObjectSimilarity(old[oldI], new[newI])
				if similarity >= threshold {
					ret = append(ret, diffs.NewUpdate(old[oldI], new[newI]))
					oldI++
					newI++ // we also consume the next "new" in this case
					continue
				}
			}

			ret = append(ret, diffs.NewDelete(old[oldI]))
			oldI++
		}
		for newI < len(new) && (lcsI >= len(lcs) || !new[newI].RawEquals(lcs[lcsI])) {
			ret = append(ret, diffs.NewCreate(new[newI]))
			newI++
		}
		if lcsI < len(lcs) {
			ret = append(ret, diffs.NewNoAction(new[newI]))

			// All of our indexes advance together now, since the line
			// is common to all three sequences.
			lcsI++
			oldI++
			newI++
		}
	}
	return ret
}

// ctyObjectSimilarity returns a number between 0 and 1 that describes
// approximately how similar the two given values are, comparing in terms of
// how many of the corresponding attributes have the same value in both
// objects.
//
// This function expects the two values to have a similar set of attribute
// names, though doesn't mind if the two slightly differ since it will
// count missing attributes as differences.
//
// This function will panic if either of the given values is not an object.
func ctyObjectSimilarity(old, new cty.Value) float64 {
	oldType := old.Type()
	newType := new.Type()
	attrNames := make(map[string]struct{})
	for name := range oldType.AttributeTypes() {
		attrNames[name] = struct{}{}
	}
	for name := range newType.AttributeTypes() {
		attrNames[name] = struct{}{}
	}

	matches := 0

	for name := range attrNames {
		if !oldType.HasAttribute(name) {
			continue
		}
		if !newType.HasAttribute(name) {
			continue
		}
		eq := old.GetAttr(name).Equals(new.GetAttr(name))
		if !eq.IsKnown() {
			continue
		}
		if eq.True() {
			matches++
		}
	}

	return float64(matches) / float64(len(attrNames))
}

func ctyEqualWithUnknown(old, new cty.Value) bool {
	if !old.IsKnown() || !new.IsKnown() {
		return false
	}
	return old.Equals(new).True()
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
