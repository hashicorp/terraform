package change

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/plans"
)

func Object(attributes map[string]Change) Renderer {
	return &objectRenderer{
		attributes:         attributes,
		overrideNullSuffix: true,
	}
}

func NestedObject(attributes map[string]Change) Renderer {
	return &objectRenderer{
		attributes:         attributes,
		overrideNullSuffix: false,
	}
}

type objectRenderer struct {
	NoWarningsRenderer

	attributes         map[string]Change
	overrideNullSuffix bool
}

func (renderer objectRenderer) Render(change Change, indent int, opts RenderOpts) string {
	if len(renderer.attributes) == 0 {
		return fmt.Sprintf("{}%s%s", change.nullSuffix(opts.overrideNullSuffix), change.forcesReplacement())
	}

	attributeOpts := opts.Clone()
	attributeOpts.overrideNullSuffix = renderer.overrideNullSuffix

	// We need to keep track of our keys in two ways. The first is the order in
	// which we will display them. The second is a mapping to their safely
	// escaped equivalent.

	maximumKeyLen := 0
	var keys []string
	escapedKeys := make(map[string]string)
	for key := range renderer.attributes {
		keys = append(keys, key)
		escapedKey := change.ensureValidAttributeName(key)
		escapedKeys[key] = escapedKey
		if maximumKeyLen < len(escapedKey) {
			maximumKeyLen = len(escapedKey)
		}
	}
	sort.Strings(keys)

	unchangedAttributes := 0
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("{%s\n", change.forcesReplacement()))
	for _, key := range keys {
		attribute := renderer.attributes[key]

		if attribute.action == plans.NoOp && !opts.showUnchangedChildren {
			// Don't render NoOp operations when we are compact display.
			unchangedAttributes++
			continue
		}

		for _, warning := range attribute.Warnings(indent + 1) {
			buf.WriteString(fmt.Sprintf("%s%s\n", change.indent(indent+1), warning))
		}
		buf.WriteString(fmt.Sprintf("%s%s %-*s = %s\n", change.indent(indent+1), format.DiffActionSymbol(attribute.action), maximumKeyLen, escapedKeys[key], attribute.Render(indent+1, attributeOpts)))
	}

	if unchangedAttributes > 0 {
		buf.WriteString(fmt.Sprintf("%s%s %s\n", change.indent(indent+1), format.DiffActionSymbol(plans.NoOp), change.unchanged("attribute", unchangedAttributes)))
	}

	buf.WriteString(fmt.Sprintf("%s%s }%s", change.indent(indent), format.DiffActionSymbol(plans.NoOp), change.nullSuffix(opts.overrideNullSuffix)))
	return buf.String()
}
