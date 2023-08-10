package s3

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

const (
	multiRegionKeyIdPattern = `mrk-[a-f0-9]{32}`
	uuidRegexPattern        = `[a-f0-9]{8}-[a-f0-9]{4}-[1-5][a-f0-9]{3}-[ab89][a-f0-9]{3}-[a-f0-9]{12}`
)

func validateKMSKey(path cty.Path, s string) (diags tfdiags.Diagnostics) {
	if arn.IsARN(s) {
		return validateKMSKeyARN(path, s)
	}
	return validateKMSKeyID(path, s)
}

func validateKMSKeyID(path cty.Path, s string) (diags tfdiags.Diagnostics) {
	keyIdRegex := regexp.MustCompile(`^` + uuidRegexPattern + `|` + multiRegionKeyIdPattern + `$`)
	if !keyIdRegex.MatchString(s) {
		diags = diags.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			"Invalid KMS Key ID",
			fmt.Sprintf("Value must be a valid KMS Key ID, got %q", s),
			path,
		))
		return diags
	}

	return diags
}

func validateKMSKeyARN(path cty.Path, s string) (diags tfdiags.Diagnostics) {
	parsedARN, err := arn.Parse(s)
	if err != nil {
		diags = diags.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			"Invalid KMS Key ARN",
			fmt.Sprintf("Value must be a valid KMS Key ARN, got %q", s),
			path,
		))
		return diags
	}

	if !isKeyARN(parsedARN) {
		diags = diags.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			"Invalid KMS Key ARN",
			fmt.Sprintf("Value must be a valid KMS Key ARN, got %q", s),
			path,
		))
		return diags
	}

	return diags
}

func isKeyARN(arn arn.ARN) bool {
	return keyIdFromARNResource(arn.Resource) != ""
}

func keyIdFromARNResource(s string) string {
	keyIdResourceRegex := regexp.MustCompile(`^key/(` + uuidRegexPattern + `|` + multiRegionKeyIdPattern + `)$`)
	matches := keyIdResourceRegex.FindStringSubmatch(s)
	if matches == nil || len(matches) != 2 {
		return ""
	}

	return matches[1]
}

type stringValidator func(val string, path cty.Path, diags *tfdiags.Diagnostics)

func validateStringNotEmpty(val string, path cty.Path, diags *tfdiags.Diagnostics) {
	val = strings.TrimSpace(val)
	if len(val) == 0 {
		*diags = diags.Append(attributeErrDiag(
			"Invalid Value",
			"The value cannot be empty or all whitespace",
			path,
		))
	}
}

func validateStringLenBetween(min, max int) stringValidator {
	return func(val string, path cty.Path, diags *tfdiags.Diagnostics) {
		if l := len(val); l < min || l > max {
			*diags = diags.Append(attributeErrDiag(
				"Invalid Value Length",
				fmt.Sprintf("Length must be between %d and %d, had %d", min, max, l),
				path,
			))
		}
	}
}

func validateStringMatches(re *regexp.Regexp, description string) stringValidator {
	return func(val string, path cty.Path, diags *tfdiags.Diagnostics) {
		if !re.MatchString(val) {
			*diags = diags.Append(attributeErrDiag(
				"Invalid Value",
				description,
				path,
			))
		}
	}
}

// S3 will strip leading slashes from an object, so while this will
// technically be accepted by S3, it will break our workspace hierarchy.
// S3 will recognize objects with a trailing slash as a directory
// so they should not be valid keys
func validateStringS3Path(val string, path cty.Path, diags *tfdiags.Diagnostics) {
	if strings.HasPrefix(val, "/") || strings.HasSuffix(val, "/") {
		*diags = diags.Append(attributeErrDiag(
			"Invalid Value",
			`The value must not start or end with "/"`,
			path,
		))
	}
}

func validateARN(validators ...arnValidator) stringValidator {
	return func(val string, path cty.Path, diags *tfdiags.Diagnostics) {
		parsedARN, err := arn.Parse(val)
		if err != nil {
			*diags = diags.Append(attributeErrDiag(
				"Invalid ARN",
				fmt.Sprintf("The value %q cannot be parsed as an ARN: %s", val, err),
				path,
			))
			return
		}

		for _, validator := range validators {
			validator(parsedARN, path, diags)
		}
	}
}

// Copied from `ValidIAMPolicyJSON` (https://github.com/hashicorp/terraform-provider-aws/blob/ffd1c8a006dcd5a6b58a643df9cc147acb5b7a53/internal/verify/validate.go#L154)
func validateIAMPolicyDocument(val string, path cty.Path, diags *tfdiags.Diagnostics) {
	// IAM Policy documents need to be valid JSON, and pass legacy parsing
	val = strings.TrimSpace(val)
	if first := val[:1]; first != "{" {
		switch val[:1] {
		case `"`:
			// There are some common mistakes that lead to strings appearing
			// here instead of objects, so we'll try some heuristics to
			// check for those so we might give more actionable feedback in
			// these situations.
			var content string
			var innerContent any
			if err := json.Unmarshal([]byte(val), &content); err == nil {
				if strings.HasSuffix(content, ".json") {
					*diags = diags.Append(attributeErrDiag(
						"Invalid IAM Policy Document",
						fmt.Sprintf(`Expected a JSON object describing the policy, had a JSON-encoded string.

The string %q looks like a filename, please pass the contents of the file instead of the filename.`,
							content,
						),
						path,
					))
					return
				} else if err := json.Unmarshal([]byte(content), &innerContent); err == nil {
					// hint = " (have you double-encoded your JSON data?)"
					*diags = diags.Append(attributeErrDiag(
						"Invalid IAM Policy Document",
						`Expected a JSON object describing the policy, had a JSON-encoded string.

The string content was valid JSON, your policy document may have been double-encoded.`,
						path,
					))
					return
				}
			}
			*diags = diags.Append(attributeErrDiag(
				"Invalid IAM Policy Document",
				"Expected a JSON object describing the policy, had a JSON-encoded string.",
				path,
			))
		default:
			// Generic error for if we didn't find something more specific to say.
			*diags = diags.Append(attributeErrDiag(
				"Invalid IAM Policy Document",
				"Expected a JSON object describing the policy",
				path,
			))
		}
	} else {
		var j any
		if err := json.Unmarshal([]byte(val), &j); err != nil {
			errStr := err.Error()
			var jsonErr *json.SyntaxError
			if errors.As(err, &jsonErr) {
				errStr += fmt.Sprintf(", at byte offset %d", jsonErr.Offset)
			}
			*diags = diags.Append(attributeErrDiag(
				"Invalid JSON Document",
				fmt.Sprintf("The JSON document contains an error: %s", errStr),
				path,
			))
		}
	}
}

// Using a val of `cty.ValueSet` would be better here, but we can't get an ElementIterator from a ValueSet
type setValidator func(val cty.Value, path cty.Path, diags *tfdiags.Diagnostics)

func validateSetStringElements(validators ...stringValidator) setValidator {
	return func(val cty.Value, path cty.Path, diags *tfdiags.Diagnostics) {
		typ := val.Type()
		if eltTyp := typ.ElementType(); eltTyp != cty.String {
			*diags = diags.Append(attributeErrDiag(
				"Internal Error",
				fmt.Sprintf(`Expected type to be %s, got: %s`, cty.Set(cty.String).FriendlyName(), val.Type().FriendlyName()),
				path,
			))
			return
		}

		eltPath := make(cty.Path, len(path)+1)
		copy(eltPath, path)
		idxIdx := len(path)

		iter := val.ElementIterator()
		for iter.Next() {
			idx, elt := iter.Element()

			eltPath[idxIdx] = cty.IndexStep{Key: idx}

			for _, validator := range validators {
				validator(elt.AsString(), eltPath, diags)
			}
		}
	}
}

type arnValidator func(val arn.ARN, path cty.Path, diags *tfdiags.Diagnostics)

func validateIAMRoleARN(val arn.ARN, path cty.Path, diags *tfdiags.Diagnostics) {
	if !strings.HasPrefix(val.Resource, "role/") {
		*diags = diags.Append(attributeErrDiag(
			"Invalid IAM Role ARN",
			fmt.Sprintf("Value must be a valid IAM Role ARN, got %q", val),
			path,
		))
	}
}

func validateIAMPolicyARN(val arn.ARN, path cty.Path, diags *tfdiags.Diagnostics) {
	if !strings.HasPrefix(val.Resource, "policy/") {
		*diags = diags.Append(attributeErrDiag(
			"Invalid IAM Policy ARN",
			fmt.Sprintf("Value must be a valid IAM Policy ARN, got %q", val),
			path,
		))
	}
}

func validateDuration(validators ...durationValidator) stringValidator {
	return func(val string, path cty.Path, diags *tfdiags.Diagnostics) {
		duration, err := time.ParseDuration(val)
		if err != nil {
			*diags = diags.Append(attributeErrDiag(
				"Invalid Duration",
				fmt.Sprintf("The value %q cannot be parsed as a duration: %s", val, err),
				path,
			))
			return
		}

		for _, validator := range validators {
			validator(duration, path, diags)
		}
	}
}

type durationValidator func(val time.Duration, path cty.Path, diags *tfdiags.Diagnostics)

func validateDurationBetween(min, max time.Duration) durationValidator {
	return func(val time.Duration, path cty.Path, diags *tfdiags.Diagnostics) {
		if val < min || val > max {
			*diags = diags.Append(attributeErrDiag(
				"Invalid Duration",
				fmt.Sprintf("Duration must be between %s and %s, had %s", min, max, val),
				path,
			))
		}
	}
}

func attributeErrDiag(summary, detail string, attrPath cty.Path) tfdiags.Diagnostic {
	return tfdiags.AttributeValue(tfdiags.Error, summary, detail, attrPath.Copy())
}

func wholeBodyErrDiag(summary, detail string) tfdiags.Diagnostic {
	return tfdiags.WholeContainingBody(tfdiags.Error, summary, detail)
}
