package change

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/plans"
)

func Object(attributes map[string]Change) Renderer {
	maximumKeyLen := 0
	for key := range attributes {
		if maximumKeyLen < len(key) {
			maximumKeyLen = len(key)
		}
	}

	return &objectRenderer{
		attributes:         attributes,
		maximumKeyLen:      maximumKeyLen,
		overrideNullSuffix: true,
	}
}

func NestedObject(attributes map[string]Change) Renderer {
	maximumKeyLen := 0
	for key := range attributes {
		if maximumKeyLen < len(key) {
			maximumKeyLen = len(key)
		}
	}

	return &objectRenderer{
		attributes:         attributes,
		maximumKeyLen:      maximumKeyLen,
		overrideNullSuffix: false,
	}
}

type objectRenderer struct {
	NoWarningsRenderer

	attributes         map[string]Change
	maximumKeyLen      int
	overrideNullSuffix bool
}

func (renderer objectRenderer) Render(change Change, indent int, opts RenderOpts) string {
	if len(renderer.attributes) == 0 {
		return fmt.Sprintf("{}%s%s", change.nullSuffix(opts.overrideNullSuffix), change.forcesReplacement())
	}

	attributeOpts := opts.Clone()
	attributeOpts.overrideNullSuffix = renderer.overrideNullSuffix

	var keys []string
	for key := range renderer.attributes {
		keys = append(keys, key)
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
		buf.WriteString(fmt.Sprintf("%s%s %-*s = %s\n", change.indent(indent+1), format.DiffActionSymbol(attribute.action), renderer.maximumKeyLen, key, attribute.Render(indent+1, attributeOpts)))
	}

	if unchangedAttributes > 0 {
		buf.WriteString(fmt.Sprintf("%s%s %s\n", change.indent(indent+1), change.emptySymbol(), change.unchanged("attribute", unchangedAttributes)))
	}

	buf.WriteString(fmt.Sprintf("%s%s }%s", change.indent(indent), change.emptySymbol(), change.nullSuffix(opts.overrideNullSuffix)))
	return buf.String()
}
