package json

import (
	"fmt"
	"strings"
)

type navigation struct {
	root node
}

// Implementation of hcled.ContextString
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

	ret := strings.Join(steps, "")
	if len(ret) > 0 && ret[0] == '.' {
		ret = ret[1:]
	}
	return ret
}

func navigationStepsRev(v node, offset int) []string {
	switch tv := v.(type) {
	case *objectVal:
		// Do any of our properties have an object that contains the target
		// offset?
		for _, attr := range tv.Attrs {
			k := attr.Name
			av := attr.Value

			switch av.(type) {
			case *objectVal, *arrayVal:
				// okay
			default:
				continue
			}

			if av.Range().ContainsOffset(offset) {
				return append(navigationStepsRev(av, offset), "."+k)
			}
		}
	case *arrayVal:
		// Do any of our elements contain the target offset?
		for i, elem := range tv.Values {

			switch elem.(type) {
			case *objectVal, *arrayVal:
				// okay
			default:
				continue
			}

			if elem.Range().ContainsOffset(offset) {
				return append(navigationStepsRev(elem, offset), fmt.Sprintf("[%d]", i))
			}
		}
	}

	return nil
}
