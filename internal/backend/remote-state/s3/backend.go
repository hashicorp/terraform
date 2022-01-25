package s3

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/s3"
	awsbase "github.com/hashicorp/aws-sdk-go-base"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/legacy/helper/schema"
	"github.com/hashicorp/terraform/internal/logging"
	"github.com/hashicorp/terraform/version"
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
				ValidateFunc: func(v interface{}, s string) ([]string, []error) {
					// s3 will strip leading slashes from an object, so while this will
					// technically be accepted by s3, it will break our workspace hierarchy.
					if strings.HasPrefix(v.(string), "/") {
						return nil, []error{errors.New("key must not start with '/'")}
					}
					return nil, nil
				},
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
				ValidateFunc: func(v interface{}, s string) ([]string, []error) {
					key := v.(string)
					if key != "" && len(key) != 44 {
						return nil, []error{errors.New("sse_customer_key must be 44 characters in length (256 bits, base64 encoded)")}
					}
					return nil, nil
				},
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
				ValidateFunc: func(v interface{}, s string) ([]string, []error) {
					prefix := v.(string)
					if strings.HasPrefix(prefix, "/") || strings.HasSuffix(prefix, "/") {
						return nil, []error{errors.New("workspace_key_prefix must not start or end with '/'")}
					}
					return nil, nil
				},
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

	result := &Backend{Backend: s}
	result.Backend.ConfigureFunc = result.configure
	return result
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

func (b *Backend) configure(ctx context.Context) error {
	if b.s3Client != nil {
		return nil
	}

	// Grab the resource data
	data := schema.FromContextBackendConfig(ctx)

	if !data.Get("skip_region_validation").(bool) {
		if err := awsbase.ValidateRegion(data.Get("region").(string)); err != nil {
			return err
		}
	}

	b.bucketName = data.Get("bucket").(string)
	b.keyName = data.Get("key").(string)
	b.acl = data.Get("acl").(string)
	b.workspaceKeyPrefix = data.Get("workspace_key_prefix").(string)
	b.serverSideEncryption = data.Get("encrypt").(bool)
	b.kmsKeyID = data.Get("kms_key_id").(string)
	b.ddbTable = data.Get("dynamodb_table").(string)

	customerKeyString := data.Get("sse_customer_key").(string)
	if customerKeyString != "" {
		if b.kmsKeyID != "" {
			return errors.New(encryptionKeyConflictError)
		}

		var err error
		b.customerEncryptionKey, err = base64.StdEncoding.DecodeString(customerKeyString)
		if err != nil {
			return fmt.Errorf("Failed to decode sse_customer_key: %s", err.Error())
		}
	}

	cfg := &awsbase.Config{
		AccessKey:                 data.Get("access_key").(string),
		AssumeRoleARN:             data.Get("role_arn").(string),
		AssumeRoleDurationSeconds: data.Get("assume_role_duration_seconds").(int),
		AssumeRoleExternalID:      data.Get("external_id").(string),
		AssumeRolePolicy:          data.Get("assume_role_policy").(string),
		AssumeRoleSessionName:     data.Get("session_name").(string),
		CallerDocumentationURL:    "https://www.terraform.io/docs/language/settings/backends/s3.html",
		CallerName:                "S3 Backend",
		CredsFilename:             data.Get("shared_credentials_file").(string),
		DebugLogging:              logging.IsDebugOrHigher(),
		IamEndpoint:               data.Get("iam_endpoint").(string),
		MaxRetries:                data.Get("max_retries").(int),
		Profile:                   data.Get("profile").(string),
		Region:                    data.Get("region").(string),
		SecretKey:                 data.Get("secret_key").(string),
		SkipCredsValidation:       data.Get("skip_credentials_validation").(bool),
		SkipMetadataApiCheck:      data.Get("skip_metadata_api_check").(bool),
		StsEndpoint:               data.Get("sts_endpoint").(string),
		Token:                     data.Get("token").(string),
		UserAgentProducts: []*awsbase.UserAgentProduct{
			{Name: "APN", Version: "1.0"},
			{Name: "HashiCorp", Version: "1.0"},
			{Name: "Terraform", Version: version.String()},
		},
	}

	if policyARNSet := data.Get("assume_role_policy_arns").(*schema.Set); policyARNSet.Len() > 0 {
		for _, policyARNRaw := range policyARNSet.List() {
			policyARN, ok := policyARNRaw.(string)

			if !ok {
				continue
			}

			cfg.AssumeRolePolicyARNs = append(cfg.AssumeRolePolicyARNs, policyARN)
		}
	}

	if tagMap := data.Get("assume_role_tags").(map[string]interface{}); len(tagMap) > 0 {
		cfg.AssumeRoleTags = make(map[string]string)

		for k, vRaw := range tagMap {
			v, ok := vRaw.(string)

			if !ok {
				continue
			}

			cfg.AssumeRoleTags[k] = v
		}
	}

	if transitiveTagKeySet := data.Get("assume_role_transitive_tag_keys").(*schema.Set); transitiveTagKeySet.Len() > 0 {
		for _, transitiveTagKeyRaw := range transitiveTagKeySet.List() {
			transitiveTagKey, ok := transitiveTagKeyRaw.(string)

			if !ok {
				continue
			}

			cfg.AssumeRoleTransitiveTagKeys = append(cfg.AssumeRoleTransitiveTagKeys, transitiveTagKey)
		}
	}

	sess, err := awsbase.GetSession(cfg)
	if err != nil {
		return fmt.Errorf("error configuring S3 Backend: %w", err)
	}

	b.dynClient = dynamodb.New(sess.Copy(&aws.Config{
		Endpoint: aws.String(data.Get("dynamodb_endpoint").(string)),
	}))
	b.s3Client = s3.New(sess.Copy(&aws.Config{
		Endpoint:         aws.String(data.Get("endpoint").(string)),
		S3ForcePathStyle: aws.Bool(data.Get("force_path_style").(bool)),
	}))

	return nil
}

const encryptionKeyConflictError = `Cannot have both kms_key_id and sse_customer_key set.

The kms_key_id is used for encryption with KMS-Managed Keys (SSE-KMS)
while sse_customer_key is used for encryption with customer-managed keys (SSE-C).
Please choose one or the other.`
