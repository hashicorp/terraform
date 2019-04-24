package format

import (
	"bufio"
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/mitchellh/colorstring"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/plans/objchange"
	"github.com/hashicorp/terraform/states"
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
	change *plans.ResourceInstanceChangeSrc,
	tainted bool,
	schema *configschema.Block,
	color *colorstring.Colorize,
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
		buf.WriteString(color.Color(fmt.Sprintf("[bold]  # %s[reset] will be created", dispAddr)))
	case plans.Read:
		buf.WriteString(color.Color(fmt.Sprintf("[bold]  # %s[reset] will be read during apply\n  # (config refers to values not yet known)", dispAddr)))
	case plans.Update:
		buf.WriteString(color.Color(fmt.Sprintf("[bold]  # %s[reset] will be updated in-place", dispAddr)))
	case plans.CreateThenDelete, plans.DeleteThenCreate:
		if tainted {
			buf.WriteString(color.Color(fmt.Sprintf("[bold]  # %s[reset] is tainted, so must be [bold][red]replaced", dispAddr)))
		} else {
			buf.WriteString(color.Color(fmt.Sprintf("[bold]  # %s[reset] must be [bold][red]replaced", dispAddr)))
		}
	case plans.Delete:
		buf.WriteString(color.Color(fmt.Sprintf("[bold]  # %s[reset] will be [bold][red]destroyed", dispAddr)))
	default:
		// should never happen, since the above is exhaustive
		buf.WriteString(fmt.Sprintf("%s has an action the plan renderer doesn't support (this is a bug)", dispAddr))
	}
	buf.WriteString(color.Color("[reset]\n"))

	switch change.Action {
	case plans.Create:
		buf.WriteString(color.Color("[green]  +[reset] "))
	case plans.Read:
		buf.WriteString(color.Color("[cyan] <=[reset] "))
	case plans.Update:
		buf.WriteString(color.Color("[yellow]  ~[reset] "))
	case plans.DeleteThenCreate:
		buf.WriteString(color.Color("[red]-[reset]/[green]+[reset] "))
	case plans.CreateThenDelete:
		buf.WriteString(color.Color("[green]+[reset]/[red]-[reset] "))
	case plans.Delete:
		buf.WriteString(color.Color("[red]  -[reset] "))
	default:
		buf.WriteString(color.Color("??? "))
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
			"data %q %q ",
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

	changeV, err := change.Decode(schema.ImpliedType())
	if err != nil {
		// Should never happen in here, since we've already been through
		// loads of layers of encode/decode of the planned changes before now.
		panic(fmt.Sprintf("failed to decode plan for %s while rendering diff: %s", addr, err))
	}

	// We currently have an opt-out that permits the legacy SDK to return values
	// that defy our usual conventions around handling of nesting blocks. To
	// avoid the rendering code from needing to handle all of these, we'll
	// normalize first.
	// (Ideally we'd do this as part of the SDK opt-out implementation in core,
	// but we've added it here for now to reduce risk of unexpected impacts
	// on other code in core.)
	changeV.Change.Before = objchange.NormalizeObjectFromLegacySDK(changeV.Change.Before, schema)
	changeV.Change.After = objchange.NormalizeObjectFromLegacySDK(changeV.Change.After, schema)

	bodyWritten := p.writeBlockBodyDiff(schema, changeV.Before, changeV.After, 6, path)
	if bodyWritten {
		buf.WriteString("\n")
		buf.WriteString(strings.Repeat(" ", 4))
	}
	buf.WriteString("}\n")

	return buf.String()
}

type blockBodyDiffPrinter struct {
	buf             *bytes.Buffer
	color           *colorstring.Colorize
	action          plans.Action
	requiredReplace cty.PathSet
}

const forcesNewResourceCaption = " [red]# forces replacement[reset]"

// writeBlockBodyDiff writes attribute or block differences
// and returns true if any differences were found and written
func (p *blockBodyDiffPrinter) writeBlockBodyDiff(schema *configschema.Block, old, new cty.Value, indent int, path cty.Path) bool {
	path = ctyEnsurePathCapacity(path, 1)

	bodyWritten := false
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

			bodyWritten = true
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

			bodyWritten = true
			p.writeNestedBlockDiffs(name, blockS, oldVal, newVal, blankBeforeBlocks, indent, path)

			// Always include a blank for any subsequent block types.
			blankBeforeBlocks = true
		}
	}

	return bodyWritten
}

func (p *blockBodyDiffPrinter) writeAttrDiff(name string, attrS *configschema.Attribute, old, new cty.Value, nameLen, indent int, path cty.Path) {
	path = append(path, cty.GetAttrStep{Name: name})
	p.buf.WriteString("\n")
	p.buf.WriteString(strings.Repeat(" ", indent))
	showJustNew := false
	var action plans.Action
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

		if blankBefore {
			p.buf.WriteRune('\n')
		}
		p.writeNestedBlockDiff(name, nil, &blockS.Block, action, old, new, indent, path)
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

		if blankBefore && (len(oldItems) > 0 || len(newItems) > 0) {
			p.buf.WriteRune('\n')
		}

		for i := 0; i < commonLen; i++ {
			path := append(path, cty.IndexStep{Key: cty.NumberIntVal(int64(i))})
			oldItem := oldItems[i]
			newItem := newItems[i]
			action := plans.Update
			if oldItem.RawEquals(newItem) {
				action = plans.NoOp
			}
			p.writeNestedBlockDiff(name, nil, &blockS.Block, action, oldItem, newItem, indent, path)
		}
		for i := commonLen; i < len(oldItems); i++ {
			path := append(path, cty.IndexStep{Key: cty.NumberIntVal(int64(i))})
			oldItem := oldItems[i]
			newItem := cty.NullVal(oldItem.Type())
			p.writeNestedBlockDiff(name, nil, &blockS.Block, plans.Delete, oldItem, newItem, indent, path)
		}
		for i := commonLen; i < len(newItems); i++ {
			path := append(path, cty.IndexStep{Key: cty.NumberIntVal(int64(i))})
			newItem := newItems[i]
			oldItem := cty.NullVal(newItem.Type())
			p.writeNestedBlockDiff(name, nil, &blockS.Block, plans.Create, oldItem, newItem, indent, path)
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
			p.writeNestedBlockDiff(name, nil, &blockS.Block, action, oldValue, newValue, indent, path)
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
			return
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

		if blankBefore {
			p.buf.WriteRune('\n')
		}

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
			p.writeNestedBlockDiff(name, &k, &blockS.Block, action, oldValue, newValue, indent, path)
		}
	}
}

func (p *blockBodyDiffPrinter) writeNestedBlockDiff(name string, label *string, blockS *configschema.Block, action plans.Action, old, new cty.Value, indent int, path cty.Path) {
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

	bodyWritten := p.writeBlockBodyDiff(blockS, old, new, indent+4, path)
	if bodyWritten {
		p.buf.WriteString("\n")
		p.buf.WriteString(strings.Repeat(" ", indent+2))
	}
	p.buf.WriteString("}")
}

func (p *blockBodyDiffPrinter) writeValue(val cty.Value, action plans.Action, indent int) {
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
				if err == nil && !ty.IsPrimitiveType() && val.AsString() != "null" {
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

			p.buf.WriteString("\n")
			p.buf.WriteString(strings.Repeat(" ", indent+2))
			p.writeActionSymbol(action)
			p.buf.WriteString(attrName)
			p.buf.WriteString(strings.Repeat(" ", nameLen-len(attrName)))
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
							// if they differ only in insigificant whitespace
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

			for it := all.ElementIterator(); it.Next(); {
				_, val := it.Element()

				p.buf.WriteString(strings.Repeat(" ", indent+2))

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

				p.writeActionSymbol(action)
				p.writeValue(val, action, indent+4)
				p.buf.WriteString(",\n")
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
			for _, elemDiff := range elemDiffs {
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

			lastK := ""
			for i, k := range allKeys {
				if i > 0 && lastK == k {
					continue // skip duplicates (list is sorted)
				}
				lastK = k

				p.buf.WriteString(strings.Repeat(" ", indent+2))
				kV := cty.StringVal(k)
				var action plans.Action
				if old.HasIndex(kV).False() {
					action = plans.Create
				} else if new.HasIndex(kV).False() {
					action = plans.Delete
				} else if eqV := old.Index(kV).Equals(new.Index(kV)); eqV.IsKnown() && eqV.True() {
					action = plans.NoOp
				} else {
					action = plans.Update
				}

				path := append(path, cty.IndexStep{Key: kV})

				p.writeActionSymbol(action)
				p.writeValue(kV, action, indent+4)
				p.buf.WriteString(strings.Repeat(" ", keyLen-len(k)))
				p.buf.WriteString(" = ")
				switch action {
				case plans.Create, plans.NoOp:
					v := new.Index(kV)
					p.writeValue(v, action, indent+4)
				case plans.Delete:
					oldV := old.Index(kV)
					newV := cty.NullVal(oldV.Type())
					p.writeValueDiff(oldV, newV, indent+4, path)
				default:
					oldV := old.Index(kV)
					newV := new.Index(kV)
					p.writeValueDiff(oldV, newV, indent+4, path)
				}

				p.buf.WriteByte('\n')
			}

			p.buf.WriteString(strings.Repeat(" ", indent))
			p.buf.WriteString("}")
			return
		case ty.IsObjectType():
			p.buf.WriteString("{")
			p.buf.WriteString("\n")

			forcesNewResource := p.pathForcesNewResource(path)

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

			lastK := ""
			for i, k := range allKeys {
				if i > 0 && lastK == k {
					continue // skip duplicates (list is sorted)
				}
				lastK = k

				p.buf.WriteString(strings.Repeat(" ", indent+2))
				kV := k
				var action plans.Action
				if !old.Type().HasAttribute(kV) {
					action = plans.Create
				} else if !new.Type().HasAttribute(kV) {
					action = plans.Delete
				} else if eqV := old.GetAttr(kV).Equals(new.GetAttr(kV)); eqV.IsKnown() && eqV.True() {
					action = plans.NoOp
				} else {
					action = plans.Update
				}

				path := append(path, cty.GetAttrStep{Name: kV})

				p.writeActionSymbol(action)
				p.buf.WriteString(k)
				p.buf.WriteString(strings.Repeat(" ", keyLen-len(k)))
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

func (p *blockBodyDiffPrinter) pathForcesNewResource(path cty.Path) bool {
	if !p.action.IsReplace() {
		// "requiredReplace" only applies when the instance is being replaced
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
	if ctyEmptyString(attrValue) {
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
		for oldI < len(old) && (lcsI >= len(lcs) || !old[oldI].RawEquals(lcs[lcsI])) {
			isObjectDiff := old[oldI].Type().IsObjectType() && (newI >= len(new) || new[newI].Type().IsObjectType())
			if isObjectDiff && newI < len(new) {
				ret = append(ret, &plans.Change{
					Action: plans.Update,
					Before: old[oldI],
					After:  new[newI],
				})
				oldI++
				newI++ // we also consume the next "new" in this case
				continue
			}

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
		if lcsI < len(lcs) {
			ret = append(ret, &plans.Change{
				Action: plans.NoOp,
				Before: new[newI],
				After:  new[newI],
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

func ctyEqualWithUnknown(old, new cty.Value) bool {
	if !old.IsWhollyKnown() || !new.IsWhollyKnown() {
		return false
	}
	return old.Equals(new).True()
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
