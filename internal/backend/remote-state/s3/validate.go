// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package s3

import (
	"fmt"
	"regexp"

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
