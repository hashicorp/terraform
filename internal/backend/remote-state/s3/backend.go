// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package s3

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

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
			},
			"session_name": {
				Type:        cty.String,
				Optional:    true,
				Description: "The session name to use when assuming the role.",
			},
			"external_id": {
				Type:        cty.String,
				Optional:    true,
				Description: "The external ID to use when assuming the role",
			},

			"assume_role_duration_seconds": {
				Type:        cty.Number,
				Optional:    true,
				Description: "Seconds to restrict the assume role session duration.",
			},

			"assume_role_policy": {
				Type:        cty.String,
				Optional:    true,
				Description: "IAM Policy JSON describing further restricting permissions for the IAM Role being assumed.",
			},

			"assume_role_policy_arns": {
				Type:        cty.Set(cty.String),
				Optional:    true,
				Description: "Amazon Resource Names (ARNs) of IAM Policies describing further restricting permissions for the IAM Role being assumed.",
			},

			"assume_role_tags": {
				Type:        cty.Map(cty.String),
				Optional:    true,
				Description: "Assume role session tags.",
			},

			"assume_role_transitive_tag_keys": {
				Type:        cty.Set(cty.String),
				Optional:    true,
				Description: "Assume role session tag keys to pass to any subsequent sessions.",
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

	if val := obj.GetAttr("bucket"); val.IsNull() || val.AsString() == "" {
		diags = diags.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			"Invalid bucket value",
			`The "bucket" attribute value must not be empty.`,
			cty.Path{cty.GetAttrStep{Name: "bucket"}},
		))
	}

	if val := obj.GetAttr("key"); val.IsNull() || val.AsString() == "" {
		diags = diags.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			"Invalid key value",
			`The "key" attribute value must not be empty.`,
			cty.Path{cty.GetAttrStep{Name: "key"}},
		))
	} else if strings.HasPrefix(val.AsString(), "/") || strings.HasSuffix(val.AsString(), "/") {
		// S3 will strip leading slashes from an object, so while this will
		// technically be accepted by S3, it will break our workspace hierarchy.
		// S3 will recognize objects with a trailing slash as a directory
		// so they should not be valid keys
		diags = diags.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			"Invalid key value",
			`The "key" attribute value must not start or end with with "/".`,
			cty.Path{cty.GetAttrStep{Name: "key"}},
		))
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

	if val := obj.GetAttr("kms_key_id"); !val.IsNull() && val.AsString() != "" {
		if val := obj.GetAttr("sse_customer_key"); !val.IsNull() && val.AsString() != "" {
			diags = diags.Append(tfdiags.AttributeValue(
				tfdiags.Error,
				"Invalid encryption configuration",
				encryptionKeyConflictError,
				cty.Path{},
			))
		} else if customerKey := os.Getenv("AWS_SSE_CUSTOMER_KEY"); customerKey != "" {
			diags = diags.Append(tfdiags.AttributeValue(
				tfdiags.Error,
				"Invalid encryption configuration",
				encryptionKeyConflictEnvVarError,
				cty.Path{},
			))
		}

		diags = diags.Append(validateKMSKey(cty.Path{cty.GetAttrStep{Name: "kms_key_id"}}, val.AsString()))
	}

	if val := obj.GetAttr("workspace_key_prefix"); !val.IsNull() {
		if v := val.AsString(); strings.HasPrefix(v, "/") || strings.HasSuffix(v, "/") {
			diags = diags.Append(tfdiags.AttributeValue(
				tfdiags.Error,
				"Invalid workspace_key_prefix value",
				`The "workspace_key_prefix" attribute value must not start with "/".`,
				cty.Path{cty.GetAttrStep{Name: "workspace_key_prefix"}},
			))
		}
	}

	return obj, diags
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
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid AWS_SSE_CUSTOMER_KEY value",
				`The environment variable "AWS_SSE_CUSTOMER_KEY" must be 44 characters in length`,
			))
		} else {
			var err error
			if b.customerEncryptionKey, err = base64.StdEncoding.DecodeString(customerKey); err != nil {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid AWS_SSE_CUSTOMER_KEY value",
					fmt.Sprintf(`The environment variable "AWS_SSE_CUSTOMER_KEY" must be base64 encoded: %s`, err),
				))
			}
		}
	}

	cfg := &awsbase.Config{
		AccessKey:                 stringAttr(obj, "access_key"),
		AssumeRoleARN:             stringAttr(obj, "role_arn"),
		AssumeRoleDurationSeconds: intAttr(obj, "assume_role_duration_seconds"),
		AssumeRoleExternalID:      stringAttr(obj, "external_id"),
		AssumeRolePolicy:          stringAttr(obj, "assume_role_policy"),
		AssumeRoleSessionName:     stringAttr(obj, "session_name"),
		CallerDocumentationURL:    "https://www.terraform.io/docs/language/settings/backends/s3.html",
		CallerName:                "S3 Backend",
		CredsFilename:             stringAttr(obj, "shared_credentials_file"),
		DebugLogging:              logging.IsDebugOrHigher(),
		IamEndpoint:               stringAttrDefaultEnvVar(obj, "iam_endpoint", "AWS_IAM_ENDPOINT"),
		MaxRetries:                intAttrDefault(obj, "max_retries", 5),
		Profile:                   stringAttr(obj, "profile"),
		Region:                    stringAttr(obj, "region"),
		SecretKey:                 stringAttr(obj, "secret_key"),
		SkipCredsValidation:       boolAttr(obj, "skip_credentials_validation"),
		SkipMetadataApiCheck:      boolAttr(obj, "skip_metadata_api_check"),
		StsEndpoint:               stringAttrDefaultEnvVar(obj, "sts_endpoint", "AWS_STS_ENDPOINT"),
		Token:                     stringAttr(obj, "token"),
		UserAgentProducts: []*awsbase.UserAgentProduct{
			{Name: "APN", Version: "1.0"},
			{Name: "HashiCorp", Version: "1.0"},
			{Name: "Terraform", Version: version.String()},
		},
	}

	if policyARNSet := obj.GetAttr("assume_role_policy_arns"); !policyARNSet.IsNull() {
		policyARNSet.ForEachElement(func(key, val cty.Value) (stop bool) {
			v, ok := stringValueOk(val)
			if ok {
				cfg.AssumeRolePolicyARNs = append(cfg.AssumeRolePolicyARNs, v)
			}
			return
		})
	}

	if tagMap := obj.GetAttr("assume_role_tags"); !tagMap.IsNull() {
		cfg.AssumeRoleTags = make(map[string]string, tagMap.LengthInt())
		tagMap.ForEachElement(func(key, val cty.Value) (stop bool) {
			k := stringValue(key)
			v, ok := stringValueOk(val)
			if ok {
				cfg.AssumeRoleTags[k] = v
			}
			return
		})
	}

	if transitiveTagKeySet := obj.GetAttr("assume_role_transitive_tag_keys"); !transitiveTagKeySet.IsNull() {
		transitiveTagKeySet.ForEachElement(func(key, val cty.Value) (stop bool) {
			v, ok := stringValueOk(val)
			if ok {
				cfg.AssumeRoleTransitiveTagKeys = append(cfg.AssumeRoleTransitiveTagKeys, v)
			}
			return
		})
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

const encryptionKeyConflictError = `Only one of "kms_key_id" and "sse_customer_key" can be set.

The "kms_key_id" is used for encryption with KMS-Managed Keys (SSE-KMS)
while "sse_customer_key" is used for encryption with customer-managed keys (SSE-C).
Please choose one or the other.`

const encryptionKeyConflictEnvVarError = `Only one of "kms_key_id" and the environment variable "AWS_SSE_CUSTOMER_KEY" can be set.

The "kms_key_id" is used for encryption with KMS-Managed Keys (SSE-KMS)
while "AWS_SSE_CUSTOMER_KEY" is used for encryption with customer-managed keys (SSE-C).
Please choose one or the other.`
