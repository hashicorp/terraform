// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package s3

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	configtesting "github.com/hashicorp/aws-sdk-go-base/v2/configtesting"
	"github.com/hashicorp/aws-sdk-go-base/v2/mockdata"
	"github.com/hashicorp/aws-sdk-go-base/v2/servicemocks"
	"github.com/hashicorp/terraform/internal/configs/hcl2shim"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

const (
	// Shockingly, this is not defined in the SDK
	sharedConfigCredentialsProvider = "SharedConfigCredentials"
)

type DiagsValidator func(*testing.T, tfdiags.Diagnostics)

func ExpectNoDiags(t *testing.T, diags tfdiags.Diagnostics) {
	t.Helper()

	expectDiagsCount(t, diags, 0)
}

func expectDiagsCount(t *testing.T, diags tfdiags.Diagnostics, c int) {
	t.Helper()

	if l := len(diags); l != c {
		t.Fatalf("Diagnostics: expected %d element, got %d\n%s", c, l, diagnosticsString(diags))
	}
}

func ExpectDiagsEqual(expected tfdiags.Diagnostics) DiagsValidator {
	return func(t *testing.T, diags tfdiags.Diagnostics) {
		t.Helper()

		if diff := cmp.Diff(diags, expected, cmp.Comparer(diagnosticComparer)); diff != "" {
			t.Fatalf("unexpected diagnostics difference: %s", diff)
		}
	}
}

// ExpectDiagMatching returns a validator expeceting a single Diagnostic with fields matching the expectation
func ExpectDiagMatching(severity tfdiags.Severity, summary matcher, detail matcher) DiagsValidator {
	return ExpectDiags(
		diagMatching(severity, summary, detail),
	)
}

type diagValidator func(*testing.T, tfdiags.Diagnostic)

func ExpectDiags(validators ...diagValidator) DiagsValidator {
	return func(t *testing.T, diags tfdiags.Diagnostics) {
		count := len(validators)
		if l := len(diags); l < count {
			count = l
		}

		for i := 0; i < count; i++ {
			validators[i](t, diags[i])
		}

		expectDiagsCount(t, diags, len(validators))
	}
}

func diagMatching(severity tfdiags.Severity, summary matcher, detail matcher) diagValidator {
	return func(t *testing.T, diag tfdiags.Diagnostic) {
		if severity != diag.Severity() || !summary.Match(diag.Description().Summary) || !detail.Match(diag.Description().Detail) {
			t.Errorf("expected Diagnostic matching %#v, got %#v",
				tfdiags.Sourceless(
					severity,
					summary.String(),
					detail.String(),
				),
				diag,
			)
		}
	}
}

type matcher interface {
	fmt.Stringer
	Match(string) bool
}

type equalsMatcher string

func (m equalsMatcher) Match(s string) bool {
	return string(m) == s
}

func (m equalsMatcher) String() string {
	return string(m)
}

type regexpMatcher struct {
	re *regexp.Regexp
}

func newRegexpMatcher(re string) regexpMatcher {
	return regexpMatcher{
		re: regexp.MustCompile(re),
	}
}

func (m regexpMatcher) Match(s string) bool {
	return m.re.MatchString(s)
}

func (m regexpMatcher) String() string {
	return m.re.String()
}

type ignoreMatcher struct{}

func (m ignoreMatcher) Match(s string) bool {
	return true
}

func (m ignoreMatcher) String() string {
	return "ignored"
}

// Corrected from aws-sdk-go-base v1 & v2
const mockStsAssumeRolePolicy = `{
	"Version": "2012-10-17",
	"Statement": {
	  "Effect": "Allow",
	  "Action": "*",
	  "Resource": "*"
	}
  }`

func TestBackendConfig_Authentication(t *testing.T) {
	testCases := map[string]struct {
		config                     map[string]any
		EnableEc2MetadataServer    bool
		EnableEcsCredentialsServer bool
		EnableWebIdentityEnvVars   bool
		// EnableWebIdentityConfig    bool // Not supported
		EnvironmentVariables     map[string]string
		ExpectedCredentialsValue aws.Credentials
		MockStsEndpoints         []*servicemocks.MockEndpoint
		SharedConfigurationFile  string
		SharedCredentialsFile    string
		ValidateDiags            DiagsValidator
	}{
		"empty config": {
			config: map[string]any{},
			ValidateDiags: ExpectDiagMatching(
				tfdiags.Error,
				equalsMatcher("No valid credential sources found"),
				ignoreMatcher{},
			),
		},

		"config AccessKey": {
			config: map[string]any{
				"access_key": servicemocks.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
			},
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			ExpectedCredentialsValue: mockdata.MockStaticCredentials,
			ValidateDiags:            ExpectNoDiags,
		},

		"config Profile shared credentials profile aws_access_key_id": {
			config: map[string]any{
				"profile": "SharedCredentialsProfile",
			},
			ExpectedCredentialsValue: aws.Credentials{
				AccessKeyID:     "ProfileSharedCredentialsAccessKey",
				SecretAccessKey: "ProfileSharedCredentialsSecretKey",
				Source:          sharedConfigCredentialsProvider,
			},
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			SharedCredentialsFile: `
[default]
aws_access_key_id = DefaultSharedCredentialsAccessKey
aws_secret_access_key = DefaultSharedCredentialsSecretKey

[SharedCredentialsProfile]
aws_access_key_id = ProfileSharedCredentialsAccessKey
aws_secret_access_key = ProfileSharedCredentialsSecretKey
`,
			ValidateDiags: ExpectNoDiags,
		},

		"environment AWS_ACCESS_KEY_ID does not override config Profile": {
			config: map[string]any{
				"profile": "SharedCredentialsProfile",
			},
			EnvironmentVariables: map[string]string{
				"AWS_ACCESS_KEY_ID":     servicemocks.MockEnvAccessKey,
				"AWS_SECRET_ACCESS_KEY": servicemocks.MockEnvSecretKey,
			},
			ExpectedCredentialsValue: aws.Credentials{
				AccessKeyID:     "ProfileSharedCredentialsAccessKey",
				SecretAccessKey: "ProfileSharedCredentialsSecretKey",
				Source:          sharedConfigCredentialsProvider,
			},
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			SharedCredentialsFile: `
[default]
aws_access_key_id = DefaultSharedCredentialsAccessKey
aws_secret_access_key = DefaultSharedCredentialsSecretKey

[SharedCredentialsProfile]
aws_access_key_id = ProfileSharedCredentialsAccessKey
aws_secret_access_key = ProfileSharedCredentialsSecretKey
`,
			ValidateDiags: ExpectNoDiags,
		},

		"environment AWS_ACCESS_KEY_ID": {
			config: map[string]any{},
			EnvironmentVariables: map[string]string{
				"AWS_ACCESS_KEY_ID":     servicemocks.MockEnvAccessKey,
				"AWS_SECRET_ACCESS_KEY": servicemocks.MockEnvSecretKey,
			},
			ExpectedCredentialsValue: mockdata.MockEnvCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			ValidateDiags: ExpectNoDiags,
		},

		"environment AWS_PROFILE shared credentials profile aws_access_key_id": {
			config: map[string]any{},
			EnvironmentVariables: map[string]string{
				"AWS_PROFILE": "SharedCredentialsProfile",
			},
			ExpectedCredentialsValue: aws.Credentials{
				AccessKeyID:     "ProfileSharedCredentialsAccessKey",
				SecretAccessKey: "ProfileSharedCredentialsSecretKey",
				Source:          sharedConfigCredentialsProvider,
			},
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			SharedCredentialsFile: `
[default]
aws_access_key_id = DefaultSharedCredentialsAccessKey
aws_secret_access_key = DefaultSharedCredentialsSecretKey

[SharedCredentialsProfile]
aws_access_key_id = ProfileSharedCredentialsAccessKey
aws_secret_access_key = ProfileSharedCredentialsSecretKey
`,
			ValidateDiags: ExpectNoDiags,
		},

		"environment AWS_SESSION_TOKEN": {
			config: map[string]any{},
			EnvironmentVariables: map[string]string{
				"AWS_ACCESS_KEY_ID":     servicemocks.MockEnvAccessKey,
				"AWS_SECRET_ACCESS_KEY": servicemocks.MockEnvSecretKey,
				"AWS_SESSION_TOKEN":     servicemocks.MockEnvSessionToken,
			},
			ExpectedCredentialsValue: mockdata.MockEnvCredentialsWithSessionToken,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
		},

		"shared credentials default aws_access_key_id": {
			config: map[string]any{},
			ExpectedCredentialsValue: aws.Credentials{
				AccessKeyID:     "DefaultSharedCredentialsAccessKey",
				SecretAccessKey: "DefaultSharedCredentialsSecretKey",
				Source:          sharedConfigCredentialsProvider,
			},
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			SharedCredentialsFile: `
[default]
aws_access_key_id = DefaultSharedCredentialsAccessKey
aws_secret_access_key = DefaultSharedCredentialsSecretKey
`,
		},

		"web identity token access key": {
			config:                   map[string]any{},
			EnableWebIdentityEnvVars: true,
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleWithWebIdentityCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleWithWebIdentityValidEndpoint,
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
		},

		"EC2 metadata access key": {
			config:                   map[string]any{},
			EnableEc2MetadataServer:  true,
			ExpectedCredentialsValue: mockdata.MockEc2MetadataCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			ValidateDiags: ExpectNoDiags,
		},

		"ECS credentials access key": {
			config:                     map[string]any{},
			EnableEcsCredentialsServer: true,
			ExpectedCredentialsValue:   mockdata.MockEcsCredentialsCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
		},

		"AssumeWebIdentity envvar AssumeRoleARN access key": {
			config: map[string]any{
				"role_arn":     servicemocks.MockStsAssumeRoleArn,
				"session_name": servicemocks.MockStsAssumeRoleSessionName,
			},
			EnableWebIdentityEnvVars: true,
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleWithWebIdentityValidEndpoint,
				servicemocks.MockStsAssumeRoleValidEndpoint,
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			ValidateDiags: ExpectDiagMatching(
				tfdiags.Warning,
				equalsMatcher("Deprecated Parameters"),
				ignoreMatcher{},
			),
		},

		"config AccessKey over environment AWS_ACCESS_KEY_ID": {
			config: map[string]any{
				"access_key": servicemocks.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
			},
			EnvironmentVariables: map[string]string{
				"AWS_ACCESS_KEY_ID":     servicemocks.MockEnvAccessKey,
				"AWS_SECRET_ACCESS_KEY": servicemocks.MockEnvSecretKey,
			},
			ExpectedCredentialsValue: mockdata.MockStaticCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			ValidateDiags: ExpectNoDiags,
		},

		"config AccessKey over shared credentials default aws_access_key_id": {
			config: map[string]any{
				"access_key": servicemocks.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
			},
			ExpectedCredentialsValue: mockdata.MockStaticCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			SharedCredentialsFile: `
[default]
aws_access_key_id = DefaultSharedCredentialsAccessKey
aws_secret_access_key = DefaultSharedCredentialsSecretKey
`,
			ValidateDiags: ExpectNoDiags,
		},

		"config AccessKey over EC2 metadata access key": {
			config: map[string]any{
				"access_key": servicemocks.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
			},
			ExpectedCredentialsValue: mockdata.MockStaticCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
		},

		"config AccessKey over ECS credentials access key": {
			config: map[string]any{
				"access_key": servicemocks.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
			},
			EnableEcsCredentialsServer: true,
			ExpectedCredentialsValue:   mockdata.MockStaticCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
		},

		"environment AWS_ACCESS_KEY_ID over shared credentials default aws_access_key_id": {
			config: map[string]any{},
			EnvironmentVariables: map[string]string{
				"AWS_ACCESS_KEY_ID":     servicemocks.MockEnvAccessKey,
				"AWS_SECRET_ACCESS_KEY": servicemocks.MockEnvSecretKey,
			},
			ExpectedCredentialsValue: mockdata.MockEnvCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			SharedCredentialsFile: `
[default]
aws_access_key_id = DefaultSharedCredentialsAccessKey
aws_secret_access_key = DefaultSharedCredentialsSecretKey
`,
			ValidateDiags: ExpectNoDiags,
		},

		"environment AWS_ACCESS_KEY_ID over EC2 metadata access key": {
			config: map[string]any{},
			EnvironmentVariables: map[string]string{
				"AWS_ACCESS_KEY_ID":     servicemocks.MockEnvAccessKey,
				"AWS_SECRET_ACCESS_KEY": servicemocks.MockEnvSecretKey,
			},
			ExpectedCredentialsValue: mockdata.MockEnvCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
		},

		"environment AWS_ACCESS_KEY_ID over ECS credentials access key": {
			config: map[string]any{},
			EnvironmentVariables: map[string]string{
				"AWS_ACCESS_KEY_ID":     servicemocks.MockEnvAccessKey,
				"AWS_SECRET_ACCESS_KEY": servicemocks.MockEnvSecretKey,
			},
			EnableEcsCredentialsServer: true,
			ExpectedCredentialsValue:   mockdata.MockEnvCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
		},

		"shared credentials default aws_access_key_id over EC2 metadata access key": {
			config: map[string]any{},
			ExpectedCredentialsValue: aws.Credentials{
				AccessKeyID:     "DefaultSharedCredentialsAccessKey",
				SecretAccessKey: "DefaultSharedCredentialsSecretKey",
				Source:          sharedConfigCredentialsProvider,
			},
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			SharedCredentialsFile: `
[default]
aws_access_key_id = DefaultSharedCredentialsAccessKey
aws_secret_access_key = DefaultSharedCredentialsSecretKey
`,
		},

		"shared credentials default aws_access_key_id over ECS credentials access key": {
			config:                     map[string]any{},
			EnableEcsCredentialsServer: true,
			ExpectedCredentialsValue: aws.Credentials{
				AccessKeyID:     "DefaultSharedCredentialsAccessKey",
				SecretAccessKey: "DefaultSharedCredentialsSecretKey",
				Source:          sharedConfigCredentialsProvider,
			},
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			SharedCredentialsFile: `
[default]
aws_access_key_id = DefaultSharedCredentialsAccessKey
aws_secret_access_key = DefaultSharedCredentialsSecretKey
`,
		},

		"ECS credentials access key over EC2 metadata access key": {
			config:                     map[string]any{},
			EnableEcsCredentialsServer: true,
			ExpectedCredentialsValue:   mockdata.MockEcsCredentialsCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
		},

		"retrieve region from shared configuration file": {
			config: map[string]any{
				"access_key": servicemocks.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
			},
			ExpectedCredentialsValue: mockdata.MockStaticCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			SharedConfigurationFile: `
[default]
region = us-east-1
`,
		},

		"skip EC2 Metadata API check": {
			config: map[string]any{
				"skip_metadata_api_check": true,
			},
			// The IMDS server must be enabled so that auth will succeed if the IMDS is called
			EnableEc2MetadataServer: true,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			ValidateDiags: ExpectDiagMatching(
				tfdiags.Error,
				equalsMatcher("No valid credential sources found"),
				ignoreMatcher{},
			),
		},

		"invalid profile name from envvar": {
			config: map[string]any{},
			EnvironmentVariables: map[string]string{
				"AWS_PROFILE": "no-such-profile",
			},
			SharedCredentialsFile: `
[some-profile]
aws_access_key_id = DefaultSharedCredentialsAccessKey
aws_secret_access_key = DefaultSharedCredentialsSecretKey
`,
			ValidateDiags: ExpectDiagMatching(
				tfdiags.Error,
				equalsMatcher("failed to get shared config profile, no-such-profile"),
				equalsMatcher(""),
			),
		},

		"invalid profile name from config": {
			config: map[string]any{
				"profile": "no-such-profile",
			},
			SharedCredentialsFile: `
[some-profile]
aws_access_key_id = DefaultSharedCredentialsAccessKey
aws_secret_access_key = DefaultSharedCredentialsSecretKey
`,
			ValidateDiags: ExpectDiagMatching(
				tfdiags.Error,
				equalsMatcher("failed to get shared config profile, no-such-profile"),
				equalsMatcher(""),
			),
		},

		"AWS_ACCESS_KEY_ID overrides AWS_PROFILE": {
			config: map[string]any{},
			EnvironmentVariables: map[string]string{
				"AWS_ACCESS_KEY_ID":     servicemocks.MockEnvAccessKey,
				"AWS_SECRET_ACCESS_KEY": servicemocks.MockEnvSecretKey,
				"AWS_PROFILE":           "SharedCredentialsProfile",
			},
			SharedCredentialsFile: `
[default]
aws_access_key_id = DefaultSharedCredentialsAccessKey
aws_secret_access_key = DefaultSharedCredentialsSecretKey

[SharedCredentialsProfile]
aws_access_key_id = ProfileSharedCredentialsAccessKey
aws_secret_access_key = ProfileSharedCredentialsSecretKey
`,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			ExpectedCredentialsValue: mockdata.MockEnvCredentials,
			ValidateDiags:            ExpectNoDiags,
		},

		"AWS_ACCESS_KEY_ID does not override invalid profile name from envvar": {
			config: map[string]any{},
			EnvironmentVariables: map[string]string{
				"AWS_ACCESS_KEY_ID":     servicemocks.MockEnvAccessKey,
				"AWS_SECRET_ACCESS_KEY": servicemocks.MockEnvSecretKey,
				"AWS_PROFILE":           "no-such-profile",
			},
			SharedCredentialsFile: `
[some-profile]
aws_access_key_id = DefaultSharedCredentialsAccessKey
aws_secret_access_key = DefaultSharedCredentialsSecretKey
`,
			ValidateDiags: ExpectDiagMatching(
				tfdiags.Error,
				equalsMatcher("failed to get shared config profile, no-such-profile"),
				equalsMatcher(""),
			),
		},
	}

	for name, tc := range testCases {
		tc := tc

		t.Run(name, func(t *testing.T) {
			servicemocks.InitSessionTestEnv(t)

			ctx := context.TODO()

			// Populate required fields
			tc.config["region"] = "us-east-1"
			tc.config["bucket"] = "bucket"
			tc.config["key"] = "key"

			if tc.ValidateDiags == nil {
				tc.ValidateDiags = ExpectNoDiags
			}

			if tc.EnableEc2MetadataServer {
				closeEc2Metadata := servicemocks.AwsMetadataApiMock(append(
					servicemocks.Ec2metadata_securityCredentialsEndpoints,
					servicemocks.Ec2metadata_instanceIdEndpoint,
					servicemocks.Ec2metadata_iamInfoEndpoint,
				))
				defer closeEc2Metadata()
			}

			if tc.EnableEcsCredentialsServer {
				closeEcsCredentials := servicemocks.EcsCredentialsApiMock()
				defer closeEcsCredentials()
			}

			if tc.EnableWebIdentityEnvVars /*|| tc.EnableWebIdentityConfig*/ {
				file, err := os.CreateTemp("", "aws-sdk-go-base-web-identity-token-file")
				if err != nil {
					t.Fatalf("unexpected error creating temporary web identity token file: %s", err)
				}

				defer os.Remove(file.Name())

				err = os.WriteFile(file.Name(), []byte(servicemocks.MockWebIdentityToken), 0600)

				if err != nil {
					t.Fatalf("unexpected error writing web identity token file: %s", err)
				}

				if tc.EnableWebIdentityEnvVars {
					t.Setenv("AWS_ROLE_ARN", servicemocks.MockStsAssumeRoleWithWebIdentityArn)
					t.Setenv("AWS_ROLE_SESSION_NAME", servicemocks.MockStsAssumeRoleWithWebIdentitySessionName)
					t.Setenv("AWS_WEB_IDENTITY_TOKEN_FILE", file.Name())
				} /*else if tc.EnableWebIdentityConfig {
					tc.Config.AssumeRoleWithWebIdentity = &AssumeRoleWithWebIdentity{
						RoleARN:              servicemocks.MockStsAssumeRoleWithWebIdentityArn,
						SessionName:          servicemocks.MockStsAssumeRoleWithWebIdentitySessionName,
						WebIdentityTokenFile: file.Name(),
					}
				}*/
			}

			ts := servicemocks.MockAwsApiServer("STS", tc.MockStsEndpoints)
			defer ts.Close()

			tc.config["endpoints"] = map[string]any{
				"sts": ts.URL,
			}

			if tc.SharedConfigurationFile != "" {
				file, err := os.CreateTemp("", "aws-sdk-go-base-shared-configuration-file")

				if err != nil {
					t.Fatalf("unexpected error creating temporary shared configuration file: %s", err)
				}

				defer os.Remove(file.Name())

				err = os.WriteFile(file.Name(), []byte(tc.SharedConfigurationFile), 0600)

				if err != nil {
					t.Fatalf("unexpected error writing shared configuration file: %s", err)
				}

				setSharedConfigFile(t, file.Name())
			}

			if tc.SharedCredentialsFile != "" {
				file, err := os.CreateTemp("", "aws-sdk-go-base-shared-credentials-file")

				if err != nil {
					t.Fatalf("unexpected error creating temporary shared credentials file: %s", err)
				}

				defer os.Remove(file.Name())

				err = os.WriteFile(file.Name(), []byte(tc.SharedCredentialsFile), 0600)

				if err != nil {
					t.Fatalf("unexpected error writing shared credentials file: %s", err)
				}

				tc.config["shared_credentials_files"] = []interface{}{file.Name()}
				if tc.ExpectedCredentialsValue.Source == sharedConfigCredentialsProvider {
					tc.ExpectedCredentialsValue.Source = sharedConfigCredentialsSource(file.Name())
				}
			}

			for k, v := range tc.EnvironmentVariables {
				t.Setenv(k, v)
			}

			b, diags := configureBackend(t, tc.config)

			tc.ValidateDiags(t, diags)

			if diags.HasErrors() {
				return
			}

			credentials, err := b.awsConfig.Credentials.Retrieve(ctx)
			if err != nil {
				t.Fatalf("Error when requesting credentials")
			}

			if diff := cmp.Diff(credentials, tc.ExpectedCredentialsValue, cmpopts.IgnoreFields(aws.Credentials{}, "Expires")); diff != "" {
				t.Fatalf("unexpected credentials: (- got, + expected)\n%s", diff)
			}
		})
	}
}

func TestBackendConfig_Authentication_AssumeRoleInline(t *testing.T) {
	testCases := map[string]struct {
		config                     map[string]any
		EnableEc2MetadataServer    bool
		EnableEcsCredentialsServer bool
		EnvironmentVariables       map[string]string
		ExpectedCredentialsValue   aws.Credentials
		MockStsEndpoints           []*servicemocks.MockEndpoint
		SharedConfigurationFile    string
		SharedCredentialsFile      string
		ValidateDiags              DiagsValidator
	}{
		// WAS: "config AccessKey config AssumeRoleARN access key"
		"from config access_key": {
			config: map[string]any{
				"access_key":   servicemocks.MockStaticAccessKey,
				"secret_key":   servicemocks.MockStaticSecretKey,
				"role_arn":     servicemocks.MockStsAssumeRoleArn,
				"session_name": servicemocks.MockStsAssumeRoleSessionName,
			},
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleValidEndpoint,
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			ValidateDiags: ExpectDiagMatching(
				tfdiags.Warning,
				equalsMatcher("Deprecated Parameters"),
				ignoreMatcher{},
			),
		},

		// WAS: "environment AWS_ACCESS_KEY_ID config AssumeRoleARN access key"
		"from environment AWS_ACCESS_KEY_ID": {
			config: map[string]any{
				"role_arn":     servicemocks.MockStsAssumeRoleArn,
				"session_name": servicemocks.MockStsAssumeRoleSessionName,
			},
			EnvironmentVariables: map[string]string{
				"AWS_ACCESS_KEY_ID":     servicemocks.MockEnvAccessKey,
				"AWS_SECRET_ACCESS_KEY": servicemocks.MockEnvSecretKey,
			},
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleValidEndpoint,
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			ValidateDiags: ExpectDiagMatching(
				tfdiags.Warning,
				equalsMatcher("Deprecated Parameters"),
				ignoreMatcher{},
			),
		},

		// WAS: "config Profile shared configuration credential_source Ec2InstanceMetadata"
		"from config Profile with Ec2InstanceMetadata source": {
			config: map[string]any{
				"profile": "SharedConfigurationProfile",
			},
			EnableEc2MetadataServer:  true,
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleValidEndpoint,
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			SharedConfigurationFile: fmt.Sprintf(`
[profile SharedConfigurationProfile]
credential_source = Ec2InstanceMetadata
role_arn = %[1]s
role_session_name = %[2]s
`, servicemocks.MockStsAssumeRoleArn, servicemocks.MockStsAssumeRoleSessionName),
		},

		// WAS: "environment AWS_PROFILE shared configuration credential_source Ec2InstanceMetadata"
		"from environment AWS_PROFILE with Ec2InstanceMetadata source": {
			config:                  map[string]any{},
			EnableEc2MetadataServer: true,
			EnvironmentVariables: map[string]string{
				"AWS_PROFILE": "SharedConfigurationProfile",
			},
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleValidEndpoint,
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			SharedConfigurationFile: fmt.Sprintf(`
[profile SharedConfigurationProfile]
credential_source = Ec2InstanceMetadata
role_arn = %[1]s
role_session_name = %[2]s
`, servicemocks.MockStsAssumeRoleArn, servicemocks.MockStsAssumeRoleSessionName),
		},

		// WAS: "config Profile shared configuration source_profile"
		"from config Profile with source profile": {
			config: map[string]any{
				"profile": "SharedConfigurationProfile",
			},
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleValidEndpoint,
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			SharedConfigurationFile: fmt.Sprintf(`
[profile SharedConfigurationProfile]
role_arn = %[1]s
role_session_name = %[2]s
source_profile = SharedConfigurationSourceProfile

[profile SharedConfigurationSourceProfile]
aws_access_key_id = SharedConfigurationSourceAccessKey
aws_secret_access_key = SharedConfigurationSourceSecretKey
`, servicemocks.MockStsAssumeRoleArn, servicemocks.MockStsAssumeRoleSessionName),
		},

		// WAS: "environment AWS_PROFILE shared configuration source_profile"
		"from environment AWS_PROFILE with source profile": {
			config: map[string]any{},
			EnvironmentVariables: map[string]string{
				"AWS_PROFILE": "SharedConfigurationProfile",
			},
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleValidEndpoint,
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			SharedConfigurationFile: fmt.Sprintf(`
[profile SharedConfigurationProfile]
role_arn = %[1]s
role_session_name = %[2]s
source_profile = SharedConfigurationSourceProfile

[profile SharedConfigurationSourceProfile]
aws_access_key_id = SharedConfigurationSourceAccessKey
aws_secret_access_key = SharedConfigurationSourceSecretKey
`, servicemocks.MockStsAssumeRoleArn, servicemocks.MockStsAssumeRoleSessionName),
		},

		// WAS: "shared credentials default aws_access_key_id config AssumeRoleARN access key"
		"from default profile": {
			config: map[string]any{
				"role_arn":     servicemocks.MockStsAssumeRoleArn,
				"session_name": servicemocks.MockStsAssumeRoleSessionName,
			},
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleValidEndpoint,
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			SharedCredentialsFile: `
[default]
aws_access_key_id = DefaultSharedCredentialsAccessKey
aws_secret_access_key = DefaultSharedCredentialsSecretKey
`,
			ValidateDiags: ExpectDiagMatching(
				tfdiags.Warning,
				equalsMatcher("Deprecated Parameters"),
				ignoreMatcher{},
			),
		},

		// WAS: "EC2 metadata access key config AssumeRoleARN access key"
		"from EC2 metadata": {
			config: map[string]any{
				"role_arn":     servicemocks.MockStsAssumeRoleArn,
				"session_name": servicemocks.MockStsAssumeRoleSessionName,
			},
			EnableEc2MetadataServer:  true,
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleValidEndpoint,
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			ValidateDiags: ExpectDiagMatching(
				tfdiags.Warning,
				equalsMatcher("Deprecated Parameters"),
				ignoreMatcher{},
			),
		},

		// WAS: "ECS credentials access key config AssumeRoleARN access key"
		"from ECS credentials": {
			config: map[string]any{
				"role_arn":     servicemocks.MockStsAssumeRoleArn,
				"session_name": servicemocks.MockStsAssumeRoleSessionName,
			},
			EnableEcsCredentialsServer: true,
			ExpectedCredentialsValue:   mockdata.MockStsAssumeRoleCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleValidEndpoint,
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			ValidateDiags: ExpectDiagMatching(
				tfdiags.Warning,
				equalsMatcher("Deprecated Parameters"),
				ignoreMatcher{},
			),
		},

		// WAS: "config AssumeRoleDuration"
		"with duration": {
			config: map[string]any{
				"access_key":                   servicemocks.MockStaticAccessKey,
				"secret_key":                   servicemocks.MockStaticSecretKey,
				"role_arn":                     servicemocks.MockStsAssumeRoleArn,
				"session_name":                 servicemocks.MockStsAssumeRoleSessionName,
				"assume_role_duration_seconds": 3600,
			},
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleValidEndpointWithOptions(map[string]string{"DurationSeconds": "3600"}),
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			ValidateDiags: ExpectDiagMatching(
				tfdiags.Warning,
				equalsMatcher("Deprecated Parameters"),
				ignoreMatcher{},
			),
		},

		// WAS: "config AssumeRoleExternalID"
		"with external ID": {
			config: map[string]any{
				"access_key":   servicemocks.MockStaticAccessKey,
				"secret_key":   servicemocks.MockStaticSecretKey,
				"role_arn":     servicemocks.MockStsAssumeRoleArn,
				"session_name": servicemocks.MockStsAssumeRoleSessionName,
				"external_id":  servicemocks.MockStsAssumeRoleExternalId,
			},
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleValidEndpointWithOptions(map[string]string{"ExternalId": servicemocks.MockStsAssumeRoleExternalId}),
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			ValidateDiags: ExpectDiagMatching(
				tfdiags.Warning,
				equalsMatcher("Deprecated Parameters"),
				ignoreMatcher{},
			),
		},

		// WAS: "config AssumeRolePolicy"
		"with policy": {
			config: map[string]any{
				"access_key":         servicemocks.MockStaticAccessKey,
				"secret_key":         servicemocks.MockStaticSecretKey,
				"role_arn":           servicemocks.MockStsAssumeRoleArn,
				"session_name":       servicemocks.MockStsAssumeRoleSessionName,
				"assume_role_policy": mockStsAssumeRolePolicy,
			},
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleValidEndpointWithOptions(map[string]string{"Policy": mockStsAssumeRolePolicy}),
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			ValidateDiags: ExpectDiagMatching(
				tfdiags.Warning,
				equalsMatcher("Deprecated Parameters"),
				ignoreMatcher{},
			),
		},

		// WAS: "config AssumeRolePolicyARNs"
		"with policy ARNs": {
			config: map[string]any{
				"access_key":              servicemocks.MockStaticAccessKey,
				"secret_key":              servicemocks.MockStaticSecretKey,
				"role_arn":                servicemocks.MockStsAssumeRoleArn,
				"session_name":            servicemocks.MockStsAssumeRoleSessionName,
				"assume_role_policy_arns": []any{servicemocks.MockStsAssumeRolePolicyArn},
			},
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleValidEndpointWithOptions(map[string]string{"PolicyArns.member.1.arn": servicemocks.MockStsAssumeRolePolicyArn}),
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			ValidateDiags: ExpectDiagMatching(
				tfdiags.Warning,
				equalsMatcher("Deprecated Parameters"),
				ignoreMatcher{},
			),
		},

		// WAS: "config AssumeRoleTags"
		"with tags": {
			config: map[string]any{
				"access_key":   servicemocks.MockStaticAccessKey,
				"secret_key":   servicemocks.MockStaticSecretKey,
				"role_arn":     servicemocks.MockStsAssumeRoleArn,
				"session_name": servicemocks.MockStsAssumeRoleSessionName,
				"assume_role_tags": map[string]any{
					servicemocks.MockStsAssumeRoleTagKey: servicemocks.MockStsAssumeRoleTagValue,
				},
			},
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleValidEndpointWithOptions(map[string]string{"Tags.member.1.Key": servicemocks.MockStsAssumeRoleTagKey, "Tags.member.1.Value": servicemocks.MockStsAssumeRoleTagValue}),
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			ValidateDiags: ExpectDiagMatching(
				tfdiags.Warning,
				equalsMatcher("Deprecated Parameters"),
				ignoreMatcher{},
			),
		},

		// WAS: "config AssumeRoleTransitiveTagKeys"
		"with transitive tags": {
			config: map[string]any{
				"access_key":   servicemocks.MockStaticAccessKey,
				"secret_key":   servicemocks.MockStaticSecretKey,
				"role_arn":     servicemocks.MockStsAssumeRoleArn,
				"session_name": servicemocks.MockStsAssumeRoleSessionName,
				"assume_role_tags": map[string]any{
					servicemocks.MockStsAssumeRoleTagKey: servicemocks.MockStsAssumeRoleTagValue,
				},
				"assume_role_transitive_tag_keys": []any{servicemocks.MockStsAssumeRoleTagKey},
			},
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleValidEndpointWithOptions(map[string]string{"Tags.member.1.Key": servicemocks.MockStsAssumeRoleTagKey, "Tags.member.1.Value": servicemocks.MockStsAssumeRoleTagValue, "TransitiveTagKeys.member.1": servicemocks.MockStsAssumeRoleTagKey}),
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			ValidateDiags: ExpectDiagMatching(
				tfdiags.Warning,
				equalsMatcher("Deprecated Parameters"),
				ignoreMatcher{},
			),
		},

		// WAS: "assume role error"
		"error": {
			config: map[string]any{
				"access_key":   servicemocks.MockStaticAccessKey,
				"secret_key":   servicemocks.MockStaticSecretKey,
				"role_arn":     servicemocks.MockStsAssumeRoleArn,
				"session_name": servicemocks.MockStsAssumeRoleSessionName,
			},
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleInvalidEndpointInvalidClientTokenId,
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			ValidateDiags: ExpectDiags(
				diagMatching(
					tfdiags.Warning,
					equalsMatcher("Deprecated Parameters"),
					ignoreMatcher{},
				),
				diagMatching(
					tfdiags.Error,
					equalsMatcher("Cannot assume IAM Role"),
					newRegexpMatcher(`IAM Role \(.+\) cannot be assumed.`),
				),
			),
		},
	}

	for name, tc := range testCases {
		tc := tc

		t.Run(name, func(t *testing.T) {
			servicemocks.InitSessionTestEnv(t)

			ctx := context.TODO()

			// Populate required fields
			tc.config["region"] = "us-east-1"
			tc.config["bucket"] = "bucket"
			tc.config["key"] = "key"

			if tc.ValidateDiags == nil {
				tc.ValidateDiags = ExpectNoDiags
			}

			if tc.EnableEc2MetadataServer {
				closeEc2Metadata := servicemocks.AwsMetadataApiMock(append(
					servicemocks.Ec2metadata_securityCredentialsEndpoints,
					servicemocks.Ec2metadata_instanceIdEndpoint,
					servicemocks.Ec2metadata_iamInfoEndpoint,
				))
				defer closeEc2Metadata()
			}

			if tc.EnableEcsCredentialsServer {
				closeEcsCredentials := servicemocks.EcsCredentialsApiMock()
				defer closeEcsCredentials()
			}

			ts := servicemocks.MockAwsApiServer("STS", tc.MockStsEndpoints)
			defer ts.Close()

			tc.config["endpoints"] = map[string]any{
				"sts": ts.URL,
			}

			if tc.SharedConfigurationFile != "" {
				file, err := os.CreateTemp("", "aws-sdk-go-base-shared-configuration-file")

				if err != nil {
					t.Fatalf("unexpected error creating temporary shared configuration file: %s", err)
				}

				defer os.Remove(file.Name())

				err = os.WriteFile(file.Name(), []byte(tc.SharedConfigurationFile), 0600)

				if err != nil {
					t.Fatalf("unexpected error writing shared configuration file: %s", err)
				}

				setSharedConfigFile(t, file.Name())
			}

			if tc.SharedCredentialsFile != "" {
				file, err := os.CreateTemp("", "aws-sdk-go-base-shared-credentials-file")

				if err != nil {
					t.Fatalf("unexpected error creating temporary shared credentials file: %s", err)
				}

				defer os.Remove(file.Name())

				err = os.WriteFile(file.Name(), []byte(tc.SharedCredentialsFile), 0600)

				if err != nil {
					t.Fatalf("unexpected error writing shared credentials file: %s", err)
				}

				tc.config["shared_credentials_files"] = []interface{}{file.Name()}
				if tc.ExpectedCredentialsValue.Source == sharedConfigCredentialsProvider {
					tc.ExpectedCredentialsValue.Source = sharedConfigCredentialsSource(file.Name())
				}
			}

			for k, v := range tc.EnvironmentVariables {
				t.Setenv(k, v)
			}

			b, diags := configureBackend(t, tc.config)

			tc.ValidateDiags(t, diags)

			if diags.HasErrors() {
				return
			}

			credentials, err := b.awsConfig.Credentials.Retrieve(ctx)
			if err != nil {
				t.Fatalf("Error when requesting credentials: %s", err)
			}

			if diff := cmp.Diff(credentials, tc.ExpectedCredentialsValue, cmpopts.IgnoreFields(aws.Credentials{}, "Expires")); diff != "" {
				t.Fatalf("unexpected credentials: (- got, + expected)\n%s", diff)
			}
		})
	}
}

func TestBackendConfig_Authentication_AssumeRoleNested(t *testing.T) {
	testCases := map[string]struct {
		config                     map[string]any
		EnableEc2MetadataServer    bool
		EnableEcsCredentialsServer bool
		EnvironmentVariables       map[string]string
		ExpectedCredentialsValue   aws.Credentials
		MockStsEndpoints           []*servicemocks.MockEndpoint
		SharedConfigurationFile    string
		SharedCredentialsFile      string
		ValidateDiags              DiagsValidator
	}{
		// WAS: "config AccessKey config AssumeRoleARN access key"
		"from config access_key": {
			config: map[string]any{
				"access_key": servicemocks.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
				"assume_role": map[string]any{
					"role_arn":     servicemocks.MockStsAssumeRoleArn,
					"session_name": servicemocks.MockStsAssumeRoleSessionName,
				},
			},
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleValidEndpoint,
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
		},

		// WAS: "environment AWS_ACCESS_KEY_ID config AssumeRoleARN access key"
		"from environment AWS_ACCESS_KEY_ID": {
			config: map[string]any{
				"assume_role": map[string]any{
					"role_arn":     servicemocks.MockStsAssumeRoleArn,
					"session_name": servicemocks.MockStsAssumeRoleSessionName,
				},
			},
			EnvironmentVariables: map[string]string{
				"AWS_ACCESS_KEY_ID":     servicemocks.MockEnvAccessKey,
				"AWS_SECRET_ACCESS_KEY": servicemocks.MockEnvSecretKey,
			},
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleValidEndpoint,
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
		},

		// WAS: "config Profile shared configuration credential_source Ec2InstanceMetadata"
		"from config Profile with Ec2InstanceMetadata source": {
			config: map[string]any{
				"profile": "SharedConfigurationProfile",
			},
			EnableEc2MetadataServer:  true,
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleValidEndpoint,
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			SharedConfigurationFile: fmt.Sprintf(`
[profile SharedConfigurationProfile]
credential_source = Ec2InstanceMetadata
role_arn = %[1]s
role_session_name = %[2]s
`, servicemocks.MockStsAssumeRoleArn, servicemocks.MockStsAssumeRoleSessionName),
		},

		// WAS: "environment AWS_PROFILE shared configuration credential_source Ec2InstanceMetadata"
		"from environment AWS_PROFILE with Ec2InstanceMetadata source": {
			config:                  map[string]any{},
			EnableEc2MetadataServer: true,
			EnvironmentVariables: map[string]string{
				"AWS_PROFILE": "SharedConfigurationProfile",
			},
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleValidEndpoint,
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			SharedConfigurationFile: fmt.Sprintf(`
[profile SharedConfigurationProfile]
credential_source = Ec2InstanceMetadata
role_arn = %[1]s
role_session_name = %[2]s
`, servicemocks.MockStsAssumeRoleArn, servicemocks.MockStsAssumeRoleSessionName),
		},

		// WAS: "config Profile shared configuration source_profile"
		"from config Profile with source profile": {
			config: map[string]any{
				"profile": "SharedConfigurationProfile",
			},
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleValidEndpoint,
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			SharedConfigurationFile: fmt.Sprintf(`
[profile SharedConfigurationProfile]
role_arn = %[1]s
role_session_name = %[2]s
source_profile = SharedConfigurationSourceProfile

[profile SharedConfigurationSourceProfile]
aws_access_key_id = SharedConfigurationSourceAccessKey
aws_secret_access_key = SharedConfigurationSourceSecretKey
`, servicemocks.MockStsAssumeRoleArn, servicemocks.MockStsAssumeRoleSessionName),
		},

		// WAS: "environment AWS_PROFILE shared configuration source_profile"
		"from environment AWS_PROFILE with source profile": {
			config: map[string]any{},
			EnvironmentVariables: map[string]string{
				"AWS_PROFILE": "SharedConfigurationProfile",
			},
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleValidEndpoint,
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			SharedConfigurationFile: fmt.Sprintf(`
[profile SharedConfigurationProfile]
role_arn = %[1]s
role_session_name = %[2]s
source_profile = SharedConfigurationSourceProfile

[profile SharedConfigurationSourceProfile]
aws_access_key_id = SharedConfigurationSourceAccessKey
aws_secret_access_key = SharedConfigurationSourceSecretKey
`, servicemocks.MockStsAssumeRoleArn, servicemocks.MockStsAssumeRoleSessionName),
		},

		// WAS: "shared credentials default aws_access_key_id config AssumeRoleARN access key"
		"from default profile": {
			config: map[string]any{
				"assume_role": map[string]any{
					"role_arn":     servicemocks.MockStsAssumeRoleArn,
					"session_name": servicemocks.MockStsAssumeRoleSessionName,
				},
			},
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleValidEndpoint,
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			SharedCredentialsFile: `
[default]
aws_access_key_id = DefaultSharedCredentialsAccessKey
aws_secret_access_key = DefaultSharedCredentialsSecretKey
`,
		},

		// WAS: "EC2 metadata access key config AssumeRoleARN access key"
		"from EC2 metadata": {
			config: map[string]any{
				"assume_role": map[string]any{
					"role_arn":     servicemocks.MockStsAssumeRoleArn,
					"session_name": servicemocks.MockStsAssumeRoleSessionName,
				},
			},
			EnableEc2MetadataServer:  true,
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleValidEndpoint,
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
		},

		// WAS: "ECS credentials access key config AssumeRoleARN access key"
		"from ECS credentials": {
			config: map[string]any{
				"assume_role": map[string]any{
					"role_arn":     servicemocks.MockStsAssumeRoleArn,
					"session_name": servicemocks.MockStsAssumeRoleSessionName,
				},
			},
			EnableEcsCredentialsServer: true,
			ExpectedCredentialsValue:   mockdata.MockStsAssumeRoleCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleValidEndpoint,
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
		},

		// WAS: "config AssumeRoleDuration"
		"with duration": {
			config: map[string]any{
				"access_key": servicemocks.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
				"assume_role": map[string]any{
					"role_arn":     servicemocks.MockStsAssumeRoleArn,
					"session_name": servicemocks.MockStsAssumeRoleSessionName,
					"duration":     "1h",
				},
			},
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleValidEndpointWithOptions(map[string]string{"DurationSeconds": "3600"}),
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
		},

		// WAS: "config AssumeRoleExternalID"
		"with external ID": {
			config: map[string]any{
				"access_key": servicemocks.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
				"assume_role": map[string]any{
					"role_arn":     servicemocks.MockStsAssumeRoleArn,
					"session_name": servicemocks.MockStsAssumeRoleSessionName,
					"external_id":  servicemocks.MockStsAssumeRoleExternalId,
				},
			},
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleValidEndpointWithOptions(map[string]string{"ExternalId": servicemocks.MockStsAssumeRoleExternalId}),
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
		},

		// WAS: "config AssumeRolePolicy"
		"with policy": {
			config: map[string]any{
				"access_key": servicemocks.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
				"assume_role": map[string]any{
					"role_arn":     servicemocks.MockStsAssumeRoleArn,
					"session_name": servicemocks.MockStsAssumeRoleSessionName,
					"policy":       mockStsAssumeRolePolicy,
				},
			},
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleValidEndpointWithOptions(map[string]string{"Policy": mockStsAssumeRolePolicy}),
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
		},

		// WAS: "config AssumeRolePolicyARNs"
		"with policy ARNs": {
			config: map[string]any{
				"access_key": servicemocks.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
				"assume_role": map[string]any{
					"role_arn":     servicemocks.MockStsAssumeRoleArn,
					"session_name": servicemocks.MockStsAssumeRoleSessionName,
					"policy_arns":  []any{servicemocks.MockStsAssumeRolePolicyArn},
				},
			},
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleValidEndpointWithOptions(map[string]string{"PolicyArns.member.1.arn": servicemocks.MockStsAssumeRolePolicyArn}),
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
		},

		// WAS: "config AssumeRoleTags"
		"with tags": {
			config: map[string]any{
				"access_key": servicemocks.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
				"assume_role": map[string]any{
					"role_arn":     servicemocks.MockStsAssumeRoleArn,
					"session_name": servicemocks.MockStsAssumeRoleSessionName,
					"tags": map[string]any{
						servicemocks.MockStsAssumeRoleTagKey: servicemocks.MockStsAssumeRoleTagValue,
					},
				},
			},
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleValidEndpointWithOptions(map[string]string{"Tags.member.1.Key": servicemocks.MockStsAssumeRoleTagKey, "Tags.member.1.Value": servicemocks.MockStsAssumeRoleTagValue}),
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
		},

		// WAS: "config AssumeRoleTransitiveTagKeys"
		"with transitive tags": {
			config: map[string]any{
				"access_key": servicemocks.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
				"assume_role": map[string]any{
					"role_arn":     servicemocks.MockStsAssumeRoleArn,
					"session_name": servicemocks.MockStsAssumeRoleSessionName,
					"tags": map[string]any{
						servicemocks.MockStsAssumeRoleTagKey: servicemocks.MockStsAssumeRoleTagValue,
					},
					"transitive_tag_keys": []any{servicemocks.MockStsAssumeRoleTagKey},
				},
			},
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleValidEndpointWithOptions(map[string]string{"Tags.member.1.Key": servicemocks.MockStsAssumeRoleTagKey, "Tags.member.1.Value": servicemocks.MockStsAssumeRoleTagValue, "TransitiveTagKeys.member.1": servicemocks.MockStsAssumeRoleTagKey}),
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
		},

		// WAS: "config AssumeRoleSourceIdentity"
		"with source identity": {
			config: map[string]any{
				"access_key": servicemocks.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
				"assume_role": map[string]any{
					"role_arn":        servicemocks.MockStsAssumeRoleArn,
					"session_name":    servicemocks.MockStsAssumeRoleSessionName,
					"source_identity": servicemocks.MockStsAssumeRoleSourceIdentity,
				},
			},
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleValidEndpointWithOptions(map[string]string{"SourceIdentity": servicemocks.MockStsAssumeRoleSourceIdentity}),
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
		},

		// WAS: "assume role error"
		"error": {
			config: map[string]any{
				"access_key": servicemocks.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
				"assume_role": map[string]any{
					"role_arn":     servicemocks.MockStsAssumeRoleArn,
					"session_name": servicemocks.MockStsAssumeRoleSessionName,
				},
			},
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleInvalidEndpointInvalidClientTokenId,
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			ValidateDiags: ExpectDiags(
				diagMatching(
					tfdiags.Error,
					equalsMatcher("Cannot assume IAM Role"),
					newRegexpMatcher(`IAM Role \(.+\) cannot be assumed.`),
				),
			),
		},
	}

	for name, tc := range testCases {
		tc := tc

		t.Run(name, func(t *testing.T) {
			servicemocks.InitSessionTestEnv(t)

			ctx := context.TODO()

			// Populate required fields
			tc.config["region"] = "us-east-1"
			tc.config["bucket"] = "bucket"
			tc.config["key"] = "key"

			if tc.ValidateDiags == nil {
				tc.ValidateDiags = ExpectNoDiags
			}

			if tc.EnableEc2MetadataServer {
				closeEc2Metadata := servicemocks.AwsMetadataApiMock(append(
					servicemocks.Ec2metadata_securityCredentialsEndpoints,
					servicemocks.Ec2metadata_instanceIdEndpoint,
					servicemocks.Ec2metadata_iamInfoEndpoint,
				))
				defer closeEc2Metadata()
			}

			if tc.EnableEcsCredentialsServer {
				closeEcsCredentials := servicemocks.EcsCredentialsApiMock()
				defer closeEcsCredentials()
			}

			ts := servicemocks.MockAwsApiServer("STS", tc.MockStsEndpoints)
			defer ts.Close()

			tc.config["endpoints"] = map[string]any{
				"sts": ts.URL,
			}

			if tc.SharedConfigurationFile != "" {
				file, err := os.CreateTemp("", "aws-sdk-go-base-shared-configuration-file")

				if err != nil {
					t.Fatalf("unexpected error creating temporary shared configuration file: %s", err)
				}

				defer os.Remove(file.Name())

				err = os.WriteFile(file.Name(), []byte(tc.SharedConfigurationFile), 0600)

				if err != nil {
					t.Fatalf("unexpected error writing shared configuration file: %s", err)
				}

				setSharedConfigFile(t, file.Name())
			}

			if tc.SharedCredentialsFile != "" {
				file, err := os.CreateTemp("", "aws-sdk-go-base-shared-credentials-file")

				if err != nil {
					t.Fatalf("unexpected error creating temporary shared credentials file: %s", err)
				}

				defer os.Remove(file.Name())

				err = os.WriteFile(file.Name(), []byte(tc.SharedCredentialsFile), 0600)

				if err != nil {
					t.Fatalf("unexpected error writing shared credentials file: %s", err)
				}

				tc.config["shared_credentials_files"] = []interface{}{file.Name()}
				if tc.ExpectedCredentialsValue.Source == sharedConfigCredentialsProvider {
					tc.ExpectedCredentialsValue.Source = sharedConfigCredentialsSource(file.Name())
				}
			}

			for k, v := range tc.EnvironmentVariables {
				t.Setenv(k, v)
			}

			b, diags := configureBackend(t, tc.config)

			tc.ValidateDiags(t, diags)

			if diags.HasErrors() {
				return
			}

			credentials, err := b.awsConfig.Credentials.Retrieve(ctx)
			if err != nil {
				t.Fatalf("Error when requesting credentials: %s", err)
			}

			if diff := cmp.Diff(credentials, tc.ExpectedCredentialsValue, cmpopts.IgnoreFields(aws.Credentials{}, "Expires")); diff != "" {
				t.Fatalf("unexpected credentials: (- got, + expected)\n%s", diff)
			}
		})
	}
}

func TestBackendConfig_Authentication_AssumeRoleWithWebIdentity(t *testing.T) {
	testCases := map[string]struct {
		config                          map[string]any
		SetConfig                       bool
		ExpandEnvVars                   bool
		EnvironmentVariables            map[string]string
		SetTokenFileEnvironmentVariable bool
		SharedConfigurationFile         string
		SetSharedConfigurationFile      bool
		ExpectedCredentialsValue        aws.Credentials
		ValidateDiags                   DiagsValidator
		MockStsEndpoints                []*servicemocks.MockEndpoint
	}{
		"config with inline token": {
			config: map[string]any{
				"assume_role_with_web_identity": map[string]any{
					"role_arn":           servicemocks.MockStsAssumeRoleWithWebIdentityArn,
					"session_name":       servicemocks.MockStsAssumeRoleWithWebIdentitySessionName,
					"web_identity_token": servicemocks.MockWebIdentityToken,
				},
			},
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleWithWebIdentityCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleWithWebIdentityValidEndpoint,
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
		},

		"config with token file": {
			config: map[string]any{
				"assume_role_with_web_identity": map[string]any{
					"role_arn":     servicemocks.MockStsAssumeRoleWithWebIdentityArn,
					"session_name": servicemocks.MockStsAssumeRoleWithWebIdentitySessionName,
				},
			},
			SetConfig:                true,
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleWithWebIdentityCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleWithWebIdentityValidEndpoint,
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
		},

		"config with expanded path": {
			config: map[string]any{
				"assume_role_with_web_identity": map[string]any{
					"role_arn":     servicemocks.MockStsAssumeRoleWithWebIdentityArn,
					"session_name": servicemocks.MockStsAssumeRoleWithWebIdentitySessionName,
				},
			},
			SetConfig:                true,
			ExpandEnvVars:            true,
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleWithWebIdentityCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleWithWebIdentityValidEndpoint,
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
		},

		"envvar": {
			config: map[string]any{},
			EnvironmentVariables: map[string]string{
				"AWS_ROLE_ARN":          servicemocks.MockStsAssumeRoleWithWebIdentityArn,
				"AWS_ROLE_SESSION_NAME": servicemocks.MockStsAssumeRoleWithWebIdentitySessionName,
			},
			SetTokenFileEnvironmentVariable: true,
			ExpectedCredentialsValue:        mockdata.MockStsAssumeRoleWithWebIdentityCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleWithWebIdentityValidEndpoint,
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
		},

		"shared configuration file": {
			config: map[string]any{},
			SharedConfigurationFile: fmt.Sprintf(`
[default]
role_arn = %[1]s
role_session_name = %[2]s
`, servicemocks.MockStsAssumeRoleWithWebIdentityArn, servicemocks.MockStsAssumeRoleWithWebIdentitySessionName),
			SetSharedConfigurationFile: true,
			ExpectedCredentialsValue:   mockdata.MockStsAssumeRoleWithWebIdentityCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleWithWebIdentityValidEndpoint,
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
		},

		"config overrides envvar": {
			config: map[string]any{
				"assume_role_with_web_identity": map[string]any{
					"role_arn":           servicemocks.MockStsAssumeRoleWithWebIdentityArn,
					"session_name":       servicemocks.MockStsAssumeRoleWithWebIdentitySessionName,
					"web_identity_token": servicemocks.MockWebIdentityToken,
				},
			},
			EnvironmentVariables: map[string]string{
				"AWS_ROLE_ARN":                servicemocks.MockStsAssumeRoleWithWebIdentityAlternateArn,
				"AWS_ROLE_SESSION_NAME":       servicemocks.MockStsAssumeRoleWithWebIdentityAlternateSessionName,
				"AWS_WEB_IDENTITY_TOKEN_FILE": "no-such-file",
			},
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleWithWebIdentityCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleWithWebIdentityValidEndpoint,
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
		},

		// "config with file envvar": {
		// config: map[string]any{
		// 	"assume_role_with_web_identity": map[string]any{
		// 		"role_arn":     servicemocks.MockStsAssumeRoleWithWebIdentityArn,
		// 		"session_name": servicemocks.MockStsAssumeRoleWithWebIdentitySessionName,
		// 	},
		// },
		// 	SetTokenFileEnvironmentVariable: true,
		// 	ExpectedCredentialsValue:        mockdata.MockStsAssumeRoleWithWebIdentityCredentials,
		// 	MockStsEndpoints: []*servicemocks.MockEndpoint{
		// 		servicemocks.MockStsAssumeRoleWithWebIdentityValidEndpoint,
		//      servicemocks.MockStsGetCallerIdentityValidEndpoint,
		// 	},
		// },

		"envvar overrides shared configuration": {
			config: map[string]any{},
			EnvironmentVariables: map[string]string{
				"AWS_ROLE_ARN":          servicemocks.MockStsAssumeRoleWithWebIdentityArn,
				"AWS_ROLE_SESSION_NAME": servicemocks.MockStsAssumeRoleWithWebIdentitySessionName,
			},
			SetTokenFileEnvironmentVariable: true,
			SharedConfigurationFile: fmt.Sprintf(`
[default]
role_arn = %[1]s
role_session_name = %[2]s
web_identity_token_file = no-such-file
`, servicemocks.MockStsAssumeRoleWithWebIdentityAlternateArn, servicemocks.MockStsAssumeRoleWithWebIdentityAlternateSessionName),
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleWithWebIdentityCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleWithWebIdentityValidEndpoint,
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
		},

		"config overrides shared configuration": {
			config: map[string]any{
				"assume_role_with_web_identity": map[string]any{
					"role_arn":           servicemocks.MockStsAssumeRoleWithWebIdentityArn,
					"session_name":       servicemocks.MockStsAssumeRoleWithWebIdentitySessionName,
					"web_identity_token": servicemocks.MockWebIdentityToken,
				},
			},
			SharedConfigurationFile: fmt.Sprintf(`
[default]
role_arn = %[1]s
role_session_name = %[2]s
web_identity_token_file = no-such-file
`, servicemocks.MockStsAssumeRoleWithWebIdentityAlternateArn, servicemocks.MockStsAssumeRoleWithWebIdentityAlternateSessionName),
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleWithWebIdentityCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleWithWebIdentityValidEndpoint,
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
		},

		"with duration": {
			config: map[string]any{
				"assume_role_with_web_identity": map[string]any{
					"role_arn":           servicemocks.MockStsAssumeRoleWithWebIdentityArn,
					"session_name":       servicemocks.MockStsAssumeRoleWithWebIdentitySessionName,
					"web_identity_token": servicemocks.MockWebIdentityToken,
					"duration":           "1h",
				},
			},
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleWithWebIdentityCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleWithWebIdentityValidWithOptions(map[string]string{"DurationSeconds": "3600"}),
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
		},

		"with policy": {
			config: map[string]any{
				"assume_role_with_web_identity": map[string]any{
					"role_arn":           servicemocks.MockStsAssumeRoleWithWebIdentityArn,
					"session_name":       servicemocks.MockStsAssumeRoleWithWebIdentitySessionName,
					"web_identity_token": servicemocks.MockWebIdentityToken,
					"policy":             "{}",
				},
			},
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleWithWebIdentityCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleWithWebIdentityValidWithOptions(map[string]string{"Policy": "{}"}),
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
		},

		"invalid empty config": {
			config: map[string]any{
				"assume_role_with_web_identity": map[string]any{},
			},
			ValidateDiags: ExpectDiagsEqual(tfdiags.Diagnostics{
				attributeErrDiag(
					"Missing Required Value",
					`Exactly one of web_identity_token, web_identity_token_file must be set.`,
					cty.GetAttrPath("assume_role_with_web_identity"),
				),
				attributeErrDiag(
					"Missing Required Value",
					`The attribute "assume_role_with_web_identity.role_arn" is required by the backend.`+"\n\n"+
						"Refer to the backend documentation for additional information which attributes are required.",
					cty.GetAttrPath("assume_role_with_web_identity").GetAttr("role_arn"),
				),
			}),
		},

		"invalid no token": {
			config: map[string]any{
				"assume_role_with_web_identity": map[string]any{
					"role_arn": servicemocks.MockStsAssumeRoleWithWebIdentityArn,
				},
			},
			ValidateDiags: ExpectDiagsEqual(tfdiags.Diagnostics{
				attributeErrDiag(
					"Missing Required Value",
					`Exactly one of web_identity_token, web_identity_token_file must be set.`,
					cty.GetAttrPath("assume_role_with_web_identity"),
				),
			}),
		},

		"invalid token config conflict": {
			config: map[string]any{
				"assume_role_with_web_identity": map[string]any{
					"role_arn":           servicemocks.MockStsAssumeRoleWithWebIdentityArn,
					"session_name":       servicemocks.MockStsAssumeRoleWithWebIdentitySessionName,
					"web_identity_token": servicemocks.MockWebIdentityToken,
				},
			},
			SetConfig: true,
			ValidateDiags: ExpectDiagsEqual(tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid Attribute Combination",
					`Only one of web_identity_token, web_identity_token_file can be set.`,
					cty.GetAttrPath("assume_role_with_web_identity"),
				),
			}),
		},
	}

	for name, tc := range testCases {
		tc := tc

		t.Run(name, func(t *testing.T) {
			servicemocks.InitSessionTestEnv(t)

			ctx := context.TODO()

			// Populate required fields
			tc.config["region"] = "us-east-1"
			tc.config["bucket"] = "bucket"
			tc.config["key"] = "key"

			if tc.ValidateDiags == nil {
				tc.ValidateDiags = ExpectNoDiags
			}

			for k, v := range tc.EnvironmentVariables {
				t.Setenv(k, v)
			}

			ts := servicemocks.MockAwsApiServer("STS", tc.MockStsEndpoints)
			defer ts.Close()

			tc.config["endpoints"] = map[string]any{
				"sts": ts.URL,
			}

			tempdir, err := os.MkdirTemp("", "temp")
			if err != nil {
				t.Fatalf("error creating temp dir: %s", err)
			}
			defer os.Remove(tempdir)
			t.Setenv("TMPDIR", tempdir)

			tokenFile, err := os.CreateTemp("", "aws-sdk-go-base-web-identity-token-file")
			if err != nil {
				t.Fatalf("unexpected error creating temporary web identity token file: %s", err)
			}
			tokenFileName := tokenFile.Name()

			defer os.Remove(tokenFileName)

			err = os.WriteFile(tokenFileName, []byte(servicemocks.MockWebIdentityToken), 0600)

			if err != nil {
				t.Fatalf("unexpected error writing web identity token file: %s", err)
			}

			if tc.ExpandEnvVars {
				tmpdir := os.Getenv("TMPDIR")
				rel, err := filepath.Rel(tmpdir, tokenFileName)
				if err != nil {
					t.Fatalf("error making path relative: %s", err)
				}
				t.Logf("relative: %s", rel)
				tokenFileName = filepath.Join("$TMPDIR", rel)
				t.Logf("env tempfile: %s", tokenFileName)
			}

			if tc.SetConfig {
				ar := tc.config["assume_role_with_web_identity"].(map[string]any)
				ar["web_identity_token_file"] = tokenFileName
			}

			if tc.SetTokenFileEnvironmentVariable {
				t.Setenv("AWS_WEB_IDENTITY_TOKEN_FILE", tokenFileName)
			}

			if tc.SharedConfigurationFile != "" {
				file, err := os.CreateTemp("", "aws-sdk-go-base-shared-configuration-file")

				if err != nil {
					t.Fatalf("unexpected error creating temporary shared configuration file: %s", err)
				}

				defer os.Remove(file.Name())

				if tc.SetSharedConfigurationFile {
					tc.SharedConfigurationFile += fmt.Sprintf("web_identity_token_file = %s\n", tokenFileName)
				}

				err = os.WriteFile(file.Name(), []byte(tc.SharedConfigurationFile), 0600)

				if err != nil {
					t.Fatalf("unexpected error writing shared configuration file: %s", err)
				}

				tc.config["shared_config_files"] = []any{file.Name()}
			}

			tc.config["skip_credentials_validation"] = true

			b, diags := configureBackend(t, tc.config)

			tc.ValidateDiags(t, diags)

			if diags.HasErrors() {
				return
			}

			credentials, err := b.awsConfig.Credentials.Retrieve(ctx)
			if err != nil {
				t.Fatalf("Error when requesting credentials: %s", err)
			}

			if diff := cmp.Diff(credentials, tc.ExpectedCredentialsValue, cmpopts.IgnoreFields(aws.Credentials{}, "Expires")); diff != "" {
				t.Fatalf("unexpected credentials: (- got, + expected)\n%s", diff)
			}
		})
	}
}

var _ configtesting.TestDriver = &testDriver{}

type testDriver struct {
	mode configtesting.TestMode
}

func (t *testDriver) Init(mode configtesting.TestMode) {
	t.mode = mode
}

func (t testDriver) TestCase() configtesting.TestCaseDriver {
	return &testCaseDriver{
		mode: t.mode,
	}
}

var _ configtesting.TestCaseDriver = &testCaseDriver{}

type testCaseDriver struct {
	mode   configtesting.TestMode
	config configurer
}

func (d *testCaseDriver) Configuration(fs []configtesting.ConfigFunc) configtesting.Configurer {
	config := d.configuration()
	for _, f := range fs {
		f(config)
	}
	return config
}

func (d *testCaseDriver) configuration() *configurer {
	if d.config == nil {
		d.config = make(configurer, 0)
	}
	return &d.config
}

func (d *testCaseDriver) Setup(t *testing.T) {
	ts := servicemocks.MockAwsApiServer("STS", []*servicemocks.MockEndpoint{
		servicemocks.MockStsGetCallerIdentityValidEndpoint,
	})
	t.Cleanup(func() {
		ts.Close()
	})
	d.config.AddEndpoint("sts", ts.URL)
}

func (d testCaseDriver) Apply(ctx context.Context, t *testing.T) (context.Context, configtesting.Thing) {
	t.Helper()

	// Populate required fields
	d.config.SetRegion("us-east-1")
	d.config.setBucket("bucket")
	d.config.setKey("key")
	if d.mode == configtesting.TestModeLocal {
		d.config.SetSkipCredsValidation(true)
		d.config.SetSkipRequestingAccountId(true)
	}

	b, diags := configureBackend(t, map[string]any(d.config))

	var expected tfdiags.Diagnostics

	if diff := cmp.Diff(diags, expected, cmp.Comparer(diagnosticComparer)); diff != "" {
		t.Errorf("unexpected diagnostics difference: %s", diff)
	}

	return ctx, thing(b.awsConfig)
}

var _ configtesting.Configurer = &configurer{}

type configurer map[string]any

func (c configurer) AddEndpoint(k, v string) {
	if endpoints, ok := c["endpoints"]; ok {
		m := endpoints.(map[string]any)
		m[k] = v
	} else {
		c["endpoints"] = map[string]any{
			k: v,
		}
	}
}

func (c configurer) AddSharedConfigFile(f string) {
	x := c["shared_config_files"]
	if x == nil {
		c["shared_config_files"] = []any{f}
	} else {
		files := x.([]any)
		files = append(files, f)
		c["shared_config_files"] = files
	}
}

func (c configurer) setBucket(s string) {
	c["bucket"] = s
}

func (c configurer) setKey(s string) {
	c["key"] = s
}

func (c configurer) SetAccessKey(s string) {
	c["access_key"] = s
}

func (c configurer) SetSecretKey(s string) {
	c["secret_key"] = s
}

func (c configurer) SetProfile(s string) {
	c["profile"] = s
}

func (c configurer) SetRegion(s string) {
	c["region"] = s
}

func (c configurer) SetUseFIPSEndpoint(b bool) {
	c["use_fips_endpoint"] = b
}

func (c configurer) SetSkipCredsValidation(b bool) {
	c["skip_credentials_validation"] = b
}

func (c configurer) SetSkipRequestingAccountId(b bool) {
	c["skip_requesting_account_id"] = b
}

var _ configtesting.Thing = thing{}

type thing aws.Config

func (t thing) GetCredentials() aws.CredentialsProvider {
	return t.Credentials
}

func (t thing) GetRegion() string {
	return t.Region
}

func TestBackendConfig_Authentication_SSO(t *testing.T) {
	configtesting.SSO(t, &testDriver{})
}

func TestBackendConfig_Authentication_LegacySSO(t *testing.T) {
	configtesting.LegacySSO(t, &testDriver{})
}

func TestBackendConfig_Region(t *testing.T) {
	testCases := map[string]struct {
		config                  map[string]any
		EnvironmentVariables    map[string]string
		IMDSRegion              string
		SharedConfigurationFile string
		ExpectedRegion          string
	}{
		// NOT SUPPORTED: region is required
		// "no configuration": {
		// 	config: map[string]any{
		// 		"access_key": servicemocks.MockStaticAccessKey,
		// 		"secret_key": servicemocks.MockStaticSecretKey,
		// 	},
		// 	ExpectedRegion: "",
		// },

		"config": {
			config: map[string]any{
				"access_key": servicemocks.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
				"region":     "us-east-1",
			},
			ExpectedRegion: "us-east-1",
		},

		"AWS_REGION": {
			config: map[string]any{
				"access_key": servicemocks.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
			},
			EnvironmentVariables: map[string]string{
				"AWS_REGION": "us-east-1",
			},
			ExpectedRegion: "us-east-1",
		},
		"AWS_DEFAULT_REGION": {
			config: map[string]any{
				"access_key": servicemocks.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
			},
			EnvironmentVariables: map[string]string{
				"AWS_DEFAULT_REGION": "us-east-1",
			},
			ExpectedRegion: "us-east-1",
		},
		"AWS_REGION overrides AWS_DEFAULT_REGION": {
			config: map[string]any{
				"access_key": servicemocks.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
			},
			EnvironmentVariables: map[string]string{
				"AWS_REGION":         "us-east-1",
				"AWS_DEFAULT_REGION": "us-west-2",
			},
			ExpectedRegion: "us-east-1",
		},

		// NOT SUPPORTED: region from shared configuration file
		// 		"shared configuration file": {
		// 			config: map[string]any{
		// 				"access_key": servicemocks.MockStaticAccessKey,
		// 				"secret_key": servicemocks.MockStaticSecretKey,
		// 			},
		// 			SharedConfigurationFile: `
		// [default]
		// region = us-east-1
		// `,
		// 			ExpectedRegion: "us-east-1",
		// 		},

		// NOT SUPPORTED: region from IMDS
		// "IMDS": {
		// 	config:         map[string]any{},
		// 	IMDSRegion:     "us-east-1",
		// 	ExpectedRegion: "us-east-1",
		// },

		"config overrides AWS_REGION": {
			config: map[string]any{
				"access_key": servicemocks.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
				"region":     "us-east-1",
			},
			EnvironmentVariables: map[string]string{
				"AWS_REGION": "us-west-2",
			},
			ExpectedRegion: "us-east-1",
		},
		"config overrides AWS_DEFAULT_REGION": {
			config: map[string]any{
				"access_key": servicemocks.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
				"region":     "us-east-1",
			},
			EnvironmentVariables: map[string]string{
				"AWS_DEFAULT_REGION": "us-west-2",
			},
			ExpectedRegion: "us-east-1",
		},

		"config overrides IMDS": {
			config: map[string]any{
				"access_key": servicemocks.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
				"region":     "us-west-2",
			},
			IMDSRegion:     "us-east-1",
			ExpectedRegion: "us-west-2",
		},

		"AWS_REGION overrides shared configuration": {
			config: map[string]any{
				"access_key": servicemocks.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
			},
			EnvironmentVariables: map[string]string{
				"AWS_REGION": "us-east-1",
			},
			SharedConfigurationFile: `
[default]
region = us-west-2
`,
			ExpectedRegion: "us-east-1",
		},
		"AWS_DEFAULT_REGION overrides shared configuration": {
			config: map[string]any{
				"access_key": servicemocks.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
			},
			EnvironmentVariables: map[string]string{
				"AWS_DEFAULT_REGION": "us-east-1",
			},
			SharedConfigurationFile: `
[default]
region = us-west-2
`,
			ExpectedRegion: "us-east-1",
		},

		"AWS_REGION overrides IMDS": {
			config: map[string]any{
				"access_key": servicemocks.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
			},
			EnvironmentVariables: map[string]string{
				"AWS_REGION": "us-east-1",
			},
			IMDSRegion:     "us-west-2",
			ExpectedRegion: "us-east-1",
		},
	}

	for name, tc := range testCases {
		tc := tc

		t.Run(name, func(t *testing.T) {
			servicemocks.InitSessionTestEnv(t)

			// Populate required fields
			tc.config["bucket"] = "bucket"
			tc.config["key"] = "key"

			for k, v := range tc.EnvironmentVariables {
				t.Setenv(k, v)
			}

			if tc.IMDSRegion != "" {
				closeEc2Metadata := servicemocks.AwsMetadataApiMock(append(
					servicemocks.Ec2metadata_securityCredentialsEndpoints,
					servicemocks.Ec2metadata_instanceIdEndpoint,
					servicemocks.Ec2metadata_iamInfoEndpoint,
					servicemocks.Ec2metadata_instanceIdentityEndpoint(tc.IMDSRegion),
				))
				defer closeEc2Metadata()
			}

			ts := servicemocks.MockAwsApiServer("STS", []*servicemocks.MockEndpoint{
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			})
			defer ts.Close()

			tc.config["endpoints"] = map[string]any{
				"sts": ts.URL,
			}

			if tc.SharedConfigurationFile != "" {
				file, err := os.CreateTemp("", "aws-sdk-go-base-shared-configuration-file")

				if err != nil {
					t.Fatalf("unexpected error creating temporary shared configuration file: %s", err)
				}

				defer os.Remove(file.Name())

				err = os.WriteFile(file.Name(), []byte(tc.SharedConfigurationFile), 0600)

				if err != nil {
					t.Fatalf("unexpected error writing shared configuration file: %s", err)
				}

				setSharedConfigFile(t, file.Name())
			}

			tc.config["skip_credentials_validation"] = true

			b, diags := configureBackend(t, tc.config)
			if diags.HasErrors() {
				t.Fatalf("configuring backend: %s", diagnosticsString(diags))
			}

			if a, e := b.awsConfig.Region, tc.ExpectedRegion; a != e {
				t.Errorf("expected Region %q, got: %q", e, a)
			}
		})
	}
}

func setSharedConfigFile(t *testing.T, filename string) {
	t.Setenv("AWS_SDK_LOAD_CONFIG", "1")
	t.Setenv("AWS_CONFIG_FILE", filename)
}

func configureBackend(t *testing.T, config map[string]any) (*Backend, tfdiags.Diagnostics) {
	b := New().(*Backend)
	configSchema := populateSchema(t, b.ConfigSchema(), hcl2shim.HCL2ValueFromConfigValue(config))

	configSchema, diags := b.PrepareConfig(configSchema)

	if diags.HasErrors() {
		return b, diags
	}

	confDiags := b.Configure(configSchema)
	diags = diags.Append(confDiags)

	return b, diags
}

func sharedConfigCredentialsSource(filename string) string {
	return fmt.Sprintf(sharedConfigCredentialsProvider+": %s", filename)
}

func TestStsEndpoint(t *testing.T) {
	type settype int
	const (
		setNone settype = iota
		setValid
		setInvalid
	)
	testcases := map[string]struct {
		Config                   map[string]any
		SetServiceEndpoint       settype
		SetServiceEndpointLegacy settype
		SetEnv                   string
		SetInvalidEnv            string
		// Use string at index 1 for valid endpoint url and index 2 for invalid endpoint url
		ConfigFile          string
		ExpectedCredentials aws.Credentials
	}{
		// Service Config

		"service config": {
			Config: map[string]any{
				"access_key": servicemocks.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
			},
			SetServiceEndpoint:  setValid,
			ExpectedCredentials: mockdata.MockStaticCredentials,
		},

		"service config overrides service envvar": {
			Config: map[string]any{
				"access_key": servicemocks.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
			},
			SetServiceEndpoint:  setValid,
			SetInvalidEnv:       "AWS_ENDPOINT_URL_STS",
			ExpectedCredentials: mockdata.MockStaticCredentials,
		},

		"service config overrides service envvar legacy": {
			Config: map[string]any{
				"access_key": servicemocks.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
			},
			SetServiceEndpoint:  setValid,
			SetInvalidEnv:       "AWS_STS_ENDPOINT",
			ExpectedCredentials: mockdata.MockStaticCredentials,
		},

		"service config overrides base envvar": {
			Config: map[string]any{
				"access_key": servicemocks.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
			},
			SetServiceEndpoint:  setValid,
			SetInvalidEnv:       "AWS_ENDPOINT_URL",
			ExpectedCredentials: mockdata.MockStaticCredentials,
		},

		"service config overrides service config_file": {
			Config: map[string]any{
				"profile": "default",
			},
			ConfigFile: `
[default]
aws_access_key_id = DefaultSharedCredentialsAccessKey
aws_secret_access_key = DefaultSharedCredentialsSecretKey
services = sts-test

[services sts-test]
sts =
	endpoint_url = %[2]s
`,
			SetServiceEndpoint: setValid,
			ExpectedCredentials: aws.Credentials{
				AccessKeyID:     "DefaultSharedCredentialsAccessKey",
				SecretAccessKey: "DefaultSharedCredentialsSecretKey",
				Source:          sharedConfigCredentialsProvider,
			},
		},

		"service config overrides base config_file": {
			Config: map[string]any{
				"profile": "default",
			},
			ConfigFile: `
[default]
aws_access_key_id = DefaultSharedCredentialsAccessKey
aws_secret_access_key = DefaultSharedCredentialsSecretKey
endpoint_url = %[2]s
`,
			SetServiceEndpoint: setValid,
			ExpectedCredentials: aws.Credentials{
				AccessKeyID:     "DefaultSharedCredentialsAccessKey",
				SecretAccessKey: "DefaultSharedCredentialsSecretKey",
				Source:          sharedConfigCredentialsProvider,
			},
		},

		// Service Config Legacy

		"service config legacy": {
			Config: map[string]any{
				"access_key": servicemocks.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
			},
			SetServiceEndpointLegacy: setValid,
			ExpectedCredentials:      mockdata.MockStaticCredentials,
		},

		"service config legacy overrides service envvar": {
			Config: map[string]any{
				"access_key": servicemocks.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
			},
			SetServiceEndpointLegacy: setValid,
			SetInvalidEnv:            "AWS_ENDPOINT_URL_STS",
			ExpectedCredentials:      mockdata.MockStaticCredentials,
		},

		"service config legacy overrides service envvar legacy": {
			Config: map[string]any{
				"access_key": servicemocks.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
			},
			SetServiceEndpointLegacy: setValid,
			SetInvalidEnv:            "AWS_STS_ENDPOINT",
			ExpectedCredentials:      mockdata.MockStaticCredentials,
		},

		"service config legacy overrides base envvar": {
			Config: map[string]any{
				"access_key": servicemocks.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
			},
			SetServiceEndpointLegacy: setValid,
			SetInvalidEnv:            "AWS_ENDPOINT_URL",
			ExpectedCredentials:      mockdata.MockStaticCredentials,
		},

		"service config legacy overrides service config_file": {
			Config: map[string]any{
				"profile": "default",
			},
			ConfigFile: `
[default]
aws_access_key_id = DefaultSharedCredentialsAccessKey
aws_secret_access_key = DefaultSharedCredentialsSecretKey
services = sts-test

[services sts-test]
sts =
	endpoint_url = %[2]s
`,
			SetServiceEndpointLegacy: setValid,
			ExpectedCredentials: aws.Credentials{
				AccessKeyID:     "DefaultSharedCredentialsAccessKey",
				SecretAccessKey: "DefaultSharedCredentialsSecretKey",
				Source:          sharedConfigCredentialsProvider,
			},
		},

		"service config legacy overrides base config_file": {
			Config: map[string]any{
				"profile": "default",
			},
			ConfigFile: `
[default]
aws_access_key_id = DefaultSharedCredentialsAccessKey
aws_secret_access_key = DefaultSharedCredentialsSecretKey
endpoint_url = %[2]s
`,
			SetServiceEndpointLegacy: setValid,
			ExpectedCredentials: aws.Credentials{
				AccessKeyID:     "DefaultSharedCredentialsAccessKey",
				SecretAccessKey: "DefaultSharedCredentialsSecretKey",
				Source:          sharedConfigCredentialsProvider,
			},
		},

		// Service Envvar

		"service envvar": {
			Config: map[string]any{
				"access_key": servicemocks.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
			},
			SetEnv:              "AWS_ENDPOINT_URL_STS",
			ExpectedCredentials: mockdata.MockStaticCredentials,
		},

		"service envvar overrides base envvar": {
			Config: map[string]any{
				"access_key": servicemocks.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
			},
			SetEnv:              "AWS_ENDPOINT_URL_STS",
			SetInvalidEnv:       "AWS_ENDPOINT_URL",
			ExpectedCredentials: mockdata.MockStaticCredentials,
		},

		"service envvar overrides service config_file": {
			Config: map[string]any{
				"profile": "default",
			},
			SetEnv: "AWS_ENDPOINT_URL_STS",
			ConfigFile: `
[default]
aws_access_key_id = DefaultSharedCredentialsAccessKey
aws_secret_access_key = DefaultSharedCredentialsSecretKey
services = sts-test

[services sts-test]
sts =
	endpoint_url = %[2]s
`,
			ExpectedCredentials: aws.Credentials{
				AccessKeyID:     "DefaultSharedCredentialsAccessKey",
				SecretAccessKey: "DefaultSharedCredentialsSecretKey",
				Source:          sharedConfigCredentialsProvider,
			},
		},

		"service envvar overrides base config_file": {
			Config: map[string]any{
				"profile": "default",
			},
			SetEnv: "AWS_ENDPOINT_URL_STS",
			ConfigFile: `
[default]
aws_access_key_id = DefaultSharedCredentialsAccessKey
aws_secret_access_key = DefaultSharedCredentialsSecretKey
endpoint_url = %[2]s
`,
			ExpectedCredentials: aws.Credentials{
				AccessKeyID:     "DefaultSharedCredentialsAccessKey",
				SecretAccessKey: "DefaultSharedCredentialsSecretKey",
				Source:          sharedConfigCredentialsProvider,
			},
		},

		"service envvar overrides service envvar legacy": {
			Config: map[string]any{
				"access_key": servicemocks.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
			},
			SetEnv:              "AWS_ENDPOINT_URL_STS",
			SetInvalidEnv:       "AWS_STS_ENDPOINT",
			ExpectedCredentials: mockdata.MockStaticCredentials,
		},

		// Service Envvar Legacy

		"service envvar legacy": {
			Config: map[string]any{
				"access_key": servicemocks.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
			},
			SetEnv:              "AWS_STS_ENDPOINT",
			ExpectedCredentials: mockdata.MockStaticCredentials,
		},

		"service envvar legacy overrides base envvar": {
			Config: map[string]any{
				"access_key": servicemocks.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
			},
			SetEnv:              "AWS_STS_ENDPOINT",
			SetInvalidEnv:       "AWS_ENDPOINT_URL",
			ExpectedCredentials: mockdata.MockStaticCredentials,
		},

		"service envvar legacy overrides service config_file": {
			Config: map[string]any{
				"profile": "default",
			},
			SetEnv: "AWS_STS_ENDPOINT",
			ConfigFile: `
[default]
aws_access_key_id = DefaultSharedCredentialsAccessKey
aws_secret_access_key = DefaultSharedCredentialsSecretKey
services = sts-test

[services sts-test]
sts =
	endpoint_url = %[2]s
`,
			ExpectedCredentials: aws.Credentials{
				AccessKeyID:     "DefaultSharedCredentialsAccessKey",
				SecretAccessKey: "DefaultSharedCredentialsSecretKey",
				Source:          sharedConfigCredentialsProvider,
			},
		},

		"service envvar legacy overrides base config_file": {
			Config: map[string]any{
				"profile": "default",
			},
			SetEnv: "AWS_STS_ENDPOINT",
			ConfigFile: `
[default]
aws_access_key_id = DefaultSharedCredentialsAccessKey
aws_secret_access_key = DefaultSharedCredentialsSecretKey
endpoint_url = %[2]s
`,
			ExpectedCredentials: aws.Credentials{
				AccessKeyID:     "DefaultSharedCredentialsAccessKey",
				SecretAccessKey: "DefaultSharedCredentialsSecretKey",
				Source:          sharedConfigCredentialsProvider,
			},
		},

		// Service Config File

		"service config_file": {
			Config: map[string]any{
				"profile": "default",
			},
			ConfigFile: `
[default]
aws_access_key_id = DefaultSharedCredentialsAccessKey
aws_secret_access_key = DefaultSharedCredentialsSecretKey
services = sts-test

[services sts-test]
sts =
	endpoint_url = %[1]s
`,
			ExpectedCredentials: aws.Credentials{
				AccessKeyID:     "DefaultSharedCredentialsAccessKey",
				SecretAccessKey: "DefaultSharedCredentialsSecretKey",
				Source:          sharedConfigCredentialsProvider,
			},
		},

		"service config_file overrides base config_file": {
			Config: map[string]any{
				"profile": "default",
			},
			ConfigFile: `
[default]
aws_access_key_id = DefaultSharedCredentialsAccessKey
aws_secret_access_key = DefaultSharedCredentialsSecretKey
services = sts-test
endpoint_url = %[2]s

[services sts-test]
sts =
	endpoint_url = %[1]s
`,
			ExpectedCredentials: aws.Credentials{
				AccessKeyID:     "DefaultSharedCredentialsAccessKey",
				SecretAccessKey: "DefaultSharedCredentialsSecretKey",
				Source:          sharedConfigCredentialsProvider,
			},
		},

		// Base envvar

		"base envvar": {
			Config: map[string]any{
				"access_key": servicemocks.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
			},
			SetEnv:              "AWS_ENDPOINT_URL",
			ExpectedCredentials: mockdata.MockStaticCredentials,
		},

		"base envvar overrides service config_file": {
			Config: map[string]any{
				"profile": "default",
			},
			SetEnv: "AWS_ENDPOINT_URL",
			ConfigFile: `
[default]
aws_access_key_id = DefaultSharedCredentialsAccessKey
aws_secret_access_key = DefaultSharedCredentialsSecretKey
services = sts-test

[services sts-test]
sts =
	endpoint_url = %[2]s
`,
			ExpectedCredentials: aws.Credentials{
				AccessKeyID:     "DefaultSharedCredentialsAccessKey",
				SecretAccessKey: "DefaultSharedCredentialsSecretKey",
				Source:          sharedConfigCredentialsProvider,
			},
		},

		"base config_file": {
			Config: map[string]any{
				"profile": "default",
			},
			ConfigFile: `
[default]
aws_access_key_id = DefaultSharedCredentialsAccessKey
aws_secret_access_key = DefaultSharedCredentialsSecretKey
endpoint_url = %[1]s
`,
			ExpectedCredentials: aws.Credentials{
				AccessKeyID:     "DefaultSharedCredentialsAccessKey",
				SecretAccessKey: "DefaultSharedCredentialsSecretKey",
				Source:          sharedConfigCredentialsProvider,
			},
		},

		"base envvar overrides base config_file": {
			Config: map[string]any{
				"profile": "default",
			},
			SetEnv: "AWS_ENDPOINT_URL",
			ConfigFile: `
[default]
aws_access_key_id = DefaultSharedCredentialsAccessKey
aws_secret_access_key = DefaultSharedCredentialsSecretKey
endpoint_url = %[2]s
`,
			ExpectedCredentials: aws.Credentials{
				AccessKeyID:     "DefaultSharedCredentialsAccessKey",
				SecretAccessKey: "DefaultSharedCredentialsSecretKey",
				Source:          sharedConfigCredentialsProvider,
			},
		},
	}

	for name, testcase := range testcases {
		testcase := testcase

		t.Run(name, func(t *testing.T) {
			servicemocks.InitSessionTestEnv(t)

			// Populate required fields
			testcase.Config["bucket"] = "bucket"
			testcase.Config["key"] = "key"
			testcase.Config["region"] = "us-west-2"

			ts := servicemocks.MockAwsApiServer("STS", []*servicemocks.MockEndpoint{
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			})
			defer ts.Close()
			stsEndpoint := ts.URL

			invalidTS := servicemocks.MockAwsApiServer("STS", []*servicemocks.MockEndpoint{
				servicemocks.MockStsGetCallerIdentityInvalidEndpointAccessDenied,
			})
			defer invalidTS.Close()
			stsInvalidEndpoint := invalidTS.URL

			if testcase.SetServiceEndpoint == setValid {
				testcase.Config["endpoints"] = map[string]any{
					"sts": stsEndpoint,
				}
			}
			if testcase.SetServiceEndpointLegacy == setValid {
				testcase.Config["sts_endpoint"] = stsEndpoint
			}
			if testcase.SetEnv != "" {
				t.Setenv(testcase.SetEnv, stsEndpoint)
			}
			if testcase.SetInvalidEnv != "" {
				t.Setenv(testcase.SetInvalidEnv, stsInvalidEndpoint)
			}
			if testcase.ConfigFile != "" {
				tempDir := t.TempDir()
				filename := writeSharedConfigFile(t, testcase.Config, tempDir, fmt.Sprintf(testcase.ConfigFile, stsEndpoint, stsInvalidEndpoint))
				testcase.ExpectedCredentials.Source = sharedConfigCredentialsSource(filename)
			}

			b, diags := configureBackend(t, testcase.Config)
			if diags.HasErrors() {
				t.Fatalf("configuring backend: %s", diagnosticsString(diags))
			}

			ctx := context.TODO()

			credentialsValue, err := b.awsConfig.Credentials.Retrieve(ctx)
			if err != nil {
				t.Fatalf("unexpected credentials Retrieve() error: %s", err)
			}

			if diff := cmp.Diff(credentialsValue, testcase.ExpectedCredentials, cmpopts.IgnoreFields(aws.Credentials{}, "Expires")); diff != "" {
				t.Fatalf("unexpected credentials: (- got, + expected)\n%s", diff)
			}
		})
	}
}

func writeSharedConfigFile(t *testing.T, config map[string]any, tempDir, content string) string {
	t.Helper()

	file, err := os.Create(filepath.Join(tempDir, "aws-sdk-go-base-shared-configuration-file"))
	if err != nil {
		t.Fatalf("creating shared configuration file: %s", err)
	}

	_, err = file.WriteString(content)
	if err != nil {
		t.Fatalf(" writing shared configuration file: %s", err)
	}

	config["shared_config_files"] = []any{file.Name()}

	return file.Name()
}
