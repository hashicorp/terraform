package s3

import (
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
	return tfdiags.AttributeValue(tfdiags.Error, summary, detail, attrPath)
}
