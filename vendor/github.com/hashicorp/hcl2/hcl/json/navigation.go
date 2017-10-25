package json

import (
	"strings"
)

type navigation struct {
	root *objectVal
}

// Implementation of zcled.ContextString
func (n navigation) ContextString(offset int) string {
	steps := navigationStepsRev(n.root, offset)
	if steps == nil {
		return ""
	}

	// We built our slice backwards, so we'll reverse it in-place now.
	half := len(steps) / 2 // integer division
	for i := 0; i < half; i++ {
		steps[i], steps[len(steps)-1-i] = steps[len(steps)-1-i], steps[i]
	}

	return strings.Join(steps, ".")
}

func navigationStepsRev(obj *objectVal, offset int) []string {
	// Do any of our properties have an object that contains the target
	// offset?
	for k, attr := range obj.Attrs {
		ov, ok := attr.Value.(*objectVal)
		if !ok {
			continue
		}

		if ov.SrcRange.ContainsOffset(offset) {
			return append(navigationStepsRev(ov, offset), k)
		}
	}
	return nil
}
