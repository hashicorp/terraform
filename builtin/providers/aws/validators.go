package aws

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/hashicorp/terraform/helper/schema"
)

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

func validateDbParamGroupNamePrefix(v interface{}, k string) (ws []string, errors []error) {
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
	if len(value) > 255 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be greater than 226 characters", k))
	}
	return
}

func validateStreamViewType(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	viewTypes := map[string]bool{
		"KEYS_ONLY":          true,
		"NEW_IMAGE":          true,
		"OLD_IMAGE":          true,
		"NEW_AND_OLD_IMAGES": true,
	}

	if !viewTypes[value] {
		errors = append(errors, fmt.Errorf("%q must be a valid DynamoDB StreamViewType", k))
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

func validateMaxLength(length int) schema.SchemaValidateFunc {
	return func(v interface{}, k string) (ws []string, errors []error) {
		value := v.(string)
		if len(value) > length {
			errors = append(errors, fmt.Errorf(
				"%q cannot be longer than %d characters: %q", k, length, value))
		}
		return
	}
}

func validateIntegerInRange(min, max int) schema.SchemaValidateFunc {
	return func(v interface{}, k string) (ws []string, errors []error) {
		value := v.(int)
		if value < min {
			errors = append(errors, fmt.Errorf(
				"%q cannot be lower than %d: %d", k, min, value))
		}
		if value > max {
			errors = append(errors, fmt.Errorf(
				"%q cannot be higher than %d: %d", k, max, value))
		}
		return
	}
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
	pattern := `^(arn:[\w-]+:lambda:)?([a-z]{2}-[a-z]+-\d{1}:)?(\d{12}:)?(function:)?([a-zA-Z0-9-_]+)(:(\$LATEST|[a-zA-Z0-9-_]+))?$`
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
			"%q must contain a valid network CIDR, expected %q, got %q",
			k, ipnet, value))
	}

	return
}

func validateHTTPMethod(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	validMethods := map[string]bool{
		"ANY":     true,
		"DELETE":  true,
		"GET":     true,
		"HEAD":    true,
		"OPTIONS": true,
		"PATCH":   true,
		"POST":    true,
		"PUT":     true,
	}

	if _, ok := validMethods[value]; !ok {
		errors = append(errors, fmt.Errorf(
			"%q contains an invalid method %q. Valid methods are either %q, %q, %q, %q, %q, %q, %q, or %q.",
			k, value, "ANY", "DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT"))
	}
	return
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

func validateS3BucketLifecycleStorageClass(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if value != s3.TransitionStorageClassStandardIa && value != s3.TransitionStorageClassGlacier {
		errors = append(errors, fmt.Errorf(
			"%q must be one of '%q', '%q'", k, s3.TransitionStorageClassStandardIa, s3.TransitionStorageClassGlacier))
	}

	return
}

func validateS3BucketReplicationRuleId(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) > 255 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be longer than 255 characters: %q", k, value))
	}

	return
}

func validateS3BucketReplicationRulePrefix(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) > 1024 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be longer than 1024 characters: %q", k, value))
	}

	return
}

func validateS3BucketReplicationDestinationStorageClass(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if value != s3.StorageClassStandard && value != s3.StorageClassStandardIa && value != s3.StorageClassReducedRedundancy {
		errors = append(errors, fmt.Errorf(
			"%q must be one of '%q', '%q' or '%q'", k, s3.StorageClassStandard, s3.StorageClassStandardIa, s3.StorageClassReducedRedundancy))
	}

	return
}

func validateS3BucketReplicationRuleStatus(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if value != s3.ReplicationRuleStatusEnabled && value != s3.ReplicationRuleStatusDisabled {
		errors = append(errors, fmt.Errorf(
			"%q must be one of '%q' or '%q'", k, s3.ReplicationRuleStatusEnabled, s3.ReplicationRuleStatusDisabled))
	}

	return
}

func validateS3BucketLifecycleRuleId(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) > 255 {
		errors = append(errors, fmt.Errorf(
			"%q cannot exceed 255 characters", k))
	}
	return
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

func validateApiGatewayIntegrationPassthroughBehavior(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if value != "WHEN_NO_MATCH" && value != "WHEN_NO_TEMPLATES" && value != "NEVER" {
		errors = append(errors, fmt.Errorf(
			"%q must be one of 'WHEN_NO_MATCH', 'WHEN_NO_TEMPLATES', 'NEVER'", k))
	}
	return
}

func validateJsonString(v interface{}, k string) (ws []string, errors []error) {
	if _, err := normalizeJsonString(v); err != nil {
		errors = append(errors, fmt.Errorf("%q contains an invalid JSON: %s", k, err))
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
	if _, err := normalizeJsonString(v); err != nil {
		errors = append(errors, fmt.Errorf("%q contains an invalid JSON: %s", k, err))
	}
	return
}

func validateCloudFormationTemplate(v interface{}, k string) (ws []string, errors []error) {
	if looksLikeJsonString(v) {
		if _, err := normalizeJsonString(v); err != nil {
			errors = append(errors, fmt.Errorf("%q contains an invalid JSON: %s", k, err))
		}
	} else {
		if _, err := checkYamlString(v); err != nil {
			errors = append(errors, fmt.Errorf("%q contains an invalid YAML: %s", k, err))
		}
	}
	return
}

func validateApiGatewayIntegrationType(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	validTypes := map[string]bool{
		"AWS":        true,
		"AWS_PROXY":  true,
		"HTTP":       true,
		"HTTP_PROXY": true,
		"MOCK":       true,
	}

	if _, ok := validTypes[value]; !ok {
		errors = append(errors, fmt.Errorf(
			"%q contains an invalid integration type %q. Valid types are either %q, %q, %q, %q, or %q.",
			k, value, "AWS", "AWS_PROXY", "HTTP", "HTTP_PROXY", "MOCK"))
	}
	return
}

func validateApiGatewayIntegrationContentHandling(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	validTypes := map[string]bool{
		"CONVERT_TO_BINARY": true,
		"CONVERT_TO_TEXT":   true,
	}

	if _, ok := validTypes[value]; !ok {
		errors = append(errors, fmt.Errorf(
			"%q contains an invalid integration type %q. Valid types are either %q or %q.",
			k, value, "CONVERT_TO_BINARY", "CONVERT_TO_TEXT"))
	}
	return
}

func validateSQSQueueName(v interface{}, k string) (errors []error) {
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

func validateSNSSubscriptionProtocol(v interface{}, k string) (ws []string, errors []error) {
	value := strings.ToLower(v.(string))
	forbidden := []string{"email", "sms"}
	for _, f := range forbidden {
		if strings.Contains(value, f) {
			errors = append(
				errors,
				fmt.Errorf("Unsupported protocol (%s) for SNS Topic", value),
			)
		}
	}
	return
}

func validateSecurityRuleType(v interface{}, k string) (ws []string, errors []error) {
	value := strings.ToLower(v.(string))

	validTypes := map[string]bool{
		"ingress": true,
		"egress":  true,
	}

	if _, ok := validTypes[value]; !ok {
		errors = append(errors, fmt.Errorf(
			"%q contains an invalid Security Group Rule type %q. Valid types are either %q or %q.",
			k, value, "ingress", "egress"))
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

func validateRoute53RecordType(v interface{}, k string) (ws []string, errors []error) {
	// Valid Record types
	// SOA, A, TXT, NS, CNAME, MX, NAPTR, PTR, SRV, SPF, AAAA
	validTypes := map[string]struct{}{
		"SOA":   {},
		"A":     {},
		"TXT":   {},
		"NS":    {},
		"CNAME": {},
		"MX":    {},
		"NAPTR": {},
		"PTR":   {},
		"SRV":   {},
		"SPF":   {},
		"AAAA":  {},
	}

	value := v.(string)
	if _, ok := validTypes[value]; !ok {
		errors = append(errors, fmt.Errorf(
			"%q must be one of [SOA, A, TXT, NS, CNAME, MX, NAPTR, PTR, SRV, SPF, AAAA]", k))
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

// Validates that an Ecs placement strategy is set correctly
// Takes type, and field as strings
func validateAwsEcsPlacementStrategy(stratType, stratField string) error {
	switch stratType {
	case "random":
		// random does not need the field attribute set, could error, but it isn't read at the API level
		return nil
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

func validateAwsEmrEbsVolumeType(v interface{}, k string) (ws []string, errors []error) {
	validTypes := map[string]struct{}{
		"gp2":      {},
		"io1":      {},
		"standard": {},
	}

	value := v.(string)

	if _, ok := validTypes[value]; !ok {
		errors = append(errors, fmt.Errorf(
			"%q must be one of ['gp2', 'io1', 'standard']", k))
	}
	return
}

func validateSfnActivityName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) > 80 {
		errors = append(errors, fmt.Errorf("%q cannot be longer than 80 characters", k))
	}

	return
}

func validateSfnStateMachineDefinition(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) > 1048576 {
		errors = append(errors, fmt.Errorf("%q cannot be longer than 1048576 characters", k))
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

func validateAppautoscalingScalableDimension(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	dimensions := map[string]bool{
		"ecs:service:DesiredCount":                     true,
		"ec2:spot-fleet-request:TargetCapacity":        true,
		"elasticmapreduce:instancegroup:InstanceCount": true,
	}

	if !dimensions[value] {
		errors = append(errors, fmt.Errorf("%q must be a valid scalable dimension value: %q", k, value))
	}
	return
}

func validateAppautoscalingServiceNamespace(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	namespaces := map[string]bool{
		"ecs":              true,
		"ec2":              true,
		"elasticmapreduce": true,
	}

	if !namespaces[value] {
		errors = append(errors, fmt.Errorf("%q must be a valid service namespace value: %q", k, value))
	}
	return
}

func validateConfigRuleSourceOwner(v interface{}, k string) (ws []string, errors []error) {
	validOwners := []string{
		"CUSTOM_LAMBDA",
		"AWS",
	}
	owner := v.(string)
	for _, o := range validOwners {
		if owner == o {
			return
		}
	}
	errors = append(errors, fmt.Errorf(
		"%q contains an invalid owner %q. Valid owners are %q.",
		k, owner, validOwners))
	return
}

func validateConfigExecutionFrequency(v interface{}, k string) (ws []string, errors []error) {
	validFrequencies := []string{
		"One_Hour",
		"Three_Hours",
		"Six_Hours",
		"Twelve_Hours",
		"TwentyFour_Hours",
	}
	frequency := v.(string)
	for _, f := range validFrequencies {
		if frequency == f {
			return
		}
	}
	errors = append(errors, fmt.Errorf(
		"%q contains an invalid frequency %q. Valid frequencies are %q.",
		k, frequency, validFrequencies))
	return
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

func validateApiGatewayApiKeyValue(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) < 30 {
		errors = append(errors, fmt.Errorf(
			"%q must be at least 30 characters long", k))
	}
	if len(value) > 128 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be longer than 128 characters", k))
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

func validateApiGatewayUsagePlanQuotaSettingsPeriod(v interface{}, k string) (ws []string, errors []error) {
	validPeriods := []string{
		apigateway.QuotaPeriodTypeDay,
		apigateway.QuotaPeriodTypeWeek,
		apigateway.QuotaPeriodTypeMonth,
	}
	period := v.(string)
	for _, f := range validPeriods {
		if period == f {
			return
		}
	}
	errors = append(errors, fmt.Errorf(
		"%q contains an invalid period %q. Valid period are %q.",
		k, period, validPeriods))
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

func validateAwsAlbTargetGroupName(v interface{}, k string) (ws []string, errors []error) {
	name := v.(string)
	if len(name) > 32 {
		errors = append(errors, fmt.Errorf("%q (%q) cannot be longer than '32' characters", k, name))
	}
	return
}

func validateAwsAlbTargetGroupNamePrefix(v interface{}, k string) (ws []string, errors []error) {
	name := v.(string)
	if len(name) > 32 {
		errors = append(errors, fmt.Errorf("%q (%q) cannot be longer than '6' characters", k, name))
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

func validateCognitoIdentityPoolName(v interface{}, k string) (ws []string, errors []error) {
	val := v.(string)
	if !regexp.MustCompile("^[\\w _]+$").MatchString(val) {
		errors = append(errors, fmt.Errorf("%q must contain only alphanumeric caracters and spaces", k))
	}

	return
}

func validateCognitoProviderDeveloperName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) > 100 {
		errors = append(errors, fmt.Errorf("%q cannot be longer than 100 caracters", k))
	}

	if !regexp.MustCompile("^[\\w._-]+$").MatchString(value) {
		errors = append(errors, fmt.Errorf("%q must contain only alphanumeric caracters, dots, underscores and hyphens", k))
	}

	return
}

func validateCognitoSupportedLoginProviders(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) < 1 {
		errors = append(errors, fmt.Errorf("%q cannot be less than 1 character", k))
	}

	if len(value) > 128 {
		errors = append(errors, fmt.Errorf("%q cannot be longer than 128 caracters", k))
	}

	if !regexp.MustCompile("^[\\w.;_/-]+$").MatchString(value) {
		errors = append(errors, fmt.Errorf("%q must contain only alphanumeric caracters, dots, semicolons, underscores, slashes and hyphens", k))
	}

	return
}

func validateCognitoIdentityProvidersClientId(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) < 1 {
		errors = append(errors, fmt.Errorf("%q cannot be less than 1 character", k))
	}

	if len(value) > 128 {
		errors = append(errors, fmt.Errorf("%q cannot be longer than 128 caracters", k))
	}

	if !regexp.MustCompile("^[\\w_]+$").MatchString(value) {
		errors = append(errors, fmt.Errorf("%q must contain only alphanumeric caracters and underscores", k))
	}

	return
}

func validateCognitoIdentityProvidersProviderName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) < 1 {
		errors = append(errors, fmt.Errorf("%q cannot be less than 1 character", k))
	}

	if len(value) > 128 {
		errors = append(errors, fmt.Errorf("%q cannot be longer than 128 caracters", k))
	}

	if !regexp.MustCompile("^[\\w._:/-]+$").MatchString(value) {
		errors = append(errors, fmt.Errorf("%q must contain only alphanumeric caracters, dots, underscores, colons, slashes and hyphens", k))
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

func validateIamRoleDescription(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	if len(value) > 1000 {
		errors = append(errors, fmt.Errorf("%q cannot be longer than 1000 caracters", k))
	}

	if !regexp.MustCompile(`[\p{L}\p{M}\p{Z}\p{S}\p{N}\p{P}]*`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"Only alphanumeric & accented characters allowed in %q: %q (Must satisfy regular expression pattern: [\\p{L}\\p{M}\\p{Z}\\p{S}\\p{N}\\p{P}]*)",
			k, value))
	}
	return
}

func validateSsmParameterType(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	types := map[string]bool{
		"String":       true,
		"StringList":   true,
		"SecureString": true,
	}

	if !types[value] {
		errors = append(errors, fmt.Errorf("Parameter type %s is invalid. Valid types are String, StringList or SecureString", value))
	}
	return
}
