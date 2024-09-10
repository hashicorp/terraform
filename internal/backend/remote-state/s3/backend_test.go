// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package s3

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/aws-sdk-go-base/v2/mockdata"
	"github.com/hashicorp/aws-sdk-go-base/v2/servicemocks"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/configs/hcl2shim"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

var (
	mockStsGetCallerIdentityRequestBody = url.Values{
		"Action":  []string{"GetCallerIdentity"},
		"Version": []string{"2011-06-15"},
	}.Encode()
)

// verify that we are doing ACC tests or the S3 tests specifically
func testACC(t *testing.T) {
	skip := os.Getenv("TF_ACC") == "" && os.Getenv("TF_S3_TEST") == ""
	if skip {
		t.Log("s3 backend tests require setting TF_ACC or TF_S3_TEST")
		t.Skip()
	}
	if os.Getenv("AWS_DEFAULT_REGION") == "" {
		os.Setenv("AWS_DEFAULT_REGION", "us-west-2")
	}
}

func TestBackend_impl(t *testing.T) {
	var _ backend.Backend = new(Backend)
}

func TestBackend_InternalValidate(t *testing.T) {
	b := New()

	schema := b.ConfigSchema()
	if err := schema.InternalValidate(); err != nil {
		t.Fatalf("failed InternalValidate: %s", err)
	}
}

func TestBackendConfig_original(t *testing.T) {
	testACC(t)

	ctx := context.TODO()

	region := "us-west-1"

	config := map[string]interface{}{
		"region":         region,
		"bucket":         "tf-test",
		"key":            "state",
		"encrypt":        true,
		"dynamodb_table": "dynamoTable",
	}

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(config)).(*Backend)

	if b.awsConfig.Region != region {
		t.Fatalf("Incorrect region was populated")
	}
	if b.awsConfig.RetryMaxAttempts != 5 {
		t.Fatalf("Default max_retries was not set")
	}
	if b.bucketName != "tf-test" {
		t.Fatalf("Incorrect bucketName was populated")
	}
	if b.keyName != "state" {
		t.Fatalf("Incorrect keyName was populated")
	}

	credentials, err := b.awsConfig.Credentials.Retrieve(ctx)
	if err != nil {
		t.Fatalf("Error when requesting credentials")
	}
	if credentials.AccessKeyID == "" {
		t.Fatalf("No Access Key Id was populated")
	}
	if credentials.SecretAccessKey == "" {
		t.Fatalf("No Secret Access Key was populated")
	}

	// Check S3 Endpoint
	expectedS3Endpoint := defaultEndpointS3(region)
	var s3Endpoint string
	_, err = b.s3Client.ListBuckets(ctx, &s3.ListBucketsInput{},
		func(opts *s3.Options) {
			opts.APIOptions = append(opts.APIOptions,
				addRetrieveEndpointURLMiddleware(t, &s3Endpoint),
				addCancelRequestMiddleware(),
			)
		},
	)
	if err == nil {
		t.Fatal("Checking S3 Endpoint: Expected an error, got none")
	} else if !errors.Is(err, errCancelOperation) {
		t.Fatalf("Checking S3 Endpoint: Unexpected error: %s", err)
	}

	if s3Endpoint != expectedS3Endpoint {
		t.Errorf("Checking S3 Endpoint: expected endpoint %q, got %q", expectedS3Endpoint, s3Endpoint)
	}

	// Check DynamoDB Endpoint
	expectedDynamoDBEndpoint := defaultEndpointDynamo(region)
	var dynamoDBEndpoint string
	_, err = b.dynClient.ListTables(ctx, &dynamodb.ListTablesInput{},
		func(opts *dynamodb.Options) {
			opts.APIOptions = append(opts.APIOptions,
				addRetrieveEndpointURLMiddleware(t, &dynamoDBEndpoint),
				addCancelRequestMiddleware(),
			)
		},
	)
	if err == nil {
		t.Fatal("Checking DynamoDB Endpoint: Expected an error, got none")
	} else if !errors.Is(err, errCancelOperation) {
		t.Fatalf("Checking DynamoDB Endpoint: Unexpected error: %s", err)
	}

	if dynamoDBEndpoint != expectedDynamoDBEndpoint {
		t.Errorf("Checking DynamoDB Endpoint: expected endpoint %q, got %q", expectedDynamoDBEndpoint, dynamoDBEndpoint)
	}
}

func TestBackendConfig_InvalidRegion(t *testing.T) {
	testACC(t)

	cases := map[string]struct {
		config        map[string]any
		expectedDiags tfdiags.Diagnostics
	}{
		"with region validation": {
			config: map[string]interface{}{
				"region":                      "nonesuch",
				"bucket":                      "tf-test",
				"key":                         "state",
				"skip_credentials_validation": true,
			},
			expectedDiags: tfdiags.Diagnostics{
				tfdiags.AttributeValue(
					tfdiags.Error,
					"Invalid region value",
					`Invalid AWS Region: nonesuch`,
					cty.GetAttrPath("region"),
				),
			},
		},
		"skip region validation": {
			config: map[string]interface{}{
				"region":                      "nonesuch",
				"bucket":                      "tf-test",
				"key":                         "state",
				"skip_region_validation":      true,
				"skip_credentials_validation": true,
			},
			expectedDiags: nil,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			b := New()
			configSchema := populateSchema(t, b.ConfigSchema(), hcl2shim.HCL2ValueFromConfigValue(tc.config))

			configSchema, diags := b.PrepareConfig(configSchema)
			if len(diags) > 0 {
				t.Fatal(diags.ErrWithWarnings())
			}

			confDiags := b.Configure(configSchema)
			diags = diags.Append(confDiags)

			if diff := cmp.Diff(diags, tc.expectedDiags, cmp.Comparer(diagnosticComparer)); diff != "" {
				t.Errorf("unexpected diagnostics difference: %s", diff)
			}
		})
	}
}

func TestBackendConfig_RegionEnvVar(t *testing.T) {
	testACC(t)
	config := map[string]interface{}{
		"bucket": "tf-test",
		"key":    "state",
	}

	cases := map[string]struct {
		vars map[string]string
	}{
		"AWS_REGION": {
			vars: map[string]string{
				"AWS_REGION": "us-west-1",
			},
		},

		"AWS_DEFAULT_REGION": {
			vars: map[string]string{
				"AWS_DEFAULT_REGION": "us-west-1",
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			for k, v := range tc.vars {
				os.Setenv(k, v)
			}
			t.Cleanup(func() {
				for k := range tc.vars {
					os.Unsetenv(k)
				}
			})

			b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(config)).(*Backend)

			if b.awsConfig.Region != "us-west-1" {
				t.Fatalf("Incorrect region was populated")
			}
		})
	}
}

func TestBackendConfig_DynamoDBEndpoint(t *testing.T) {
	testACC(t)

	ctx := context.TODO()

	region := "us-west-1"

	cases := map[string]struct {
		config           map[string]any
		vars             map[string]string
		expectedEndpoint string
		expectedDiags    tfdiags.Diagnostics
	}{
		"none": {
			expectedEndpoint: defaultEndpointDynamo(region),
		},
		"config URL": {
			config: map[string]any{
				"endpoints": map[string]any{
					"dynamodb": "https://dynamo.test",
				},
			},
			expectedEndpoint: "https://dynamo.test/",
		},
		"config hostname": {
			config: map[string]any{
				"endpoints": map[string]any{
					"dynamodb": "dynamo.test",
				},
			},
			expectedEndpoint: "dynamo.test/",
			expectedDiags: tfdiags.Diagnostics{
				legacyIncompleteURLDiag("dynamo.test", cty.GetAttrPath("endpoints").GetAttr("dynamodb")),
			},
		},
		"deprecated config URL": {
			config: map[string]any{
				"dynamodb_endpoint": "https://dynamo.test",
			},
			expectedEndpoint: "https://dynamo.test/",
			expectedDiags: tfdiags.Diagnostics{
				deprecatedAttrDiag(cty.GetAttrPath("dynamodb_endpoint"), cty.GetAttrPath("endpoints").GetAttr("dynamodb")),
			},
		},
		"deprecated config hostname": {
			config: map[string]any{
				"dynamodb_endpoint": "dynamo.test",
			},
			expectedEndpoint: "dynamo.test/",
			expectedDiags: tfdiags.Diagnostics{
				deprecatedAttrDiag(cty.GetAttrPath("dynamodb_endpoint"), cty.GetAttrPath("endpoints").GetAttr("dynamodb")),
				legacyIncompleteURLDiag("dynamo.test", cty.GetAttrPath("dynamodb_endpoint")),
			},
		},
		"config conflict": {
			config: map[string]any{
				"dynamodb_endpoint": "https://dynamo.test",
				"endpoints": map[string]any{
					"dynamodb": "https://dynamo.test",
				},
			},
			expectedDiags: tfdiags.Diagnostics{
				deprecatedAttrDiag(cty.GetAttrPath("dynamodb_endpoint"), cty.GetAttrPath("endpoints").GetAttr("dynamodb")),
				wholeBodyErrDiag(
					"Conflicting Parameters",
					fmt.Sprintf(`The parameters "%s" and "%s" cannot be configured together.`,
						pathString(cty.GetAttrPath("dynamodb_endpoint")),
						pathString(cty.GetAttrPath("endpoints").GetAttr("dynamodb")),
					),
				)},
		},
		"envvar": {
			vars: map[string]string{
				"AWS_ENDPOINT_URL_DYNAMODB": "https://dynamo.test",
			},
			expectedEndpoint: "https://dynamo.test/",
		},
		"deprecated envvar": {
			vars: map[string]string{
				"AWS_DYNAMODB_ENDPOINT": "https://dynamo.test",
			},
			expectedEndpoint: "https://dynamo.test/",
			expectedDiags: tfdiags.Diagnostics{
				deprecatedEnvVarDiag("AWS_DYNAMODB_ENDPOINT", "AWS_ENDPOINT_URL_DYNAMODB"),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			config := map[string]interface{}{
				"region": region,
				"bucket": "tf-test",
				"key":    "state",
			}

			if tc.vars != nil {
				for k, v := range tc.vars {
					os.Setenv(k, v)
				}
				t.Cleanup(func() {
					for k := range tc.vars {
						os.Unsetenv(k)
					}
				})
			}

			if tc.config != nil {
				for k, v := range tc.config {
					config[k] = v
				}
			}

			raw, diags := testBackendConfigDiags(t, New(), backend.TestWrapConfig(config))
			b := raw.(*Backend)

			if diff := cmp.Diff(diags, tc.expectedDiags, cmp.Comparer(diagnosticComparer)); diff != "" {
				t.Errorf("unexpected diagnostics difference: %s", diff)
			}

			if !diags.HasErrors() {
				var dynamoDBEndpoint string
				_, err := b.dynClient.ListTables(ctx, &dynamodb.ListTablesInput{},
					func(opts *dynamodb.Options) {
						opts.APIOptions = append(opts.APIOptions,
							addRetrieveEndpointURLMiddleware(t, &dynamoDBEndpoint),
							addCancelRequestMiddleware(),
						)
					},
				)
				if err == nil {
					t.Fatal("Expected an error, got none")
				} else if !errors.Is(err, errCancelOperation) {
					t.Fatalf("Unexpected error: %s", err)
				}

				if dynamoDBEndpoint != tc.expectedEndpoint {
					t.Errorf("expected endpoint %q, got %q", tc.expectedEndpoint, dynamoDBEndpoint)
				}
			}
		})
	}
}

func TestBackendConfig_IAMEndpoint(t *testing.T) {
	testACC(t)

	// Doesn't test for expected endpoint, since the IAM endpoint is used internally to `aws-sdk-go-base`
	// The mocked tests won't work if the config parameter doesn't work
	cases := map[string]struct {
		config        map[string]any
		vars          map[string]string
		expectedDiags tfdiags.Diagnostics
	}{
		"none": {},
		"config URL": {
			config: map[string]any{
				"endpoints": map[string]any{
					"iam": "https://iam.test",
				},
			},
		},
		"config hostname": {
			config: map[string]any{
				"endpoints": map[string]any{
					"iam": "iam.test",
				},
			},
			expectedDiags: tfdiags.Diagnostics{
				legacyIncompleteURLDiag("iam.test", cty.GetAttrPath("endpoints").GetAttr("iam")),
			},
		},
		"deprecated config URL": {
			config: map[string]any{
				"iam_endpoint": "https://iam.test",
			},
			expectedDiags: tfdiags.Diagnostics{
				deprecatedAttrDiag(cty.GetAttrPath("iam_endpoint"), cty.GetAttrPath("endpoints").GetAttr("iam")),
			},
		},
		"deprecated config hostname": {
			config: map[string]any{
				"iam_endpoint": "iam.test",
			},
			expectedDiags: tfdiags.Diagnostics{
				deprecatedAttrDiag(cty.GetAttrPath("iam_endpoint"), cty.GetAttrPath("endpoints").GetAttr("iam")),
				legacyIncompleteURLDiag("iam.test", cty.GetAttrPath("iam_endpoint")),
			},
		},
		"config conflict": {
			config: map[string]any{
				"iam_endpoint": "https://iam.test",
				"endpoints": map[string]any{
					"iam": "https://iam.test",
				},
			},
			expectedDiags: tfdiags.Diagnostics{
				deprecatedAttrDiag(cty.GetAttrPath("iam_endpoint"), cty.GetAttrPath("endpoints").GetAttr("iam")),
				wholeBodyErrDiag(
					"Conflicting Parameters",
					fmt.Sprintf(`The parameters "%s" and "%s" cannot be configured together.`,
						pathString(cty.GetAttrPath("iam_endpoint")),
						pathString(cty.GetAttrPath("endpoints").GetAttr("iam")),
					),
				)},
		},
		"envvar": {
			vars: map[string]string{
				"AWS_ENDPOINT_URL_IAM": "https://iam.test",
			},
		},
		"deprecated envvar": {
			vars: map[string]string{
				"AWS_IAM_ENDPOINT": "https://iam.test",
			},
			expectedDiags: tfdiags.Diagnostics{
				deprecatedEnvVarDiag("AWS_IAM_ENDPOINT", "AWS_ENDPOINT_URL_IAM"),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			config := map[string]interface{}{
				"region": "us-west-1",
				"bucket": "tf-test",
				"key":    "state",
			}

			if tc.vars != nil {
				for k, v := range tc.vars {
					os.Setenv(k, v)
				}
				t.Cleanup(func() {
					for k := range tc.vars {
						os.Unsetenv(k)
					}
				})
			}

			if tc.config != nil {
				for k, v := range tc.config {
					config[k] = v
				}
			}

			_, diags := testBackendConfigDiags(t, New(), backend.TestWrapConfig(config))

			if diff := cmp.Diff(diags, tc.expectedDiags, cmp.Comparer(diagnosticComparer)); diff != "" {
				t.Errorf("unexpected diagnostics difference: %s", diff)
			}
		})
	}
}

func TestBackendConfig_S3Endpoint(t *testing.T) {
	testACC(t)

	ctx := context.TODO()

	region := "us-west-1"

	cases := map[string]struct {
		config           map[string]any
		vars             map[string]string
		expectedEndpoint string
		expectedDiags    tfdiags.Diagnostics
	}{
		"none": {
			expectedEndpoint: defaultEndpointS3(region),
		},
		"config URL": {
			config: map[string]any{
				"endpoints": map[string]any{
					"s3": "https://s3.test",
				},
			},
			expectedEndpoint: "https://s3.test/",
		},
		"config hostname": {
			config: map[string]any{
				"endpoints": map[string]any{
					"s3": "s3.test",
				},
			},
			expectedEndpoint: "/s3.test",
			expectedDiags: tfdiags.Diagnostics{
				legacyIncompleteURLDiag("s3.test", cty.GetAttrPath("endpoints").GetAttr("s3")),
			},
		},
		"deprecated config URL": {
			config: map[string]any{
				"endpoint": "https://s3.test",
			},
			expectedEndpoint: "https://s3.test/",
			expectedDiags: tfdiags.Diagnostics{
				deprecatedAttrDiag(cty.GetAttrPath("endpoint"), cty.GetAttrPath("endpoints").GetAttr("s3")),
			},
		},
		"deprecated config hostname": {
			config: map[string]any{
				"endpoint": "s3.test",
			},
			expectedEndpoint: "/s3.test",
			expectedDiags: tfdiags.Diagnostics{
				deprecatedAttrDiag(cty.GetAttrPath("endpoint"), cty.GetAttrPath("endpoints").GetAttr("s3")),
				legacyIncompleteURLDiag("s3.test", cty.GetAttrPath("endpoint")),
			},
		},
		"config conflict": {
			config: map[string]any{
				"endpoint": "https://s3.test",
				"endpoints": map[string]any{
					"s3": "https://s3.test",
				},
			},
			expectedDiags: tfdiags.Diagnostics{
				deprecatedAttrDiag(cty.GetAttrPath("endpoint"), cty.GetAttrPath("endpoints").GetAttr("s3")),
				wholeBodyErrDiag(
					"Conflicting Parameters",
					fmt.Sprintf(`The parameters "%s" and "%s" cannot be configured together.`,
						pathString(cty.GetAttrPath("endpoint")),
						pathString(cty.GetAttrPath("endpoints").GetAttr("s3")),
					),
				)},
		},
		"envvar": {
			vars: map[string]string{
				"AWS_ENDPOINT_URL_S3": "https://s3.test",
			},
			expectedEndpoint: "https://s3.test/",
		},
		"deprecated envvar": {
			vars: map[string]string{
				"AWS_S3_ENDPOINT": "https://s3.test",
			},
			expectedEndpoint: "https://s3.test/",
			expectedDiags: tfdiags.Diagnostics{
				deprecatedEnvVarDiag("AWS_S3_ENDPOINT", "AWS_ENDPOINT_URL_S3"),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			config := map[string]interface{}{
				"region": region,
				"bucket": "tf-test",
				"key":    "state",
			}

			if tc.vars != nil {
				for k, v := range tc.vars {
					os.Setenv(k, v)
				}
				t.Cleanup(func() {
					for k := range tc.vars {
						os.Unsetenv(k)
					}
				})
			}

			if tc.config != nil {
				for k, v := range tc.config {
					config[k] = v
				}
			}

			raw, diags := testBackendConfigDiags(t, New(), backend.TestWrapConfig(config))
			b := raw.(*Backend)

			if diff := cmp.Diff(diags, tc.expectedDiags, cmp.Comparer(diagnosticComparer)); diff != "" {
				t.Errorf("unexpected diagnostics difference: %s", diff)
			}

			if !diags.HasErrors() {
				var s3Endpoint string
				_, err := b.s3Client.ListBuckets(ctx, &s3.ListBucketsInput{},
					func(opts *s3.Options) {
						opts.APIOptions = append(opts.APIOptions,
							addRetrieveEndpointURLMiddleware(t, &s3Endpoint),
							addCancelRequestMiddleware(),
						)
					},
				)
				if err == nil {
					t.Fatal("Expected an error, got none")
				} else if !errors.Is(err, errCancelOperation) {
					t.Fatalf("Unexpected error: %s", err)
				}

				if s3Endpoint != tc.expectedEndpoint {
					t.Errorf("expected endpoint %q, got %q", tc.expectedEndpoint, s3Endpoint)
				}
			}
		})
	}
}

func TestBackendConfig_EC2MetadataEndpoint(t *testing.T) {
	testACC(t)

	ctx := context.TODO()

	cases := map[string]struct {
		config           map[string]any
		vars             map[string]string
		expectedEndpoint string
		expectedDiags    tfdiags.Diagnostics
	}{
		"none": {
			expectedEndpoint: "http://169.254.169.254/latest/meta-data",
		},
		"config URL": {
			config: map[string]any{
				"ec2_metadata_service_endpoint": "https://ec2.test",
			},
			expectedEndpoint: "https://ec2.test/latest/meta-data",
		},
		"config hostname": {
			config: map[string]any{
				"ec2_metadata_service_endpoint": "ec2.test",
			},
			expectedDiags: tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid Value",
					`The value must be a valid URL containing at least a scheme and hostname. Had "ec2.test"`,
					cty.GetAttrPath("ec2_metadata_service_endpoint"),
				),
			},
		},
		"config IPv4 mode": {
			config: map[string]any{
				"ec2_metadata_service_endpoint":      "https://ec2.test",
				"ec2_metadata_service_endpoint_mode": "IPv4",
			},
			expectedEndpoint: "https://ec2.test/latest/meta-data",
		},
		"config IPv6 mode": {
			config: map[string]any{
				"ec2_metadata_service_endpoint":      "https://ec2.test",
				"ec2_metadata_service_endpoint_mode": "IPv6",
			},
			expectedEndpoint: "https://ec2.test/latest/meta-data",
		},
		"config invalid mode": {
			config: map[string]any{
				"ec2_metadata_service_endpoint":      "https://ec2.test",
				"ec2_metadata_service_endpoint_mode": "invalid",
			},
			expectedDiags: tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid Value",
					"Value must be one of [IPv4, IPv6]",
					cty.GetAttrPath("ec2_metadata_service_endpoint_mode"),
				),
			},
		},
		"envvar": {
			vars: map[string]string{
				"AWS_EC2_METADATA_SERVICE_ENDPOINT": "https://ec2.test",
			},
			expectedEndpoint: "https://ec2.test/latest/meta-data",
		},
		"envvar IPv4 mode": {
			vars: map[string]string{
				"AWS_EC2_METADATA_SERVICE_ENDPOINT":      "https://ec2.test",
				"AWS_EC2_METADATA_SERVICE_ENDPOINT_MODE": "IPv4",
			},
			expectedEndpoint: "https://ec2.test/latest/meta-data",
		},
		"envvar IPv6 mode": {
			vars: map[string]string{
				"AWS_EC2_METADATA_SERVICE_ENDPOINT":      "https://ec2.test",
				"AWS_EC2_METADATA_SERVICE_ENDPOINT_MODE": "IPv6",
			},
			expectedEndpoint: "https://ec2.test/latest/meta-data",
		},
		"envvar invalid mode": {
			vars: map[string]string{
				"AWS_EC2_METADATA_SERVICE_ENDPOINT":      "https://ec2.test",
				"AWS_EC2_METADATA_SERVICE_ENDPOINT_MODE": "invalid",
			},
			// expectedEndpoint: "ec2.test",
			expectedDiags: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"unknown EC2 IMDS endpoint mode, must be either IPv6 or IPv4",
					"",
				),
			},
		},
		"deprecated envvar": {
			vars: map[string]string{
				"AWS_METADATA_URL": "https://ec2.test",
			},
			expectedEndpoint: "https://ec2.test/latest/meta-data",
			expectedDiags: tfdiags.Diagnostics{
				deprecatedEnvVarDiag("AWS_METADATA_URL", "AWS_EC2_METADATA_SERVICE_ENDPOINT"),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			config := map[string]interface{}{
				"region": "us-west-1",
				"bucket": "tf-test",
				"key":    "state",
			}

			if tc.vars != nil {
				for k, v := range tc.vars {
					os.Setenv(k, v)
				}
				t.Cleanup(func() {
					for k := range tc.vars {
						os.Unsetenv(k)
					}
				})
			}

			if tc.config != nil {
				for k, v := range tc.config {
					config[k] = v
				}
			}

			raw, diags := testBackendConfigDiags(t, New(), backend.TestWrapConfig(config))
			b := raw.(*Backend)

			if diff := cmp.Diff(diags, tc.expectedDiags, cmp.Comparer(diagnosticComparer)); diff != "" {
				t.Errorf("unexpected diagnostics difference: %s", diff)
			}

			if !diags.HasErrors() {
				var imdsEndpoint string
				imdsClient := imds.NewFromConfig(b.awsConfig)
				_, err := imdsClient.GetMetadata(ctx, &imds.GetMetadataInput{},
					func(opts *imds.Options) {
						opts.APIOptions = append(opts.APIOptions,
							addRetrieveEndpointURLMiddleware(t, &imdsEndpoint),
							addCancelRequestMiddleware(),
						)
					},
				)
				if err == nil {
					t.Fatal("Expected an error, got none")
				} else if !errors.Is(err, errCancelOperation) {
					t.Fatalf("Unexpected error: %s", err)
				}

				if imdsEndpoint != tc.expectedEndpoint {
					t.Errorf("expected endpoint %q, got %q", tc.expectedEndpoint, imdsEndpoint)
				}
			}
		})
	}
}

func TestBackendConfig_AssumeRole(t *testing.T) {
	testACC(t)

	testCases := map[string]struct {
		Config           map[string]interface{}
		MockStsEndpoints []*servicemocks.MockEndpoint
	}{
		"role_arn": {
			Config: map[string]interface{}{
				"bucket":       "tf-test",
				"key":          "state",
				"region":       "us-west-1",
				"role_arn":     servicemocks.MockStsAssumeRoleArn,
				"session_name": servicemocks.MockStsAssumeRoleSessionName,
			},
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				{
					Request: &servicemocks.MockRequest{Method: "POST", Uri: "/", Body: url.Values{
						"Action":          []string{"AssumeRole"},
						"DurationSeconds": []string{"900"},
						"RoleArn":         []string{servicemocks.MockStsAssumeRoleArn},
						"RoleSessionName": []string{servicemocks.MockStsAssumeRoleSessionName},
						"Version":         []string{"2011-06-15"},
					}.Encode()},
					Response: &servicemocks.MockResponse{StatusCode: 200, Body: servicemocks.MockStsAssumeRoleValidResponseBody, ContentType: "text/xml"},
				},
				{
					Request:  &servicemocks.MockRequest{Method: "POST", Uri: "/", Body: mockStsGetCallerIdentityRequestBody},
					Response: &servicemocks.MockResponse{StatusCode: 200, Body: servicemocks.MockStsGetCallerIdentityValidResponseBody, ContentType: "text/xml"},
				},
			},
		},
		"assume_role_duration_seconds": {
			Config: map[string]interface{}{
				"assume_role_duration_seconds": 3600,
				"bucket":                       "tf-test",
				"key":                          "state",
				"region":                       "us-west-1",
				"role_arn":                     servicemocks.MockStsAssumeRoleArn,
				"session_name":                 servicemocks.MockStsAssumeRoleSessionName,
			},
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				{
					Request: &servicemocks.MockRequest{Method: "POST", Uri: "/", Body: url.Values{
						"Action":          []string{"AssumeRole"},
						"DurationSeconds": []string{"3600"},
						"RoleArn":         []string{servicemocks.MockStsAssumeRoleArn},
						"RoleSessionName": []string{servicemocks.MockStsAssumeRoleSessionName},
						"Version":         []string{"2011-06-15"},
					}.Encode()},
					Response: &servicemocks.MockResponse{StatusCode: 200, Body: servicemocks.MockStsAssumeRoleValidResponseBody, ContentType: "text/xml"},
				},
				{
					Request:  &servicemocks.MockRequest{Method: "POST", Uri: "/", Body: mockStsGetCallerIdentityRequestBody},
					Response: &servicemocks.MockResponse{StatusCode: 200, Body: servicemocks.MockStsGetCallerIdentityValidResponseBody, ContentType: "text/xml"},
				},
			},
		},
		"external_id": {
			Config: map[string]interface{}{
				"bucket":       "tf-test",
				"external_id":  servicemocks.MockStsAssumeRoleExternalId,
				"key":          "state",
				"region":       "us-west-1",
				"role_arn":     servicemocks.MockStsAssumeRoleArn,
				"session_name": servicemocks.MockStsAssumeRoleSessionName,
			},
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				{
					Request: &servicemocks.MockRequest{Method: "POST", Uri: "/", Body: url.Values{
						"Action":          []string{"AssumeRole"},
						"DurationSeconds": []string{"900"},
						"ExternalId":      []string{servicemocks.MockStsAssumeRoleExternalId},
						"RoleArn":         []string{servicemocks.MockStsAssumeRoleArn},
						"RoleSessionName": []string{servicemocks.MockStsAssumeRoleSessionName},
						"Version":         []string{"2011-06-15"},
					}.Encode()},
					Response: &servicemocks.MockResponse{StatusCode: 200, Body: servicemocks.MockStsAssumeRoleValidResponseBody, ContentType: "text/xml"},
				},
				{
					Request:  &servicemocks.MockRequest{Method: "POST", Uri: "/", Body: mockStsGetCallerIdentityRequestBody},
					Response: &servicemocks.MockResponse{StatusCode: 200, Body: servicemocks.MockStsGetCallerIdentityValidResponseBody, ContentType: "text/xml"},
				},
			},
		},
		"assume_role_policy": {
			Config: map[string]interface{}{
				"assume_role_policy": servicemocks.MockStsAssumeRolePolicy,
				"bucket":             "tf-test",
				"key":                "state",
				"region":             "us-west-1",
				"role_arn":           servicemocks.MockStsAssumeRoleArn,
				"session_name":       servicemocks.MockStsAssumeRoleSessionName,
			},
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				{
					Request: &servicemocks.MockRequest{Method: "POST", Uri: "/", Body: url.Values{
						"Action":          []string{"AssumeRole"},
						"DurationSeconds": []string{"900"},
						"Policy":          []string{servicemocks.MockStsAssumeRolePolicy},
						"RoleArn":         []string{servicemocks.MockStsAssumeRoleArn},
						"RoleSessionName": []string{servicemocks.MockStsAssumeRoleSessionName},
						"Version":         []string{"2011-06-15"},
					}.Encode()},
					Response: &servicemocks.MockResponse{StatusCode: 200, Body: servicemocks.MockStsAssumeRoleValidResponseBody, ContentType: "text/xml"},
				},
				{
					Request:  &servicemocks.MockRequest{Method: "POST", Uri: "/", Body: mockStsGetCallerIdentityRequestBody},
					Response: &servicemocks.MockResponse{StatusCode: 200, Body: servicemocks.MockStsGetCallerIdentityValidResponseBody, ContentType: "text/xml"},
				},
			},
		},
		"assume_role_policy_arns": {
			Config: map[string]interface{}{
				"assume_role_policy_arns": []interface{}{servicemocks.MockStsAssumeRolePolicyArn},
				"bucket":                  "tf-test",
				"key":                     "state",
				"region":                  "us-west-1",
				"role_arn":                servicemocks.MockStsAssumeRoleArn,
				"session_name":            servicemocks.MockStsAssumeRoleSessionName,
			},
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				{
					Request: &servicemocks.MockRequest{Method: "POST", Uri: "/", Body: url.Values{
						"Action":                  []string{"AssumeRole"},
						"DurationSeconds":         []string{"900"},
						"PolicyArns.member.1.arn": []string{servicemocks.MockStsAssumeRolePolicyArn},
						"RoleArn":                 []string{servicemocks.MockStsAssumeRoleArn},
						"RoleSessionName":         []string{servicemocks.MockStsAssumeRoleSessionName},
						"Version":                 []string{"2011-06-15"},
					}.Encode()},
					Response: &servicemocks.MockResponse{StatusCode: 200, Body: servicemocks.MockStsAssumeRoleValidResponseBody, ContentType: "text/xml"},
				},
				{
					Request:  &servicemocks.MockRequest{Method: "POST", Uri: "/", Body: mockStsGetCallerIdentityRequestBody},
					Response: &servicemocks.MockResponse{StatusCode: 200, Body: servicemocks.MockStsGetCallerIdentityValidResponseBody, ContentType: "text/xml"},
				},
			},
		},
		"assume_role_tags": {
			Config: map[string]interface{}{
				"assume_role_tags": map[string]interface{}{
					servicemocks.MockStsAssumeRoleTagKey: servicemocks.MockStsAssumeRoleTagValue,
				},
				"bucket":       "tf-test",
				"key":          "state",
				"region":       "us-west-1",
				"role_arn":     servicemocks.MockStsAssumeRoleArn,
				"session_name": servicemocks.MockStsAssumeRoleSessionName,
			},
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				{
					Request: &servicemocks.MockRequest{Method: "POST", Uri: "/", Body: url.Values{
						"Action":              []string{"AssumeRole"},
						"DurationSeconds":     []string{"900"},
						"RoleArn":             []string{servicemocks.MockStsAssumeRoleArn},
						"RoleSessionName":     []string{servicemocks.MockStsAssumeRoleSessionName},
						"Tags.member.1.Key":   []string{servicemocks.MockStsAssumeRoleTagKey},
						"Tags.member.1.Value": []string{servicemocks.MockStsAssumeRoleTagValue},
						"Version":             []string{"2011-06-15"},
					}.Encode()},
					Response: &servicemocks.MockResponse{StatusCode: 200, Body: servicemocks.MockStsAssumeRoleValidResponseBody, ContentType: "text/xml"},
				},
				{
					Request:  &servicemocks.MockRequest{Method: "POST", Uri: "/", Body: mockStsGetCallerIdentityRequestBody},
					Response: &servicemocks.MockResponse{StatusCode: 200, Body: servicemocks.MockStsGetCallerIdentityValidResponseBody, ContentType: "text/xml"},
				},
			},
		},
		"assume_role_transitive_tag_keys": {
			Config: map[string]interface{}{
				"assume_role_tags": map[string]interface{}{
					servicemocks.MockStsAssumeRoleTagKey: servicemocks.MockStsAssumeRoleTagValue,
				},
				"assume_role_transitive_tag_keys": []interface{}{servicemocks.MockStsAssumeRoleTagKey},
				"bucket":                          "tf-test",
				"key":                             "state",
				"region":                          "us-west-1",
				"role_arn":                        servicemocks.MockStsAssumeRoleArn,
				"session_name":                    servicemocks.MockStsAssumeRoleSessionName,
			},
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				{
					Request: &servicemocks.MockRequest{Method: "POST", Uri: "/", Body: url.Values{
						"Action":                     []string{"AssumeRole"},
						"DurationSeconds":            []string{"900"},
						"RoleArn":                    []string{servicemocks.MockStsAssumeRoleArn},
						"RoleSessionName":            []string{servicemocks.MockStsAssumeRoleSessionName},
						"Tags.member.1.Key":          []string{servicemocks.MockStsAssumeRoleTagKey},
						"Tags.member.1.Value":        []string{servicemocks.MockStsAssumeRoleTagValue},
						"TransitiveTagKeys.member.1": []string{servicemocks.MockStsAssumeRoleTagKey},
						"Version":                    []string{"2011-06-15"},
					}.Encode()},
					Response: &servicemocks.MockResponse{StatusCode: 200, Body: servicemocks.MockStsAssumeRoleValidResponseBody, ContentType: "text/xml"},
				},
				{
					Request:  &servicemocks.MockRequest{Method: "POST", Uri: "/", Body: mockStsGetCallerIdentityRequestBody},
					Response: &servicemocks.MockResponse{StatusCode: 200, Body: servicemocks.MockStsGetCallerIdentityValidResponseBody, ContentType: "text/xml"},
				},
			},
		},
	}

	for testName, testCase := range testCases {
		testCase := testCase

		t.Run(testName, func(t *testing.T) {
			closeSts, _, stsEndpoint := mockdata.GetMockedAwsApiSession("STS", testCase.MockStsEndpoints)
			defer closeSts()

			testCase.Config["sts_endpoint"] = stsEndpoint

			b := New()
			diags := b.Configure(populateSchema(t, b.ConfigSchema(), hcl2shim.HCL2ValueFromConfigValue(testCase.Config)))

			if diags.HasErrors() {
				for _, diag := range diags {
					t.Errorf("unexpected error: %s", diag.Description().Summary)
				}
			}
		})
	}
}

func TestBackendConfig_PrepareConfigValidation(t *testing.T) {
	cases := map[string]struct {
		config        cty.Value
		expectedDiags tfdiags.Diagnostics
	}{
		"null bucket": {
			config: cty.ObjectVal(map[string]cty.Value{
				"bucket": cty.NullVal(cty.String),
				"key":    cty.StringVal("test"),
				"region": cty.StringVal("us-west-2"),
			}),
			expectedDiags: tfdiags.Diagnostics{
				requiredAttributeErrDiag(cty.GetAttrPath("bucket")),
			},
		},
		"empty bucket": {
			config: cty.ObjectVal(map[string]cty.Value{
				"bucket": cty.StringVal(""),
				"key":    cty.StringVal("test"),
				"region": cty.StringVal("us-west-2"),
			}),
			expectedDiags: tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid Value",
					"The value cannot be empty or all whitespace",
					cty.GetAttrPath("bucket"),
				),
			},
		},

		"null key": {
			config: cty.ObjectVal(map[string]cty.Value{
				"bucket": cty.StringVal("test"),
				"key":    cty.NullVal(cty.String),
				"region": cty.StringVal("us-west-2"),
			}),
			expectedDiags: tfdiags.Diagnostics{
				requiredAttributeErrDiag(cty.GetAttrPath("key")),
			},
		},
		"empty key": {
			config: cty.ObjectVal(map[string]cty.Value{
				"bucket": cty.StringVal("test"),
				"key":    cty.StringVal(""),
				"region": cty.StringVal("us-west-2"),
			}),
			expectedDiags: tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid Value",
					"The value cannot be empty or all whitespace",
					cty.GetAttrPath("key"),
				),
			},
		},
		"key with leading slash": {
			config: cty.ObjectVal(map[string]cty.Value{
				"bucket": cty.StringVal("test"),
				"key":    cty.StringVal("/leading-slash"),
				"region": cty.StringVal("us-west-2"),
			}),
			expectedDiags: tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid Value",
					`The value must not start or end with "/"`,
					cty.GetAttrPath("key"),
				),
			},
		},
		"key with trailing slash": {
			config: cty.ObjectVal(map[string]cty.Value{
				"bucket": cty.StringVal("test"),
				"key":    cty.StringVal("trailing-slash/"),
				"region": cty.StringVal("us-west-2"),
			}),
			expectedDiags: tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid Value",
					`The value must not start or end with "/"`,
					cty.GetAttrPath("key"),
				),
			},
		},
		"key with double slash": {
			config: cty.ObjectVal(map[string]cty.Value{
				"bucket": cty.StringVal("test"),
				"key":    cty.StringVal("test/with/double//slash"),
				"region": cty.StringVal("us-west-2"),
			}),
			expectedDiags: tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid Value",
					`Value must not contain "//"`,
					cty.GetAttrPath("key"),
				),
			},
		},

		"null region": {
			config: cty.ObjectVal(map[string]cty.Value{
				"bucket": cty.StringVal("test"),
				"key":    cty.StringVal("test"),
				"region": cty.NullVal(cty.String),
			}),
			expectedDiags: tfdiags.Diagnostics{
				attributeErrDiag(
					"Missing region value",
					`The "region" attribute or the "AWS_REGION" or "AWS_DEFAULT_REGION" environment variables must be set.`,
					cty.GetAttrPath("region"),
				),
			},
		},
		"empty region": {
			config: cty.ObjectVal(map[string]cty.Value{
				"bucket": cty.StringVal("test"),
				"key":    cty.StringVal("test"),
				"region": cty.StringVal(""),
			}),
			expectedDiags: tfdiags.Diagnostics{
				attributeErrDiag(
					"Missing region value",
					`The "region" attribute or the "AWS_REGION" or "AWS_DEFAULT_REGION" environment variables must be set.`,
					cty.GetAttrPath("region"),
				),
			},
		},

		"workspace_key_prefix with leading slash": {
			config: cty.ObjectVal(map[string]cty.Value{
				"bucket":               cty.StringVal("test"),
				"key":                  cty.StringVal("test"),
				"region":               cty.StringVal("us-west-2"),
				"workspace_key_prefix": cty.StringVal("/env"),
			}),
			expectedDiags: tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid Value",
					`The value must not start or end with "/"`,
					cty.GetAttrPath("workspace_key_prefix"),
				),
			},
		},
		"workspace_key_prefix with trailing slash": {
			config: cty.ObjectVal(map[string]cty.Value{
				"bucket":               cty.StringVal("test"),
				"key":                  cty.StringVal("test"),
				"region":               cty.StringVal("us-west-2"),
				"workspace_key_prefix": cty.StringVal("env/"),
			}),
			expectedDiags: tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid Value",
					`The value must not start or end with "/"`,
					cty.GetAttrPath("workspace_key_prefix"),
				),
			},
		},

		"encyrption key conflict": {
			config: cty.ObjectVal(map[string]cty.Value{
				"bucket":               cty.StringVal("test"),
				"key":                  cty.StringVal("test"),
				"region":               cty.StringVal("us-west-2"),
				"workspace_key_prefix": cty.StringVal("env"),
				"sse_customer_key":     cty.StringVal("1hwbcNPGWL+AwDiyGmRidTWAEVmCWMKbEHA+Es8w75o="),
				"kms_key_id":           cty.StringVal("arn:aws:kms:us-west-2:111122223333:key/1234abcd-12ab-34cd-ab56-1234567890ab"),
			}),
			expectedDiags: tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid Attribute Combination",
					`Only one of kms_key_id, sse_customer_key can be set.`,
					cty.Path{},
				),
			},
		},

		"shared credentials file conflict": {
			config: cty.ObjectVal(map[string]cty.Value{
				"bucket":                   cty.StringVal("test"),
				"key":                      cty.StringVal("test"),
				"region":                   cty.StringVal("us-west-2"),
				"shared_credentials_file":  cty.StringVal("test"),
				"shared_credentials_files": cty.SetVal([]cty.Value{cty.StringVal("test2")}),
			}),
			expectedDiags: tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid Attribute Combination",
					`Only one of shared_credentials_file, shared_credentials_files can be set.`,
					cty.Path{},
				),
				attributeWarningDiag(
					"Deprecated Parameter",
					`The parameter "shared_credentials_file" is deprecated. Use parameter "shared_credentials_files" instead.`,
					cty.GetAttrPath("shared_credentials_file"),
				),
			},
		},

		"allowed forbidden account ids conflict": {
			config: cty.ObjectVal(map[string]cty.Value{
				"bucket":                cty.StringVal("test"),
				"key":                   cty.StringVal("test"),
				"region":                cty.StringVal("us-west-2"),
				"allowed_account_ids":   cty.SetVal([]cty.Value{cty.StringVal("012345678901")}),
				"forbidden_account_ids": cty.SetVal([]cty.Value{cty.StringVal("012345678901")}),
			}),
			expectedDiags: tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid Attribute Combination",
					`Only one of allowed_account_ids, forbidden_account_ids can be set.`,
					cty.Path{},
				),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			servicemocks.StashEnv(t)

			b := New()

			_, valDiags := b.PrepareConfig(populateSchema(t, b.ConfigSchema(), tc.config))

			if diff := cmp.Diff(valDiags, tc.expectedDiags, cmp.Comparer(diagnosticComparer)); diff != "" {
				t.Errorf("unexpected diagnostics difference: %s", diff)
			}
		})
	}
}

func TestBackendConfig_PrepareConfigWithEnvVars(t *testing.T) {
	cases := map[string]struct {
		config      cty.Value
		vars        map[string]string
		expectedErr string
	}{
		"region env var AWS_REGION": {
			config: cty.ObjectVal(map[string]cty.Value{
				"bucket": cty.StringVal("test"),
				"key":    cty.StringVal("test"),
				"region": cty.NullVal(cty.String),
			}),
			vars: map[string]string{
				"AWS_REGION": "us-west-1",
			},
		},
		"region env var AWS_DEFAULT_REGION": {
			config: cty.ObjectVal(map[string]cty.Value{
				"bucket": cty.StringVal("test"),
				"key":    cty.StringVal("test"),
				"region": cty.NullVal(cty.String),
			}),
			vars: map[string]string{
				"AWS_DEFAULT_REGION": "us-west-1",
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			servicemocks.StashEnv(t)

			b := New()

			for k, v := range tc.vars {
				os.Setenv(k, v)
			}

			_, valDiags := b.PrepareConfig(populateSchema(t, b.ConfigSchema(), tc.config))
			if tc.expectedErr != "" {
				if valDiags.Err() != nil {
					actualErr := valDiags.Err().Error()
					if !strings.Contains(actualErr, tc.expectedErr) {
						t.Fatalf("unexpected validation result: %v", valDiags.Err())
					}
				} else {
					t.Fatal("expected an error, got none")
				}
			} else if valDiags.Err() != nil {
				t.Fatalf("expected no error, got %s", valDiags.Err())
			}
		})
	}
}

type proxyCase struct {
	url           string
	expectedProxy string
}

func TestBackendConfig_Proxy(t *testing.T) {
	cases := map[string]struct {
		config               map[string]any
		environmentVariables map[string]string
		expectedDiags        tfdiags.Diagnostics
		urls                 []proxyCase
	}{
		"no config": {
			config: map[string]any{},
			urls: []proxyCase{
				{
					url:           "http://example.com",
					expectedProxy: "",
				},
				{
					url:           "https://example.com",
					expectedProxy: "",
				},
			},
		},

		"http_proxy empty string": {
			config: map[string]any{
				"http_proxy": "",
			},
			urls: []proxyCase{
				{
					url:           "http://example.com",
					expectedProxy: "",
				},
				{
					url:           "https://example.com",
					expectedProxy: "",
				},
			},
		},

		"http_proxy config": {
			config: map[string]any{
				"http_proxy": "http://http-proxy.test:1234",
			},
			expectedDiags: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Warning,
					"Missing HTTPS Proxy",
					fmt.Sprintf(
						"An HTTP proxy was set but no HTTPS proxy was. Using HTTP proxy %q for HTTPS requests. This behavior may change in future versions.\n\n"+
							"To specify no proxy for HTTPS, set the HTTPS to an empty string.",
						"http://http-proxy.test:1234"),
				),
			},
			urls: []proxyCase{
				{
					url:           "http://example.com",
					expectedProxy: "http://http-proxy.test:1234",
				},
				{
					url:           "https://example.com",
					expectedProxy: "http://http-proxy.test:1234",
				},
			},
		},

		"https_proxy config": {
			config: map[string]any{
				"https_proxy": "http://https-proxy.test:1234",
			},
			urls: []proxyCase{
				{
					url:           "http://example.com",
					expectedProxy: "",
				},
				{
					url:           "https://example.com",
					expectedProxy: "http://https-proxy.test:1234",
				},
			},
		},

		"http_proxy config https_proxy config": {
			config: map[string]any{
				"http_proxy":  "http://http-proxy.test:1234",
				"https_proxy": "http://https-proxy.test:1234",
			},
			urls: []proxyCase{
				{
					url:           "http://example.com",
					expectedProxy: "http://http-proxy.test:1234",
				},
				{
					url:           "https://example.com",
					expectedProxy: "http://https-proxy.test:1234",
				},
			},
		},

		"http_proxy config https_proxy config empty string": {
			config: map[string]any{
				"http_proxy":  "http://http-proxy.test:1234",
				"https_proxy": "",
			},
			urls: []proxyCase{
				{
					url:           "http://example.com",
					expectedProxy: "http://http-proxy.test:1234",
				},
				{
					url:           "https://example.com",
					expectedProxy: "",
				},
			},
		},

		"https_proxy config http_proxy config empty string": {
			config: map[string]any{
				"http_proxy":  "",
				"https_proxy": "http://https-proxy.test:1234",
			},
			urls: []proxyCase{
				{
					url:           "http://example.com",
					expectedProxy: "",
				},
				{
					url:           "https://example.com",
					expectedProxy: "http://https-proxy.test:1234",
				},
			},
		},

		"http_proxy config https_proxy config no_proxy config": {
			config: map[string]any{
				"http_proxy":  "http://http-proxy.test:1234",
				"https_proxy": "http://https-proxy.test:1234",
				"no_proxy":    "dont-proxy.test",
			},
			urls: []proxyCase{
				{
					url:           "http://example.com",
					expectedProxy: "http://http-proxy.test:1234",
				},
				{
					url:           "http://dont-proxy.test",
					expectedProxy: "",
				},
				{
					url:           "https://example.com",
					expectedProxy: "http://https-proxy.test:1234",
				},
				{
					url:           "https://dont-proxy.test",
					expectedProxy: "",
				},
			},
		},

		"HTTP_PROXY envvar": {
			config: map[string]any{},
			environmentVariables: map[string]string{
				"HTTP_PROXY": "http://http-proxy.test:1234",
			},
			urls: []proxyCase{
				{
					url:           "http://example.com",
					expectedProxy: "http://http-proxy.test:1234",
				},
				{
					url:           "https://example.com",
					expectedProxy: "",
				},
			},
		},

		"http_proxy envvar": {
			config: map[string]any{},
			environmentVariables: map[string]string{
				"http_proxy": "http://http-proxy.test:1234",
			},
			urls: []proxyCase{
				{
					url:           "http://example.com",
					expectedProxy: "http://http-proxy.test:1234",
				},
				{
					url:           "https://example.com",
					expectedProxy: "",
				},
			},
		},

		"HTTPS_PROXY envvar": {
			config: map[string]any{},
			environmentVariables: map[string]string{
				"HTTPS_PROXY": "http://https-proxy.test:1234",
			},
			urls: []proxyCase{
				{
					url:           "http://example.com",
					expectedProxy: "",
				},
				{
					url:           "https://example.com",
					expectedProxy: "http://https-proxy.test:1234",
				},
			},
		},

		"https_proxy envvar": {
			config: map[string]any{},
			environmentVariables: map[string]string{
				"https_proxy": "http://https-proxy.test:1234",
			},
			urls: []proxyCase{
				{
					url:           "http://example.com",
					expectedProxy: "",
				},
				{
					url:           "https://example.com",
					expectedProxy: "http://https-proxy.test:1234",
				},
			},
		},

		"http_proxy config HTTPS_PROXY envvar": {
			config: map[string]any{
				"http_proxy": "http://http-proxy.test:1234",
			},
			environmentVariables: map[string]string{
				"HTTPS_PROXY": "http://https-proxy.test:1234",
			},
			urls: []proxyCase{
				{
					url:           "http://example.com",
					expectedProxy: "http://http-proxy.test:1234",
				},
				{
					url:           "https://example.com",
					expectedProxy: "http://https-proxy.test:1234",
				},
			},
		},

		"http_proxy config https_proxy envvar": {
			config: map[string]any{
				"http_proxy": "http://http-proxy.test:1234",
			},
			environmentVariables: map[string]string{
				"https_proxy": "http://https-proxy.test:1234",
			},
			urls: []proxyCase{
				{
					url:           "http://example.com",
					expectedProxy: "http://http-proxy.test:1234",
				},
				{
					url:           "https://example.com",
					expectedProxy: "http://https-proxy.test:1234",
				},
			},
		},

		"http_proxy config NO_PROXY envvar": {
			config: map[string]any{
				"http_proxy": "http://http-proxy.test:1234",
			},
			environmentVariables: map[string]string{
				"NO_PROXY": "dont-proxy.test",
			},
			expectedDiags: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Warning,
					"Missing HTTPS Proxy",
					fmt.Sprintf(
						"An HTTP proxy was set but no HTTPS proxy was. Using HTTP proxy %q for HTTPS requests. This behavior may change in future versions.\n\n"+
							"To specify no proxy for HTTPS, set the HTTPS to an empty string.",
						"http://http-proxy.test:1234"),
				),
			},
			urls: []proxyCase{
				{
					url:           "http://example.com",
					expectedProxy: "http://http-proxy.test:1234",
				},
				{
					url:           "http://dont-proxy.test",
					expectedProxy: "",
				},
				{
					url:           "https://example.com",
					expectedProxy: "http://http-proxy.test:1234",
				},
				{
					url:           "https://dont-proxy.test",
					expectedProxy: "",
				},
			},
		},

		"http_proxy config no_proxy envvar": {
			config: map[string]any{
				"http_proxy": "http://http-proxy.test:1234",
			},
			environmentVariables: map[string]string{
				"no_proxy": "dont-proxy.test",
			},
			expectedDiags: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Warning,
					"Missing HTTPS Proxy",
					fmt.Sprintf(
						"An HTTP proxy was set but no HTTPS proxy was. Using HTTP proxy %q for HTTPS requests. This behavior may change in future versions.\n\n"+
							"To specify no proxy for HTTPS, set the HTTPS to an empty string.",
						"http://http-proxy.test:1234"),
				),
			},
			urls: []proxyCase{
				{
					url:           "http://example.com",
					expectedProxy: "http://http-proxy.test:1234",
				},
				{
					url:           "http://dont-proxy.test",
					expectedProxy: "",
				},
				{
					url:           "https://example.com",
					expectedProxy: "http://http-proxy.test:1234",
				},
				{
					url:           "https://dont-proxy.test",
					expectedProxy: "",
				},
			},
		},

		"HTTP_PROXY envvar HTTPS_PROXY envvar NO_PROXY envvar": {
			config: map[string]any{},
			environmentVariables: map[string]string{
				"HTTP_PROXY":  "http://http-proxy.test:1234",
				"HTTPS_PROXY": "http://https-proxy.test:1234",
				"NO_PROXY":    "dont-proxy.test",
			},
			urls: []proxyCase{
				{
					url:           "http://example.com",
					expectedProxy: "http://http-proxy.test:1234",
				},
				{
					url:           "http://dont-proxy.test",
					expectedProxy: "",
				},
				{
					url:           "https://example.com",
					expectedProxy: "http://https-proxy.test:1234",
				},
				{
					url:           "https://dont-proxy.test",
					expectedProxy: "",
				},
			},
		},

		"http_proxy config overrides HTTP_PROXY envvar": {
			config: map[string]any{
				"http_proxy": "http://config-proxy.test:1234",
			},
			environmentVariables: map[string]string{
				"HTTP_PROXY": "http://envvar-proxy.test:1234",
			},
			expectedDiags: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Warning,
					"Missing HTTPS Proxy",
					fmt.Sprintf(
						"An HTTP proxy was set but no HTTPS proxy was. Using HTTP proxy %q for HTTPS requests. This behavior may change in future versions.\n\n"+
							"To specify no proxy for HTTPS, set the HTTPS to an empty string.",
						"http://config-proxy.test:1234"),
				),
			},
			urls: []proxyCase{
				{
					url:           "http://example.com",
					expectedProxy: "http://config-proxy.test:1234",
				},
				{
					url:           "https://example.com",
					expectedProxy: "http://config-proxy.test:1234",
				},
			},
		},

		"https_proxy config overrides HTTPS_PROXY envvar": {
			config: map[string]any{
				"https_proxy": "http://config-proxy.test:1234",
			},
			environmentVariables: map[string]string{
				"HTTPS_PROXY": "http://envvar-proxy.test:1234",
			},
			urls: []proxyCase{
				{
					url:           "http://example.com",
					expectedProxy: "",
				},
				{
					url:           "https://example.com",
					expectedProxy: "http://config-proxy.test:1234",
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			config := map[string]any{
				"region":                      "us-west-2",
				"bucket":                      "tf-test",
				"key":                         "state",
				"skip_credentials_validation": true,
				"skip_requesting_account_id":  true,
				"access_key":                  servicemocks.MockStaticAccessKey,
				"secret_key":                  servicemocks.MockStaticSecretKey,
			}

			for k, v := range tc.environmentVariables {
				t.Setenv(k, v)
			}

			maps.Copy(config, tc.config)

			raw, diags := testBackendConfigDiags(t, New(), backend.TestWrapConfig(config))
			b := raw.(*Backend)

			if diff := cmp.Diff(diags, tc.expectedDiags, cmp.Comparer(diagnosticComparer)); diff != "" {
				t.Errorf("unexpected diagnostics difference: %s", diff)
			}

			client := b.awsConfig.HTTPClient
			bClient, ok := client.(*awshttp.BuildableClient)
			if !ok {
				t.Fatalf("expected awshttp.BuildableClient, got %T", client)
			}
			transport := bClient.GetTransport()
			proxyF := transport.Proxy

			for _, url := range tc.urls {
				req, _ := http.NewRequest("GET", url.url, nil)
				pUrl, err := proxyF(req)
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
				if url.expectedProxy != "" {
					if pUrl == nil {
						t.Errorf("expected proxy for %q, got none", url.url)
					} else if pUrl.String() != url.expectedProxy {
						t.Errorf("expected proxy %q for %q, got %q", url.expectedProxy, url.url, pUrl.String())
					}
				} else {
					if pUrl != nil {
						t.Errorf("expected no proxy for %q, got %q", url.url, pUrl.String())
					}
				}
			}
		})
	}
}

func TestBackendBasic(t *testing.T) {
	testACC(t)

	ctx := context.TODO()

	bucketName := fmt.Sprintf("terraform-remote-s3-test-%x", time.Now().Unix())
	keyName := "testState"

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":  bucketName,
		"key":     keyName,
		"encrypt": true,
		"region":  "us-west-1",
	})).(*Backend)

	createS3Bucket(ctx, t, b.s3Client, bucketName, b.awsConfig.Region)
	defer deleteS3Bucket(ctx, t, b.s3Client, bucketName, b.awsConfig.Region)

	backend.TestBackendStates(t, b)
}

func TestBackendLocked(t *testing.T) {
	testACC(t)

	ctx := context.TODO()

	bucketName := fmt.Sprintf("terraform-remote-s3-test-%x", time.Now().Unix())
	keyName := "test/state"

	b1 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":         bucketName,
		"key":            keyName,
		"encrypt":        true,
		"dynamodb_table": bucketName,
		"region":         "us-west-1",
	})).(*Backend)

	b2 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":         bucketName,
		"key":            keyName,
		"encrypt":        true,
		"dynamodb_table": bucketName,
		"region":         "us-west-1",
	})).(*Backend)

	createS3Bucket(ctx, t, b1.s3Client, bucketName, b1.awsConfig.Region)
	defer deleteS3Bucket(ctx, t, b1.s3Client, bucketName, b1.awsConfig.Region)
	createDynamoDBTable(ctx, t, b1.dynClient, bucketName)
	defer deleteDynamoDBTable(ctx, t, b1.dynClient, bucketName)

	backend.TestBackendStateLocks(t, b1, b2)
	backend.TestBackendStateForceUnlock(t, b1, b2)
}

func TestBackendKmsKeyId(t *testing.T) {
	testACC(t)

	testCases := map[string]struct {
		config        map[string]any
		expectedKeyId string
		expectedDiags tfdiags.Diagnostics
	}{
		"valid": {
			config: map[string]any{
				"kms_key_id": "arn:aws:kms:us-west-2:111122223333:key/1234abcd-12ab-34cd-ab56-1234567890ab",
			},
			expectedKeyId: "arn:aws:kms:us-west-2:111122223333:key/1234abcd-12ab-34cd-ab56-1234567890ab",
		},

		"invalid": {
			config: map[string]any{
				"kms_key_id": "not-an-arn",
			},
			expectedDiags: tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid KMS Key ID",
					`Value must be a valid KMS Key ID, got "not-an-arn"`,
					cty.GetAttrPath("kms_key_id"),
				),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			bucketName := fmt.Sprintf("terraform-remote-s3-test-%x", time.Now().Unix())
			config := map[string]any{
				"bucket":  bucketName,
				"encrypt": true,
				"key":     "test-SSE-KMS",
				"region":  "us-west-1",
			}
			maps.Copy(config, tc.config)

			b := New().(*Backend)
			configSchema := populateSchema(t, b.ConfigSchema(), hcl2shim.HCL2ValueFromConfigValue(config))

			configSchema, diags := b.PrepareConfig(configSchema)

			if !diags.HasErrors() {
				confDiags := b.Configure(configSchema)
				diags = diags.Append(confDiags)
			}

			if diff := cmp.Diff(diags, tc.expectedDiags, cmp.Comparer(diagnosticComparer)); diff != "" {
				t.Fatalf("unexpected diagnostics difference: %s", diff)
			}

			if tc.expectedKeyId != "" {
				if string(b.kmsKeyID) != tc.expectedKeyId {
					t.Fatal("unexpected value for KMS key Id")
				}
			}
		})
	}
}

func TestBackendSSECustomerKey(t *testing.T) {
	testACC(t)

	ctx := context.TODO()

	testCases := map[string]struct {
		config               map[string]any
		environmentVariables map[string]string
		expectedKey          string
		expectedDiags        tfdiags.Diagnostics
	}{
		// config
		"config valid": {
			config: map[string]any{
				"sse_customer_key": "4Dm1n4rphuFgawxuzY/bEfvLf6rYK0gIjfaDSLlfXNk=",
			},
			expectedKey: string(must(base64.StdEncoding.DecodeString("4Dm1n4rphuFgawxuzY/bEfvLf6rYK0gIjfaDSLlfXNk="))),
		},
		"config invalid length": {
			config: map[string]any{
				"sse_customer_key": "test",
			},
			expectedDiags: tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid sse_customer_key value",
					"sse_customer_key must be 44 characters in length",
					cty.GetAttrPath("sse_customer_key"),
				),
			},
		},
		"config invalid encoding": {
			config: map[string]any{
				"sse_customer_key": "====CT70aTYB2JGff7AjQtwbiLkwH4npICay1PWtmdka",
			},
			expectedDiags: tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid sse_customer_key value",
					"sse_customer_key must be base64 encoded: illegal base64 data at input byte 0",
					cty.GetAttrPath("sse_customer_key"),
				),
			},
		},

		// env var
		"envvar valid": {
			environmentVariables: map[string]string{
				"AWS_SSE_CUSTOMER_KEY": "4Dm1n4rphuFgawxuzY/bEfvLf6rYK0gIjfaDSLlfXNk=",
			},
			expectedKey: string(must(base64.StdEncoding.DecodeString("4Dm1n4rphuFgawxuzY/bEfvLf6rYK0gIjfaDSLlfXNk="))),
		},
		"envvar invalid length": {
			environmentVariables: map[string]string{
				"AWS_SSE_CUSTOMER_KEY": "test",
			},
			expectedDiags: tfdiags.Diagnostics{
				wholeBodyErrDiag(
					"Invalid AWS_SSE_CUSTOMER_KEY value",
					`The environment variable "AWS_SSE_CUSTOMER_KEY" must be 44 characters in length`,
				),
			},
		},
		"envvar invalid encoding": {
			environmentVariables: map[string]string{
				"AWS_SSE_CUSTOMER_KEY": "====CT70aTYB2JGff7AjQtwbiLkwH4npICay1PWtmdka",
			},
			expectedDiags: tfdiags.Diagnostics{
				wholeBodyErrDiag(
					"Invalid AWS_SSE_CUSTOMER_KEY value",
					`The environment variable "AWS_SSE_CUSTOMER_KEY" must be base64 encoded: illegal base64 data at input byte 0`,
				),
			},
		},

		// conflict
		"config kms_key_id and envvar AWS_SSE_CUSTOMER_KEY": {
			config: map[string]any{
				"kms_key_id": "arn:aws:kms:us-west-2:111122223333:key/1234abcd-12ab-34cd-ab56-1234567890ab",
			},
			environmentVariables: map[string]string{
				"AWS_SSE_CUSTOMER_KEY": "4Dm1n4rphuFgawxuzY/bEfvLf6rYK0gIjfaDSLlfXNk=",
			},
			expectedDiags: tfdiags.Diagnostics{
				wholeBodyErrDiag(
					"Invalid encryption configuration",
					encryptionKeyConflictEnvVarError,
				),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			bucketName := fmt.Sprintf("terraform-remote-s3-test-%x", time.Now().Unix())
			config := map[string]any{
				"bucket":  bucketName,
				"encrypt": true,
				"key":     "test-SSE-C",
				"region":  "us-west-1",
			}
			maps.Copy(config, tc.config)

			oldEnv := os.Environ() // For now, save without clearing
			defer servicemocks.PopEnv(oldEnv)
			for k, v := range tc.environmentVariables {
				os.Setenv(k, v)
			}

			b := New().(*Backend)
			configSchema := populateSchema(t, b.ConfigSchema(), hcl2shim.HCL2ValueFromConfigValue(config))

			configSchema, diags := b.PrepareConfig(configSchema)

			if !diags.HasErrors() {
				confDiags := b.Configure(configSchema)
				diags = diags.Append(confDiags)
			}

			if diff := cmp.Diff(diags, tc.expectedDiags, cmp.Comparer(diagnosticComparer)); diff != "" {
				t.Fatalf("unexpected diagnostics difference: %s", diff)
			}

			if tc.expectedKey != "" {
				if string(b.customerEncryptionKey) != tc.expectedKey {
					t.Fatal("unexpected value for customer encryption key")
				}
			}

			if !diags.HasErrors() {
				createS3Bucket(ctx, t, b.s3Client, bucketName, b.awsConfig.Region)
				defer deleteS3Bucket(ctx, t, b.s3Client, bucketName, b.awsConfig.Region)

				backend.TestBackendStates(t, b)
			}
		})
	}
}

// add some extra junk in S3 to try and confuse the env listing.
func TestBackendExtraPaths(t *testing.T) {
	testACC(t)

	ctx := context.TODO()

	bucketName := fmt.Sprintf("terraform-remote-s3-test-%x", time.Now().Unix())
	keyName := "test/state/tfstate"

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":  bucketName,
		"key":     keyName,
		"encrypt": true,
	})).(*Backend)

	createS3Bucket(ctx, t, b.s3Client, bucketName, b.awsConfig.Region)
	defer deleteS3Bucket(ctx, t, b.s3Client, bucketName, b.awsConfig.Region)

	// put multiple states in old env paths.
	s1 := states.NewState()
	s2 := states.NewState()

	// RemoteClient to Put things in various paths
	client := &RemoteClient{
		s3Client:             b.s3Client,
		dynClient:            b.dynClient,
		bucketName:           b.bucketName,
		path:                 b.path("s1"),
		serverSideEncryption: b.serverSideEncryption,
		acl:                  b.acl,
		kmsKeyID:             b.kmsKeyID,
		ddbTable:             b.ddbTable,
	}

	// Write the first state
	stateMgr := &remote.State{Client: client}
	if err := stateMgr.WriteState(s1); err != nil {
		t.Fatal(err)
	}
	if err := stateMgr.PersistState(nil); err != nil {
		t.Fatal(err)
	}

	// Write the second state
	// Note a new state manager - otherwise, because these
	// states are equal, the state will not Put to the remote
	client.path = b.path("s2")
	stateMgr2 := &remote.State{Client: client}
	if err := stateMgr2.WriteState(s2); err != nil {
		t.Fatal(err)
	}
	if err := stateMgr2.PersistState(nil); err != nil {
		t.Fatal(err)
	}

	s2Lineage := stateMgr2.StateSnapshotMeta().Lineage

	if err := checkStateList(b, []string{"default", "s1", "s2"}); err != nil {
		t.Fatal(err)
	}

	// put a state in an env directory name
	client.path = b.workspaceKeyPrefix + "/error"
	if err := stateMgr.WriteState(states.NewState()); err != nil {
		t.Fatal(err)
	}
	if err := stateMgr.PersistState(nil); err != nil {
		t.Fatal(err)
	}
	if err := checkStateList(b, []string{"default", "s1", "s2"}); err != nil {
		t.Fatal(err)
	}

	// add state with the wrong key for an existing env
	client.path = b.workspaceKeyPrefix + "/s2/notTestState"
	if err := stateMgr.WriteState(states.NewState()); err != nil {
		t.Fatal(err)
	}
	if err := stateMgr.PersistState(nil); err != nil {
		t.Fatal(err)
	}
	if err := checkStateList(b, []string{"default", "s1", "s2"}); err != nil {
		t.Fatal(err)
	}

	// remove the state with extra subkey
	if err := client.Delete(); err != nil {
		t.Fatal(err)
	}

	// delete the real workspace
	if err := b.DeleteWorkspace("s2", true); err != nil {
		t.Fatal(err)
	}

	if err := checkStateList(b, []string{"default", "s1"}); err != nil {
		t.Fatal(err)
	}

	// fetch that state again, which should produce a new lineage
	s2Mgr, err := b.StateMgr("s2")
	if err != nil {
		t.Fatal(err)
	}
	if err := s2Mgr.RefreshState(); err != nil {
		t.Fatal(err)
	}

	if s2Mgr.(*remote.State).StateSnapshotMeta().Lineage == s2Lineage {
		t.Fatal("state s2 was not deleted")
	}
	_ = s2Mgr.State() // We need the side-effect
	s2Lineage = stateMgr.StateSnapshotMeta().Lineage

	// add a state with a key that matches an existing environment dir name
	client.path = b.workspaceKeyPrefix + "/s2/"
	if err := stateMgr.WriteState(states.NewState()); err != nil {
		t.Fatal(err)
	}
	if err := stateMgr.PersistState(nil); err != nil {
		t.Fatal(err)
	}

	// make sure s2 is OK
	s2Mgr, err = b.StateMgr("s2")
	if err != nil {
		t.Fatal(err)
	}
	if err := s2Mgr.RefreshState(); err != nil {
		t.Fatal(err)
	}

	if stateMgr.StateSnapshotMeta().Lineage != s2Lineage {
		t.Fatal("we got the wrong state for s2")
	}

	if err := checkStateList(b, []string{"default", "s1", "s2"}); err != nil {
		t.Fatal(err)
	}
}

// ensure we can separate the workspace prefix when it also matches the prefix
// of the workspace name itself.
func TestBackendPrefixInWorkspace(t *testing.T) {
	testACC(t)

	ctx := context.TODO()

	bucketName := fmt.Sprintf("terraform-remote-s3-test-%x", time.Now().Unix())

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":               bucketName,
		"key":                  "test-env.tfstate",
		"workspace_key_prefix": "env",
	})).(*Backend)

	createS3Bucket(ctx, t, b.s3Client, bucketName, b.awsConfig.Region)
	defer deleteS3Bucket(ctx, t, b.s3Client, bucketName, b.awsConfig.Region)

	// get a state that contains the prefix as a substring
	sMgr, err := b.StateMgr("env-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := sMgr.RefreshState(); err != nil {
		t.Fatal(err)
	}

	if err := checkStateList(b, []string{"default", "env-1"}); err != nil {
		t.Fatal(err)
	}
}

func TestBackendRestrictedRoot_Default(t *testing.T) {
	testACC(t)

	ctx := context.TODO()

	bucketName := fmt.Sprintf("terraform-remote-s3-test-%x", time.Now().Unix())
	workspacePrefix := defaultWorkspaceKeyPrefix

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket": bucketName,
		"key":    "test/test-env.tfstate",
	})).(*Backend)

	createS3Bucket(ctx, t, b.s3Client, bucketName, b.awsConfig.Region, s3BucketWithPolicy(fmt.Sprintf(`{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Sid": "Statement1",
			"Effect": "Deny",
			"Principal": "*",
			"Action": "s3:ListBucket",
			"Resource": "arn:aws:s3:::%[1]s",
			"Condition": {
				"StringLike": {
					"s3:prefix": "%[2]s/*"
				}
			}
		}
	]
}`, bucketName, workspacePrefix)))
	defer deleteS3Bucket(ctx, t, b.s3Client, bucketName, b.awsConfig.Region)

	sMgr, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}
	if err := sMgr.RefreshState(); err != nil {
		t.Fatal(err)
	}

	if err := checkStateList(b, []string{"default"}); err != nil {
		t.Fatal(err)
	}
}

func TestBackendRestrictedRoot_NamedPrefix(t *testing.T) {
	testACC(t)

	ctx := context.TODO()

	bucketName := fmt.Sprintf("terraform-remote-s3-test-%x", time.Now().Unix())
	workspacePrefix := "prefix"

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":               bucketName,
		"key":                  "test/test-env.tfstate",
		"workspace_key_prefix": workspacePrefix,
	})).(*Backend)

	createS3Bucket(ctx, t, b.s3Client, bucketName, b.awsConfig.Region, s3BucketWithPolicy(fmt.Sprintf(`{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Sid": "Statement1",
			"Effect": "Deny",
			"Principal": "*",
			"Action": "s3:ListBucket",
			"Resource": "arn:aws:s3:::%[1]s",
			"Condition": {
				"StringLike": {
					"s3:prefix": "%[2]s/*"
				}
			}
		}
	]
}`, bucketName, workspacePrefix)))
	defer deleteS3Bucket(ctx, t, b.s3Client, bucketName, b.awsConfig.Region)

	_, err := b.StateMgr(backend.DefaultStateName)
	if err == nil {
		t.Fatal("expected AccessDenied error, got none")
	}
	if s := err.Error(); !strings.Contains(s, fmt.Sprintf("Unable to list objects in S3 bucket %q with prefix %q:", bucketName, workspacePrefix+"/")) {
		t.Fatalf("expected AccessDenied error, got: %s", s)
	}
}

func TestBackendWrongRegion(t *testing.T) {
	testACC(t)

	ctx := context.TODO()

	bucketName := fmt.Sprintf("terraform-remote-s3-test-%x", time.Now().Unix())
	keyName := "testState"

	bucketRegion := "us-west-1"
	backendRegion := "us-east-1"
	if backendRegion == bucketRegion {
		t.Fatalf("bucket region and backend region must not be the same")
	}

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":  bucketName,
		"key":     keyName,
		"encrypt": true,
		"region":  backendRegion,
	})).(*Backend)

	createS3Bucket(ctx, t, b.s3Client, bucketName, bucketRegion)
	defer deleteS3Bucket(ctx, t, b.s3Client, bucketName, bucketRegion)

	if _, err := b.StateMgr(backend.DefaultStateName); err == nil {
		t.Fatal("expected error, got none")
	} else {
		if regionErr, ok := As[bucketRegionError](err); ok {
			if a, e := regionErr.bucketRegion, bucketRegion; a != e {
				t.Errorf("expected bucket region %q, got %q", e, a)
			}
			if a, e := regionErr.requestRegion, backendRegion; a != e {
				t.Errorf("expected request region %q, got %q", e, a)
			}
		} else {
			t.Fatalf("expected bucket region error, got: %v", err)
		}
	}
}

func TestBackendS3ObjectLock(t *testing.T) {
	testACC(t)

	ctx := context.TODO()

	bucketName := fmt.Sprintf("terraform-remote-s3-test-%x", time.Now().Unix())
	keyName := "testState"

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":  bucketName,
		"key":     keyName,
		"encrypt": true,
		"region":  "us-west-1",
	})).(*Backend)

	createS3Bucket(ctx, t, b.s3Client, bucketName, b.awsConfig.Region,
		s3BucketWithVersioning,
		s3BucketWithObjectLock(s3types.ObjectLockRetentionModeCompliance),
	)
	defer deleteS3Bucket(ctx, t, b.s3Client, bucketName, b.awsConfig.Region)

	backend.TestBackendStates(t, b)
}

func TestKeyEnv(t *testing.T) {
	testACC(t)

	ctx := context.TODO()

	keyName := "some/paths/tfstate"

	bucket0Name := fmt.Sprintf("terraform-remote-s3-test-%x-0", time.Now().Unix())
	b0 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":               bucket0Name,
		"key":                  keyName,
		"encrypt":              true,
		"workspace_key_prefix": "",
	})).(*Backend)

	createS3Bucket(ctx, t, b0.s3Client, bucket0Name, b0.awsConfig.Region)
	defer deleteS3Bucket(ctx, t, b0.s3Client, bucket0Name, b0.awsConfig.Region)

	bucket1Name := fmt.Sprintf("terraform-remote-s3-test-%x-1", time.Now().Unix())
	b1 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":               bucket1Name,
		"key":                  keyName,
		"encrypt":              true,
		"workspace_key_prefix": "project/env:",
	})).(*Backend)

	createS3Bucket(ctx, t, b1.s3Client, bucket1Name, b1.awsConfig.Region)
	defer deleteS3Bucket(ctx, t, b1.s3Client, bucket1Name, b1.awsConfig.Region)

	bucket2Name := fmt.Sprintf("terraform-remote-s3-test-%x-2", time.Now().Unix())
	b2 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":  bucket2Name,
		"key":     keyName,
		"encrypt": true,
	})).(*Backend)

	createS3Bucket(ctx, t, b2.s3Client, bucket2Name, b2.awsConfig.Region)
	defer deleteS3Bucket(ctx, t, b2.s3Client, bucket2Name, b2.awsConfig.Region)

	if err := testGetWorkspaceForKey(b0, "some/paths/tfstate", ""); err != nil {
		t.Fatal(err)
	}

	if err := testGetWorkspaceForKey(b0, "ws1/some/paths/tfstate", "ws1"); err != nil {
		t.Fatal(err)
	}

	if err := testGetWorkspaceForKey(b1, "project/env:/ws1/some/paths/tfstate", "ws1"); err != nil {
		t.Fatal(err)
	}

	if err := testGetWorkspaceForKey(b1, "project/env:/ws2/some/paths/tfstate", "ws2"); err != nil {
		t.Fatal(err)
	}

	if err := testGetWorkspaceForKey(b2, "env:/ws3/some/paths/tfstate", "ws3"); err != nil {
		t.Fatal(err)
	}

	backend.TestBackendStates(t, b0)
	backend.TestBackendStates(t, b1)
	backend.TestBackendStates(t, b2)
}

func TestAssumeRole_PrepareConfigValidation(t *testing.T) {
	path := cty.GetAttrPath("field")

	cases := map[string]struct {
		config        map[string]cty.Value
		expectedDiags tfdiags.Diagnostics
	}{
		"basic": {
			config: map[string]cty.Value{
				"role_arn": cty.StringVal("arn:aws:iam::123456789012:role/testrole"),
			},
		},

		"invalid ARN": {
			config: map[string]cty.Value{
				"role_arn": cty.StringVal("not an arn"),
			},
			expectedDiags: tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid ARN",
					`The value "not an arn" cannot be parsed as an ARN: arn: invalid prefix`,
					path.IndexInt(0).GetAttr("role_arn"),
				),
			},
		},

		"no role_arn": {
			config: map[string]cty.Value{},
			expectedDiags: tfdiags.Diagnostics{
				requiredAttributeErrDiag(path.IndexInt(0).GetAttr("role_arn")),
			},
		},

		"with duration": {
			config: map[string]cty.Value{
				"role_arn": cty.StringVal("arn:aws:iam::123456789012:role/testrole"),
				"duration": cty.StringVal("2h"),
			},
		},

		"invalid duration": {
			config: map[string]cty.Value{
				"role_arn": cty.StringVal("arn:aws:iam::123456789012:role/testrole"),
				"duration": cty.StringVal("two hours"),
			},
			expectedDiags: tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid Duration",
					`The value "two hours" cannot be parsed as a duration: time: invalid duration "two hours"`,
					path.IndexInt(0).GetAttr("duration"),
				),
			},
		},

		"with external_id": {
			config: map[string]cty.Value{
				"role_arn":    cty.StringVal("arn:aws:iam::123456789012:role/testrole"),
				"external_id": cty.StringVal("external-id"),
			},
		},

		"empty external_id": {
			config: map[string]cty.Value{
				"role_arn":    cty.StringVal("arn:aws:iam::123456789012:role/testrole"),
				"external_id": cty.StringVal(""),
			},
			expectedDiags: tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid Value Length",
					`Length must be between 2 and 1224, had 0`,
					path.IndexInt(0).GetAttr("external_id"),
				),
			},
		},

		"with policy": {
			config: map[string]cty.Value{
				"role_arn": cty.StringVal("arn:aws:iam::123456789012:role/testrole"),
				"policy":   cty.StringVal("{}"),
			},
		},

		"invalid policy": {
			config: map[string]cty.Value{
				"role_arn": cty.StringVal("arn:aws:iam::123456789012:role/testrole"),
				"policy":   cty.StringVal(""),
			},
			expectedDiags: tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid Value",
					`The value cannot be empty or all whitespace`,
					path.IndexInt(0).GetAttr("policy"),
				),
			},
		},

		"with policy_arns": {
			config: map[string]cty.Value{
				"role_arn": cty.StringVal("arn:aws:iam::123456789012:role/testrole"),
				"policy_arns": cty.SetVal([]cty.Value{
					cty.StringVal("arn:aws:iam::123456789012:policy/testpolicy"),
				}),
			},
		},

		"invalid policy_arns": {
			config: map[string]cty.Value{
				"role_arn": cty.StringVal("arn:aws:iam::123456789012:role/testrole"),
				"policy_arns": cty.SetVal([]cty.Value{
					cty.StringVal("not an arn"),
				}),
			},
			expectedDiags: tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid ARN",
					`The value "not an arn" cannot be parsed as an ARN: arn: invalid prefix`,
					path.IndexInt(0).GetAttr("policy_arns").IndexString("not an arn"),
				),
			},
		},

		"with session_name": {
			config: map[string]cty.Value{
				"role_arn":     cty.StringVal("arn:aws:iam::123456789012:role/testrole"),
				"session_name": cty.StringVal("session-name"),
			},
		},

		"source_identity": {
			config: map[string]cty.Value{
				"role_arn":        cty.StringVal("arn:aws:iam::123456789012:role/testrole"),
				"source_identity": cty.StringVal("source-identity"),
			},
		},

		"with tags": {
			config: map[string]cty.Value{
				"role_arn": cty.StringVal("arn:aws:iam::123456789012:role/testrole"),
				"tags": cty.MapVal(map[string]cty.Value{
					"tag-key": cty.StringVal("tag-value"),
				}),
			},
		},

		"with transitive_tag_keys": {
			config: map[string]cty.Value{
				"role_arn": cty.StringVal("arn:aws:iam::123456789012:role/testrole"),
				"transitive_tag_keys": cty.SetVal([]cty.Value{
					cty.StringVal("tag-key"),
				}),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			schema := assumeRoleSchema.NestedObject.Attributes
			vals := make(map[string]cty.Value, len(schema))
			for name, attrSchema := range schema {
				if val, ok := tc.config[name]; ok {
					vals[name] = val
				} else {
					vals[name] = cty.NullVal(attrSchema.SchemaAttribute().Type)
				}
			}
			config := cty.ListVal([]cty.Value{cty.ObjectVal(vals)})

			var diags tfdiags.Diagnostics
			validateListNestedAttribute(assumeRoleSchema, config, path, &diags)

			if diff := cmp.Diff(diags, tc.expectedDiags, cmp.Comparer(diagnosticComparer)); diff != "" {
				t.Errorf("unexpected diagnostics difference: %s", diff)
			}
		})
	}
}

// TestBackend_CoerceValue verifies a cty.Object can be coerced into
// an s3 backend Block
//
// This serves as a smoke test for use of the terraform_remote_state
// data source with the s3 backend, replicating the process that
// data source uses. The returned value is ignored as the object is
// large (representing the entire s3 backend schema) and the focus of
// this test is early detection of coercion failures.
func TestBackend_CoerceValue(t *testing.T) {
	testCases := map[string]struct {
		Input   cty.Value
		WantErr string
	}{
		"basic": {
			Input: cty.ObjectVal(map[string]cty.Value{
				"bucket": cty.StringVal("test"),
				"key":    cty.StringVal("test"),
			}),
		},
		"missing bucket": {
			Input: cty.ObjectVal(map[string]cty.Value{
				"key": cty.StringVal("test"),
			}),
			WantErr: `attribute "bucket" is required`,
		},
		"missing key": {
			Input: cty.ObjectVal(map[string]cty.Value{
				"bucket": cty.StringVal("test"),
			}),
			WantErr: `attribute "key" is required`,
		},
		"assume_role": {
			Input: cty.ObjectVal(map[string]cty.Value{
				"bucket": cty.StringVal("test"),
				"key":    cty.StringVal("test"),
				"assume_role": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"role_arn": cty.StringVal("test"),
					}),
				}),
			}),
		},
		"assume_role missing role_arn": {
			Input: cty.ObjectVal(map[string]cty.Value{
				"bucket": cty.StringVal("test"),
				"key":    cty.StringVal("test"),
				"assume_role": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{}),
				}),
			}),
			WantErr: `.assume_role: incorrect list element type: attribute "role_arn" is required`,
		},
		"assume_role_with_web_identity": {
			Input: cty.ObjectVal(map[string]cty.Value{
				"bucket": cty.StringVal("test"),
				"key":    cty.StringVal("test"),
				"assume_role_with_web_identity": cty.ObjectVal(map[string]cty.Value{
					"role_arn": cty.StringVal("test"),
				}),
			}),
		},
		"assume_role_with_web_identity missing role_arn": {
			Input: cty.ObjectVal(map[string]cty.Value{
				"bucket":                        cty.StringVal("test"),
				"key":                           cty.StringVal("test"),
				"assume_role_with_web_identity": cty.ObjectVal(map[string]cty.Value{}),
			}),
			WantErr: `.assume_role_with_web_identity: attribute "role_arn" is required`,
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			b := Backend{}
			// Skip checking the returned cty.Value as this object will be large.
			_, gotErrObj := b.ConfigSchema().CoerceValue(test.Input)

			if gotErrObj == nil {
				if test.WantErr != "" {
					t.Fatalf("coersion succeeded; want error: %q", test.WantErr)
				}
			} else {
				gotErr := tfdiags.FormatError(gotErrObj)
				if gotErr != test.WantErr {
					t.Fatalf("wrong error\ngot:  %s\nwant: %s", gotErr, test.WantErr)
				}
			}
		})
	}
}

func testGetWorkspaceForKey(b *Backend, key string, expected string) error {
	if actual := b.keyEnv(key); actual != expected {
		return fmt.Errorf("incorrect workspace for key[%q]. Expected[%q]: Actual[%q]", key, expected, actual)
	}
	return nil
}

func checkStateList(b backend.Backend, expected []string) error {
	states, err := b.Workspaces()
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(states, expected) {
		return fmt.Errorf("incorrect states listed: %q", states)
	}
	return nil
}

type createS3BucketOptions struct {
	versioning     bool
	objectLockMode s3types.ObjectLockRetentionMode
	policy         string
}

type createS3BucketOptionsFunc func(*createS3BucketOptions)

func createS3Bucket(ctx context.Context, t *testing.T, s3Client *s3.Client, bucketName, region string, optFns ...createS3BucketOptionsFunc) {
	t.Helper()

	var opts createS3BucketOptions
	for _, f := range optFns {
		f(&opts)
	}

	createBucketReq := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}
	if region != "us-east-1" {
		createBucketReq.CreateBucketConfiguration = &s3types.CreateBucketConfiguration{
			LocationConstraint: s3types.BucketLocationConstraint(region),
		}
	}
	if opts.objectLockMode != "" {
		createBucketReq.ObjectLockEnabledForBucket = aws.Bool(true)
	}

	// Be clear about what we're doing in case the user needs to clean
	// this up later.
	t.Logf("creating S3 bucket %s in %s", bucketName, region)
	_, err := s3Client.CreateBucket(ctx, createBucketReq, s3WithRegion(region))
	if err != nil {
		t.Fatal("failed to create test S3 bucket:", err)
	}

	if opts.versioning {
		_, err := s3Client.PutBucketVersioning(ctx, &s3.PutBucketVersioningInput{
			Bucket: aws.String(bucketName),
			VersioningConfiguration: &s3types.VersioningConfiguration{
				Status: s3types.BucketVersioningStatusEnabled,
			},
		})
		if err != nil {
			t.Fatalf("failed enabling versioning: %s", err)
		}
	}

	if opts.objectLockMode != "" {
		_, err := s3Client.PutObjectLockConfiguration(ctx, &s3.PutObjectLockConfigurationInput{
			Bucket: aws.String(bucketName),
			ObjectLockConfiguration: &s3types.ObjectLockConfiguration{
				ObjectLockEnabled: s3types.ObjectLockEnabledEnabled,
				Rule: &s3types.ObjectLockRule{
					DefaultRetention: &s3types.DefaultRetention{
						Days: aws.Int32(1),
						Mode: opts.objectLockMode,
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("failed enabling object locking: %s", err)
		}
	}

	if opts.policy != "" {
		_, err := s3Client.PutBucketPolicy(ctx, &s3.PutBucketPolicyInput{
			Bucket: aws.String(bucketName),
			Policy: &opts.policy,
		})
		if err != nil {
			t.Fatalf("failed setting bucket policy: %s", err)
		}
	}
}

func s3BucketWithVersioning(opts *createS3BucketOptions) {
	opts.versioning = true
}

func s3BucketWithObjectLock(mode s3types.ObjectLockRetentionMode) createS3BucketOptionsFunc {
	return func(opts *createS3BucketOptions) {
		opts.objectLockMode = mode
	}
}

func s3BucketWithPolicy(policy string) createS3BucketOptionsFunc {
	return func(opts *createS3BucketOptions) {
		opts.policy = policy
	}
}

func deleteS3Bucket(ctx context.Context, t *testing.T, s3Client *s3.Client, bucketName, region string) {
	t.Helper()

	warning := "WARNING: Failed to delete the test S3 bucket. It may have been left in your AWS account and may incur storage charges. (error was %s)"

	// first we have to get rid of the env objects, or we can't delete the bucket
	resp, err := s3Client.ListObjects(ctx, &s3.ListObjectsInput{Bucket: &bucketName}, s3WithRegion(region))
	if err != nil {
		t.Logf(warning, err)
		return
	}
	for _, obj := range resp.Contents {
		if _, err := s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &bucketName, Key: obj.Key}, s3WithRegion(region)); err != nil {
			// this will need cleanup no matter what, so just warn and exit
			t.Logf(warning, err)
			return
		}
	}

	if _, err := s3Client.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: &bucketName}, s3WithRegion(region)); err != nil {
		t.Logf(warning, err)
	}
}

func s3WithRegion(region string) func(o *s3.Options) {
	return func(o *s3.Options) {
		o.Region = region
	}
}

// create the dynamoDB table, and wait until we can query it.
func createDynamoDBTable(ctx context.Context, t *testing.T, dynClient *dynamodb.Client, tableName string) {
	createInput := &dynamodb.CreateTableInput{
		AttributeDefinitions: []dynamodbtypes.AttributeDefinition{
			{
				AttributeName: aws.String("LockID"),
				AttributeType: dynamodbtypes.ScalarAttributeTypeS,
			},
		},
		KeySchema: []dynamodbtypes.KeySchemaElement{
			{
				AttributeName: aws.String("LockID"),
				KeyType:       dynamodbtypes.KeyTypeHash,
			},
		},
		ProvisionedThroughput: &dynamodbtypes.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(5),
			WriteCapacityUnits: aws.Int64(5),
		},
		TableName: aws.String(tableName),
	}

	_, err := dynClient.CreateTable(ctx, createInput)
	if err != nil {
		t.Fatal(err)
	}

	// now wait until it's ACTIVE
	start := time.Now()
	time.Sleep(time.Second)

	describeInput := &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	}

	for {
		resp, err := dynClient.DescribeTable(ctx, describeInput)
		if err != nil {
			t.Fatal(err)
		}

		if resp.Table.TableStatus == dynamodbtypes.TableStatusActive {
			return
		}

		if time.Since(start) > time.Minute {
			t.Fatalf("timed out creating DynamoDB table %s", tableName)
		}

		time.Sleep(3 * time.Second)
	}

}

func deleteDynamoDBTable(ctx context.Context, t *testing.T, dynClient *dynamodb.Client, tableName string) {
	params := &dynamodb.DeleteTableInput{
		TableName: aws.String(tableName),
	}
	_, err := dynClient.DeleteTable(ctx, params)
	if err != nil {
		t.Logf("WARNING: Failed to delete the test DynamoDB table %q. It has been left in your AWS account and may incur charges. (error was %s)", tableName, err)
	}
}

func populateSchema(t *testing.T, schema *configschema.Block, value cty.Value) cty.Value {
	ty := schema.ImpliedType()
	var path cty.Path
	val, err := unmarshal(value, ty, path)
	if err != nil {
		t.Fatalf("populating schema: %s", err)
	}
	return val
}

func unmarshal(value cty.Value, ty cty.Type, path cty.Path) (cty.Value, error) {
	switch {
	case ty.IsPrimitiveType():
		return value, nil
	case ty.IsListType():
		return unmarshalList(value, ty.ElementType(), path)
	case ty.IsSetType():
		return unmarshalSet(value, ty.ElementType(), path)
	case ty.IsMapType():
		return unmarshalMap(value, ty.ElementType(), path)
	// case ty.IsTupleType():
	// 	return unmarshalTuple(value, ty.TupleElementTypes(), path)
	case ty.IsObjectType():
		return unmarshalObject(value, ty.AttributeTypes(), path)
	default:
		return cty.NilVal, path.NewErrorf("unsupported type %s", ty.FriendlyName())
	}
}

func unmarshalSet(dec cty.Value, ety cty.Type, path cty.Path) (cty.Value, error) {
	if dec.IsNull() {
		return dec, nil
	}

	length := dec.LengthInt()

	if length == 0 {
		return cty.SetValEmpty(ety), nil
	}

	vals := make([]cty.Value, 0, length)
	dec.ForEachElement(func(key, val cty.Value) (stop bool) {
		// vals = append(vals, must(unmarshal(val, ety, path.Index(key))))
		vals = append(vals, val)
		return
	})

	return cty.SetVal(vals), nil
}

func unmarshalList(dec cty.Value, ety cty.Type, path cty.Path) (cty.Value, error) {
	if dec.IsNull() {
		return dec, nil
	}

	length := dec.LengthInt()

	if length == 0 {
		return cty.ListValEmpty(ety), nil
	}

	vals := make([]cty.Value, 0, length)
	dec.ForEachElement(func(key, val cty.Value) (stop bool) {
		vals = append(vals, must(unmarshal(val, ety, path.Index(key))))
		return
	})

	return cty.ListVal(vals), nil
}

func unmarshalMap(dec cty.Value, ety cty.Type, path cty.Path) (cty.Value, error) {
	if dec.IsNull() {
		return dec, nil
	}

	length := dec.LengthInt()

	if length == 0 {
		return cty.MapValEmpty(ety), nil
	}

	vals := make(map[string]cty.Value, length)
	dec.ForEachElement(func(key, val cty.Value) (stop bool) {
		k := stringValue(key)
		// vals[k] = must(unmarshal(val, ety, path.Index(key)))
		vals[k] = val
		return
	})

	return cty.MapVal(vals), nil
}

func unmarshalObject(dec cty.Value, atys map[string]cty.Type, path cty.Path) (cty.Value, error) {
	if dec.IsNull() {
		return dec, nil
	}
	valueTy := dec.Type()

	vals := make(map[string]cty.Value, len(atys))
	path = append(path, nil)
	for key, aty := range atys {
		path[len(path)-1] = cty.IndexStep{
			Key: cty.StringVal(key),
		}

		if !valueTy.HasAttribute(key) {
			vals[key] = cty.NullVal(aty)
		} else {
			val, err := unmarshal(dec.GetAttr(key), aty, path)
			if err != nil {
				return cty.DynamicVal, err
			}
			vals[key] = val
		}
	}

	return cty.ObjectVal(vals), nil
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	} else {
		return v
	}
}

// testBackendConfigDiags is an equivalent to `backend.TestBackendConfig` which returns the diags to the caller
// instead of failing the test
func testBackendConfigDiags(t *testing.T, b backend.Backend, c hcl.Body) (backend.Backend, tfdiags.Diagnostics) {
	t.Helper()

	t.Logf("TestBackendConfig on %T with %#v", b, c)

	var diags tfdiags.Diagnostics

	// To make things easier for test authors, we'll allow a nil body here
	// (even though that's not normally valid) and just treat it as an empty
	// body.
	if c == nil {
		c = hcl.EmptyBody()
	}

	schema := b.ConfigSchema()
	spec := schema.DecoderSpec()
	obj, decDiags := hcldec.Decode(c, spec, nil)
	diags = diags.Append(decDiags)

	newObj, valDiags := b.PrepareConfig(obj)
	diags = diags.Append(valDiags.InConfigBody(c, ""))

	// it's valid for a Backend to have warnings (e.g. a Deprecation) as such we should only raise on errors
	if diags.HasErrors() {
		return b, diags
	}

	obj = newObj

	confDiags := b.Configure(obj)

	return b, diags.Append(confDiags)
}

func addRetrieveEndpointURLMiddleware(t *testing.T, endpoint *string) func(*middleware.Stack) error {
	return func(stack *middleware.Stack) error {
		return stack.Finalize.Add(
			retrieveEndpointURLMiddleware(t, endpoint),
			middleware.After,
		)
	}
}

func retrieveEndpointURLMiddleware(t *testing.T, endpoint *string) middleware.FinalizeMiddleware {
	return middleware.FinalizeMiddlewareFunc(
		"Test: Retrieve Endpoint",
		func(ctx context.Context, in middleware.FinalizeInput, next middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
			t.Helper()

			request, ok := in.Request.(*smithyhttp.Request)
			if !ok {
				t.Fatalf("Expected *github.com/aws/smithy-go/transport/http.Request, got %s", fullTypeName(in.Request))
			}

			url := request.URL
			url.RawQuery = ""

			*endpoint = url.String()

			return next.HandleFinalize(ctx, in)
		})
}

var errCancelOperation = fmt.Errorf("Test: Cancelling request")

func addCancelRequestMiddleware() func(*middleware.Stack) error {
	return func(stack *middleware.Stack) error {
		return stack.Finalize.Add(
			cancelRequestMiddleware(),
			middleware.After,
		)
	}
}

// cancelRequestMiddleware creates a Smithy middleware that intercepts the request before sending and cancels it
func cancelRequestMiddleware() middleware.FinalizeMiddleware {
	return middleware.FinalizeMiddlewareFunc(
		"Test: Cancel Requests",
		func(_ context.Context, in middleware.FinalizeInput, next middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
			return middleware.FinalizeOutput{}, middleware.Metadata{}, errCancelOperation
		})
}

func fullTypeName(i interface{}) string {
	return fullValueTypeName(reflect.ValueOf(i))
}

func fullValueTypeName(v reflect.Value) string {
	if v.Kind() == reflect.Ptr {
		return "*" + fullValueTypeName(reflect.Indirect(v))
	}

	requestType := v.Type()
	return fmt.Sprintf("%s.%s", requestType.PkgPath(), requestType.Name())
}

func defaultEndpointDynamo(region string) string {
	r := dynamodb.NewDefaultEndpointResolverV2()

	ep, err := r.ResolveEndpoint(context.TODO(), dynamodb.EndpointParameters{
		Region: aws.String(region),
	})
	if err != nil {
		return err.Error()
	}

	if ep.URI.Path == "" {
		ep.URI.Path = "/"
	}

	return ep.URI.String()
}

func defaultEndpointS3(region string) string {
	r := s3.NewDefaultEndpointResolverV2()

	ep, err := r.ResolveEndpoint(context.TODO(), s3.EndpointParameters{
		Region: aws.String(region),
	})
	if err != nil {
		return err.Error()
	}

	if ep.URI.Path == "" {
		ep.URI.Path = "/"
	}

	return ep.URI.String()
}
