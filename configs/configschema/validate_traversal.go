package configschema

import (
	"fmt"
	"sort"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/helper/didyoumean"
	"github.com/hashicorp/terraform/tfdiags"
)

// StaticValidateTraversal checks whether the given traversal (which must be
// relative) refers to a construct in the receiving schema, returning error
// diagnostics if any problems are found.
//
// This method is "optimistic" in that it will not return errors for possible
// problems that cannot be detected statically. It is possible that an
// traversal which passed static validation will still fail when evaluated.
func (b *Block) StaticValidateTraversal(traversal hcl.Traversal) tfdiags.Diagnostics {
	if !traversal.IsRelative() {
		panic("StaticValidateTraversal on absolute traversal")
	}
	if len(traversal) == 0 {
		return nil
	}

	var diags tfdiags.Diagnostics

	next := traversal[0]
	after := traversal[1:]

	var name string
	switch step := next.(type) {
	case hcl.TraverseAttr:
		name = step.Name
	case hcl.TraverseIndex:
		// No other traversal step types are allowed directly at a block.
		// If it looks like the user was trying to use index syntax to
		// access an attribute then we'll produce a specialized message.
		key := step.Key
		if key.Type() == cty.String && key.IsKnown() && !key.IsNull() {
			maybeName := key.AsString()
			if hclsyntax.ValidIdentifier(maybeName) {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  `Invalid index operation`,
					Detail:   fmt.Sprintf(`Only attribute access is allowed here. Did you mean to access attribute %q using the dot operator?`, maybeName),
					Subject:  &step.SrcRange,
				})
				return diags
			}
		}
		// If it looks like some other kind of index then we'll use a generic error.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Invalid index operation`,
			Detail:   `Only attribute access is allowed here, using the dot operator.`,
			Subject:  &step.SrcRange,
		})
		return diags
	default:
		// No other traversal types should appear in a normal valid traversal,
		// but we'll handle this with a generic error anyway to be robust.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Invalid operation`,
			Detail:   `Only attribute access is allowed here, using the dot operator.`,
			Subject:  next.SourceRange().Ptr(),
		})
		return diags
	}

	if attrS, exists := b.Attributes[name]; exists {
		// For attribute validation we will just apply the rest of the
		// traversal to an unknown value of the attribute type and pass
		// through HCL's own errors, since we don't want to replicate all of
		// HCL's type checking rules here.
		val := cty.UnknownVal(attrS.Type)
		_, hclDiags := after.TraverseRel(val)
		diags = diags.Append(hclDiags)
		return diags
	}

	if blockS, exists := b.BlockTypes[name]; exists {
		moreDiags := blockS.staticValidateTraversal(name, after)
		diags = diags.Append(moreDiags)
		return diags
	}

	// If we get here then the name isn't valid at all. We'll collect up
	// all of the names that _are_ valid to use as suggestions.
	var suggestions []string
	for name := range b.Attributes {
		suggestions = append(suggestions, name)
	}
	for name := range b.BlockTypes {
		suggestions = append(suggestions, name)
	}
	sort.Strings(suggestions)
	suggestion := didyoumean.NameSuggestion(name, suggestions)
	if suggestion != "" {
		suggestion = fmt.Sprintf(" Did you mean %q?", suggestion)
	}
	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  `Unsupported attribute`,
		Detail:   fmt.Sprintf(`This object has no argument, nested block, or exported attribute named %q.%s`, name, suggestion),
		Subject:  next.SourceRange().Ptr(),
	})

	return diags
}

func (b *NestedBlock) staticValidateTraversal(typeName string, traversal hcl.Traversal) tfdiags.Diagnostics {
	if b.Nesting == NestingSingle || b.Nesting == NestingGroup {
		// Single blocks are easy: just pass right through.
		return b.Block.StaticValidateTraversal(traversal)
	}

	if len(traversal) == 0 {
		// It's always valid to access a nested block's attribute directly.
		return nil
	}

	var diags tfdiags.Diagnostics
	next := traversal[0]
	after := traversal[1:]

	switch b.Nesting {

	case NestingSet:
		// Can't traverse into a set at all, since it does not have any keys
		// to index with.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Cannot index a set value`,
			Detail:   fmt.Sprintf(`Block type %q is represented by a set of objects, and set elements do not have addressable keys. To find elements matching specific criteria, use a "for" expression with an "if" clause.`, typeName),
			Subject:  next.SourceRange().Ptr(),
		})
		return diags

	case NestingList:
		if _, ok := next.(hcl.TraverseIndex); ok {
			moreDiags := b.Block.StaticValidateTraversal(after)
			diags = diags.Append(moreDiags)
		} else {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  `Invalid operation`,
				Detail:   fmt.Sprintf(`Block type %q is represented by a list of objects, so it must be indexed using a numeric key, like .%s[0].`, typeName, typeName),
				Subject:  next.SourceRange().Ptr(),
			})
		}
		return diags

	case NestingMap:
		// Both attribute and index steps are valid for maps, so we'll just
		// pass through here and let normal evaluation catch an
		// incorrectly-typed index key later, if present.
		moreDiags := b.Block.StaticValidateTraversal(after)
		diags = diags.Append(moreDiags)
		return diags

	default:
		// Invalid nesting type is just ignored. It's checked by
		// InternalValidate. (Note that we handled NestingSingle separately
		// back at the start of this function.)
		return nil
	}
}
