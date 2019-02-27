package s3

import (
	"context"
	"errors"
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
				Description: "The region of the S3 bucket.",
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

			"lock_table": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "DynamoDB table for state locking",
				Default:     "",
				Deprecated:  "please use the dynamodb_table attribute",
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

			"skip_get_ec2_platforms": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Skip getting the supported EC2 platforms.",
				Default:     false,
				Deprecated:  "The S3 Backend does not require EC2 functionality and this attribute is no longer used.",
			},

			"skip_region_validation": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Skip static validation of region name.",
				Default:     false,
			},

			"skip_requesting_account_id": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Skip requesting the account ID.",
				Default:     false,
				Deprecated:  "The S3 Backend no longer automatically looks up the AWS Account ID and this attribute is no longer used.",
			},

			"skip_metadata_api_check": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Skip the AWS Metadata API check.",
				Default:     false,
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

	bucketName           string
	keyName              string
	serverSideEncryption bool
	acl                  string
	kmsKeyID             string
	ddbTable             string
	workspaceKeyPrefix   string
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
	b.serverSideEncryption = data.Get("encrypt").(bool)
	b.acl = data.Get("acl").(string)
	b.kmsKeyID = data.Get("kms_key_id").(string)
	b.workspaceKeyPrefix = data.Get("workspace_key_prefix").(string)

	b.ddbTable = data.Get("dynamodb_table").(string)
	if b.ddbTable == "" {
		// try the deprecated field
		b.ddbTable = data.Get("lock_table").(string)
	}

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
		Endpoint: aws.String(data.Get("dynamodb_endpoint").(string)),
	}))
	b.s3Client = s3.New(sess.Copy(&aws.Config{
		Endpoint:         aws.String(data.Get("endpoint").(string)),
		S3ForcePathStyle: aws.Bool(data.Get("force_path_style").(bool)),
	}))

	return nil
}
