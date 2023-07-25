package s3

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/google/go-cmp/cmp"
	awsbase "github.com/hashicorp/aws-sdk-go-base"
	mockdata "github.com/hashicorp/aws-sdk-go-base"
	servicemocks "github.com/hashicorp/aws-sdk-go-base"
	"github.com/hashicorp/terraform/internal/configs/hcl2shim"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type DiagsValidator func(*testing.T, tfdiags.Diagnostics)

func ExpectNoDiags(t *testing.T, diags tfdiags.Diagnostics) {
	expectDiagsCount(t, diags, 0)
}

func expectDiagsCount(t *testing.T, diags tfdiags.Diagnostics, c int) {
	if l := len(diags); l != c {
		t.Fatalf("Diagnostics: expected %d element, got %d\n%#v", c, l, diags)
	}
}

func ExpectDiagsEqual(expected tfdiags.Diagnostics) DiagsValidator {
	return func(t *testing.T, diags tfdiags.Diagnostics) {
		if diff := cmp.Diff(diags, expected, cmp.Comparer(diagnosticComparer)); diff != "" {
			t.Fatalf("unexpected diagnostics difference: %s", diff)
		}
	}
}

// ExpectDiagMatching returns a validator expeceting a single Diagnostic with fields matching the expectation
func ExpectDiagMatching(severity tfdiags.Severity, summary matcher, detail matcher) DiagsValidator {
	return func(t *testing.T, diags tfdiags.Diagnostics) {
		for _, d := range diags {
			if !summary.Match(d.Description().Summary) || !detail.Match(d.Description().Detail) {
				t.Fatalf("expected Diagnostic matching %#v, got %#v",
					tfdiags.Sourceless(
						severity,
						summary.String(),
						detail.String(),
					),
					d,
				)
			}
		}

		expectDiagsCount(t, diags, 1)
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

func TestBackendConfig_Authentication(t *testing.T) {
	testCases := map[string]struct {
		config                     map[string]any
		EnableEc2MetadataServer    bool
		EnableEcsCredentialsServer bool
		EnableWebIdentityEnvVars   bool
		// EnableWebIdentityConfig    bool // Not supported
		EnvironmentVariables     map[string]string
		ExpectedCredentialsValue credentials.Value
		MockStsEndpoints         []*servicemocks.MockEndpoint
		SharedConfigurationFile  string
		SharedCredentialsFile    string
		ValidateDiags            DiagsValidator
	}{
		"empty config": {
			config: map[string]any{},
			ValidateDiags: ExpectDiagMatching(
				tfdiags.Error,
				equalsMatcher("Failed to configure AWS client"),
				newRegexpMatcher("no valid credential sources for S3 Backend found"),
			),
		},

		"config AccessKey": {
			config: map[string]any{
				"access_key": awsbase.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
			},
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			ExpectedCredentialsValue: mockdata.MockStaticCredentials,
			ValidateDiags:            ExpectNoDiags,
		},

		"config AccessKey config AssumeRoleARN access key": {
			config: map[string]any{
				"access_key":   awsbase.MockStaticAccessKey,
				"secret_key":   servicemocks.MockStaticSecretKey,
				"role_arn":     servicemocks.MockStsAssumeRoleArn,
				"session_name": servicemocks.MockStsAssumeRoleSessionName,
			},
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleValidEndpoint,
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
		},

		"config AssumeRoleDuration": {
			config: map[string]any{
				"access_key":                   awsbase.MockStaticAccessKey,
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
		},

		"config AssumeRoleExternalID": {
			config: map[string]any{
				"access_key":   awsbase.MockStaticAccessKey,
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
		},

		"config AssumeRolePolicy": {
			config: map[string]any{
				"access_key":         awsbase.MockStaticAccessKey,
				"secret_key":         servicemocks.MockStaticSecretKey,
				"role_arn":           servicemocks.MockStsAssumeRoleArn,
				"session_name":       servicemocks.MockStsAssumeRoleSessionName,
				"assume_role_policy": servicemocks.MockStsAssumeRolePolicy,
			},
			ExpectedCredentialsValue: mockdata.MockStsAssumeRoleCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleValidEndpointWithOptions(map[string]string{"Policy": servicemocks.MockStsAssumeRolePolicy}),
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
		},

		"config AssumeRolePolicyARNs": {
			config: map[string]any{
				"access_key":              awsbase.MockStaticAccessKey,
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
		},

		"config AssumeRoleTags": {
			config: map[string]any{
				"access_key":   awsbase.MockStaticAccessKey,
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
		},

		"config AssumeRoleTransitiveTagKeys": {
			config: map[string]any{
				"access_key":   awsbase.MockStaticAccessKey,
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
		},

		// NOT SUPPORTED: AssumeRoleSourceIdentity
		// "config AssumeRoleSourceIdentity": {
		// 	config: map[string]any{
		// 		"access_key":                  awsbase.MockStaticAccessKey,
		// 		"secret_key":                  servicemocks.MockStaticSecretKey,
		// 		"role_arn":                    servicemocks.MockStsAssumeRoleArn,
		// 		"session_name":                servicemocks.MockStsAssumeRoleSessionName,
		// 		"assume_role_source_identity": servicemocks.MockStsAssumeRoleSourceIdentity,
		// 	},
		// 	ExpectedCredentialsValue: mockdata.MockStsAssumeRoleCredentials,
		// 	MockStsEndpoints: []*servicemocks.MockEndpoint{
		// 		servicemocks.MockStsAssumeRoleValidEndpointWithOptions(map[string]string{"SourceIdentity": servicemocks.MockStsAssumeRoleSourceIdentity}),
		// 		servicemocks.MockStsGetCallerIdentityValidEndpoint,
		// 	},
		// },

		"config Profile shared credentials profile aws_access_key_id": {
			config: map[string]any{
				"profile": "SharedCredentialsProfile",
			},
			ExpectedCredentialsValue: credentials.Value{
				AccessKeyID:     "ProfileSharedCredentialsAccessKey",
				SecretAccessKey: "ProfileSharedCredentialsSecretKey",
				ProviderName:    credentials.SharedCredsProviderName,
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

		"environment AWS_ACCESS_KEY_ID overrides config Profile": { // Legacy behavior
			config: map[string]any{
				"profile": "SharedCredentialsProfile",
			},
			EnvironmentVariables: map[string]string{
				"AWS_ACCESS_KEY_ID":     servicemocks.MockEnvAccessKey,
				"AWS_SECRET_ACCESS_KEY": servicemocks.MockEnvSecretKey,
			},
			ExpectedCredentialsValue: servicemocks.MockEnvCredentials,
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

		"config Profile shared configuration credential_source Ec2InstanceMetadata": {
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

		"config Profile shared configuration source_profile": {
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

		"environment AWS_ACCESS_KEY_ID config AssumeRoleARN access key": {
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
		},

		"environment AWS_PROFILE shared credentials profile aws_access_key_id": {
			config: map[string]any{},
			EnvironmentVariables: map[string]string{
				"AWS_PROFILE": "SharedCredentialsProfile",
			},
			ExpectedCredentialsValue: credentials.Value{
				AccessKeyID:     "ProfileSharedCredentialsAccessKey",
				SecretAccessKey: "ProfileSharedCredentialsSecretKey",
				ProviderName:    credentials.SharedCredsProviderName,
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

		"environment AWS_PROFILE shared configuration credential_source Ec2InstanceMetadata": {
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

		"environment AWS_PROFILE shared configuration source_profile": {
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
			ExpectedCredentialsValue: credentials.Value{
				AccessKeyID:     "DefaultSharedCredentialsAccessKey",
				SecretAccessKey: "DefaultSharedCredentialsSecretKey",
				ProviderName:    credentials.SharedCredsProviderName,
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

		"shared credentials default aws_access_key_id config AssumeRoleARN access key": {
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

		"EC2 metadata access key config AssumeRoleARN access key": {
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
		},

		"ECS credentials access key": {
			config:                     map[string]any{},
			EnableEcsCredentialsServer: true,
			ExpectedCredentialsValue:   mockdata.MockEcsCredentialsCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
		},

		"ECS credentials access key config AssumeRoleARN access key": {
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
		},

		// NOT SUPPORTED: AssumeWebIdentity config
		// "AssumeWebIdentity config AssumeRoleARN access key": {
		// 	config: map[string]any{
		// 		"role_arn":     servicemocks.MockStsAssumeRoleArn,
		// 		"session_name": servicemocks.MockStsAssumeRoleSessionName,
		// 	},
		// 	EnableWebIdentityConfig:  true,
		// 	ExpectedCredentialsValue: mockdata.MockStsAssumeRoleCredentials,
		// 	MockStsEndpoints: []*servicemocks.MockEndpoint{
		// 		servicemocks.MockStsAssumeRoleWithWebIdentityValidEndpoint,
		// 		servicemocks.MockStsAssumeRoleValidEndpoint,
		// 		servicemocks.MockStsGetCallerIdentityValidEndpoint,
		// 	},
		// },

		"config AccessKey over environment AWS_ACCESS_KEY_ID": {
			config: map[string]any{
				"access_key": awsbase.MockStaticAccessKey,
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
				"access_key": awsbase.MockStaticAccessKey,
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
				"access_key": awsbase.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
			},
			ExpectedCredentialsValue: mockdata.MockStaticCredentials,
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
		},

		"config AccessKey over ECS credentials access key": {
			config: map[string]any{
				"access_key": awsbase.MockStaticAccessKey,
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
			ExpectedCredentialsValue: credentials.Value{
				AccessKeyID:     "DefaultSharedCredentialsAccessKey",
				SecretAccessKey: "DefaultSharedCredentialsSecretKey",
				ProviderName:    credentials.SharedCredsProviderName,
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
			ExpectedCredentialsValue: credentials.Value{
				AccessKeyID:     "DefaultSharedCredentialsAccessKey",
				SecretAccessKey: "DefaultSharedCredentialsSecretKey",
				ProviderName:    credentials.SharedCredsProviderName,
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
				"access_key": awsbase.MockStaticAccessKey,
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

		"assume role error": {
			config: map[string]any{
				"access_key":   awsbase.MockStaticAccessKey,
				"secret_key":   servicemocks.MockStaticSecretKey,
				"role_arn":     servicemocks.MockStsAssumeRoleArn,
				"session_name": servicemocks.MockStsAssumeRoleSessionName,
			},
			MockStsEndpoints: []*servicemocks.MockEndpoint{
				servicemocks.MockStsAssumeRoleInvalidEndpointInvalidClientTokenId,
				servicemocks.MockStsGetCallerIdentityValidEndpoint,
			},
			ValidateDiags: ExpectDiagMatching(
				tfdiags.Error,
				equalsMatcher("Failed to configure AWS client"),
				newRegexpMatcher(`IAM Role \(.+\) cannot be assumed.`),
			),
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
				equalsMatcher("Failed to configure AWS client"),
				newRegexpMatcher("no valid credential sources for S3 Backend found"),
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
				equalsMatcher("Failed to configure AWS client"),
				newRegexpMatcher("no valid credential sources for S3 Backend found"),
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
				equalsMatcher("Failed to configure AWS client"),
				newRegexpMatcher("no valid credential sources for S3 Backend found"),
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
				equalsMatcher("Failed to configure AWS client"),
				newRegexpMatcher("error validating provider credentials:"),
			),
		},
	}

	for name, tc := range testCases {
		tc := tc

		t.Run(name, func(t *testing.T) {
			oldEnv := initSessionTestEnv()
			defer popEnv(oldEnv)

			// Populate required fields
			tc.config["region"] = "us-east-1"
			tc.config["bucket"] = "bucket"
			tc.config["key"] = "key"

			if tc.ValidateDiags == nil {
				tc.ValidateDiags = ExpectNoDiags
			}

			if tc.EnableEc2MetadataServer {
				closeEc2Metadata := awsMetadataApiMock(append(
					ec2metadata_securityCredentialsEndpoints,
					ec2metadata_instanceIdEndpoint,
					ec2metadata_iamInfoEndpoint,
				))
				defer closeEc2Metadata()
			}

			if tc.EnableEcsCredentialsServer {
				closeEcsCredentials := ecsCredentialsApiMock()
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
					os.Setenv("AWS_ROLE_ARN", servicemocks.MockStsAssumeRoleWithWebIdentityArn)
					os.Setenv("AWS_ROLE_SESSION_NAME", servicemocks.MockStsAssumeRoleWithWebIdentitySessionName)
					os.Setenv("AWS_WEB_IDENTITY_TOKEN_FILE", file.Name())
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

			tc.config["sts_endpoint"] = ts.URL

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

				setSharedConfigFile(file.Name())
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

				tc.config["shared_credentials_file"] = file.Name()
			}

			for k, v := range tc.EnvironmentVariables {
				os.Setenv(k, v)
			}

			b, diags := configureBackend(t, tc.config)

			tc.ValidateDiags(t, diags)

			if diags.HasErrors() {
				return
			}

			credentials, err := b.s3Client.Config.Credentials.Get()
			if err != nil {
				t.Fatalf("Error when requesting credentials: %s", err)
			}

			if diff := cmp.Diff(credentials, tc.ExpectedCredentialsValue); diff != "" {
				t.Fatalf("unexpected credentials: (- got, + expected)\n%s", diff)
			}
		})
	}
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
		// 		"access_key": awsbase.MockStaticAccessKey,
		// 		"secret_key": servicemocks.MockStaticSecretKey,
		// 	},
		// 	ExpectedRegion: "",
		// },

		"config": {
			config: map[string]any{
				"access_key": awsbase.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
				"region":     "us-east-1",
			},
			ExpectedRegion: "us-east-1",
		},

		"AWS_REGION": {
			config: map[string]any{
				"access_key": awsbase.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
			},
			EnvironmentVariables: map[string]string{
				"AWS_REGION": "us-east-1",
			},
			ExpectedRegion: "us-east-1",
		},
		"AWS_DEFAULT_REGION": {
			config: map[string]any{
				"access_key": awsbase.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
			},
			EnvironmentVariables: map[string]string{
				"AWS_DEFAULT_REGION": "us-east-1",
			},
			ExpectedRegion: "us-east-1",
		},
		"AWS_REGION overrides AWS_DEFAULT_REGION": {
			config: map[string]any{
				"access_key": awsbase.MockStaticAccessKey,
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
		// 				"access_key": awsbase.MockStaticAccessKey,
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
				"access_key": awsbase.MockStaticAccessKey,
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
				"access_key": awsbase.MockStaticAccessKey,
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
				"access_key": awsbase.MockStaticAccessKey,
				"secret_key": servicemocks.MockStaticSecretKey,
				"region":     "us-west-2",
			},
			IMDSRegion:     "us-east-1",
			ExpectedRegion: "us-west-2",
		},

		"AWS_REGION overrides shared configuration": {
			config: map[string]any{
				"access_key": awsbase.MockStaticAccessKey,
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
				"access_key": awsbase.MockStaticAccessKey,
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
				"access_key": awsbase.MockStaticAccessKey,
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
			oldEnv := initSessionTestEnv()
			defer popEnv(oldEnv)

			// Populate required fields
			tc.config["bucket"] = "bucket"
			tc.config["key"] = "key"

			for k, v := range tc.EnvironmentVariables {
				os.Setenv(k, v)
			}

			if tc.IMDSRegion != "" {
				closeEc2Metadata := awsMetadataApiMock(append(
					ec2metadata_securityCredentialsEndpoints,
					ec2metadata_instanceIdEndpoint,
					ec2metadata_iamInfoEndpoint,
					ec2metadata_instanceIdentityEndpoint(tc.IMDSRegion),
				))
				defer closeEc2Metadata()
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

				setSharedConfigFile(file.Name())
			}

			tc.config["skip_credentials_validation"] = true

			b, diags := configureBackend(t, tc.config)
			if diags.HasErrors() {
				t.Fatalf("configuring backend: %s", diagnosticsString(diags))
			}

			if a, e := aws.StringValue(b.s3Client.Config.Region), tc.ExpectedRegion; a != e {
				t.Errorf("expected Region %q, got: %q", e, a)
			}
		})
	}
}

func setSharedConfigFile(filename string) {
	os.Setenv("AWS_SDK_LOAD_CONFIG", "1")
	os.Setenv("AWS_CONFIG_FILE", filename)
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
