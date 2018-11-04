package aws

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/aws/aws-sdk-go/service/cognitoidentity"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go/service/configservice"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/waf"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/structure"
	"github.com/hashicorp/terraform/helper/validation"
)

// validateTypeStringNullableBoolean provides custom error messaging for TypeString booleans
// Some arguments require three values: true, false, and "" (unspecified).
// This ValidateFunc returns a custom message since the message with
// validation.StringInSlice([]string{"", "false", "true"}, false) is confusing:
// to be one of [ false true], got 1
func validateTypeStringNullableBoolean(v interface{}, k string) (ws []string, es []error) {
	value, ok := v.(string)
	if !ok {
		es = append(es, fmt.Errorf("expected type of %s to be string", k))
		return
	}

	for _, str := range []string{"", "0", "1", "false", "true"} {
		if value == str {
			return
		}
	}

	es = append(es, fmt.Errorf("expected %s to be one of [\"\", false, true], got %s", k, value))
	return
}

func validateRdsIdentifier(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[0-9a-z-]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only lowercase alphanumeric characters and hyphens allowed in %q", k))
	}
	if !regexp.MustCompile(`^[a-z]`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"first character of %q must be a letter", k))
	}
	if regexp.MustCompile(`--`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q cannot contain two consecutive hyphens", k))
	}
	if regexp.MustCompile(`-$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q cannot end with a hyphen", k))
	}
	return
}

func validateNeptuneIdentifier(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[0-9a-z-]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only lowercase alphanumeric characters and hyphens allowed in %q", k))
	}
	if !regexp.MustCompile(`^[a-z]`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"first character of %q must be a letter", k))
	}
	if regexp.MustCompile(`--`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q cannot contain two consecutive hyphens", k))
	}
	if regexp.MustCompile(`-$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q cannot end with a hyphen", k))
	}
	return
}

func validateRdsIdentifierPrefix(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[0-9a-z-]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only lowercase alphanumeric characters and hyphens allowed in %q", k))
	}
	if !regexp.MustCompile(`^[a-z]`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"first character of %q must be a letter", k))
	}
	if regexp.MustCompile(`--`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q cannot contain two consecutive hyphens", k))
	}
	return
}

func validateNeptuneIdentifierPrefix(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[0-9a-z-]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only lowercase alphanumeric characters and hyphens allowed in %q", k))
	}
	if !regexp.MustCompile(`^[a-z]`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"first character of %q must be a letter", k))
	}
	if regexp.MustCompile(`--`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q cannot contain two consecutive hyphens", k))
	}
	return
}

func validateRdsEngine() schema.SchemaValidateFunc {
	return validation.StringInSlice([]string{
		"aurora",
		"aurora-mysql",
		"aurora-postgresql",
	}, false)
}

func validateNeptuneEngine() schema.SchemaValidateFunc {
	return validation.StringInSlice([]string{
		"neptune",
	}, false)
}

func validateElastiCacheClusterId(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if (len(value) < 1) || (len(value) > 20) {
		errors = append(errors, fmt.Errorf(
			"%q (%q) must contain from 1 to 20 alphanumeric characters or hyphens", k, value))
	}
	if !regexp.MustCompile(`^[0-9a-z-]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only lowercase alphanumeric characters and hyphens allowed in %q (%q)", k, value))
	}
	if !regexp.MustCompile(`^[a-z]`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"first character of %q (%q) must be a letter", k, value))
	}
	if regexp.MustCompile(`--`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q (%q) cannot contain two consecutive hyphens", k, value))
	}
	if regexp.MustCompile(`-$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q (%q) cannot end with a hyphen", k, value))
	}
	return
}

func validateASGScheduleTimestamp(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	_, err := time.Parse(awsAutoscalingScheduleTimeLayout, value)
	if err != nil {
		errors = append(errors, fmt.Errorf(
			"%q cannot be parsed as iso8601 Timestamp Format", value))
	}

	return
}

// validateTagFilters confirms the "value" component of a tag filter is one of
// AWS's three allowed types.
func validateTagFilters(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if value != "KEY_ONLY" && value != "VALUE_ONLY" && value != "KEY_AND_VALUE" {
		errors = append(errors, fmt.Errorf(
			"%q must be one of \"KEY_ONLY\", \"VALUE_ONLY\", or \"KEY_AND_VALUE\"", k))
	}
	return
}

func validateDbParamGroupName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[0-9a-z-]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only lowercase alphanumeric characters and hyphens allowed in parameter group %q", k))
	}
	if !regexp.MustCompile(`^[a-z]`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"first character of parameter group %q must be a letter", k))
	}
	if regexp.MustCompile(`--`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"parameter group %q cannot contain two consecutive hyphens", k))
	}
	if regexp.MustCompile(`-$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"parameter group %q cannot end with a hyphen", k))
	}
	if len(value) > 255 {
		errors = append(errors, fmt.Errorf(
			"parameter group %q cannot be greater than 255 characters", k))
	}
	return
}

func validateDbParamGroupNamePrefix(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[0-9a-z-]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only lowercase alphanumeric characters and hyphens allowed in parameter group %q", k))
	}
	if !regexp.MustCompile(`^[a-z]`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"first character of parameter group %q must be a letter", k))
	}
	if regexp.MustCompile(`--`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"parameter group %q cannot contain two consecutive hyphens", k))
	}
	if len(value) > 255 {
		errors = append(errors, fmt.Errorf(
			"parameter group %q cannot be greater than 226 characters", k))
	}
	return
}

func validateElbName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) == 0 {
		return // short-circuit
	}
	if len(value) > 32 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be longer than 32 characters: %q", k, value))
	}
	if !regexp.MustCompile(`^[0-9A-Za-z-]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only alphanumeric characters and hyphens allowed in %q: %q",
			k, value))
	}
	if regexp.MustCompile(`^-`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q cannot begin with a hyphen: %q", k, value))
	}
	if regexp.MustCompile(`-$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q cannot end with a hyphen: %q", k, value))
	}
	return
}

func validateElbNamePrefix(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[0-9A-Za-z-]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only alphanumeric characters and hyphens allowed in %q: %q",
			k, value))
	}
	if len(value) > 6 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be longer than 6 characters: %q", k, value))
	}
	if regexp.MustCompile(`^-`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q cannot begin with a hyphen: %q", k, value))
	}
	return
}

func validateEcrRepositoryName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) < 2 {
		errors = append(errors, fmt.Errorf(
			"%q must be at least 2 characters long: %q", k, value))
	}
	if len(value) > 256 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be longer than 256 characters: %q", k, value))
	}

	// http://docs.aws.amazon.com/AmazonECR/latest/APIReference/API_CreateRepository.html
	pattern := `^(?:[a-z0-9]+(?:[._-][a-z0-9]+)*/)*[a-z0-9]+(?:[._-][a-z0-9]+)*$`
	if !regexp.MustCompile(pattern).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q doesn't comply with restrictions (%q): %q",
			k, pattern, value))
	}

	return
}

func validateCloudWatchDashboardName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) > 255 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be longer than 255 characters: %q", k, value))
	}

	// http://docs.aws.amazon.com/AmazonCloudWatch/latest/APIReference/API_PutDashboard.html
	pattern := `^[\-_A-Za-z0-9]+$`
	if !regexp.MustCompile(pattern).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q doesn't comply with restrictions (%q): %q",
			k, pattern, value))
	}

	return
}

func validateCloudWatchEventRuleName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) > 64 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be longer than 64 characters: %q", k, value))
	}

	// http://docs.aws.amazon.com/AmazonCloudWatchEvents/latest/APIReference/API_PutRule.html
	pattern := `^[\.\-_A-Za-z0-9]+$`
	if !regexp.MustCompile(pattern).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q doesn't comply with restrictions (%q): %q",
			k, pattern, value))
	}

	return
}

func validateCloudWatchLogResourcePolicyDocument(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	// http://docs.aws.amazon.com/AmazonCloudWatchLogs/latest/APIReference/API_PutResourcePolicy.html
	if len(value) > 5120 || (len(value) == 0) {
		errors = append(errors, fmt.Errorf("CloudWatch log resource policy document must be between 1 and 5120 characters."))
	}
	if _, err := structure.NormalizeJsonString(v); err != nil {
		errors = append(errors, fmt.Errorf("%q contains an invalid JSON: %s", k, err))
	}
	return
}

func validateCloudWatchEventTargetId(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) > 64 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be longer than 64 characters: %q", k, value))
	}

	// http://docs.aws.amazon.com/AmazonCloudWatchEvents/latest/APIReference/API_Target.html
	pattern := `^[\.\-_A-Za-z0-9]+$`
	if !regexp.MustCompile(pattern).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q doesn't comply with restrictions (%q): %q",
			k, pattern, value))
	}

	return
}

func validateLambdaFunctionName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) > 140 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be longer than 140 characters: %q", k, value))
	}
	// http://docs.aws.amazon.com/lambda/latest/dg/API_AddPermission.html
	pattern := `^(arn:[\w-]+:lambda:)?([a-z]{2}-(?:[a-z]+-){1,2}\d{1}:)?(\d{12}:)?(function:)?([a-zA-Z0-9-_]+)(:(\$LATEST|[a-zA-Z0-9-_]+))?$`
	if !regexp.MustCompile(pattern).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q doesn't comply with restrictions (%q): %q",
			k, pattern, value))
	}

	return
}

func validateLambdaQualifier(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) > 128 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be longer than 128 characters: %q", k, value))
	}
	// http://docs.aws.amazon.com/lambda/latest/dg/API_AddPermission.html
	pattern := `^[a-zA-Z0-9$_-]+$`
	if !regexp.MustCompile(pattern).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q doesn't comply with restrictions (%q): %q",
			k, pattern, value))
	}

	return
}

func validateLambdaPermissionAction(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	// http://docs.aws.amazon.com/lambda/latest/dg/API_AddPermission.html
	pattern := `^(lambda:[*]|lambda:[a-zA-Z]+|[*])$`
	if !regexp.MustCompile(pattern).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q doesn't comply with restrictions (%q): %q",
			k, pattern, value))
	}

	return
}

func validateLambdaPermissionEventSourceToken(v interface{}, k string) (ws []string, errors []error) {
	// https://docs.aws.amazon.com/lambda/latest/dg/API_AddPermission.html
	value := v.(string)

	if len(value) > 256 {
		errors = append(errors, fmt.Errorf("%q cannot be longer than 256 characters: %q", k, value))
	}

	pattern := `^[a-zA-Z0-9._\-]+$`
	if !regexp.MustCompile(pattern).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q doesn't comply with restrictions (%q): %q",
			k, pattern, value))
	}

	return
}

func validateAwsAccountId(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	// http://docs.aws.amazon.com/lambda/latest/dg/API_AddPermission.html
	pattern := `^\d{12}$`
	if !regexp.MustCompile(pattern).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q doesn't look like AWS Account ID (exactly 12 digits): %q",
			k, value))
	}

	return
}

func validateArn(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	if value == "" {
		return
	}

	// http://docs.aws.amazon.com/lambda/latest/dg/API_AddPermission.html
	pattern := `^arn:[\w-]+:([a-zA-Z0-9\-])+:([a-z]{2}-(gov-)?[a-z]+-\d{1})?:(\d{12})?:(.*)$`
	if !regexp.MustCompile(pattern).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q doesn't look like a valid ARN (%q): %q",
			k, pattern, value))
	}

	return
}

func validatePolicyStatementId(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	if len(value) > 100 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be longer than 100 characters: %q", k, value))
	}

	// http://docs.aws.amazon.com/lambda/latest/dg/API_AddPermission.html
	pattern := `^[a-zA-Z0-9-_]+$`
	if !regexp.MustCompile(pattern).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q doesn't look like a valid statement ID (%q): %q",
			k, pattern, value))
	}

	return
}

// validateCIDRNetworkAddress ensures that the string value is a valid CIDR that
// represents a network address - it adds an error otherwise
func validateCIDRNetworkAddress(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	_, ipnet, err := net.ParseCIDR(value)
	if err != nil {
		errors = append(errors, fmt.Errorf(
			"%q must contain a valid CIDR, got error parsing: %s", k, err))
		return
	}

	if ipnet == nil || value != ipnet.String() {
		errors = append(errors, fmt.Errorf(
			"%q must contain a valid network CIDR, got %q", k, value))
	}

	return
}

func validateHTTPMethod() schema.SchemaValidateFunc {
	return validation.StringInSlice([]string{
		"ANY",
		"DELETE",
		"GET",
		"HEAD",
		"OPTIONS",
		"PATCH",
		"POST",
		"PUT",
	}, false)
}

func validateLogMetricFilterName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	if len(value) > 512 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be longer than 512 characters: %q", k, value))
	}

	// http://docs.aws.amazon.com/AmazonCloudWatchLogs/latest/APIReference/API_PutMetricFilter.html
	pattern := `^[^:*]+$`
	if !regexp.MustCompile(pattern).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q isn't a valid log metric name (must not contain colon nor asterisk): %q",
			k, value))
	}

	return
}

func validateLogMetricFilterTransformationName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	if len(value) > 255 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be longer than 255 characters: %q", k, value))
	}

	// http://docs.aws.amazon.com/AmazonCloudWatchLogs/latest/APIReference/API_MetricTransformation.html
	pattern := `^[^:*$]*$`
	if !regexp.MustCompile(pattern).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q isn't a valid log metric transformation name (must not contain"+
				" colon, asterisk nor dollar sign): %q",
			k, value))
	}

	return
}

func validateLogGroupName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	if len(value) > 512 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be longer than 512 characters: %q", k, value))
	}

	// http://docs.aws.amazon.com/AmazonCloudWatchLogs/latest/APIReference/API_CreateLogGroup.html
	pattern := `^[\.\-_/#A-Za-z0-9]+$`
	if !regexp.MustCompile(pattern).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q isn't a valid log group name (alphanumeric characters, underscores,"+
				" hyphens, slashes, hash signs and dots are allowed): %q",
			k, value))
	}

	return
}

func validateLogGroupNamePrefix(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	if len(value) > 483 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be longer than 483 characters: %q", k, value))
	}

	// http://docs.aws.amazon.com/AmazonCloudWatchLogs/latest/APIReference/API_CreateLogGroup.html
	pattern := `^[\.\-_/#A-Za-z0-9]+$`
	if !regexp.MustCompile(pattern).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q isn't a valid log group name (alphanumeric characters, underscores,"+
				" hyphens, slashes, hash signs and dots are allowed): %q",
			k, value))
	}

	return
}

func validateS3BucketLifecycleTimestamp(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	_, err := time.Parse(time.RFC3339, fmt.Sprintf("%sT00:00:00Z", value))
	if err != nil {
		errors = append(errors, fmt.Errorf(
			"%q cannot be parsed as RFC3339 Timestamp Format", value))
	}

	return
}

func validateS3BucketLifecycleStorageClass() schema.SchemaValidateFunc {
	return validation.StringInSlice([]string{
		s3.TransitionStorageClassOnezoneIa,
		s3.TransitionStorageClassStandardIa,
		s3.TransitionStorageClassGlacier,
	}, false)
}

func validateDbEventSubscriptionName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[0-9A-Za-z-]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only alphanumeric characters and hyphens allowed in %q", k))
	}
	if len(value) > 255 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be longer than 255 characters", k))
	}
	return
}

func validateIAMPolicyJson(v interface{}, k string) (ws []string, errors []error) {
	// IAM Policy documents need to be valid JSON, and pass legacy parsing
	value := v.(string)
	if len(value) < 1 {
		errors = append(errors, fmt.Errorf("%q contains an invalid JSON policy", k))
		return
	}
	if value[:1] != "{" {
		errors = append(errors, fmt.Errorf("%q contains an invalid JSON policy", k))
		return
	}
	if _, err := structure.NormalizeJsonString(v); err != nil {
		errors = append(errors, fmt.Errorf("%q contains an invalid JSON: %s", k, err))
	}
	return
}

func validateCloudFormationTemplate(v interface{}, k string) (ws []string, errors []error) {
	if looksLikeJsonString(v) {
		if _, err := structure.NormalizeJsonString(v); err != nil {
			errors = append(errors, fmt.Errorf("%q contains an invalid JSON: %s", k, err))
		}
	} else {
		if _, err := checkYamlString(v); err != nil {
			errors = append(errors, fmt.Errorf("%q contains an invalid YAML: %s", k, err))
		}
	}
	return
}

func validateApiGatewayIntegrationContentHandling() schema.SchemaValidateFunc {
	return validation.StringInSlice([]string{
		apigateway.ContentHandlingStrategyConvertToBinary,
		apigateway.ContentHandlingStrategyConvertToText,
	}, false)
}

func validateSQSQueueName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) > 80 {
		errors = append(errors, fmt.Errorf("%q cannot be longer than 80 characters", k))
	}

	if !regexp.MustCompile(`^[0-9A-Za-z-_]+(\.fifo)?$`).MatchString(value) {
		errors = append(errors, fmt.Errorf("only alphanumeric characters and hyphens allowed in %q", k))
	}
	return
}

func validateSQSNonFifoQueueName(v interface{}, k string) (errors []error) {
	value := v.(string)
	if len(value) > 80 {
		errors = append(errors, fmt.Errorf("%q cannot be longer than 80 characters", k))
	}

	if !regexp.MustCompile(`^[0-9A-Za-z-_]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf("only alphanumeric characters and hyphens allowed in %q", k))
	}
	return
}

func validateSQSFifoQueueName(v interface{}, k string) (errors []error) {
	value := v.(string)

	if len(value) > 80 {
		errors = append(errors, fmt.Errorf("%q cannot be longer than 80 characters", k))
	}

	if !regexp.MustCompile(`^[0-9A-Za-z-_.]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf("only alphanumeric characters and hyphens allowed in %q", k))
	}

	if regexp.MustCompile(`^[^a-zA-Z0-9-_]`).MatchString(value) {
		errors = append(errors, fmt.Errorf("FIFO queue name must start with one of these characters [a-zA-Z0-9-_]: %v", value))
	}

	if !regexp.MustCompile(`\.fifo$`).MatchString(value) {
		errors = append(errors, fmt.Errorf("FIFO queue name should ends with \".fifo\": %v", value))
	}

	return
}

func validateOnceAWeekWindowFormat(v interface{}, k string) (ws []string, errors []error) {
	// valid time format is "ddd:hh24:mi"
	validTimeFormat := "(sun|mon|tue|wed|thu|fri|sat):([0-1][0-9]|2[0-3]):([0-5][0-9])"
	validTimeFormatConsolidated := "^(" + validTimeFormat + "-" + validTimeFormat + "|)$"

	value := strings.ToLower(v.(string))
	if !regexp.MustCompile(validTimeFormatConsolidated).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q must satisfy the format of \"ddd:hh24:mi-ddd:hh24:mi\".", k))
	}
	return
}

func validateOnceADayWindowFormat(v interface{}, k string) (ws []string, errors []error) {
	// valid time format is "hh24:mi"
	validTimeFormat := "([0-1][0-9]|2[0-3]):([0-5][0-9])"
	validTimeFormatConsolidated := "^(" + validTimeFormat + "-" + validTimeFormat + "|)$"

	value := v.(string)
	if !regexp.MustCompile(validTimeFormatConsolidated).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q must satisfy the format of \"hh24:mi-hh24:mi\".", k))
	}
	return
}

// Validates that ECS Placement Constraints are set correctly
// Takes type, and expression as strings
func validateAwsEcsPlacementConstraint(constType, constExpr string) error {
	switch constType {
	case "distinctInstance":
		// Expression can be nil for distinctInstance
		return nil
	case "memberOf":
		if constExpr == "" {
			return fmt.Errorf("Expression cannot be nil for 'memberOf' type")
		}
	default:
		return fmt.Errorf("Unknown type provided: %q", constType)
	}
	return nil
}

// http://docs.aws.amazon.com/amazondynamodb/latest/APIReference/API_CreateGlobalTable.html
func validateAwsDynamoDbGlobalTableName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if (len(value) > 255) || (len(value) < 3) {
		errors = append(errors, fmt.Errorf("%s length must be between 3 and 255 characters: %q", k, value))
	}
	pattern := `^[a-zA-Z0-9_.-]+$`
	if !regexp.MustCompile(pattern).MatchString(value) {
		errors = append(errors, fmt.Errorf("%s must only include alphanumeric, underscore, period, or hyphen characters: %q", k, value))
	}
	return
}

// Validates that an Ecs placement strategy is set correctly
// Takes type, and field as strings
func validateAwsEcsPlacementStrategy(stratType, stratField string) error {
	switch stratType {
	case "random":
		// random requires the field attribute to be unset.
		if stratField != "" {
			return fmt.Errorf("Random type requires the field attribute to be unset. Got: %s",
				stratField)
		}
	case "spread":
		//  For the spread placement strategy, valid values are instanceId
		// (or host, which has the same effect), or any platform or custom attribute
		// that is applied to a container instance
		// stratField is already cased to a string
		return nil
	case "binpack":
		if stratField != "cpu" && stratField != "memory" {
			return fmt.Errorf("Binpack type requires the field attribute to be either 'cpu' or 'memory'. Got: %s",
				stratField)
		}
	default:
		return fmt.Errorf("Unknown type %s. Must be one of 'random', 'spread', or 'binpack'.", stratType)
	}
	return nil
}

func validateAwsEmrEbsVolumeType() schema.SchemaValidateFunc {
	return validation.StringInSlice([]string{
		"gp2",
		"io1",
		"standard",
		"st1",
	}, false)
}

func validateAwsEmrCustomAmiId(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) > 256 {
		errors = append(errors, fmt.Errorf("%q cannot be longer than 256 characters", k))
	}

	if !regexp.MustCompile(`^ami\-[a-z0-9]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q must begin with 'ami-' and be comprised of only [a-z0-9]: %v", k, value))
	}

	return
}

func validateSfnStateMachineName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) > 80 {
		errors = append(errors, fmt.Errorf("%q cannot be longer than 80 characters", k))
	}

	if !regexp.MustCompile(`^[a-zA-Z0-9-_]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q must be composed with only these characters [a-zA-Z0-9-_]: %v", k, value))
	}
	return
}

func validateDmsCertificateId(v interface{}, k string) (ws []string, es []error) {
	val := v.(string)

	if len(val) > 255 {
		es = append(es, fmt.Errorf("%q must not be longer than 255 characters", k))
	}
	if !regexp.MustCompile("^[a-zA-Z][a-zA-Z0-9-]+$").MatchString(val) {
		es = append(es, fmt.Errorf("%q must start with a letter, only contain alphanumeric characters and hyphens", k))
	}
	if strings.Contains(val, "--") {
		es = append(es, fmt.Errorf("%q must not contain consecutive hyphens", k))
	}
	if strings.HasSuffix(val, "-") {
		es = append(es, fmt.Errorf("%q must not end in a hyphen", k))
	}

	return
}

func validateDmsEndpointId(v interface{}, k string) (ws []string, es []error) {
	val := v.(string)

	if len(val) > 255 {
		es = append(es, fmt.Errorf("%q must not be longer than 255 characters", k))
	}
	if !regexp.MustCompile("^[a-zA-Z][a-zA-Z0-9-]+$").MatchString(val) {
		es = append(es, fmt.Errorf("%q must start with a letter, only contain alphanumeric characters and hyphens", k))
	}
	if strings.Contains(val, "--") {
		es = append(es, fmt.Errorf("%q must not contain consecutive hyphens", k))
	}
	if strings.HasSuffix(val, "-") {
		es = append(es, fmt.Errorf("%q must not end in a hyphen", k))
	}

	return
}

func validateDmsReplicationInstanceId(v interface{}, k string) (ws []string, es []error) {
	val := v.(string)

	if len(val) > 63 {
		es = append(es, fmt.Errorf("%q must not be longer than 63 characters", k))
	}
	if !regexp.MustCompile("^[a-zA-Z][a-zA-Z0-9-]+$").MatchString(val) {
		es = append(es, fmt.Errorf("%q must start with a letter, only contain alphanumeric characters and hyphens", k))
	}
	if strings.Contains(val, "--") {
		es = append(es, fmt.Errorf("%q must not contain consecutive hyphens", k))
	}
	if strings.HasSuffix(val, "-") {
		es = append(es, fmt.Errorf("%q must not end in a hyphen", k))
	}

	return
}

func validateDmsReplicationSubnetGroupId(v interface{}, k string) (ws []string, es []error) {
	val := v.(string)

	if val == "default" {
		es = append(es, fmt.Errorf("%q must not be default", k))
	}
	if len(val) > 255 {
		es = append(es, fmt.Errorf("%q must not be longer than 255 characters", k))
	}
	if !regexp.MustCompile(`^[a-zA-Z0-9. _-]+$`).MatchString(val) {
		es = append(es, fmt.Errorf("%q must only contain alphanumeric characters, periods, spaces, underscores and hyphens", k))
	}

	return
}

func validateDmsReplicationTaskId(v interface{}, k string) (ws []string, es []error) {
	val := v.(string)

	if len(val) > 255 {
		es = append(es, fmt.Errorf("%q must not be longer than 255 characters", k))
	}
	if !regexp.MustCompile("^[a-zA-Z][a-zA-Z0-9-]+$").MatchString(val) {
		es = append(es, fmt.Errorf("%q must start with a letter, only contain alphanumeric characters and hyphens", k))
	}
	if strings.Contains(val, "--") {
		es = append(es, fmt.Errorf("%q must not contain consecutive hyphens", k))
	}
	if strings.HasSuffix(val, "-") {
		es = append(es, fmt.Errorf("%q must not end in a hyphen", k))
	}

	return
}

func validateConfigExecutionFrequency() schema.SchemaValidateFunc {
	return validation.StringInSlice([]string{
		configservice.MaximumExecutionFrequencyOneHour,
		configservice.MaximumExecutionFrequencyThreeHours,
		configservice.MaximumExecutionFrequencySixHours,
		configservice.MaximumExecutionFrequencyTwelveHours,
		configservice.MaximumExecutionFrequencyTwentyFourHours,
	}, false)
}

func validateAccountAlias(v interface{}, k string) (ws []string, es []error) {
	val := v.(string)

	if (len(val) < 3) || (len(val) > 63) {
		es = append(es, fmt.Errorf("%q must contain from 3 to 63 alphanumeric characters or hyphens", k))
	}
	if !regexp.MustCompile("^[a-z0-9][a-z0-9-]+$").MatchString(val) {
		es = append(es, fmt.Errorf("%q must start with an alphanumeric character and only contain lowercase alphanumeric characters and hyphens", k))
	}
	if strings.Contains(val, "--") {
		es = append(es, fmt.Errorf("%q must not contain consecutive hyphens", k))
	}
	if strings.HasSuffix(val, "-") {
		es = append(es, fmt.Errorf("%q must not end in a hyphen", k))
	}
	return
}

func validateIamRolePolicyName(v interface{}, k string) (ws []string, errors []error) {
	// https://github.com/boto/botocore/blob/2485f5c/botocore/data/iam/2010-05-08/service-2.json#L8291-L8296
	value := v.(string)
	if len(value) > 128 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be longer than 128 characters", k))
	}
	if !regexp.MustCompile("^[\\w+=,.@-]+$").MatchString(value) {
		errors = append(errors, fmt.Errorf("%q must match [\\w+=,.@-]", k))
	}
	return
}

func validateIamRolePolicyNamePrefix(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) > 100 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be longer than 100 characters", k))
	}
	if !regexp.MustCompile("^[\\w+=,.@-]+$").MatchString(value) {
		errors = append(errors, fmt.Errorf("%q must match [\\w+=,.@-]", k))
	}
	return
}

func validateApiGatewayUsagePlanQuotaSettings(v map[string]interface{}) (errors []error) {
	period := v["period"].(string)
	offset := v["offset"].(int)

	if period == apigateway.QuotaPeriodTypeDay && offset != 0 {
		errors = append(errors, fmt.Errorf("Usage Plan quota offset must be zero in the DAY period"))
	}

	if period == apigateway.QuotaPeriodTypeWeek && (offset < 0 || offset > 6) {
		errors = append(errors, fmt.Errorf("Usage Plan quota offset must be between 0 and 6 inclusive in the WEEK period"))
	}

	if period == apigateway.QuotaPeriodTypeMonth && (offset < 0 || offset > 27) {
		errors = append(errors, fmt.Errorf("Usage Plan quota offset must be between 0 and 27 inclusive in the MONTH period"))
	}

	return
}

func validateDbSubnetGroupName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[ .0-9a-z-_]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only lowercase alphanumeric characters, hyphens, underscores, periods, and spaces allowed in %q", k))
	}
	if len(value) > 255 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be longer than 255 characters", k))
	}
	if regexp.MustCompile(`(?i)^default$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q is not allowed as %q", "Default", k))
	}
	return
}

func validateNeptuneSubnetGroupName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[ .0-9a-z-_]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only lowercase alphanumeric characters, hyphens, underscores, periods, and spaces allowed in %q", k))
	}
	if len(value) > 255 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be longer than 255 characters", k))
	}
	if value == "default" {
		errors = append(errors, fmt.Errorf(
			"%q is not allowed as %q", "Default", k))
	}
	return
}

func validateDbSubnetGroupNamePrefix(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[ .0-9a-z-_]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only lowercase alphanumeric characters, hyphens, underscores, periods, and spaces allowed in %q", k))
	}
	if len(value) > 229 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be longer than 229 characters", k))
	}
	return
}

func validateNeptuneSubnetGroupNamePrefix(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[ .0-9a-z-_]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only lowercase alphanumeric characters, hyphens, underscores, periods, and spaces allowed in %q", k))
	}
	prefixMaxLength := 255 - resource.UniqueIDSuffixLength
	if len(value) > prefixMaxLength {
		errors = append(errors, fmt.Errorf(
			"%q cannot be longer than %d characters", k, prefixMaxLength))
	}
	return
}

func validateDbOptionGroupName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[a-z]`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"first character of %q must be a letter", k))
	}
	if !regexp.MustCompile(`^[0-9a-z-]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only lowercase alphanumeric characters and hyphens allowed in %q", k))
	}
	if regexp.MustCompile(`--`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q cannot contain two consecutive hyphens", k))
	}
	if regexp.MustCompile(`-$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q cannot end with a hyphen", k))
	}
	if len(value) > 255 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be greater than 255 characters", k))
	}
	return
}

func validateDbOptionGroupNamePrefix(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[a-z]`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"first character of %q must be a letter", k))
	}
	if !regexp.MustCompile(`^[0-9a-z-]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only alphanumeric characters and hyphens allowed in %q", k))
	}
	if regexp.MustCompile(`--`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q cannot contain two consecutive hyphens", k))
	}
	if len(value) > 229 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be greater than 229 characters", k))
	}
	return
}

func validateOpenIdURL(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	u, err := url.Parse(value)
	if err != nil {
		errors = append(errors, fmt.Errorf("%q has to be a valid URL", k))
		return
	}
	if u.Scheme != "https" {
		errors = append(errors, fmt.Errorf("%q has to use HTTPS scheme (i.e. begin with https://)", k))
	}
	if len(u.Query()) > 0 {
		errors = append(errors, fmt.Errorf("%q cannot contain query parameters per the OIDC standard", k))
	}
	return
}

func validateAwsKmsName(v interface{}, k string) (ws []string, es []error) {
	value := v.(string)
	if !regexp.MustCompile(`^(alias\/)[a-zA-Z0-9:/_-]+$`).MatchString(value) {
		es = append(es, fmt.Errorf(
			"%q must begin with 'alias/' and be comprised of only [a-zA-Z0-9:/_-]", k))
	}
	return
}

func validateAwsKmsGrantName(v interface{}, k string) (ws []string, es []error) {
	value := v.(string)

	if len(value) > 256 {
		es = append(es, fmt.Errorf("%s can not be greater than 256 characters", k))
	}

	if !regexp.MustCompile(`^[a-zA-Z0-9:/_-]+$`).MatchString(value) {
		es = append(es, fmt.Errorf("%s must only contain [a-zA-Z0-9:/_-]", k))
	}

	return
}

func validateCognitoIdentityPoolName(v interface{}, k string) (ws []string, errors []error) {
	val := v.(string)
	if !regexp.MustCompile("^[\\w _]+$").MatchString(val) {
		errors = append(errors, fmt.Errorf("%q must contain only alphanumeric characters and spaces", k))
	}

	return
}

func validateCognitoProviderDeveloperName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) > 100 {
		errors = append(errors, fmt.Errorf("%q cannot be longer than 100 characters", k))
	}

	if !regexp.MustCompile("^[\\w._-]+$").MatchString(value) {
		errors = append(errors, fmt.Errorf("%q must contain only alphanumeric characters, dots, underscores and hyphens", k))
	}

	return
}

func validateCognitoSupportedLoginProviders(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) < 1 {
		errors = append(errors, fmt.Errorf("%q cannot be less than 1 character", k))
	}

	if len(value) > 128 {
		errors = append(errors, fmt.Errorf("%q cannot be longer than 128 characters", k))
	}

	if !regexp.MustCompile("^[\\w.;_/-]+$").MatchString(value) {
		errors = append(errors, fmt.Errorf("%q must contain only alphanumeric characters, dots, semicolons, underscores, slashes and hyphens", k))
	}

	return
}

func validateCognitoIdentityProvidersClientId(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) < 1 {
		errors = append(errors, fmt.Errorf("%q cannot be less than 1 character", k))
	}

	if len(value) > 128 {
		errors = append(errors, fmt.Errorf("%q cannot be longer than 128 characters", k))
	}

	if !regexp.MustCompile("^[\\w_]+$").MatchString(value) {
		errors = append(errors, fmt.Errorf("%q must contain only alphanumeric characters and underscores", k))
	}

	return
}

func validateCognitoIdentityProvidersProviderName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) < 1 {
		errors = append(errors, fmt.Errorf("%q cannot be less than 1 character", k))
	}

	if len(value) > 128 {
		errors = append(errors, fmt.Errorf("%q cannot be longer than 128 characters", k))
	}

	if !regexp.MustCompile("^[\\w._:/-]+$").MatchString(value) {
		errors = append(errors, fmt.Errorf("%q must contain only alphanumeric characters, dots, underscores, colons, slashes and hyphens", k))
	}

	return
}

func validateCognitoUserGroupName(v interface{}, k string) (ws []string, es []error) {
	value := v.(string)
	if len(value) < 1 {
		es = append(es, fmt.Errorf("%q cannot be less than 1 character", k))
	}

	if len(value) > 128 {
		es = append(es, fmt.Errorf("%q cannot be longer than 128 character", k))
	}

	if !regexp.MustCompile(`[\p{L}\p{M}\p{S}\p{N}\p{P}]+`).MatchString(value) {
		es = append(es, fmt.Errorf("%q must satisfy regular expression pattern: [\\p{L}\\p{M}\\p{S}\\p{N}\\p{P}]+", k))
	}
	return
}

func validateCognitoUserPoolEmailVerificationMessage(v interface{}, k string) (ws []string, es []error) {
	value := v.(string)
	if len(value) < 6 {
		es = append(es, fmt.Errorf("%q cannot be less than 6 characters", k))
	}

	if len(value) > 20000 {
		es = append(es, fmt.Errorf("%q cannot be longer than 20000 characters", k))
	}

	if !regexp.MustCompile(`[\p{L}\p{M}\p{S}\p{N}\p{P}\s*]*\{####\}[\p{L}\p{M}\p{S}\p{N}\p{P}\s*]*`).MatchString(value) {
		es = append(es, fmt.Errorf("%q does not contain {####}", k))
	}
	return
}

func validateCognitoUserPoolEmailVerificationSubject(v interface{}, k string) (ws []string, es []error) {
	value := v.(string)
	if len(value) < 6 {
		es = append(es, fmt.Errorf("%q cannot be less than 6 characters", k))
	}

	if len(value) > 140 {
		es = append(es, fmt.Errorf("%q cannot be longer than 140 characters", k))
	}

	if !regexp.MustCompile(`[\p{L}\p{M}\p{S}\p{N}\p{P}\s]+`).MatchString(value) {
		es = append(es, fmt.Errorf("%q can be composed of any kind of letter, symbols, numeric character, punctuation and whitespaces", k))
	}
	return
}

func validateCognitoUserPoolId(v interface{}, k string) (ws []string, es []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[\w-]+_[0-9a-zA-Z]+$`).MatchString(value) {
		es = append(es, fmt.Errorf("%q must be the region name followed by an underscore and then alphanumeric pattern", k))
	}
	return
}

func validateCognitoUserPoolSmsAuthenticationMessage(v interface{}, k string) (ws []string, es []error) {
	value := v.(string)
	if len(value) < 6 {
		es = append(es, fmt.Errorf("%q cannot be less than 6 characters", k))
	}

	if len(value) > 140 {
		es = append(es, fmt.Errorf("%q cannot be longer than 140 characters", k))
	}

	if !regexp.MustCompile(`.*\{####\}.*`).MatchString(value) {
		es = append(es, fmt.Errorf("%q does not contain {####}", k))
	}
	return
}

func validateCognitoUserPoolSmsVerificationMessage(v interface{}, k string) (ws []string, es []error) {
	value := v.(string)
	if len(value) < 6 {
		es = append(es, fmt.Errorf("%q cannot be less than 6 characters", k))
	}

	if len(value) > 140 {
		es = append(es, fmt.Errorf("%q cannot be longer than 140 characters", k))
	}

	if !regexp.MustCompile(`.*\{####\}.*`).MatchString(value) {
		es = append(es, fmt.Errorf("%q does not contain {####}", k))
	}
	return
}

func validateCognitoUserPoolClientAuthFlows(v interface{}, k string) (ws []string, es []error) {
	validValues := []string{
		cognitoidentityprovider.AuthFlowTypeAdminNoSrpAuth,
		cognitoidentityprovider.AuthFlowTypeCustomAuth,
	}
	period := v.(string)
	for _, f := range validValues {
		if period == f {
			return
		}
	}
	es = append(es, fmt.Errorf(
		"%q contains an invalid auth flow %q. Valid auth flows are %q.",
		k, period, validValues))
	return
}

func validateCognitoUserPoolTemplateEmailMessage(v interface{}, k string) (ws []string, es []error) {
	value := v.(string)
	if len(value) < 6 {
		es = append(es, fmt.Errorf("%q cannot be less than 6 characters", k))
	}

	if len(value) > 20000 {
		es = append(es, fmt.Errorf("%q cannot be longer than 20000 characters", k))
	}

	if !regexp.MustCompile(`[\p{L}\p{M}\p{S}\p{N}\p{P}\s*]*\{####\}[\p{L}\p{M}\p{S}\p{N}\p{P}\s*]*`).MatchString(value) {
		es = append(es, fmt.Errorf("%q does not contain {####}", k))
	}
	return
}

func validateCognitoUserPoolTemplateEmailMessageByLink(v interface{}, k string) (ws []string, es []error) {
	value := v.(string)
	if len(value) < 1 {
		es = append(es, fmt.Errorf("%q cannot be less than 1 character", k))
	}

	if len(value) > 20000 {
		es = append(es, fmt.Errorf("%q cannot be longer than 20000 characters", k))
	}

	if !regexp.MustCompile(`[\p{L}\p{M}\p{S}\p{N}\p{P}\s*]*\{##[\p{L}\p{M}\p{S}\p{N}\p{P}\s*]*##\}[\p{L}\p{M}\p{S}\p{N}\p{P}\s*]*`).MatchString(value) {
		es = append(es, fmt.Errorf("%q must satisfy regular expression pattern: [\\p{L}\\p{M}\\p{S}\\p{N}\\p{P}\\s*]*\\{##[\\p{L}\\p{M}\\p{S}\\p{N}\\p{P}\\s*]*##\\}[\\p{L}\\p{M}\\p{S}\\p{N}\\p{P}\\s*]*", k))
	}
	return
}

func validateCognitoUserPoolTemplateEmailSubject(v interface{}, k string) (ws []string, es []error) {
	value := v.(string)
	if len(value) < 1 {
		es = append(es, fmt.Errorf("%q cannot be less than 1 character", k))
	}

	if len(value) > 140 {
		es = append(es, fmt.Errorf("%q cannot be longer than 140 characters", k))
	}

	if !regexp.MustCompile(`[\p{L}\p{M}\p{S}\p{N}\p{P}\s]+`).MatchString(value) {
		es = append(es, fmt.Errorf("%q must satisfy regular expression pattern: [\\p{L}\\p{M}\\p{S}\\p{N}\\p{P}\\s]+", k))
	}
	return
}

func validateCognitoUserPoolTemplateEmailSubjectByLink(v interface{}, k string) (ws []string, es []error) {
	value := v.(string)
	if len(value) < 1 {
		es = append(es, fmt.Errorf("%q cannot be less than 1 character", k))
	}

	if len(value) > 140 {
		es = append(es, fmt.Errorf("%q cannot be longer than 140 characters", k))
	}

	if !regexp.MustCompile(`[\p{L}\p{M}\p{S}\p{N}\p{P}\s]+`).MatchString(value) {
		es = append(es, fmt.Errorf("%q must satisfy regular expression pattern: [\\p{L}\\p{M}\\p{S}\\p{N}\\p{P}\\s]+", k))
	}
	return
}

func validateCognitoUserPoolTemplateSmsMessage(v interface{}, k string) (ws []string, es []error) {
	value := v.(string)
	if len(value) < 6 {
		es = append(es, fmt.Errorf("%q cannot be less than 6 characters", k))
	}

	if len(value) > 140 {
		es = append(es, fmt.Errorf("%q cannot be longer than 140 characters", k))
	}

	if !regexp.MustCompile(`.*\{####\}.*`).MatchString(value) {
		es = append(es, fmt.Errorf("%q does not contain {####}", k))
	}
	return
}

func validateCognitoUserPoolInviteTemplateEmailMessage(v interface{}, k string) (ws []string, es []error) {
	value := v.(string)
	if len(value) < 6 {
		es = append(es, fmt.Errorf("%q cannot be less than 6 characters", k))
	}

	if len(value) > 20000 {
		es = append(es, fmt.Errorf("%q cannot be longer than 20000 characters", k))
	}

	if !regexp.MustCompile(`[\p{L}\p{M}\p{S}\p{N}\p{P}\s*]*\{####\}[\p{L}\p{M}\p{S}\p{N}\p{P}\s*]*`).MatchString(value) {
		es = append(es, fmt.Errorf("%q does not contain {####}", k))
	}

	if !regexp.MustCompile(`.*\{username\}.*`).MatchString(value) {
		es = append(es, fmt.Errorf("%q does not contain {username}", k))
	}
	return
}

func validateCognitoUserPoolInviteTemplateSmsMessage(v interface{}, k string) (ws []string, es []error) {
	value := v.(string)
	if len(value) < 6 {
		es = append(es, fmt.Errorf("%q cannot be less than 6 characters", k))
	}

	if len(value) > 140 {
		es = append(es, fmt.Errorf("%q cannot be longer than 140 characters", k))
	}

	if !regexp.MustCompile(`.*\{####\}.*`).MatchString(value) {
		es = append(es, fmt.Errorf("%q does not contain {####}", k))
	}

	if !regexp.MustCompile(`.*\{username\}.*`).MatchString(value) {
		es = append(es, fmt.Errorf("%q does not contain {username}", k))
	}
	return
}

func validateCognitoUserPoolReplyEmailAddress(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	if !regexp.MustCompile(`[\p{L}\p{M}\p{S}\p{N}\p{P}]+@[\p{L}\p{M}\p{S}\p{N}\p{P}]+`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q must satisfy regular expression pattern: [\\p{L}\\p{M}\\p{S}\\p{N}\\p{P}]+@[\\p{L}\\p{M}\\p{S}\\p{N}\\p{P}]+", k))
	}
	return
}

func validateCognitoUserPoolSchemaName(v interface{}, k string) (ws []string, es []error) {
	value := v.(string)
	if len(value) < 1 {
		es = append(es, fmt.Errorf("%q cannot be less than 1 character", k))
	}

	if len(value) > 20 {
		es = append(es, fmt.Errorf("%q cannot be longer than 20 character", k))
	}

	if !regexp.MustCompile(`[\p{L}\p{M}\p{S}\p{N}\p{P}]+`).MatchString(value) {
		es = append(es, fmt.Errorf("%q must satisfy regular expression pattern: [\\p{L}\\p{M}\\p{S}\\p{N}\\p{P}]+", k))
	}
	return
}

func validateCognitoUserPoolClientURL(v interface{}, k string) (ws []string, es []error) {
	value := v.(string)
	if len(value) < 1 {
		es = append(es, fmt.Errorf("%q cannot be less than 1 character", k))
	}

	if len(value) > 1024 {
		es = append(es, fmt.Errorf("%q cannot be longer than 1024 character", k))
	}

	if !regexp.MustCompile(`[\p{L}\p{M}\p{S}\p{N}\p{P}]+`).MatchString(value) {
		es = append(es, fmt.Errorf("%q must satisfy regular expression pattern: [\\p{L}\\p{M}\\p{S}\\p{N}\\p{P}]+", k))
	}
	return
}

func validateCognitoResourceServerScopeName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	if len(value) < 1 {
		errors = append(errors, fmt.Errorf("%q cannot be less than 1 character", k))
	}
	if len(value) > 256 {
		errors = append(errors, fmt.Errorf("%q cannot be longer than 256 character", k))
	}
	if !regexp.MustCompile(`[\x21\x23-\x2E\x30-\x5B\x5D-\x7E]+`).MatchString(value) {
		errors = append(errors, fmt.Errorf("%q must satisfy regular expression pattern: [\\x21\\x23-\\x2E\\x30-\\x5B\\x5D-\\x7E]+", k))
	}
	return
}

func validateWafMetricName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[0-9A-Za-z]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"Only alphanumeric characters allowed in %q: %q",
			k, value))
	}
	return
}

func validateWafPredicatesType() schema.SchemaValidateFunc {
	return validation.StringInSlice([]string{
		waf.PredicateTypeByteMatch,
		waf.PredicateTypeGeoMatch,
		waf.PredicateTypeIpmatch,
		waf.PredicateTypeRegexMatch,
		waf.PredicateTypeSizeConstraint,
		waf.PredicateTypeSqlInjectionMatch,
		waf.PredicateTypeXssMatch,
	}, false)
}

func validateIamRoleDescription(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	if len(value) > 1000 {
		errors = append(errors, fmt.Errorf("%q cannot be longer than 1000 characters", k))
	}

	if !regexp.MustCompile(`[\p{L}\p{M}\p{Z}\p{S}\p{N}\p{P}]*`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"Only alphanumeric & accented characters allowed in %q: %q (Must satisfy regular expression pattern: [\\p{L}\\p{M}\\p{Z}\\p{S}\\p{N}\\p{P}]*)",
			k, value))
	}
	return
}

func validateAwsSSMName(v interface{}, k string) (ws []string, errors []error) {
	// http://docs.aws.amazon.com/systems-manager/latest/APIReference/API_CreateDocument.html#EC2-CreateDocument-request-Name
	value := v.(string)

	if !regexp.MustCompile(`^[a-zA-Z0-9_\-.]{3,128}$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"Only alphanumeric characters, hyphens, dots & underscores allowed in %q: %q (Must satisfy regular expression pattern: ^[a-zA-Z0-9_\\-.]{3,128}$)",
			k, value))
	}

	return
}

func validateBatchName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[0-9a-zA-Z]{1}[0-9a-zA-Z_\-]{0,127}$`).MatchString(value) {
		errors = append(errors, fmt.Errorf("%q (%q) must be up to 128 letters (uppercase and lowercase), numbers, underscores and dashes, and must start with an alphanumeric.", k, v))
	}
	return
}

func validateSecurityGroupRuleDescription(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) > 255 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be longer than 255 characters: %q", k, value))
	}

	// https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_IpRange.html. Note that
	// "" is an allowable description value.
	pattern := `^[A-Za-z0-9 \.\_\-\:\/\(\)\#\,\@\[\]\+\=\;\{\}\!\$\*]*$`
	if !regexp.MustCompile(pattern).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q doesn't comply with restrictions (%q): %q",
			k, pattern, value))
	}
	return
}

func validateIoTTopicRuleName(v interface{}, s string) ([]string, []error) {
	name := v.(string)
	if len(name) < 1 || len(name) > 128 {
		return nil, []error{fmt.Errorf("Name must between 1 and 128 characters long")}
	}

	matched, err := regexp.MatchReader("^[a-zA-Z0-9_]+$", strings.NewReader(name))

	if err != nil {
		return nil, []error{err}
	}

	if !matched {
		return nil, []error{fmt.Errorf("Name must match the pattern ^[a-zA-Z0-9_]+$")}
	}

	return nil, nil
}

func validateIoTTopicRuleCloudWatchAlarmStateValue(v interface{}, s string) ([]string, []error) {
	switch v.(string) {
	case
		"OK",
		"ALARM",
		"INSUFFICIENT_DATA":
		return nil, nil
	}

	return nil, []error{fmt.Errorf("State must be one of OK, ALARM, or INSUFFICIENT_DATA")}
}

func validateIoTTopicRuleCloudWatchMetricTimestamp(v interface{}, s string) ([]string, []error) {
	dateString := v.(string)

	// https://docs.aws.amazon.com/iot/latest/apireference/API_CloudwatchMetricAction.html
	if _, err := time.Parse(time.RFC3339, dateString); err != nil {
		return nil, []error{err}
	}
	return nil, nil
}

func validateIoTTopicRuleElasticSearchEndpoint(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	// https://docs.aws.amazon.com/iot/latest/apireference/API_ElasticsearchAction.html
	if !regexp.MustCompile(`https?://.*`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q should be an URL: %q",
			k, value))
	}
	return
}

func validateIoTTopicRuleFirehoseSeparator(v interface{}, s string) ([]string, []error) {
	switch v.(string) {
	case
		",",
		"\t",
		"\n",
		"\r\n":
		return nil, nil
	}

	return nil, []error{fmt.Errorf("Separator must be one of ',' (comma), '\\t' (tab) '\\n' (newline) or '\\r\\n' (Windows newline)")}
}

func validateCognitoRoleMappingsAmbiguousRoleResolutionAgainstType(v map[string]interface{}) (errors []error) {
	t := v["type"].(string)
	isRequired := t == cognitoidentity.RoleMappingTypeToken || t == cognitoidentity.RoleMappingTypeRules

	if value, ok := v["ambiguous_role_resolution"]; (!ok || value == "") && isRequired {
		errors = append(errors, fmt.Errorf("Ambiguous Role Resolution must be defined when \"type\" equals \"Token\" or \"Rules\""))
	}

	return
}

func validateCognitoRoleMappingsRulesConfiguration(v map[string]interface{}) (errors []error) {
	t := v["type"].(string)
	valLength := 0
	if value, ok := v["mapping_rule"]; ok {
		valLength = len(value.([]interface{}))
	}

	if (valLength == 0) && t == cognitoidentity.RoleMappingTypeRules {
		errors = append(errors, fmt.Errorf("mapping_rule is required for Rules"))
	}

	if (valLength > 0) && t == cognitoidentity.RoleMappingTypeToken {
		errors = append(errors, fmt.Errorf("mapping_rule must not be set for Token based role mapping"))
	}

	return
}

func validateCognitoRoleMappingsRulesClaim(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	if !regexp.MustCompile("^[\\p{L}\\p{M}\\p{S}\\p{N}\\p{P}]+$").MatchString(value) {
		errors = append(errors, fmt.Errorf("%q must contain only alphanumeric characters, dots, underscores, colons, slashes and hyphens", k))
	}

	return
}

// Validates that either authenticated or unauthenticated is defined
func validateCognitoRoles(v map[string]interface{}, k string) (errors []error) {
	_, hasAuthenticated := v["authenticated"].(string)
	_, hasUnauthenticated := v["unauthenticated"].(string)

	if !hasAuthenticated && !hasUnauthenticated {
		errors = append(errors, fmt.Errorf("%q: Either \"authenticated\" or \"unauthenticated\" must be defined", k))
	}

	return
}

func validateCognitoUserPoolDomain(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[a-z0-9](?:[a-z0-9\-]{0,61}[a-z0-9])?$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only lowercase alphanumeric characters and hyphens (max length 63 chars) allowed in %q", k))
	}
	return
}

func validateDxConnectionBandWidth() schema.SchemaValidateFunc {
	return validation.StringInSlice([]string{
		"1Gbps",
		"10Gbps",
		"50Mbps",
		"100Mbps",
		"200Mbps",
		"300Mbps",
		"400Mbps",
		"500Mbps"}, false)
}

func validateKmsKey(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	arnPrefixPattern := `arn:[^:]+:kms:[^:]+:[^:]+:`
	keyIdPattern := "[A-Za-z0-9-]+"
	keyArnPattern := arnPrefixPattern + "key/" + keyIdPattern
	aliasNamePattern := "alias/[a-zA-Z0-9:/_-]+"
	aliasArnPattern := arnPrefixPattern + aliasNamePattern
	if !regexp.MustCompile(fmt.Sprintf("^%s$", keyIdPattern)).MatchString(value) &&
		!regexp.MustCompile(fmt.Sprintf("^%s$", keyArnPattern)).MatchString(value) &&
		!regexp.MustCompile(fmt.Sprintf("^%s$", aliasNamePattern)).MatchString(value) &&
		!regexp.MustCompile(fmt.Sprintf("^%s$", aliasArnPattern)).MatchString(value) {
		errors = append(errors, fmt.Errorf("%q must be one of the following patterns: %s, %s, %s or %s", k, keyIdPattern, keyArnPattern, aliasNamePattern, aliasArnPattern))
	}
	return
}

func validateAwsElastiCacheReplicationGroupAuthToken(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if (len(value) < 16) || (len(value) > 128) {
		errors = append(errors, fmt.Errorf(
			"%q must contain from 16 to 128 alphanumeric characters or symbols (excluding @, \", and /)", k))
	}
	if !regexp.MustCompile(`^[^@"\/]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only alphanumeric characters or symbols (excluding @, \", and /) allowed in %q", k))
	}
	return
}

func validateDynamoDbStreamSpec(d *schema.ResourceDiff) error {
	enabled := d.Get("stream_enabled").(bool)
	if enabled {
		if v, ok := d.GetOk("stream_view_type"); ok {
			value := v.(string)
			if len(value) == 0 {
				return errors.New("stream_view_type must be non-empty when stream_enabled = true")
			}
			return nil
		}
		return errors.New("stream_view_type is required when stream_enabled = true")
	}
	return nil
}

func validateVpnGatewayAmazonSideAsn(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_CreateVpnGateway.html
	asn, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		errors = append(errors, fmt.Errorf("%q (%q) must be a 64-bit integer", k, v))
		return
	}

	// https://github.com/terraform-providers/terraform-provider-aws/issues/5263
	isLegacyAsn := func(a int64) bool {
		return a == 7224 || a == 9059 || a == 10124 || a == 17493
	}

	if !isLegacyAsn(asn) && ((asn < 64512) || (asn > 65534 && asn < 4200000000) || (asn > 4294967294)) {
		errors = append(errors, fmt.Errorf("%q (%q) must be 7224, 9059, 10124 or 17493 or in the range 64512 to 65534 or 4200000000 to 4294967294", k, v))
	}
	return
}

func validateDxGatewayAmazonSideAsn(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	// https://docs.aws.amazon.com/directconnect/latest/APIReference/API_CreateDirectConnectGateway.html
	asn, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		errors = append(errors, fmt.Errorf("%q (%q) must be a 64-bit integer", k, v))
		return
	}

	if (asn < 64512) || (asn > 65534 && asn < 4200000000) || (asn > 4294967294) {
		errors = append(errors, fmt.Errorf("%q (%q) must be in the range 64512 to 65534 or 4200000000 to 4294967294", k, v))
	}
	return
}

func validateIotThingTypeName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`[a-zA-Z0-9:_-]+`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only alphanumeric characters, colons, underscores and hyphens allowed in %q", k))
	}
	return
}

func validateIotThingTypeDescription(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) > 2028 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be longer than 2028 characters", k))
	}
	if !regexp.MustCompile(`[\\p{Graph}\\x20]*`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q must match pattern [\\p{Graph}\\x20]*", k))
	}
	return
}

func validateIotThingTypeSearchableAttribute(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) > 128 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be longer than 128 characters", k))
	}
	if !regexp.MustCompile(`[a-zA-Z0-9_.,@/:#-]+`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only alphanumeric characters, underscores, dots, commas, arobases, slashes, colons, hashes and hyphens allowed in %q", k))
	}
	return
}

func validateDynamoDbTableAttributes(d *schema.ResourceDiff) error {
	// Collect all indexed attributes
	primaryHashKey := d.Get("hash_key").(string)
	indexedAttributes := map[string]bool{
		primaryHashKey: true,
	}
	if v, ok := d.GetOk("range_key"); ok {
		indexedAttributes[v.(string)] = true
	}
	if v, ok := d.GetOk("local_secondary_index"); ok {
		indexes := v.(*schema.Set).List()
		for _, idx := range indexes {
			index := idx.(map[string]interface{})
			rangeKey := index["range_key"].(string)
			indexedAttributes[rangeKey] = true
		}
	}
	if v, ok := d.GetOk("global_secondary_index"); ok {
		indexes := v.(*schema.Set).List()
		for _, idx := range indexes {
			index := idx.(map[string]interface{})

			hashKey := index["hash_key"].(string)
			indexedAttributes[hashKey] = true

			if rk, ok := index["range_key"]; ok {
				indexedAttributes[rk.(string)] = true
			}
		}
	}

	// Check if all indexed attributes have an attribute definition
	attributes := d.Get("attribute").(*schema.Set).List()
	missingAttrDefs := []string{}
	for _, attr := range attributes {
		attribute := attr.(map[string]interface{})
		attrName := attribute["name"].(string)

		if _, ok := indexedAttributes[attrName]; !ok {
			missingAttrDefs = append(missingAttrDefs, attrName)
		}
	}

	if len(missingAttrDefs) > 0 {
		return fmt.Errorf("All attributes must be indexed. Unused attributes: %q", missingAttrDefs)
	}

	return nil
}

func validateLaunchTemplateName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) < 3 {
		errors = append(errors, fmt.Errorf("%q cannot be less than 3 characters", k))
	} else if strings.HasSuffix(k, "prefix") && len(value) > 99 {
		errors = append(errors, fmt.Errorf("%q cannot be longer than 99 characters, name is limited to 125", k))
	} else if !strings.HasSuffix(k, "prefix") && len(value) > 125 {
		errors = append(errors, fmt.Errorf("%q cannot be longer than 125 characters", k))
	} else if !regexp.MustCompile(`^[0-9a-zA-Z()./_\-]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf("%q can only alphanumeric characters and ()./_- symbols", k))
	}
	return
}

func validateLaunchTemplateId(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) < 1 {
		errors = append(errors, fmt.Errorf("%q cannot be shorter than 1 character", k))
	} else if len(value) > 255 {
		errors = append(errors, fmt.Errorf("%q cannot be longer than 255 characters", k))
	} else if !regexp.MustCompile(`^lt\-[a-z0-9]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q must begin with 'lt-' and be comprised of only alphanumeric characters: %v", k, value))
	}
	return
}

func validateNeptuneParamGroupName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[0-9a-z-]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only lowercase alphanumeric characters and hyphens allowed in %q", k))
	}
	if !regexp.MustCompile(`^[a-z]`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"first character of %q must be a letter", k))
	}
	if regexp.MustCompile(`--`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q cannot contain two consecutive hyphens", k))
	}
	if regexp.MustCompile(`-$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q cannot end with a hyphen", k))
	}
	if len(value) > 255 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be greater than 255 characters", k))
	}
	return
}

func validateNeptuneParamGroupNamePrefix(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[0-9a-z-]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only lowercase alphanumeric characters and hyphens allowed in %q", k))
	}
	if !regexp.MustCompile(`^[a-z]`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"first character of %q must be a letter", k))
	}
	if regexp.MustCompile(`--`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q cannot contain two consecutive hyphens", k))
	}
	prefixMaxLength := 255 - resource.UniqueIDSuffixLength
	if len(value) > prefixMaxLength {
		errors = append(errors, fmt.Errorf(
			"%q cannot be greater than %d characters", k, prefixMaxLength))
	}
	return
}

func validateNeptuneEventSubscriptionName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[0-9A-Za-z-]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only alphanumeric characters and hyphens allowed in %q", k))
	}
	if len(value) > 255 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be greater than 255 characters", k))
	}
	return
}

func validateNeptuneEventSubscriptionNamePrefix(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[0-9A-Za-z-]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only alphanumeric characters and hyphens allowed in %q", k))
	}
	prefixMaxLength := 255 - resource.UniqueIDSuffixLength
	if len(value) > prefixMaxLength {
		errors = append(errors, fmt.Errorf(
			"%q cannot be greater than %d characters", k, prefixMaxLength))
	}
	return
}

func validateCloudFrontPublicKeyName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[0-9A-Za-z_-]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only alphanumeric characters, underscores and hyphens allowed in %q", k))
	}
	if len(value) > 128 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be greater than 128 characters", k))
	}
	return
}

func validateCloudFrontPublicKeyNamePrefix(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[0-9A-Za-z_-]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only alphanumeric characters, underscores and hyphens allowed in %q", k))
	}
	prefixMaxLength := 128 - resource.UniqueIDSuffixLength
	if len(value) > prefixMaxLength {
		errors = append(errors, fmt.Errorf(
			"%q cannot be greater than %d characters", k, prefixMaxLength))
	}
	return
}
