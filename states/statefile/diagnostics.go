package statefile

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform/tfdiags"
)

const invalidFormat = "Invalid state file format"

// jsonUnmarshalDiags is a helper that translates errors returned from
// json.Unmarshal into hopefully-more-helpful diagnostics messages.
func jsonUnmarshalDiags(err error) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	if err == nil {
		return diags
	}

	switch tErr := err.(type) {
	case *json.SyntaxError:
		// We've usually already successfully parsed a source file as JSON at
		// least once before we'd use jsonUnmarshalDiags with it (to sniff
		// the version number) so this particular error should not appear much
		// in practice.
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			invalidFormat,
			fmt.Sprintf("The state file could not be parsed as JSON: syntax error at byte offset %d.", tErr.Offset),
		))
	case *json.UnmarshalTypeError:
		// This is likely to be the most common area, describing a
		// non-conformance between the file and the expected file format
		// at a semantic level.
		if tErr.Field != "" {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				invalidFormat,
				fmt.Sprintf("The state file field %q has invalid value %s", tErr.Field, tErr.Value),
			))
			break
		} else {
			// Without a field name, we can't really say anything helpful.
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				invalidFormat,
				"The state file does not conform to the expected JSON data structure.",
			))
		}
	default:
		// Fallback for all other types of errors. This can happen only for
		// custom UnmarshalJSON implementations, so should be encountered
		// only rarely.
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			invalidFormat,
			fmt.Sprintf("The state file does not conform to the expected JSON data structure: %s.", err.Error()),
		))
	}

	return diags
}
