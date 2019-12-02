package dynamodb

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
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/helper/logging"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/version"
)

// New creates a new backend for S3 remote state.
func New() backend.Backend {
	s := &schema.Backend{
		Schema: map[string]*schema.Schema{
			"state_table": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the DynamoDB Table used for state.",
			},

			"hash": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the hashKey of state file inside the dynamodb table.",
				//ValidateFunc: func(v interface{}, s string) ([]string, []error) {
				//	// s3 will strip leading slashes from an object, so while this will
				//	// technically be accepted by s3, it will break our workspace hierarchy.
				//	if strings.HasPrefix(v.(string), "/") {
				//		return nil, []error{errors.New("key must not start with '/'")}
				//	}
				//	return nil, nil
				//},
			},

			"region": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The region of the DynamoDB Table.",
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{
					"AWS_REGION",
					"AWS_DEFAULT_REGION",
				}, nil),
			},

			"endpoint": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A custom endpoint for the DynamoDB API",
				DefaultFunc: schema.EnvDefaultFunc("AWS_DYNAMODB_ENDPOINT", ""),
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

			"lock_table": {
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

			"assume_role_policy": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The permissions applied when assuming a role.",
				Default:     "",
			},

			"workspace_key_prefix": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The prefix applied to the non-default state path inside the bucket.",
				Default:     "workspace",
				ValidateFunc: func(v interface{}, s string) ([]string, []error) {
					prefix := v.(string)
					if strings.Contains(prefix, ":") {
						return nil, []error{errors.New("workspace_key_prefix must not contains '='")}
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
	if b.dynClient != nil {
		return nil
	}

	// Grab the resource data
	data := schema.FromContextBackendConfig(ctx)

	if !data.Get("skip_region_validation").(bool) {
		if err := awsbase.ValidateRegion(data.Get("region").(string)); err != nil {
			return err
		}
	}

	b.bucketName = data.Get("state_table").(string)
	b.keyName = data.Get("hash").(string)
	b.acl = data.Get("acl").(string)
	b.workspaceKeyPrefix = data.Get("workspace_key_prefix").(string)
	b.serverSideEncryption = data.Get("encrypt").(bool)
	b.kmsKeyID = data.Get("kms_key_id").(string)

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

	b.ddbTable = data.Get("lock_table").(string)

	cfg := &awsbase.Config{
		AccessKey:             data.Get("access_key").(string),
		AssumeRoleARN:         data.Get("role_arn").(string),
		AssumeRoleExternalID:  data.Get("external_id").(string),
		AssumeRolePolicy:      data.Get("assume_role_policy").(string),
		AssumeRoleSessionName: data.Get("session_name").(string),
		CredsFilename:         data.Get("shared_credentials_file").(string),
		DebugLogging:          logging.IsDebugOrHigher(),
		IamEndpoint:           data.Get("iam_endpoint").(string),
		MaxRetries:            data.Get("max_retries").(int),
		Profile:               data.Get("profile").(string),
		Region:                data.Get("region").(string),
		SecretKey:             data.Get("secret_key").(string),
		SkipCredsValidation:   data.Get("skip_credentials_validation").(bool),
		SkipMetadataApiCheck:  data.Get("skip_metadata_api_check").(bool),
		StsEndpoint:           data.Get("sts_endpoint").(string),
		Token:                 data.Get("token").(string),
		UserAgentProducts: []*awsbase.UserAgentProduct{
			{Name: "APN", Version: "1.0"},
			{Name: "HashiCorp", Version: "1.0"},
			{Name: "Terraform", Version: version.String()},
		},
	}

	sess, err := awsbase.GetSession(cfg)
	if err != nil {
		return err
	}

	b.dynClient = dynamodb.New(sess.Copy(&aws.Config{
		Endpoint: aws.String(data.Get("endpoint").(string)),
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
