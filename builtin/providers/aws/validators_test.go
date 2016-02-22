package aws

import (
	"strings"
	"testing"
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
		// lenght > 140
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
		// lenght > 128
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
	validNames := []string{
		"arn:aws:elasticbeanstalk:us-east-1:123456789012:environment/My App/MyEnvironment", // Beanstalk
		"arn:aws:iam::123456789012:user/David",                                             // IAM User
		"arn:aws:rds:eu-west-1:123456789012:db:mysql-db",                                   // RDS
		"arn:aws:s3:::my_corporate_bucket/exampleobject.png",                               // S3 object
		"arn:aws:events:us-east-1:319201112229:rule/rule_name",                             // CloudWatch Rule
		"arn:aws:lambda:eu-west-1:319201112229:function:myCustomFunction",                  // Lambda function
		"arn:aws:lambda:eu-west-1:319201112229:function:myCustomFunction:Qualifier",        // Lambda func qualifier
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
