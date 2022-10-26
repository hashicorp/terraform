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
	"github.com/hashicorp/terraform/internal/legacy/helper/schema"
	"github.com/hashicorp/terraform/internal/logging"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/hashicorp/terraform/version"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

// New creates a new backend for S3 remote state.
func New() backend.Backend {
	s := &schema.Backend{
		Schema: map[string]*schema.Schema{
			"bucket": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the S3 bucket",
			},

			"key": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The path to the state file inside the bucket",
			},

			"region": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "AWS region of the S3 Bucket and DynamoDB Table (if used).",
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{
					"AWS_REGION",
					"AWS_DEFAULT_REGION",
				}, nil),
			},

			"dynamodb_endpoint": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A custom endpoint for the DynamoDB API",
				DefaultFunc: schema.EnvDefaultFunc("AWS_DYNAMODB_ENDPOINT", ""),
			},

			"endpoint": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A custom endpoint for the S3 API",
				DefaultFunc: schema.EnvDefaultFunc("AWS_S3_ENDPOINT", ""),
			},

			"iam_endpoint": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A custom endpoint for the IAM API",
				DefaultFunc: schema.EnvDefaultFunc("AWS_IAM_ENDPOINT", ""),
			},

			"sts_endpoint": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A custom endpoint for the STS API",
				DefaultFunc: schema.EnvDefaultFunc("AWS_STS_ENDPOINT", ""),
			},

			"encrypt": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Whether to enable server side encryption of the state file",
				Default:     false,
			},

			"acl": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Canned ACL to be applied to the state file",
				Default:     "",
			},

			"access_key": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "AWS access key",
				Default:     "",
			},

			"secret_key": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "AWS secret key",
				Default:     "",
			},

			"kms_key_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The ARN of a KMS Key to use for encrypting the state",
				Default:     "",
			},

			"dynamodb_table": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "DynamoDB table for state locking and consistency",
				Default:     "",
			},

			"profile": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "AWS profile name",
				Default:     "",
			},

			"shared_credentials_file": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Path to a shared credentials file",
				Default:     "",
			},

			"token": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "MFA token",
				Default:     "",
			},

			"skip_credentials_validation": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Skip the credentials validation via STS API.",
				Default:     false,
			},

			"skip_region_validation": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Skip static validation of region name.",
				Default:     false,
			},

			"skip_metadata_api_check": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Skip the AWS Metadata API check.",
				Default:     false,
			},

			"sse_customer_key": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The base64-encoded encryption key to use for server-side encryption with customer-provided keys (SSE-C).",
				DefaultFunc: schema.EnvDefaultFunc("AWS_SSE_CUSTOMER_KEY", ""),
				Sensitive:   true,
			},

			"role_arn": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The role to be assumed",
				Default:     "",
			},

			"session_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The session name to use when assuming the role.",
				Default:     "",
			},

			"external_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The external ID to use when assuming the role",
				Default:     "",
			},

			"assume_role_duration_seconds": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Seconds to restrict the assume role session duration.",
			},

			"assume_role_policy": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "IAM Policy JSON describing further restricting permissions for the IAM Role being assumed.",
				Default:     "",
			},

			"assume_role_policy_arns": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Amazon Resource Names (ARNs) of IAM Policies describing further restricting permissions for the IAM Role being assumed.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},

			"assume_role_tags": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Assume role session tags.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},

			"assume_role_transitive_tag_keys": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Assume role session tag keys to pass to any subsequent sessions.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},

			"workspace_key_prefix": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The prefix applied to the non-default state path inside the bucket.",
				Default:     "env:",
			},

			"force_path_style": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Force s3 to use path style api.",
				Default:     false,
			},

			"max_retries": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The maximum number of times an AWS API request is retried on retryable failure.",
				Default:     5,
			},
		},
	}

	return &Backend{Backend: s}
}

type Backend struct {
	*schema.Backend

	// The fields below are set from configure
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

func (b *Backend) PrepareConfig(obj cty.Value) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	if obj.IsNull() {
		return obj, diags
	}

	if val := obj.GetAttr("key"); val.IsNull() || val.AsString() == "" {
		diags = diags.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			"Invalid key value",
			`"key": required field is not set`,
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
			"key must not start or end with '/'",
			cty.Path{cty.GetAttrStep{Name: "key"}},
		))
	}

	if val := obj.GetAttr("region"); val.IsNull() {
		if os.Getenv("AWS_REGION") == "" && os.Getenv("AWS_DEFAULT_REGION") == "" {
			diags = diags.Append(tfdiags.AttributeValue(
				tfdiags.Error,
				"Missing region value",
				`"region": required field is not set`,
				cty.Path{cty.GetAttrStep{Name: "region"}},
			))
		}
	}

	if val := obj.GetAttr("sse_customer_key"); !val.IsNull() {
		s := val.AsString()
		if len(s) != 44 {
			diags = diags.Append(tfdiags.AttributeValue(
				tfdiags.Error,
				"Invalid sse_customer_key value",
				"sse_customer_key must be 44 characters in length",
				cty.Path{cty.GetAttrStep{Name: "sse_customer_key"}},
			))
		} else {
			var err error
			_, err = base64.StdEncoding.DecodeString(s)
			if err != nil {
				diags = diags.Append(tfdiags.AttributeValue(
					tfdiags.Error,
					"Invalid sse_customer_key value",
					fmt.Sprintf("sse_customer_key must be base64 encoded: %s", err),
					cty.Path{cty.GetAttrStep{Name: "sse_customer_key"}},
				))
			}
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
		}
	}

	if val := obj.GetAttr("workspace_key_prefix"); !val.IsNull() {
		if v := val.AsString(); strings.HasPrefix(v, "/") || strings.HasSuffix(v, "/") {
			diags = diags.Append(tfdiags.AttributeValue(
				tfdiags.Error,
				"Invalid workspace_key_prefix value",
				"workspace_key_prefix must not start or end with '/'",
				cty.Path{cty.GetAttrStep{Name: "workspace_key_prefix"}},
			))
		}
	}

	return obj, diags
}

func (b *Backend) Configure(obj cty.Value) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	if obj.IsNull() {
		return diags
	}

	var region string
	if v, ok := stringAttrOk(obj, "region"); ok {
		region = v
	}

	if boolAttr(obj, "skip_region_validation") {
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

	if customerKeyString, ok := stringAttrOk(obj, "sse_customer_key"); ok {
		// Validation is handled in PrepareConfig, so ignore it here
		b.customerEncryptionKey, _ = base64.StdEncoding.DecodeString(customerKeyString)
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
		IamEndpoint:               stringAttr(obj, "iam_endpoint"),
		MaxRetries:                intAttrDefault(obj, "max_retries", 5),
		Profile:                   stringAttr(obj, "profile"),
		Region:                    stringAttr(obj, "region"),
		SecretKey:                 stringAttr(obj, "secret_key"),
		SkipCredsValidation:       boolAttr(obj, "skip_credentials_validation"),
		SkipMetadataApiCheck:      boolAttr(obj, "skip_metadata_api_check"),
		StsEndpoint:               stringAttr(obj, "sts_endpoint"),
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

	b.dynClient = dynamodb.New(sess.Copy(&aws.Config{
		Endpoint: aws.String(stringAttr(obj, "dynamodb_endpoint")),
	}))
	b.s3Client = s3.New(sess.Copy(&aws.Config{
		Endpoint:         aws.String(stringAttr(obj, "endpoint")),
		S3ForcePathStyle: aws.Bool(boolAttr(obj, "force_path_style")),
	}))

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

The kms_key_id is used for encryption with KMS-Managed Keys (SSE-KMS)
while sse_customer_key is used for encryption with customer-managed keys (SSE-C).
Please choose one or the other.`
