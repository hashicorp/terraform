// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package s3

import (
	"encoding/base64"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/s3"
	awsbase "github.com/hashicorp/aws-sdk-go-base"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/logging"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/hashicorp/terraform/version"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

func New() backend.Backend {
	return &Backend{}
}

type Backend struct {
	s3Client  *s3.S3
	dynClient *dynamodb.DynamoDB

	bucketName            string
	keyName               string
	serverSideEncryption  bool
	customerEncryptionKey []byte
	acl                   string
	kmsKeyID              string
	ddbTable              string
	workspaceKeyPrefix    string
}

// ConfigSchema returns a description of the expected configuration
// structure for the receiving backend.
func (b *Backend) ConfigSchema() *configschema.Block {
	return &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"bucket": {
				Type:        cty.String,
				Required:    true,
				Description: "The name of the S3 bucket",
			},
			"key": {
				Type:        cty.String,
				Required:    true,
				Description: "The path to the state file inside the bucket",
			},
			"region": {
				Type:        cty.String,
				Optional:    true,
				Description: "AWS region of the S3 Bucket and DynamoDB Table (if used).",
			},
			"dynamodb_endpoint": {
				Type:        cty.String,
				Optional:    true,
				Description: "A custom endpoint for the DynamoDB API",
			},
			"endpoint": {
				Type:        cty.String,
				Optional:    true,
				Description: "A custom endpoint for the S3 API",
			},
			"iam_endpoint": {
				Type:        cty.String,
				Optional:    true,
				Description: "A custom endpoint for the IAM API",
			},
			"sts_endpoint": {
				Type:        cty.String,
				Optional:    true,
				Description: "A custom endpoint for the STS API",
			},
			"encrypt": {
				Type:        cty.Bool,
				Optional:    true,
				Description: "Whether to enable server side encryption of the state file",
			},
			"acl": {
				Type:        cty.String,
				Optional:    true,
				Description: "Canned ACL to be applied to the state file",
			},
			"access_key": {
				Type:        cty.String,
				Optional:    true,
				Description: "AWS access key",
			},
			"secret_key": {
				Type:        cty.String,
				Optional:    true,
				Description: "AWS secret key",
			},
			"kms_key_id": {
				Type:        cty.String,
				Optional:    true,
				Description: "The ARN of a KMS Key to use for encrypting the state",
			},
			"dynamodb_table": {
				Type:        cty.String,
				Optional:    true,
				Description: "DynamoDB table for state locking and consistency",
			},
			"profile": {
				Type:        cty.String,
				Optional:    true,
				Description: "AWS profile name",
			},
			"shared_credentials_file": {
				Type:        cty.String,
				Optional:    true,
				Description: "Path to a shared credentials file",
			},
			"token": {
				Type:        cty.String,
				Optional:    true,
				Description: "MFA token",
			},
			"skip_credentials_validation": {
				Type:        cty.Bool,
				Optional:    true,
				Description: "Skip the credentials validation via STS API.",
			},
			"skip_metadata_api_check": {
				Type:        cty.Bool,
				Optional:    true,
				Description: "Skip the AWS Metadata API check.",
			},
			"skip_region_validation": {
				Type:        cty.Bool,
				Optional:    true,
				Description: "Skip static validation of region name.",
			},
			"sse_customer_key": {
				Type:        cty.String,
				Optional:    true,
				Description: "The base64-encoded encryption key to use for server-side encryption with customer-provided keys (SSE-C).",
				Sensitive:   true,
			},
			"role_arn": {
				Type:        cty.String,
				Optional:    true,
				Description: "The role to be assumed",
				Deprecated:  true,
			},
			"session_name": {
				Type:        cty.String,
				Optional:    true,
				Description: "The session name to use when assuming the role.",
				Deprecated:  true,
			},
			"external_id": {
				Type:        cty.String,
				Optional:    true,
				Description: "The external ID to use when assuming the role",
				Deprecated:  true,
			},

			"assume_role_duration_seconds": {
				Type:        cty.Number,
				Optional:    true,
				Description: "Seconds to restrict the assume role session duration.",
				Deprecated:  true,
			},

			"assume_role_policy": {
				Type:        cty.String,
				Optional:    true,
				Description: "IAM Policy JSON describing further restricting permissions for the IAM Role being assumed.",
				Deprecated:  true,
			},

			"assume_role_policy_arns": {
				Type:        cty.Set(cty.String),
				Optional:    true,
				Description: "Amazon Resource Names (ARNs) of IAM Policies describing further restricting permissions for the IAM Role being assumed.",
				Deprecated:  true,
			},

			"assume_role_tags": {
				Type:        cty.Map(cty.String),
				Optional:    true,
				Description: "Assume role session tags.",
				Deprecated:  true,
			},

			"assume_role_transitive_tag_keys": {
				Type:        cty.Set(cty.String),
				Optional:    true,
				Description: "Assume role session tag keys to pass to any subsequent sessions.",
				Deprecated:  true,
			},

			"workspace_key_prefix": {
				Type:        cty.String,
				Optional:    true,
				Description: "The prefix applied to the non-default state path inside the bucket.",
			},

			"force_path_style": {
				Type:        cty.Bool,
				Optional:    true,
				Description: "Force s3 to use path style api.",
			},

			"max_retries": {
				Type:        cty.Number,
				Optional:    true,
				Description: "The maximum number of times an AWS API request is retried on retryable failure.",
			},

			"assume_role": {
				NestedType: &configschema.Object{
					Nesting:    configschema.NestingSingle,
					Attributes: assumeRoleFullSchema().SchemaAttributes(),
				},
			},
		},
	}
}

// PrepareConfig checks the validity of the values in the given
// configuration, and inserts any missing defaults, assuming that its
// structure has already been validated per the schema returned by
// ConfigSchema.
func (b *Backend) PrepareConfig(obj cty.Value) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	if obj.IsNull() {
		return obj, diags
	}

	var attrPath cty.Path

	attrPath = cty.Path{cty.GetAttrStep{Name: "bucket"}}
	if val := obj.GetAttr("bucket"); val.IsNull() {
		diags = diags.Append(requiredAttributeErrDiag(attrPath))
	} else {
		bucketValidators := validateString{
			Validators: []stringValidator{
				validateStringNotEmpty,
			},
		}
		bucketValidators.ValidateAttr(val, attrPath, &diags)
	}

	attrPath = cty.Path{cty.GetAttrStep{Name: "key"}}
	if val := obj.GetAttr("key"); val.IsNull() {
		diags = diags.Append(requiredAttributeErrDiag(attrPath))
	} else {
		keyValidators := validateString{
			Validators: []stringValidator{
				validateStringNotEmpty,
				validateStringS3Path,
			},
		}
		keyValidators.ValidateAttr(val, attrPath, &diags)
	}

	if val := obj.GetAttr("region"); val.IsNull() || val.AsString() == "" {
		if os.Getenv("AWS_REGION") == "" && os.Getenv("AWS_DEFAULT_REGION") == "" {
			diags = diags.Append(tfdiags.AttributeValue(
				tfdiags.Error,
				"Missing region value",
				`The "region" attribute or the "AWS_REGION" or "AWS_DEFAULT_REGION" environment variables must be set.`,
				cty.Path{cty.GetAttrStep{Name: "region"}},
			))
		}
	}

	validateAttributesConflict(
		cty.GetAttrPath("kms_key_id"),
		cty.GetAttrPath("sse_customer_key"),
	)(obj, cty.Path{}, &diags)

	if val := obj.GetAttr("kms_key_id"); !val.IsNull() && val.AsString() != "" {
		if customerKey := os.Getenv("AWS_SSE_CUSTOMER_KEY"); customerKey != "" {
			diags = diags.Append(wholeBodyErrDiag(
				"Invalid encryption configuration",
				encryptionKeyConflictEnvVarError,
			))
		}

		diags = diags.Append(validateKMSKey(cty.Path{cty.GetAttrStep{Name: "kms_key_id"}}, val.AsString()))
	}

	attrPath = cty.Path{cty.GetAttrStep{Name: "workspace_key_prefix"}}
	if val := obj.GetAttr("workspace_key_prefix"); !val.IsNull() {
		keyPrefixValidators := validateString{
			Validators: []stringValidator{
				validateStringS3Path,
			},
		}
		keyPrefixValidators.ValidateAttr(val, attrPath, &diags)
	}

	var assumeRoleDeprecatedFields = map[string]string{
		"role_arn":                        "assume_role.role_arn",
		"session_name":                    "assume_role.session_name",
		"external_id":                     "assume_role.external_id",
		"assume_role_duration_seconds":    "assume_role.duration",
		"assume_role_policy":              "assume_role.policy",
		"assume_role_policy_arns":         "assume_role.policy_arns",
		"assume_role_tags":                "assume_role.tags",
		"assume_role_transitive_tag_keys": "assume_role.transitive_tag_keys",
	}

	if val := obj.GetAttr("assume_role"); !val.IsNull() {
		diags = diags.Append(prepareAssumeRoleConfig(val, cty.Path{cty.GetAttrStep{Name: "assume_role"}}))

		if defined := findDeprecatedFields(obj, assumeRoleDeprecatedFields); len(defined) != 0 {
			diags = diags.Append(tfdiags.WholeContainingBody(
				tfdiags.Error,
				"Conflicting Parameters",
				`The following deprecated parameters conflict with the parameter "assume_role". Replace them as follows:`+"\n"+
					formatDeprecations(defined),
			))
		}
	} else {
		if defined := findDeprecatedFields(obj, assumeRoleDeprecatedFields); len(defined) != 0 {
			diags = diags.Append(tfdiags.WholeContainingBody(
				tfdiags.Warning,
				"Deprecated Parameters",
				`The following parameters have been deprecated. Replace them as follows:`+"\n"+
					formatDeprecations(defined),
			))
		}
	}

	return obj, diags
}

func findDeprecatedFields(obj cty.Value, attrs map[string]string) map[string]string {
	defined := make(map[string]string)
	for attr, v := range attrs {
		if val := obj.GetAttr(attr); !val.IsNull() {
			defined[attr] = v
		}
	}
	return defined
}

func formatDeprecations(attrs map[string]string) string {
	names := make([]string, 0, len(attrs))
	var maxLen int
	for attr := range attrs {
		names = append(names, attr)
		if l := len(attr); l > maxLen {
			maxLen = l
		}
	}
	sort.Strings(names)

	var buf strings.Builder

	for _, attr := range names {
		replacement := attrs[attr]
		fmt.Fprintf(&buf, "  * %-[1]*[2]s -> %[3]s\n", maxLen, attr, replacement)
	}
	return buf.String()
}

// Configure uses the provided configuration to set configuration fields
// within the backend.
//
// The given configuration is assumed to have already been validated
// against the schema returned by ConfigSchema and passed validation
// via PrepareConfig.
func (b *Backend) Configure(obj cty.Value) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	if obj.IsNull() {
		return diags
	}

	var region string
	if v, ok := stringAttrOk(obj, "region"); ok {
		region = v
	}

	if region != "" && !boolAttr(obj, "skip_region_validation") {
		if err := awsbase.ValidateRegion(region); err != nil {
			diags = diags.Append(tfdiags.AttributeValue(
				tfdiags.Error,
				"Invalid region value",
				err.Error(),
				cty.Path{cty.GetAttrStep{Name: "region"}},
			))
			return diags
		}
	}

	b.bucketName = stringAttr(obj, "bucket")
	b.keyName = stringAttr(obj, "key")
	b.acl = stringAttr(obj, "acl")
	b.workspaceKeyPrefix = stringAttrDefault(obj, "workspace_key_prefix", "env:")
	b.serverSideEncryption = boolAttr(obj, "encrypt")
	b.kmsKeyID = stringAttr(obj, "kms_key_id")
	b.ddbTable = stringAttr(obj, "dynamodb_table")

	if customerKey, ok := stringAttrOk(obj, "sse_customer_key"); ok {
		if len(customerKey) != 44 {
			diags = diags.Append(tfdiags.AttributeValue(
				tfdiags.Error,
				"Invalid sse_customer_key value",
				"sse_customer_key must be 44 characters in length",
				cty.Path{cty.GetAttrStep{Name: "sse_customer_key"}},
			))
		} else {
			var err error
			if b.customerEncryptionKey, err = base64.StdEncoding.DecodeString(customerKey); err != nil {
				diags = diags.Append(tfdiags.AttributeValue(
					tfdiags.Error,
					"Invalid sse_customer_key value",
					fmt.Sprintf("sse_customer_key must be base64 encoded: %s", err),
					cty.Path{cty.GetAttrStep{Name: "sse_customer_key"}},
				))
			}
		}
	} else if customerKey := os.Getenv("AWS_SSE_CUSTOMER_KEY"); customerKey != "" {
		if len(customerKey) != 44 {
			diags = diags.Append(tfdiags.WholeContainingBody(
				tfdiags.Error,
				"Invalid AWS_SSE_CUSTOMER_KEY value",
				`The environment variable "AWS_SSE_CUSTOMER_KEY" must be 44 characters in length`,
			))
		} else {
			var err error
			if b.customerEncryptionKey, err = base64.StdEncoding.DecodeString(customerKey); err != nil {
				diags = diags.Append(tfdiags.WholeContainingBody(
					tfdiags.Error,
					"Invalid AWS_SSE_CUSTOMER_KEY value",
					fmt.Sprintf(`The environment variable "AWS_SSE_CUSTOMER_KEY" must be base64 encoded: %s`, err),
				))
			}
		}
	}

	cfg := &awsbase.Config{
		AccessKey:              stringAttr(obj, "access_key"),
		CallerDocumentationURL: "https://www.terraform.io/docs/language/settings/backends/s3.html",
		CallerName:             "S3 Backend",
		CredsFilename:          stringAttr(obj, "shared_credentials_file"),
		DebugLogging:           logging.IsDebugOrHigher(),
		IamEndpoint:            stringAttrDefaultEnvVar(obj, "iam_endpoint", "AWS_IAM_ENDPOINT"),
		MaxRetries:             intAttrDefault(obj, "max_retries", 5),
		Profile:                stringAttr(obj, "profile"),
		Region:                 stringAttr(obj, "region"),
		SecretKey:              stringAttr(obj, "secret_key"),
		SkipCredsValidation:    boolAttr(obj, "skip_credentials_validation"),
		SkipMetadataApiCheck:   boolAttr(obj, "skip_metadata_api_check"),
		StsEndpoint:            stringAttrDefaultEnvVar(obj, "sts_endpoint", "AWS_STS_ENDPOINT"),
		Token:                  stringAttr(obj, "token"),
		UserAgentProducts: []*awsbase.UserAgentProduct{
			{Name: "APN", Version: "1.0"},
			{Name: "HashiCorp", Version: "1.0"},
			{Name: "Terraform", Version: version.String()},
		},
	}

	if assumeRole := obj.GetAttr("assume_role"); !assumeRole.IsNull() {
		if val, ok := stringAttrOk(assumeRole, "role_arn"); ok {
			cfg.AssumeRoleARN = val
		}
		if val, ok := stringAttrOk(assumeRole, "duration"); ok {
			duration, _ := time.ParseDuration(val)
			cfg.AssumeRoleDurationSeconds = int(duration.Seconds())
		}
		if val, ok := stringAttrOk(assumeRole, "external_id"); ok {
			cfg.AssumeRoleExternalID = val
		}
		if val, ok := stringAttrOk(assumeRole, "policy"); ok {
			cfg.AssumeRolePolicy = strings.TrimSpace(val)
		}
		if val, ok := stringSetAttrOk(assumeRole, "policy_arns"); ok {
			cfg.AssumeRolePolicyARNs = val
		}
		if val, ok := stringAttrOk(assumeRole, "session_name"); ok {
			cfg.AssumeRoleSessionName = val
		}
		if val, ok := stringMapAttrOk(assumeRole, "tags"); ok {
			cfg.AssumeRoleTags = val
		}
		if val, ok := stringSetAttrOk(assumeRole, "transitive_tag_keys"); ok {
			cfg.AssumeRoleTransitiveTagKeys = val
		}
	} else {
		cfg.AssumeRoleARN = stringAttr(obj, "role_arn")
		cfg.AssumeRoleSessionName = stringAttr(obj, "session_name")
		cfg.AssumeRoleDurationSeconds = intAttr(obj, "assume_role_duration_seconds")
		cfg.AssumeRoleExternalID = stringAttr(obj, "external_id")
		if val, ok := stringAttrOk(obj, "assume_role_policy"); ok {
			cfg.AssumeRolePolicy = strings.TrimSpace(val)
		}
		if val, ok := stringSetAttrOk(obj, "assume_role_policy_arns"); ok {
			cfg.AssumeRolePolicyARNs = val
		}

		if val, ok := stringMapAttrOk(obj, "assume_role_tags"); ok {
			cfg.AssumeRoleTags = val
		}

		if val, ok := stringSetAttrOk(obj, "assume_role_transitive_tag_keys"); ok {
			cfg.AssumeRoleTransitiveTagKeys = val
		}
	}

	sess, err := awsbase.GetSession(cfg)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to configure AWS client",
			fmt.Sprintf(`The "S3" backend encountered an unexpected error while creating the AWS client: %s`, err),
		))
		return diags
	}

	var dynamoConfig aws.Config
	if v, ok := stringAttrDefaultEnvVarOk(obj, "dynamodb_endpoint", "AWS_DYNAMODB_ENDPOINT"); ok {
		dynamoConfig.Endpoint = aws.String(v)
	}
	b.dynClient = dynamodb.New(sess.Copy(&dynamoConfig))

	var s3Config aws.Config
	if v, ok := stringAttrDefaultEnvVarOk(obj, "endpoint", "AWS_S3_ENDPOINT"); ok {
		s3Config.Endpoint = aws.String(v)
	}
	if v, ok := boolAttrOk(obj, "force_path_style"); ok {
		s3Config.S3ForcePathStyle = aws.Bool(v)
	}
	b.s3Client = s3.New(sess.Copy(&s3Config))

	return diags
}

func stringValue(val cty.Value) string {
	v, _ := stringValueOk(val)
	return v
}

func stringValueOk(val cty.Value) (string, bool) {
	if val.IsNull() {
		return "", false
	} else {
		return val.AsString(), true
	}
}

func stringAttr(obj cty.Value, name string) string {
	return stringValue(obj.GetAttr(name))
}

func stringAttrOk(obj cty.Value, name string) (string, bool) {
	return stringValueOk(obj.GetAttr(name))
}

func stringAttrDefault(obj cty.Value, name, def string) string {
	if v, ok := stringAttrOk(obj, name); !ok {
		return def
	} else {
		return v
	}
}

func stringAttrDefaultEnvVar(obj cty.Value, name string, envvars ...string) string {
	if v, ok := stringAttrDefaultEnvVarOk(obj, name, envvars...); !ok {
		return ""
	} else {
		return v
	}
}

func stringAttrDefaultEnvVarOk(obj cty.Value, name string, envvars ...string) (string, bool) {
	if v, ok := stringAttrOk(obj, name); !ok {
		for _, envvar := range envvars {
			if v := os.Getenv(envvar); v != "" {
				return v, true
			}
		}
		return "", false
	} else {
		return v, true
	}
}

func stringSetValueOk(val cty.Value) ([]string, bool) {
	var list []string
	typ := val.Type()
	if !typ.IsSetType() {
		return nil, false
	}
	err := gocty.FromCtyValue(val, &list)
	if err != nil {
		return nil, false
	}
	return list, true
}

func stringSetAttrOk(obj cty.Value, name string) ([]string, bool) {
	return stringSetValueOk(obj.GetAttr(name))
}

func stringMapValueOk(val cty.Value) (map[string]string, bool) {
	var m map[string]string
	err := gocty.FromCtyValue(val, &m)
	if err != nil {
		return nil, false
	}
	return m, true
}

func stringMapAttrOk(obj cty.Value, name string) (map[string]string, bool) {
	return stringMapValueOk(obj.GetAttr(name))
}

func boolAttr(obj cty.Value, name string) bool {
	v, _ := boolAttrOk(obj, name)
	return v
}

func boolAttrOk(obj cty.Value, name string) (bool, bool) {
	if val := obj.GetAttr(name); val.IsNull() {
		return false, false
	} else {
		return val.True(), true
	}
}

func intAttr(obj cty.Value, name string) int {
	v, _ := intAttrOk(obj, name)
	return v
}

func intAttrOk(obj cty.Value, name string) (int, bool) {
	if val := obj.GetAttr(name); val.IsNull() {
		return 0, false
	} else {
		var v int
		if err := gocty.FromCtyValue(val, &v); err != nil {
			return 0, false
		}
		return v, true
	}
}

func intAttrDefault(obj cty.Value, name string, def int) int {
	if v, ok := intAttrOk(obj, name); !ok {
		return def
	} else {
		return v
	}
}

const encryptionKeyConflictEnvVarError = `Only one of "kms_key_id" and the environment variable "AWS_SSE_CUSTOMER_KEY" can be set.

The "kms_key_id" is used for encryption with KMS-Managed Keys (SSE-KMS)
while "AWS_SSE_CUSTOMER_KEY" is used for encryption with customer-managed keys (SSE-C).
Please choose one or the other.`

func prepareAssumeRoleConfig(obj cty.Value, objPath cty.Path) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	if obj.IsNull() {
		return diags
	}

	for name, attrSchema := range assumeRoleFullSchema() {
		attrPath := objPath.GetAttr(name)
		attrVal := obj.GetAttr(name)

		if a, e := attrVal.Type(), attrSchema.SchemaAttribute().Type; a != e {
			diags = diags.Append(attributeErrDiag(
				"Internal Error",
				fmt.Sprintf(`Expected type to be %s, got: %s`, e.FriendlyName(), a.FriendlyName()),
				attrPath,
			))
			continue
		}

		if attrVal.IsNull() {
			if attrSchema.SchemaAttribute().Required {
				diags = diags.Append(requiredAttributeErrDiag(attrPath))
			}
			continue
		}

		validator := attrSchema.Validator()
		validator.ValidateAttr(attrVal, attrPath, &diags)
	}

	return diags
}

func requiredAttributeErrDiag(path cty.Path) tfdiags.Diagnostic {
	return attributeErrDiag(
		"Missing Required Value",
		fmt.Sprintf("The attribute %q is required by the backend.\n\n", pathString(path))+
			"Refer to the backend documentation for additional information which attributes are required.",
		path,
	)
}

func pathString(path cty.Path) string {
	var buf strings.Builder
	for i, step := range path {
		switch x := step.(type) {
		case cty.GetAttrStep:
			if i != 0 {
				buf.WriteString(".")
			}
			buf.WriteString(x.Name)
		case cty.IndexStep:
			val := x.Key
			typ := val.Type()
			var s string
			switch {
			case typ == cty.String:
				s = val.AsString()
			case typ == cty.Number:
				num := val.AsBigFloat()
				s = num.String()
			default:
				s = fmt.Sprintf("<unexpected index: %s>", typ.FriendlyName())
			}
			buf.WriteString(fmt.Sprintf("[%s]", s))
		default:
			if i != 0 {
				buf.WriteString(".")
			}
			buf.WriteString(fmt.Sprintf("<unexpected step: %[1]T %[1]v>", x))
		}
	}
	return buf.String()
}

type validateSchema interface {
	ValidateAttr(cty.Value, cty.Path, *tfdiags.Diagnostics)
}

type validateString struct {
	Validators []stringValidator
}

func (v validateString) ValidateAttr(val cty.Value, attrPath cty.Path, diags *tfdiags.Diagnostics) {
	s := val.AsString()
	for _, validator := range v.Validators {
		validator(s, attrPath, diags)
		if diags.HasErrors() {
			return
		}
	}
}

type validateMap struct{}

func (v validateMap) ValidateAttr(val cty.Value, attrPath cty.Path, diags *tfdiags.Diagnostics) {}

type validateSet struct {
	Validators []setValidator
}

func (v validateSet) ValidateAttr(val cty.Value, attrPath cty.Path, diags *tfdiags.Diagnostics) {
	for _, validator := range v.Validators {
		validator(val, attrPath, diags)
		if diags.HasErrors() {
			return
		}
	}
}

type schemaAttribute interface {
	SchemaAttribute() *configschema.Attribute
	Validator() validateSchema
}

type stringAttribute struct {
	configschema.Attribute
	validateString
}

func (a stringAttribute) SchemaAttribute() *configschema.Attribute {
	return &a.Attribute
}

func (a stringAttribute) Validator() validateSchema {
	return a.validateString
}

type setAttribute struct {
	configschema.Attribute
	validateSet
}

func (a setAttribute) SchemaAttribute() *configschema.Attribute {
	return &a.Attribute
}

func (a setAttribute) Validator() validateSchema {
	return a.validateSet
}

type mapAttribute struct {
	configschema.Attribute
	validateMap
}

func (a mapAttribute) SchemaAttribute() *configschema.Attribute {
	return &a.Attribute
}

func (a mapAttribute) Validator() validateSchema {
	return a.validateMap
}

type objectSchema map[string]schemaAttribute

func (s objectSchema) SchemaAttributes() map[string]*configschema.Attribute {
	m := make(map[string]*configschema.Attribute, len(s))
	for k, v := range s {
		m[k] = v.SchemaAttribute()
	}
	return m
}

func assumeRoleFullSchema() objectSchema {
	return map[string]schemaAttribute{
		"role_arn": stringAttribute{
			configschema.Attribute{
				Type:        cty.String,
				Required:    true,
				Description: "The role to be assumed.",
			},
			validateString{
				Validators: []stringValidator{
					validateARN(
						validateIAMRoleARN,
					),
				},
			},
		},

		"duration": stringAttribute{
			configschema.Attribute{
				Type:        cty.String,
				Optional:    true,
				Description: "The duration, between 15 minutes and 12 hours, of the role session. Valid time units are ns, us (or µs), ms, s, h, or m.",
			},
			validateString{
				Validators: []stringValidator{
					validateDuration(
						validateDurationBetween(15*time.Minute, 12*time.Hour),
					),
				},
			},
		},

		"external_id": stringAttribute{
			configschema.Attribute{
				Type:        cty.String,
				Optional:    true,
				Description: "The external ID to use when assuming the role",
			},
			validateString{
				Validators: []stringValidator{
					validateStringLenBetween(2, 1224),
					validateStringMatches(
						regexp.MustCompile(`^[\w+=,.@:\/\-]*$`),
						`Value can only contain letters, numbers, or the following characters: =,.@/-`,
					),
				},
			},
		},

		"policy": stringAttribute{
			configschema.Attribute{
				Type:        cty.String,
				Optional:    true,
				Description: "IAM Policy JSON describing further restricting permissions for the IAM Role being assumed.",
			},
			validateString{
				Validators: []stringValidator{
					validateStringNotEmpty,
					validateIAMPolicyDocument,
				},
			},
		},

		"policy_arns": setAttribute{
			configschema.Attribute{
				Type:        cty.Set(cty.String),
				Optional:    true,
				Description: "Amazon Resource Names (ARNs) of IAM Policies describing further restricting permissions for the IAM Role being assumed.",
			},
			validateSet{
				Validators: []setValidator{
					validateSetStringElements(
						validateARN(
							validateIAMPolicyARN,
						),
					),
				},
			},
		},

		"session_name": stringAttribute{
			configschema.Attribute{
				Type:        cty.String,
				Optional:    true,
				Description: "The session name to use when assuming the role.",
			},
			validateString{
				Validators: []stringValidator{
					validateStringLenBetween(2, 64),
					validateStringMatches(
						regexp.MustCompile(`^[\w+=,.@\-]*$`),
						`Value can only contain letters, numbers, or the following characters: =,.@-`,
					),
				},
			},
		},

		// NOT SUPPORTED by `aws-sdk-go-base/v1`
		// "source_identity": stringAttribute{
		// 	configschema.Attribute{
		// 		Type:         cty.String,
		// 		Optional:     true,
		// 		Description:  "Source identity specified by the principal assuming the role.",
		// 		ValidateFunc: validAssumeRoleSourceIdentity,
		// 	},
		// },

		"tags": mapAttribute{
			configschema.Attribute{
				Type:        cty.Map(cty.String),
				Optional:    true,
				Description: "Assume role session tags.",
			},
			validateMap{},
		},

		"transitive_tag_keys": setAttribute{
			configschema.Attribute{
				Type:        cty.Set(cty.String),
				Optional:    true,
				Description: "Assume role session tag keys to pass to any subsequent sessions.",
			},
			validateSet{},
		},
	}
}
