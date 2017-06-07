package aws

import (
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/service/s3"
)

func TestValidateEcrRepositoryName(t *testing.T) {
	validNames := []string{
		"nginx-web-app",
		"project-a/nginx-web-app",
		"domain.ltd/nginx-web-app",
		"3chosome-thing.com/01different-pattern",
		"0123456789/999999999",
		"double/forward/slash",
		"000000000000000",
	}
	for _, v := range validNames {
		_, errors := validateEcrRepositoryName(v, "name")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid ECR repository name: %q", v, errors)
		}
	}

	invalidNames := []string{
		// length > 256
		"3cho_some-thing.com/01different.-_pattern01different.-_pattern01diff" +
			"erent.-_pattern01different.-_pattern01different.-_pattern01different" +
			".-_pattern01different.-_pattern01different.-_pattern01different.-_pa" +
			"ttern01different.-_pattern01different.-_pattern234567",
		// length < 2
		"i",
		"special@character",
		"different+special=character",
		"double//slash",
		"double..dot",
		"/slash-at-the-beginning",
		"slash-at-the-end/",
	}
	for _, v := range invalidNames {
		_, errors := validateEcrRepositoryName(v, "name")
		if len(errors) == 0 {
			t.Fatalf("%q should be an invalid ECR repository name", v)
		}
	}
}

func TestValidateCloudWatchEventRuleName(t *testing.T) {
	validNames := []string{
		"HelloWorl_d",
		"hello-world",
		"hello.World0125",
	}
	for _, v := range validNames {
		_, errors := validateCloudWatchEventRuleName(v, "name")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid CW event rule name: %q", v, errors)
		}
	}

	invalidNames := []string{
		"special@character",
		"slash/in-the-middle",
		// Length > 64
		"TooLooooooooooooooooooooooooooooooooooooooooooooooooooooooongName",
	}
	for _, v := range invalidNames {
		_, errors := validateCloudWatchEventRuleName(v, "name")
		if len(errors) == 0 {
			t.Fatalf("%q should be an invalid CW event rule name", v)
		}
	}
}

func TestValidateLambdaFunctionName(t *testing.T) {
	validNames := []string{
		"arn:aws:lambda:us-west-2:123456789012:function:ThumbNail",
		"arn:aws-us-gov:lambda:us-west-2:123456789012:function:ThumbNail",
		"FunctionName",
		"function-name",
	}
	for _, v := range validNames {
		_, errors := validateLambdaFunctionName(v, "name")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid Lambda function name: %q", v, errors)
		}
	}

	invalidNames := []string{
		"/FunctionNameWithSlash",
		"function.name.with.dots",
		// length > 140
		"arn:aws:lambda:us-west-2:123456789012:function:TooLoooooo" +
			"ooooooooooooooooooooooooooooooooooooooooooooooooooooooo" +
			"ooooooooooooooooongFunctionName",
	}
	for _, v := range invalidNames {
		_, errors := validateLambdaFunctionName(v, "name")
		if len(errors) == 0 {
			t.Fatalf("%q should be an invalid Lambda function name", v)
		}
	}
}

func TestValidateLambdaQualifier(t *testing.T) {
	validNames := []string{
		"123",
		"prod",
		"PROD",
		"MyTestEnv",
		"contains-dashes",
		"contains_underscores",
		"$LATEST",
	}
	for _, v := range validNames {
		_, errors := validateLambdaQualifier(v, "name")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid Lambda function qualifier: %q", v, errors)
		}
	}

	invalidNames := []string{
		// No ARNs allowed
		"arn:aws:lambda:us-west-2:123456789012:function:prod",
		// length > 128
		"TooLooooooooooooooooooooooooooooooooooooooooooooooooooo" +
			"ooooooooooooooooooooooooooooooooooooooooooooooooooo" +
			"oooooooooooongQualifier",
	}
	for _, v := range invalidNames {
		_, errors := validateLambdaQualifier(v, "name")
		if len(errors) == 0 {
			t.Fatalf("%q should be an invalid Lambda function qualifier", v)
		}
	}
}

func TestValidateLambdaPermissionAction(t *testing.T) {
	validNames := []string{
		"lambda:*",
		"lambda:InvokeFunction",
		"*",
	}
	for _, v := range validNames {
		_, errors := validateLambdaPermissionAction(v, "action")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid Lambda permission action: %q", v, errors)
		}
	}

	invalidNames := []string{
		"yada",
		"lambda:123",
		"*:*",
		"lambda:Invoke*",
	}
	for _, v := range invalidNames {
		_, errors := validateLambdaPermissionAction(v, "action")
		if len(errors) == 0 {
			t.Fatalf("%q should be an invalid Lambda permission action", v)
		}
	}
}

func TestValidateAwsAccountId(t *testing.T) {
	validNames := []string{
		"123456789012",
		"999999999999",
	}
	for _, v := range validNames {
		_, errors := validateAwsAccountId(v, "account_id")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid AWS Account ID: %q", v, errors)
		}
	}

	invalidNames := []string{
		"12345678901",   // too short
		"1234567890123", // too long
		"invalid",
		"x123456789012",
	}
	for _, v := range invalidNames {
		_, errors := validateAwsAccountId(v, "account_id")
		if len(errors) == 0 {
			t.Fatalf("%q should be an invalid AWS Account ID", v)
		}
	}
}

func TestValidateArn(t *testing.T) {
	v := ""
	_, errors := validateArn(v, "arn")
	if len(errors) != 0 {
		t.Fatalf("%q should not be validated as an ARN: %q", v, errors)
	}

	validNames := []string{
		"arn:aws:elasticbeanstalk:us-east-1:123456789012:environment/My App/MyEnvironment", // Beanstalk
		"arn:aws:iam::123456789012:user/David",                                             // IAM User
		"arn:aws:rds:eu-west-1:123456789012:db:mysql-db",                                   // RDS
		"arn:aws:s3:::my_corporate_bucket/exampleobject.png",                               // S3 object
		"arn:aws:events:us-east-1:319201112229:rule/rule_name",                             // CloudWatch Rule
		"arn:aws:lambda:eu-west-1:319201112229:function:myCustomFunction",                  // Lambda function
		"arn:aws:lambda:eu-west-1:319201112229:function:myCustomFunction:Qualifier",        // Lambda func qualifier
		"arn:aws-us-gov:s3:::corp_bucket/object.png",                                       // GovCloud ARN
		"arn:aws-us-gov:kms:us-gov-west-1:123456789012:key/some-uuid-abc123",               // GovCloud KMS ARN
	}
	for _, v := range validNames {
		_, errors := validateArn(v, "arn")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid ARN: %q", v, errors)
		}
	}

	invalidNames := []string{
		"arn",
		"123456789012",
		"arn:aws",
		"arn:aws:logs",
		"arn:aws:logs:region:*:*",
	}
	for _, v := range invalidNames {
		_, errors := validateArn(v, "arn")
		if len(errors) == 0 {
			t.Fatalf("%q should be an invalid ARN", v)
		}
	}
}

func TestValidatePolicyStatementId(t *testing.T) {
	validNames := []string{
		"YadaHereAndThere",
		"Valid-5tatement_Id",
		"1234",
	}
	for _, v := range validNames {
		_, errors := validatePolicyStatementId(v, "statement_id")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid Statement ID: %q", v, errors)
		}
	}

	invalidNames := []string{
		"Invalid/StatementId/with/slashes",
		"InvalidStatementId.with.dots",
		// length > 100
		"TooooLoooooooooooooooooooooooooooooooooooooooooooo" +
			"ooooooooooooooooooooooooooooooooooooooooStatementId",
	}
	for _, v := range invalidNames {
		_, errors := validatePolicyStatementId(v, "statement_id")
		if len(errors) == 0 {
			t.Fatalf("%q should be an invalid Statement ID", v)
		}
	}
}

func TestValidateCIDRNetworkAddress(t *testing.T) {
	cases := []struct {
		CIDR              string
		ExpectedErrSubstr string
	}{
		{"notacidr", `must contain a valid CIDR`},
		{"10.0.1.0/16", `must contain a valid network CIDR`},
		{"10.0.1.0/24", ``},
	}

	for i, tc := range cases {
		_, errs := validateCIDRNetworkAddress(tc.CIDR, "foo")
		if tc.ExpectedErrSubstr == "" {
			if len(errs) != 0 {
				t.Fatalf("%d/%d: Expected no error, got errs: %#v",
					i+1, len(cases), errs)
			}
		} else {
			if len(errs) != 1 {
				t.Fatalf("%d/%d: Expected 1 err containing %q, got %d errs",
					i+1, len(cases), tc.ExpectedErrSubstr, len(errs))
			}
			if !strings.Contains(errs[0].Error(), tc.ExpectedErrSubstr) {
				t.Fatalf("%d/%d: Expected err: %q, to include %q",
					i+1, len(cases), errs[0], tc.ExpectedErrSubstr)
			}
		}
	}
}

func TestValidateHTTPMethod(t *testing.T) {
	type testCases struct {
		Value    string
		ErrCount int
	}

	invalidCases := []testCases{
		{
			Value:    "incorrect",
			ErrCount: 1,
		},
		{
			Value:    "delete",
			ErrCount: 1,
		},
	}

	for _, tc := range invalidCases {
		_, errors := validateHTTPMethod(tc.Value, "http_method")
		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected %q to trigger a validation error.", tc.Value)
		}
	}

	validCases := []testCases{
		{
			Value:    "ANY",
			ErrCount: 0,
		},
		{
			Value:    "DELETE",
			ErrCount: 0,
		},
		{
			Value:    "OPTIONS",
			ErrCount: 0,
		},
	}

	for _, tc := range validCases {
		_, errors := validateHTTPMethod(tc.Value, "http_method")
		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected %q not to trigger a validation error.", tc.Value)
		}
	}
}

func TestValidateLogMetricFilterName(t *testing.T) {
	validNames := []string{
		"YadaHereAndThere",
		"Valid-5Metric_Name",
		"This . is also %% valid@!)+(",
		"1234",
		strings.Repeat("W", 512),
	}
	for _, v := range validNames {
		_, errors := validateLogMetricFilterName(v, "name")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid Log Metric Filter Name: %q", v, errors)
		}
	}

	invalidNames := []string{
		"Here is a name with: colon",
		"and here is another * invalid name",
		"*",
		// length > 512
		strings.Repeat("W", 513),
	}
	for _, v := range invalidNames {
		_, errors := validateLogMetricFilterName(v, "name")
		if len(errors) == 0 {
			t.Fatalf("%q should be an invalid Log Metric Filter Name", v)
		}
	}
}

func TestValidateLogMetricTransformationName(t *testing.T) {
	validNames := []string{
		"YadaHereAndThere",
		"Valid-5Metric_Name",
		"This . is also %% valid@!)+(",
		"1234",
		"",
		strings.Repeat("W", 255),
	}
	for _, v := range validNames {
		_, errors := validateLogMetricFilterTransformationName(v, "name")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid Log Metric Filter Transformation Name: %q", v, errors)
		}
	}

	invalidNames := []string{
		"Here is a name with: colon",
		"and here is another * invalid name",
		"also $ invalid",
		"*",
		// length > 255
		strings.Repeat("W", 256),
	}
	for _, v := range invalidNames {
		_, errors := validateLogMetricFilterTransformationName(v, "name")
		if len(errors) == 0 {
			t.Fatalf("%q should be an invalid Log Metric Filter Transformation Name", v)
		}
	}
}

func TestValidateLogGroupName(t *testing.T) {
	validNames := []string{
		"ValidLogGroupName",
		"ValidLogGroup.Name",
		"valid/Log-group",
		"1234",
		"YadaValid#0123",
		"Also_valid-name",
		strings.Repeat("W", 512),
	}
	for _, v := range validNames {
		_, errors := validateLogGroupName(v, "name")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid Log Group name: %q", v, errors)
		}
	}

	invalidNames := []string{
		"Here is a name with: colon",
		"and here is another * invalid name",
		"also $ invalid",
		"This . is also %% invalid@!)+(",
		"*",
		"",
		// length > 512
		strings.Repeat("W", 513),
	}
	for _, v := range invalidNames {
		_, errors := validateLogGroupName(v, "name")
		if len(errors) == 0 {
			t.Fatalf("%q should be an invalid Log Group name", v)
		}
	}
}

func TestValidateLogGroupNamePrefix(t *testing.T) {
	validNames := []string{
		"ValidLogGroupName",
		"ValidLogGroup.Name",
		"valid/Log-group",
		"1234",
		"YadaValid#0123",
		"Also_valid-name",
		strings.Repeat("W", 483),
	}
	for _, v := range validNames {
		_, errors := validateLogGroupNamePrefix(v, "name_prefix")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid Log Group name prefix: %q", v, errors)
		}
	}

	invalidNames := []string{
		"Here is a name with: colon",
		"and here is another * invalid name",
		"also $ invalid",
		"This . is also %% invalid@!)+(",
		"*",
		"",
		// length > 483
		strings.Repeat("W", 484),
	}
	for _, v := range invalidNames {
		_, errors := validateLogGroupNamePrefix(v, "name_prefix")
		if len(errors) == 0 {
			t.Fatalf("%q should be an invalid Log Group name prefix", v)
		}
	}
}

func TestValidateS3BucketLifecycleTimestamp(t *testing.T) {
	validDates := []string{
		"2016-01-01",
		"2006-01-02",
	}

	for _, v := range validDates {
		_, errors := validateS3BucketLifecycleTimestamp(v, "date")
		if len(errors) != 0 {
			t.Fatalf("%q should be valid date: %q", v, errors)
		}
	}

	invalidDates := []string{
		"Jan 01 2016",
		"20160101",
	}

	for _, v := range invalidDates {
		_, errors := validateS3BucketLifecycleTimestamp(v, "date")
		if len(errors) == 0 {
			t.Fatalf("%q should be invalid date", v)
		}
	}
}

func TestValidateS3BucketLifecycleStorageClass(t *testing.T) {
	validStorageClass := []string{
		"STANDARD_IA",
		"GLACIER",
	}

	for _, v := range validStorageClass {
		_, errors := validateS3BucketLifecycleStorageClass(v, "storage_class")
		if len(errors) != 0 {
			t.Fatalf("%q should be valid storage class: %q", v, errors)
		}
	}

	invalidStorageClass := []string{
		"STANDARD",
		"1234",
	}
	for _, v := range invalidStorageClass {
		_, errors := validateS3BucketLifecycleStorageClass(v, "storage_class")
		if len(errors) == 0 {
			t.Fatalf("%q should be invalid storage class", v)
		}
	}
}

func TestValidateS3BucketReplicationRuleId(t *testing.T) {
	validId := []string{
		"YadaHereAndThere",
		"Valid-5Rule_ID",
		"This . is also %% valid@!)+*(:ID",
		"1234",
		strings.Repeat("W", 255),
	}
	for _, v := range validId {
		_, errors := validateS3BucketReplicationRuleId(v, "id")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid lifecycle rule id: %q", v, errors)
		}
	}

	invalidId := []string{
		// length > 255
		strings.Repeat("W", 256),
	}
	for _, v := range invalidId {
		_, errors := validateS3BucketReplicationRuleId(v, "id")
		if len(errors) == 0 {
			t.Fatalf("%q should be an invalid replication configuration rule id", v)
		}
	}
}

func TestValidateS3BucketReplicationRulePrefix(t *testing.T) {
	validId := []string{
		"YadaHereAndThere",
		"Valid-5Rule_ID",
		"This . is also %% valid@!)+*(:ID",
		"1234",
		strings.Repeat("W", 1024),
	}
	for _, v := range validId {
		_, errors := validateS3BucketReplicationRulePrefix(v, "id")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid lifecycle rule id: %q", v, errors)
		}
	}

	invalidId := []string{
		// length > 1024
		strings.Repeat("W", 1025),
	}
	for _, v := range invalidId {
		_, errors := validateS3BucketReplicationRulePrefix(v, "id")
		if len(errors) == 0 {
			t.Fatalf("%q should be an invalid replication configuration rule id", v)
		}
	}
}

func TestValidateS3BucketReplicationDestinationStorageClass(t *testing.T) {
	validStorageClass := []string{
		s3.StorageClassStandard,
		s3.StorageClassStandardIa,
		s3.StorageClassReducedRedundancy,
	}

	for _, v := range validStorageClass {
		_, errors := validateS3BucketReplicationDestinationStorageClass(v, "storage_class")
		if len(errors) != 0 {
			t.Fatalf("%q should be valid storage class: %q", v, errors)
		}
	}

	invalidStorageClass := []string{
		"FOO",
		"1234",
	}
	for _, v := range invalidStorageClass {
		_, errors := validateS3BucketReplicationDestinationStorageClass(v, "storage_class")
		if len(errors) == 0 {
			t.Fatalf("%q should be invalid storage class", v)
		}
	}
}

func TestValidateS3BucketReplicationRuleStatus(t *testing.T) {
	validRuleStatuses := []string{
		s3.ReplicationRuleStatusEnabled,
		s3.ReplicationRuleStatusDisabled,
	}

	for _, v := range validRuleStatuses {
		_, errors := validateS3BucketReplicationRuleStatus(v, "status")
		if len(errors) != 0 {
			t.Fatalf("%q should be valid rule status: %q", v, errors)
		}
	}

	invalidRuleStatuses := []string{
		"FOO",
		"1234",
	}
	for _, v := range invalidRuleStatuses {
		_, errors := validateS3BucketReplicationRuleStatus(v, "status")
		if len(errors) == 0 {
			t.Fatalf("%q should be invalid rule status", v)
		}
	}
}

func TestValidateS3BucketLifecycleRuleId(t *testing.T) {
	validId := []string{
		"YadaHereAndThere",
		"Valid-5Rule_ID",
		"This . is also %% valid@!)+*(:ID",
		"1234",
		strings.Repeat("W", 255),
	}
	for _, v := range validId {
		_, errors := validateS3BucketLifecycleRuleId(v, "id")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid lifecycle rule id: %q", v, errors)
		}
	}

	invalidId := []string{
		// length > 255
		strings.Repeat("W", 256),
	}
	for _, v := range invalidId {
		_, errors := validateS3BucketLifecycleRuleId(v, "id")
		if len(errors) == 0 {
			t.Fatalf("%q should be an invalid lifecycle rule id", v)
		}
	}
}

func TestValidateIntegerInRange(t *testing.T) {
	validIntegers := []int{-259, 0, 1, 5, 999}
	min := -259
	max := 999
	for _, v := range validIntegers {
		_, errors := validateIntegerInRange(min, max)(v, "name")
		if len(errors) != 0 {
			t.Fatalf("%q should be an integer in range (%d, %d): %q", v, min, max, errors)
		}
	}

	invalidIntegers := []int{-260, -99999, 1000, 25678}
	for _, v := range invalidIntegers {
		_, errors := validateIntegerInRange(min, max)(v, "name")
		if len(errors) == 0 {
			t.Fatalf("%q should be an integer outside range (%d, %d)", v, min, max)
		}
	}
}

func TestResourceAWSElastiCacheClusterIdValidation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "tEsting",
			ErrCount: 1,
		},
		{
			Value:    "t.sting",
			ErrCount: 1,
		},
		{
			Value:    "t--sting",
			ErrCount: 1,
		},
		{
			Value:    "1testing",
			ErrCount: 1,
		},
		{
			Value:    "testing-",
			ErrCount: 1,
		},
		{
			Value:    randomString(65),
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateElastiCacheClusterId(tc.Value, "aws_elasticache_cluster_cluster_id")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the ElastiCache Cluster cluster_id to trigger a validation error")
		}
	}
}

func TestValidateDbEventSubscriptionName(t *testing.T) {
	validNames := []string{
		"valid-name",
		"valid02-name",
		"Valid-Name1",
	}
	for _, v := range validNames {
		_, errors := validateDbEventSubscriptionName(v, "name")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid RDS Event Subscription Name: %q", v, errors)
		}
	}

	invalidNames := []string{
		"Here is a name with: colon",
		"and here is another * invalid name",
		"also $ invalid",
		"This . is also %% invalid@!)+(",
		"*",
		"",
		" ",
		"_",
		// length > 255
		strings.Repeat("W", 256),
	}
	for _, v := range invalidNames {
		_, errors := validateDbEventSubscriptionName(v, "name")
		if len(errors) == 0 {
			t.Fatalf("%q should be an invalid RDS Event Subscription Name", v)
		}
	}
}

func TestValidateJsonString(t *testing.T) {
	type testCases struct {
		Value    string
		ErrCount int
	}

	invalidCases := []testCases{
		{
			Value:    `{0:"1"}`,
			ErrCount: 1,
		},
		{
			Value:    `{'abc':1}`,
			ErrCount: 1,
		},
		{
			Value:    `{"def":}`,
			ErrCount: 1,
		},
		{
			Value:    `{"xyz":[}}`,
			ErrCount: 1,
		},
	}

	for _, tc := range invalidCases {
		_, errors := validateJsonString(tc.Value, "json")
		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected %q to trigger a validation error.", tc.Value)
		}
	}

	validCases := []testCases{
		{
			Value:    ``,
			ErrCount: 0,
		},
		{
			Value:    `{}`,
			ErrCount: 0,
		},
		{
			Value:    `{"abc":["1","2"]}`,
			ErrCount: 0,
		},
	}

	for _, tc := range validCases {
		_, errors := validateJsonString(tc.Value, "json")
		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected %q not to trigger a validation error.", tc.Value)
		}
	}
}

func TestValidateIAMPolicyJsonString(t *testing.T) {
	type testCases struct {
		Value    string
		ErrCount int
	}

	invalidCases := []testCases{
		{
			Value:    `{0:"1"}`,
			ErrCount: 1,
		},
		{
			Value:    `{'abc':1}`,
			ErrCount: 1,
		},
		{
			Value:    `{"def":}`,
			ErrCount: 1,
		},
		{
			Value:    `{"xyz":[}}`,
			ErrCount: 1,
		},
		{
			Value:    ``,
			ErrCount: 1,
		},
		{
			Value:    `    {"xyz": "foo"}`,
			ErrCount: 1,
		},
	}

	for _, tc := range invalidCases {
		_, errors := validateIAMPolicyJson(tc.Value, "json")
		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected %q to trigger a validation error.", tc.Value)
		}
	}

	validCases := []testCases{
		{
			Value:    `{}`,
			ErrCount: 0,
		},
		{
			Value:    `{"abc":["1","2"]}`,
			ErrCount: 0,
		},
	}

	for _, tc := range validCases {
		_, errors := validateIAMPolicyJson(tc.Value, "json")
		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected %q not to trigger a validation error.", tc.Value)
		}
	}
}

func TestValidateCloudFormationTemplate(t *testing.T) {
	type testCases struct {
		Value    string
		ErrCount int
	}

	invalidCases := []testCases{
		{
			Value:    `{"abc":"`,
			ErrCount: 1,
		},
		{
			Value:    "abc: [",
			ErrCount: 1,
		},
	}

	for _, tc := range invalidCases {
		_, errors := validateCloudFormationTemplate(tc.Value, "template")
		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected %q to trigger a validation error.", tc.Value)
		}
	}

	validCases := []testCases{
		{
			Value:    `{"abc":"1"}`,
			ErrCount: 0,
		},
		{
			Value:    `abc: 1`,
			ErrCount: 0,
		},
	}

	for _, tc := range validCases {
		_, errors := validateCloudFormationTemplate(tc.Value, "template")
		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected %q not to trigger a validation error.", tc.Value)
		}
	}
}

func TestValidateApiGatewayIntegrationType(t *testing.T) {
	type testCases struct {
		Value    string
		ErrCount int
	}

	invalidCases := []testCases{
		{
			Value:    "incorrect",
			ErrCount: 1,
		},
		{
			Value:    "aws_proxy",
			ErrCount: 1,
		},
	}

	for _, tc := range invalidCases {
		_, errors := validateApiGatewayIntegrationType(tc.Value, "types")
		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected %q to trigger a validation error.", tc.Value)
		}
	}

	validCases := []testCases{
		{
			Value:    "MOCK",
			ErrCount: 0,
		},
		{
			Value:    "AWS_PROXY",
			ErrCount: 0,
		},
	}

	for _, tc := range validCases {
		_, errors := validateApiGatewayIntegrationType(tc.Value, "types")
		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected %q not to trigger a validation error.", tc.Value)
		}
	}
}

func TestValidateSQSQueueName(t *testing.T) {
	validNames := []string{
		"valid-name",
		"valid02-name",
		"Valid-Name1",
		"_",
		"-",
		strings.Repeat("W", 80),
	}
	for _, v := range validNames {
		if errors := validateSQSQueueName(v, "name"); len(errors) > 0 {
			t.Fatalf("%q should be a valid SQS queue Name", v)
		}
	}

	invalidNames := []string{
		"Here is a name with: colon",
		"another * invalid name",
		"also $ invalid",
		"This . is also %% invalid@!)+(",
		"*",
		"",
		" ",
		".",
		strings.Repeat("W", 81), // length > 80
	}
	for _, v := range invalidNames {
		if errors := validateSQSQueueName(v, "name"); len(errors) == 0 {
			t.Fatalf("%q should be an invalid SQS queue Name", v)
		}
	}
}

func TestValidateSQSFifoQueueName(t *testing.T) {
	validNames := []string{
		"valid-name.fifo",
		"valid02-name.fifo",
		"Valid-Name1.fifo",
		"_.fifo",
		"a.fifo",
		"A.fifo",
		"9.fifo",
		"-.fifo",
		fmt.Sprintf("%s.fifo", strings.Repeat("W", 75)),
	}
	for _, v := range validNames {
		if errors := validateSQSFifoQueueName(v, "name"); len(errors) > 0 {
			t.Fatalf("%q should be a valid SQS FIFO queue Name: %v", v, errors)
		}
	}

	invalidNames := []string{
		"Here is a name with: colon",
		"another * invalid name",
		"also $ invalid",
		"This . is also %% invalid@!)+(",
		".fifo",
		"*",
		"",
		" ",
		".",
		strings.Repeat("W", 81), // length > 80
	}
	for _, v := range invalidNames {
		if errors := validateSQSFifoQueueName(v, "name"); len(errors) == 0 {
			t.Fatalf("%q should be an invalid SQS FIFO queue Name: %v", v, errors)
		}
	}
}

func TestValidateSNSSubscriptionProtocol(t *testing.T) {
	validProtocols := []string{
		"lambda",
		"sqs",
		"sqs",
		"application",
		"http",
		"https",
	}
	for _, v := range validProtocols {
		if _, errors := validateSNSSubscriptionProtocol(v, "protocol"); len(errors) > 0 {
			t.Fatalf("%q should be a valid SNS Subscription protocol: %v", v, errors)
		}
	}

	invalidProtocols := []string{
		"Email",
		"email",
		"Email-JSON",
		"email-json",
		"SMS",
		"sms",
	}
	for _, v := range invalidProtocols {
		if _, errors := validateSNSSubscriptionProtocol(v, "protocol"); len(errors) == 0 {
			t.Fatalf("%q should be an invalid SNS Subscription protocol: %v", v, errors)
		}
	}
}

func TestValidateSecurityRuleType(t *testing.T) {
	validTypes := []string{
		"ingress",
		"egress",
	}
	for _, v := range validTypes {
		if _, errors := validateSecurityRuleType(v, "type"); len(errors) > 0 {
			t.Fatalf("%q should be a valid Security Group Rule type: %v", v, errors)
		}
	}

	invalidTypes := []string{
		"foo",
		"ingresss",
	}
	for _, v := range invalidTypes {
		if _, errors := validateSecurityRuleType(v, "type"); len(errors) == 0 {
			t.Fatalf("%q should be an invalid Security Group Rule type: %v", v, errors)
		}
	}
}

func TestValidateOnceAWeekWindowFormat(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			// once a day window format
			Value:    "04:00-05:00",
			ErrCount: 1,
		},
		{
			// invalid day of week
			Value:    "san:04:00-san:05:00",
			ErrCount: 1,
		},
		{
			// invalid hour
			Value:    "sun:24:00-san:25:00",
			ErrCount: 1,
		},
		{
			// invalid min
			Value:    "sun:04:00-sun:04:60",
			ErrCount: 1,
		},
		{
			// valid format
			Value:    "sun:04:00-sun:05:00",
			ErrCount: 0,
		},
		{
			// "Sun" can also be used
			Value:    "Sun:04:00-Sun:05:00",
			ErrCount: 0,
		},
		{
			// valid format
			Value:    "",
			ErrCount: 0,
		},
	}

	for _, tc := range cases {
		_, errors := validateOnceAWeekWindowFormat(tc.Value, "maintenance_window")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected %d validation errors, But got %d errors for \"%s\"", tc.ErrCount, len(errors), tc.Value)
		}
	}
}

func TestValidateOnceADayWindowFormat(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			// once a week window format
			Value:    "sun:04:00-sun:05:00",
			ErrCount: 1,
		},
		{
			// invalid hour
			Value:    "24:00-25:00",
			ErrCount: 1,
		},
		{
			// invalid min
			Value:    "04:00-04:60",
			ErrCount: 1,
		},
		{
			// valid format
			Value:    "04:00-05:00",
			ErrCount: 0,
		},
		{
			// valid format
			Value:    "",
			ErrCount: 0,
		},
	}

	for _, tc := range cases {
		_, errors := validateOnceADayWindowFormat(tc.Value, "backup_window")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected %d validation errors, But got %d errors for \"%s\"", tc.ErrCount, len(errors), tc.Value)
		}
	}
}

func TestValidateRoute53RecordType(t *testing.T) {
	validTypes := []string{
		"AAAA",
		"SOA",
		"A",
		"TXT",
		"CNAME",
		"MX",
		"NAPTR",
		"PTR",
		"SPF",
		"SRV",
		"NS",
	}

	invalidTypes := []string{
		"a",
		"alias",
		"SpF",
		"Txt",
		"AaAA",
	}

	for _, v := range validTypes {
		_, errors := validateRoute53RecordType(v, "route53_record")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid Route53 record type: %v", v, errors)
		}
	}

	for _, v := range invalidTypes {
		_, errors := validateRoute53RecordType(v, "route53_record")
		if len(errors) == 0 {
			t.Fatalf("%q should not be a valid Route53 record type", v)
		}
	}
}

func TestValidateEcsPlacementConstraint(t *testing.T) {
	cases := []struct {
		constType string
		constExpr string
		Err       bool
	}{
		{
			constType: "distinctInstance",
			constExpr: "",
			Err:       false,
		},
		{
			constType: "memberOf",
			constExpr: "",
			Err:       true,
		},
		{
			constType: "distinctInstance",
			constExpr: "expression",
			Err:       false,
		},
		{
			constType: "memberOf",
			constExpr: "expression",
			Err:       false,
		},
	}

	for _, tc := range cases {
		if err := validateAwsEcsPlacementConstraint(tc.constType, tc.constExpr); err != nil && !tc.Err {
			t.Fatalf("Unexpected validation error for \"%s:%s\": %s",
				tc.constType, tc.constExpr, err)
		}

	}
}

func TestValidateEcsPlacementStrategy(t *testing.T) {
	cases := []struct {
		stratType  string
		stratField string
		Err        bool
	}{
		{
			stratType:  "random",
			stratField: "",
			Err:        false,
		},
		{
			stratType:  "spread",
			stratField: "instanceID",
			Err:        false,
		},
		{
			stratType:  "binpack",
			stratField: "cpu",
			Err:        false,
		},
		{
			stratType:  "binpack",
			stratField: "memory",
			Err:        false,
		},
		{
			stratType:  "binpack",
			stratField: "disk",
			Err:        true,
		},
		{
			stratType:  "fakeType",
			stratField: "",
			Err:        true,
		},
	}

	for _, tc := range cases {
		if err := validateAwsEcsPlacementStrategy(tc.stratType, tc.stratField); err != nil && !tc.Err {
			t.Fatalf("Unexpected validation error for \"%s:%s\": %s",
				tc.stratType, tc.stratField, err)
		}
	}
}

func TestValidateStepFunctionActivityName(t *testing.T) {
	validTypes := []string{
		"foo",
		"FooBar123",
	}

	invalidTypes := []string{
		strings.Repeat("W", 81), // length > 80
	}

	for _, v := range validTypes {
		_, errors := validateSfnActivityName(v, "name")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid Step Function Activity name: %v", v, errors)
		}
	}

	for _, v := range invalidTypes {
		_, errors := validateSfnActivityName(v, "name")
		if len(errors) == 0 {
			t.Fatalf("%q should not be a valid Step Function Activity name", v)
		}
	}
}

func TestValidateStepFunctionStateMachineDefinition(t *testing.T) {
	validDefinitions := []string{
		"foobar",
		strings.Repeat("W", 1048576),
	}

	invalidDefinitions := []string{
		strings.Repeat("W", 1048577), // length > 1048576
	}

	for _, v := range validDefinitions {
		_, errors := validateSfnStateMachineDefinition(v, "definition")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid Step Function State Machine definition: %v", v, errors)
		}
	}

	for _, v := range invalidDefinitions {
		_, errors := validateSfnStateMachineDefinition(v, "definition")
		if len(errors) == 0 {
			t.Fatalf("%q should not be a valid Step Function State Machine definition", v)
		}
	}
}

func TestValidateStepFunctionStateMachineName(t *testing.T) {
	validTypes := []string{
		"foo",
		"BAR",
		"FooBar123",
		"FooBar123Baz-_",
	}

	invalidTypes := []string{
		"foo bar",
		"foo<bar>",
		"foo{bar}",
		"foo[bar]",
		"foo*bar",
		"foo?bar",
		"foo#bar",
		"foo%bar",
		"foo\bar",
		"foo^bar",
		"foo|bar",
		"foo~bar",
		"foo$bar",
		"foo&bar",
		"foo,bar",
		"foo:bar",
		"foo;bar",
		"foo/bar",
		strings.Repeat("W", 81), // length > 80
	}

	for _, v := range validTypes {
		_, errors := validateSfnStateMachineName(v, "name")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid Step Function State Machine name: %v", v, errors)
		}
	}

	for _, v := range invalidTypes {
		_, errors := validateSfnStateMachineName(v, "name")
		if len(errors) == 0 {
			t.Fatalf("%q should not be a valid Step Function State Machine name", v)
		}
	}
}

func TestValidateEmrEbsVolumeType(t *testing.T) {
	cases := []struct {
		VolType  string
		ErrCount int
	}{
		{
			VolType:  "gp2",
			ErrCount: 0,
		},
		{
			VolType:  "io1",
			ErrCount: 0,
		},
		{
			VolType:  "standard",
			ErrCount: 0,
		},
		{
			VolType:  "stand",
			ErrCount: 1,
		},
		{
			VolType:  "io",
			ErrCount: 1,
		},
		{
			VolType:  "gp1",
			ErrCount: 1,
		},
		{
			VolType:  "fast-disk",
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateAwsEmrEbsVolumeType(tc.VolType, "volume")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected %d errors, got %d: %s", tc.ErrCount, len(errors), errors)
		}
	}
}

func TestValidateAppautoscalingScalableDimension(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "ecs:service:DesiredCount",
			ErrCount: 0,
		},
		{
			Value:    "ec2:spot-fleet-request:TargetCapacity",
			ErrCount: 0,
		},
		{
			Value:    "ec2:service:DesiredCount",
			ErrCount: 1,
		},
		{
			Value:    "ecs:spot-fleet-request:TargetCapacity",
			ErrCount: 1,
		},
		{
			Value:    "",
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateAppautoscalingScalableDimension(tc.Value, "scalable_dimension")
		if len(errors) != tc.ErrCount {
			t.Fatalf("Scalable Dimension validation failed for value %q: %q", tc.Value, errors)
		}
	}
}

func TestValidateAppautoscalingServiceNamespace(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "ecs",
			ErrCount: 0,
		},
		{
			Value:    "ec2",
			ErrCount: 0,
		},
		{
			Value:    "autoscaling",
			ErrCount: 1,
		},
		{
			Value:    "s3",
			ErrCount: 1,
		},
		{
			Value:    "es",
			ErrCount: 1,
		},
		{
			Value:    "",
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateAppautoscalingServiceNamespace(tc.Value, "service_namespace")
		if len(errors) != tc.ErrCount {
			t.Fatalf("Service Namespace validation failed for value %q: %q", tc.Value, errors)
		}
	}
}

func TestValidateDmsEndpointId(t *testing.T) {
	validIds := []string{
		"tf-test-endpoint-1",
		"tfTestEndpoint",
	}

	for _, s := range validIds {
		_, errors := validateDmsEndpointId(s, "endpoint_id")
		if len(errors) > 0 {
			t.Fatalf("%q should be a valid endpoint id: %v", s, errors)
		}
	}

	invalidIds := []string{
		"tf_test_endpoint_1",
		"tf.test.endpoint.1",
		"tf test endpoint 1",
		"tf-test-endpoint-1!",
		"tf-test-endpoint-1-",
		"tf-test-endpoint--1",
		"tf-test-endpoint-1tf-test-endpoint-1tf-test-endpoint-1tf-test-endpoint-1tf-test-endpoint-1tf-test-endpoint-1tf-test-endpoint-1tf-test-endpoint-1tf-test-endpoint-1tf-test-endpoint-1tf-test-endpoint-1tf-test-endpoint-1tf-test-endpoint-1tf-test-endpoint-1tf-test-endpoint-1",
	}

	for _, s := range invalidIds {
		_, errors := validateDmsEndpointId(s, "endpoint_id")
		if len(errors) == 0 {
			t.Fatalf("%q should not be a valid endpoint id: %v", s, errors)
		}
	}
}

func TestValidateDmsCertificateId(t *testing.T) {
	validIds := []string{
		"tf-test-certificate-1",
		"tfTestEndpoint",
	}

	for _, s := range validIds {
		_, errors := validateDmsCertificateId(s, "certificate_id")
		if len(errors) > 0 {
			t.Fatalf("%q should be a valid certificate id: %v", s, errors)
		}
	}

	invalidIds := []string{
		"tf_test_certificate_1",
		"tf.test.certificate.1",
		"tf test certificate 1",
		"tf-test-certificate-1!",
		"tf-test-certificate-1-",
		"tf-test-certificate--1",
		"tf-test-certificate-1tf-test-certificate-1tf-test-certificate-1tf-test-certificate-1tf-test-certificate-1tf-test-certificate-1tf-test-certificate-1tf-test-certificate-1tf-test-certificate-1tf-test-certificate-1tf-test-certificate-1tf-test-certificate-1tf-test-certificate-1tf-test-certificate-1tf-test-certificate-1",
	}

	for _, s := range invalidIds {
		_, errors := validateDmsEndpointId(s, "certificate_id")
		if len(errors) == 0 {
			t.Fatalf("%q should not be a valid certificate id: %v", s, errors)
		}
	}
}

func TestValidateDmsReplicationInstanceId(t *testing.T) {
	validIds := []string{
		"tf-test-replication-instance-1",
		"tfTestReplicaitonInstance",
	}

	for _, s := range validIds {
		_, errors := validateDmsReplicationInstanceId(s, "replicaiton_instance_id")
		if len(errors) > 0 {
			t.Fatalf("%q should be a valid replication instance id: %v", s, errors)
		}
	}

	invalidIds := []string{
		"tf_test_replication-instance_1",
		"tf.test.replication.instance.1",
		"tf test replication instance 1",
		"tf-test-replication-instance-1!",
		"tf-test-replication-instance-1-",
		"tf-test-replication-instance--1",
		"tf-test-replication-instance-1tf-test-replication-instance-1tf-test-replication-instance-1",
	}

	for _, s := range invalidIds {
		_, errors := validateDmsReplicationInstanceId(s, "replication_instance_id")
		if len(errors) == 0 {
			t.Fatalf("%q should not be a valid replication instance id: %v", s, errors)
		}
	}
}

func TestValidateDmsReplicationSubnetGroupId(t *testing.T) {
	validIds := []string{
		"tf-test-replication-subnet-group-1",
		"tf_test_replication_subnet_group_1",
		"tf.test.replication.subnet.group.1",
		"tf test replication subnet group 1",
		"tfTestReplicationSubnetGroup",
	}

	for _, s := range validIds {
		_, errors := validateDmsReplicationSubnetGroupId(s, "replication_subnet_group_id")
		if len(errors) > 0 {
			t.Fatalf("%q should be a valid replication subnet group id: %v", s, errors)
		}
	}

	invalidIds := []string{
		"default",
		"tf-test-replication-subnet-group-1!",
		"tf-test-replication-subnet-group-1tf-test-replication-subnet-group-1tf-test-replication-subnet-group-1tf-test-replication-subnet-group-1tf-test-replication-subnet-group-1tf-test-replication-subnet-group-1tf-test-replication-subnet-group-1tf-test-replication-subnet-group-1",
	}

	for _, s := range invalidIds {
		_, errors := validateDmsReplicationSubnetGroupId(s, "replication_subnet_group_id")
		if len(errors) == 0 {
			t.Fatalf("%q should not be a valid replication subnet group id: %v", s, errors)
		}
	}
}

func TestValidateDmsReplicationTaskId(t *testing.T) {
	validIds := []string{
		"tf-test-replication-task-1",
		"tfTestReplicationTask",
	}

	for _, s := range validIds {
		_, errors := validateDmsReplicationTaskId(s, "replication_task_id")
		if len(errors) > 0 {
			t.Fatalf("%q should be a valid replication task id: %v", s, errors)
		}
	}

	invalidIds := []string{
		"tf_test_replication_task_1",
		"tf.test.replication.task.1",
		"tf test replication task 1",
		"tf-test-replication-task-1!",
		"tf-test-replication-task-1-",
		"tf-test-replication-task--1",
		"tf-test-replication-task-1tf-test-replication-task-1tf-test-replication-task-1tf-test-replication-task-1tf-test-replication-task-1tf-test-replication-task-1tf-test-replication-task-1tf-test-replication-task-1tf-test-replication-task-1tf-test-replication-task-1",
	}

	for _, s := range invalidIds {
		_, errors := validateDmsReplicationTaskId(s, "replication_task_id")
		if len(errors) == 0 {
			t.Fatalf("%q should not be a valid replication task id: %v", s, errors)
		}
	}
}

func TestValidateAccountAlias(t *testing.T) {
	validAliases := []string{
		"tf-alias",
		"0tf-alias1",
	}

	for _, s := range validAliases {
		_, errors := validateAccountAlias(s, "account_alias")
		if len(errors) > 0 {
			t.Fatalf("%q should be a valid account alias: %v", s, errors)
		}
	}

	invalidAliases := []string{
		"tf",
		"-tf",
		"tf-",
		"TF-Alias",
		"tf-alias-tf-alias-tf-alias-tf-alias-tf-alias-tf-alias-tf-alias-tf-alias",
	}

	for _, s := range invalidAliases {
		_, errors := validateAccountAlias(s, "account_alias")
		if len(errors) == 0 {
			t.Fatalf("%q should not be a valid account alias: %v", s, errors)
		}
	}
}

func TestValidateIamRoleProfileName(t *testing.T) {
	validNames := []string{
		"tf-test-role-profile-1",
	}

	for _, s := range validNames {
		_, errors := validateIamRolePolicyName(s, "name")
		if len(errors) > 0 {
			t.Fatalf("%q should be a valid IAM role policy name: %v", s, errors)
		}
	}

	invalidNames := []string{
		"invalid#name",
		"this-is-a-very-long-role-policy-name-this-is-a-very-long-role-policy-name-this-is-a-very-long-role-policy-name-this-is-a-very-long",
	}

	for _, s := range invalidNames {
		_, errors := validateIamRolePolicyName(s, "name")
		if len(errors) == 0 {
			t.Fatalf("%q should not be a valid IAM role policy name: %v", s, errors)
		}
	}
}

func TestValidateIamRoleProfileNamePrefix(t *testing.T) {
	validNamePrefixes := []string{
		"tf-test-role-profile-",
	}

	for _, s := range validNamePrefixes {
		_, errors := validateIamRolePolicyNamePrefix(s, "name_prefix")
		if len(errors) > 0 {
			t.Fatalf("%q should be a valid IAM role policy name prefix: %v", s, errors)
		}
	}

	invalidNamePrefixes := []string{
		"invalid#name_prefix",
		"this-is-a-very-long-role-policy-name-prefix-this-is-a-very-long-role-policy-name-prefix-this-is-a-very-",
	}

	for _, s := range invalidNamePrefixes {
		_, errors := validateIamRolePolicyNamePrefix(s, "name_prefix")
		if len(errors) == 0 {
			t.Fatalf("%q should not be a valid IAM role policy name prefix: %v", s, errors)
		}
	}
}

func TestValidateApiGatewayUsagePlanQuotaSettingsPeriod(t *testing.T) {
	validEntries := []string{
		"DAY",
		"WEEK",
		"MONTH",
	}

	invalidEntries := []string{
		"fooBAR",
		"foobar45Baz",
		"foobar45Baz@!",
	}

	for _, v := range validEntries {
		_, errors := validateApiGatewayUsagePlanQuotaSettingsPeriod(v, "name")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid API Gateway Quota Settings Period: %v", v, errors)
		}
	}

	for _, v := range invalidEntries {
		_, errors := validateApiGatewayUsagePlanQuotaSettingsPeriod(v, "name")
		if len(errors) == 0 {
			t.Fatalf("%q should not be a API Gateway Quota Settings Period", v)
		}
	}
}

func TestValidateApiGatewayUsagePlanQuotaSettings(t *testing.T) {
	cases := []struct {
		Offset   int
		Period   string
		ErrCount int
	}{
		{
			Offset:   0,
			Period:   "DAY",
			ErrCount: 0,
		},
		{
			Offset:   -1,
			Period:   "DAY",
			ErrCount: 1,
		},
		{
			Offset:   1,
			Period:   "DAY",
			ErrCount: 1,
		},
		{
			Offset:   0,
			Period:   "WEEK",
			ErrCount: 0,
		},
		{
			Offset:   6,
			Period:   "WEEK",
			ErrCount: 0,
		},
		{
			Offset:   -1,
			Period:   "WEEK",
			ErrCount: 1,
		},
		{
			Offset:   7,
			Period:   "WEEK",
			ErrCount: 1,
		},
		{
			Offset:   0,
			Period:   "MONTH",
			ErrCount: 0,
		},
		{
			Offset:   27,
			Period:   "MONTH",
			ErrCount: 0,
		},
		{
			Offset:   -1,
			Period:   "MONTH",
			ErrCount: 1,
		},
		{
			Offset:   28,
			Period:   "MONTH",
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		m := make(map[string]interface{})
		m["offset"] = tc.Offset
		m["period"] = tc.Period

		errors := validateApiGatewayUsagePlanQuotaSettings(m)
		if len(errors) != tc.ErrCount {
			t.Fatalf("API Gateway Usage Plan Quota Settings validation failed: %v", errors)
		}
	}
}

func TestValidateElbName(t *testing.T) {
	validNames := []string{
		"tf-test-elb",
	}

	for _, s := range validNames {
		_, errors := validateElbName(s, "name")
		if len(errors) > 0 {
			t.Fatalf("%q should be a valid ELB name: %v", s, errors)
		}
	}

	invalidNames := []string{
		"tf.test.elb.1",
		"tf-test-elb-tf-test-elb-tf-test-elb",
		"-tf-test-elb",
		"tf-test-elb-",
	}

	for _, s := range invalidNames {
		_, errors := validateElbName(s, "name")
		if len(errors) == 0 {
			t.Fatalf("%q should not be a valid ELB name: %v", s, errors)
		}
	}
}

func TestValidateElbNamePrefix(t *testing.T) {
	validNamePrefixes := []string{
		"test-",
	}

	for _, s := range validNamePrefixes {
		_, errors := validateElbNamePrefix(s, "name_prefix")
		if len(errors) > 0 {
			t.Fatalf("%q should be a valid ELB name prefix: %v", s, errors)
		}
	}

	invalidNamePrefixes := []string{
		"tf.test.elb.",
		"tf-test",
		"-test",
	}

	for _, s := range invalidNamePrefixes {
		_, errors := validateElbNamePrefix(s, "name_prefix")
		if len(errors) == 0 {
			t.Fatalf("%q should not be a valid ELB name prefix: %v", s, errors)
		}
	}
}

func TestValidateDbSubnetGroupName(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "tEsting",
			ErrCount: 1,
		},
		{
			Value:    "testing?",
			ErrCount: 1,
		},
		{
			Value:    "default",
			ErrCount: 1,
		},
		{
			Value:    randomString(300),
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateDbSubnetGroupName(tc.Value, "aws_db_subnet_group")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the DB Subnet Group name to trigger a validation error")
		}
	}
}

func TestValidateDbSubnetGroupNamePrefix(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "tEsting",
			ErrCount: 1,
		},
		{
			Value:    "testing?",
			ErrCount: 1,
		},
		{
			Value:    randomString(230),
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateDbSubnetGroupNamePrefix(tc.Value, "aws_db_subnet_group")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the DB Subnet Group name prefix to trigger a validation error")
		}
	}
}

func TestValidateDbOptionGroupName(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "testing123!",
			ErrCount: 1,
		},
		{
			Value:    "1testing123",
			ErrCount: 1,
		},
		{
			Value:    "testing--123",
			ErrCount: 1,
		},
		{
			Value:    "testing123-",
			ErrCount: 1,
		},
		{
			Value:    randomString(256),
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateDbOptionGroupName(tc.Value, "aws_db_option_group_name")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the DB Option Group Name to trigger a validation error")
		}
	}
}

func TestValidateDbOptionGroupNamePrefix(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "testing123!",
			ErrCount: 1,
		},
		{
			Value:    "1testing123",
			ErrCount: 1,
		},
		{
			Value:    "testing--123",
			ErrCount: 1,
		},
		{
			Value:    randomString(230),
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateDbOptionGroupNamePrefix(tc.Value, "aws_db_option_group_name")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the DB Option Group name prefix to trigger a validation error")
		}
	}
}

func TestValidateOpenIdURL(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "http://wrong.scheme.com",
			ErrCount: 1,
		},
		{
			Value:    "ftp://wrong.scheme.co.uk",
			ErrCount: 1,
		},
		{
			Value:    "%@invalidUrl",
			ErrCount: 1,
		},
		{
			Value:    "https://example.com/?query=param",
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateOpenIdURL(tc.Value, "url")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected %d of OpenID URL validation errors, got %d", tc.ErrCount, len(errors))
		}
	}
}

func TestValidateAwsKmsName(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "alias/aws/s3",
			ErrCount: 0,
		},
		{
			Value:    "alias/hashicorp",
			ErrCount: 0,
		},
		{
			Value:    "hashicorp",
			ErrCount: 1,
		},
		{
			Value:    "hashicorp/terraform",
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateAwsKmsName(tc.Value, "name")
		if len(errors) != tc.ErrCount {
			t.Fatalf("AWS KMS Alias Name validation failed: %v", errors)
		}
	}
}

func TestValidateCognitoIdentityPoolName(t *testing.T) {
	validValues := []string{
		"123",
		"1 2 3",
		"foo",
		"foo bar",
		"foo_bar",
		"1foo 2bar 3",
	}

	for _, s := range validValues {
		_, errors := validateCognitoIdentityPoolName(s, "identity_pool_name")
		if len(errors) > 0 {
			t.Fatalf("%q should be a valid Cognito Identity Pool Name: %v", s, errors)
		}
	}

	invalidValues := []string{
		"1-2-3",
		"foo!",
		"foo-bar",
		"foo-bar",
		"foo1-bar2",
	}

	for _, s := range invalidValues {
		_, errors := validateCognitoIdentityPoolName(s, "identity_pool_name")
		if len(errors) == 0 {
			t.Fatalf("%q should not be a valid Cognito Identity Pool Name: %v", s, errors)
		}
	}
}

func TestValidateCognitoProviderDeveloperName(t *testing.T) {
	validValues := []string{
		"1",
		"foo",
		"1.2",
		"foo1-bar2-baz3",
		"foo_bar",
	}

	for _, s := range validValues {
		_, errors := validateCognitoProviderDeveloperName(s, "developer_provider_name")
		if len(errors) > 0 {
			t.Fatalf("%q should be a valid Cognito Provider Developer Name: %v", s, errors)
		}
	}

	invalidValues := []string{
		"foo!",
		"foo:bar",
		"foo/bar",
		"foo;bar",
	}

	for _, s := range invalidValues {
		_, errors := validateCognitoProviderDeveloperName(s, "developer_provider_name")
		if len(errors) == 0 {
			t.Fatalf("%q should not be a valid Cognito Provider Developer Name: %v", s, errors)
		}
	}
}

func TestValidateCognitoSupportedLoginProviders(t *testing.T) {
	validValues := []string{
		"foo",
		"7346241598935552",
		"123456789012.apps.googleusercontent.com",
		"foo_bar",
		"foo;bar",
		"foo/bar",
		"foo-bar",
		"xvz1evFS4wEEPTGEFPHBog;kAcSOqF21Fu85e7zjz7ZN2U4ZRhfV3WpwPAoE3Z7kBw",
		strings.Repeat("W", 128),
	}

	for _, s := range validValues {
		_, errors := validateCognitoSupportedLoginProviders(s, "supported_login_providers")
		if len(errors) > 0 {
			t.Fatalf("%q should be a valid Cognito Supported Login Providers: %v", s, errors)
		}
	}

	invalidValues := []string{
		"",
		strings.Repeat("W", 129), // > 128
		"foo:bar_baz",
		"foobar,foobaz",
		"foobar=foobaz",
	}

	for _, s := range invalidValues {
		_, errors := validateCognitoSupportedLoginProviders(s, "supported_login_providers")
		if len(errors) == 0 {
			t.Fatalf("%q should not be a valid Cognito Supported Login Providers: %v", s, errors)
		}
	}
}

func TestValidateCognitoIdentityProvidersClientId(t *testing.T) {
	validValues := []string{
		"7lhlkkfbfb4q5kpp90urffao",
		"12345678",
		"foo_123",
		strings.Repeat("W", 128),
	}

	for _, s := range validValues {
		_, errors := validateCognitoIdentityProvidersClientId(s, "client_id")
		if len(errors) > 0 {
			t.Fatalf("%q should be a valid Cognito Identity Provider Client ID: %v", s, errors)
		}
	}

	invalidValues := []string{
		"",
		strings.Repeat("W", 129), // > 128
		"foo-bar",
		"foo:bar",
		"foo;bar",
	}

	for _, s := range invalidValues {
		_, errors := validateCognitoIdentityProvidersClientId(s, "client_id")
		if len(errors) == 0 {
			t.Fatalf("%q should not be a valid Cognito Identity Provider Client ID: %v", s, errors)
		}
	}
}

func TestValidateCognitoIdentityProvidersProviderName(t *testing.T) {
	validValues := []string{
		"foo",
		"7346241598935552",
		"foo_bar",
		"foo:bar",
		"foo/bar",
		"foo-bar",
		"cognito-idp.us-east-1.amazonaws.com/us-east-1_Zr231apJu",
		strings.Repeat("W", 128),
	}

	for _, s := range validValues {
		_, errors := validateCognitoIdentityProvidersProviderName(s, "provider_name")
		if len(errors) > 0 {
			t.Fatalf("%q should be a valid Cognito Identity Provider Name: %v", s, errors)
		}
	}

	invalidValues := []string{
		"",
		strings.Repeat("W", 129), // > 128
		"foo;bar_baz",
		"foobar,foobaz",
		"foobar=foobaz",
	}

	for _, s := range invalidValues {
		_, errors := validateCognitoIdentityProvidersProviderName(s, "provider_name")
		if len(errors) == 0 {
			t.Fatalf("%q should not be a valid Cognito Identity Provider Name: %v", s, errors)
		}
	}
}

func TestValidateWafMetricName(t *testing.T) {
	validNames := []string{
		"testrule",
		"testRule",
		"testRule123",
	}
	for _, v := range validNames {
		_, errors := validateWafMetricName(v, "name")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid WAF metric name: %q", v, errors)
		}
	}

	invalidNames := []string{
		"!",
		"/",
		" ",
		":",
		";",
		"white space",
		"/slash-at-the-beginning",
		"slash-at-the-end/",
	}
	for _, v := range invalidNames {
		_, errors := validateWafMetricName(v, "name")
		if len(errors) == 0 {
			t.Fatalf("%q should be an invalid WAF metric name", v)
		}
	}
}

func TestValidateIamRoleDescription(t *testing.T) {
	validNames := []string{
		"This 1s a D3scr!pti0n with weird content: @ #^ù£ê®æ ø]ŒîÏî~ÈÙ£÷=,ë",
		strings.Repeat("W", 1000),
	}
	for _, v := range validNames {
		_, errors := validateIamRoleDescription(v, "description")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid IAM Role Description: %q", v, errors)
		}
	}

	invalidNames := []string{
		strings.Repeat("W", 1001), // > 1000
	}
	for _, v := range invalidNames {
		_, errors := validateIamRoleDescription(v, "description")
		if len(errors) == 0 {
			t.Fatalf("%q should be an invalid IAM Role Description", v)
		}
	}
}

func TestValidateSsmParameterType(t *testing.T) {
	validTypes := []string{
		"String",
		"StringList",
		"SecureString",
	}
	for _, v := range validTypes {
		_, errors := validateSsmParameterType(v, "name")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid SSM parameter type: %q", v, errors)
		}
	}

	invalidTypes := []string{
		"foo",
		"string",
		"Securestring",
	}
	for _, v := range invalidTypes {
		_, errors := validateSsmParameterType(v, "name")
		if len(errors) == 0 {
			t.Fatalf("%q should be an invalid SSM parameter type", v)
		}
	}
}
