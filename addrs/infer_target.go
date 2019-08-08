package addrs

import (
	"strconv"

	"github.com/hashicorp/terraform/tfdiags"
)

// InferAbsResourceInstanceStr Attempts to infer a resource instance address with invalid syntax
// The current implementation will only quote any invalid resource index values
// as shell environments can easily strip quotes (e.g. type.name["quoted"]
// becoming type.name[quoted]). This function is currently only intended for
// CLI command handling.
func InferAbsResourceInstanceStr(originalStr string) (AbsResourceInstance, tfdiags.Diagnostics) {
	addr, addrDiags := ParseAbsResourceInstanceStr(originalStr)

	if !addrDiags.HasErrors() {
		return addr, addrDiags
	}

	previousStr := originalStr
	currentStr := originalStr

	for {
		for _, diag := range addrDiags {
			if diag.Description().Summary != "Index value required" {
				continue
			}

			indexValueStart := diag.Source().Subject.Start.Column - 1
			indexValueEnd := diag.Source().Subject.End.Column - 1
			indexValue := currentStr[indexValueStart:indexValueEnd]

			// If the index value is numeric, skip handling
			if _, err := strconv.Atoi(indexValue); err == nil {
				break
			}

			// Add quotes around index value
			currentStr = currentStr[:indexValueStart] + `"` + indexValue + `"` + currentStr[indexValueEnd:]

			addr, addrDiags = ParseAbsResourceInstanceStr(currentStr)

			break
		}

		if !addrDiags.HasErrors() {
			break
		}

		// Prevent infinite looping on unresolveable conditions
		if currentStr == previousStr {
			break
		}

		previousStr = currentStr
	}

	return addr, addrDiags
}
