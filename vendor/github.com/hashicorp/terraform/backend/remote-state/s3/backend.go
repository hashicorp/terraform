package s3

import (
	"context"

	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/helper/schema"

	terraformAWS "github.com/hashicorp/terraform/builtin/providers/aws"
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
				Description: "The region of the S3 bucket.",
				DefaultFunc: schema.EnvDefaultFunc("AWS_DEFAULT_REGION", nil),
			},

			"endpoint": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A custom endpoint for the S3 API",
				DefaultFunc: schema.EnvDefaultFunc("AWS_S3_ENDPOINT", ""),
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
}

func (b *Backend) configure(ctx context.Context) error {
	if b.s3Client != nil {
		return nil
	}

	// Grab the resource data
	data := schema.FromContextBackendConfig(ctx)

	b.bucketName = data.Get("bucket").(string)
	b.keyName = data.Get("key").(string)
	b.serverSideEncryption = data.Get("encrypt").(bool)
	b.acl = data.Get("acl").(string)
	b.kmsKeyID = data.Get("kms_key_id").(string)

	b.ddbTable = data.Get("dynamodb_table").(string)
	if b.ddbTable == "" {
		// try the depracted field
		b.ddbTable = data.Get("lock_table").(string)
	}

	cfg := &terraformAWS.Config{
		AccessKey:             data.Get("access_key").(string),
		AssumeRoleARN:         data.Get("role_arn").(string),
		AssumeRoleExternalID:  data.Get("external_id").(string),
		AssumeRolePolicy:      data.Get("assume_role_policy").(string),
		AssumeRoleSessionName: data.Get("session_name").(string),
		CredsFilename:         data.Get("shared_credentials_file").(string),
		Profile:               data.Get("profile").(string),
		Region:                data.Get("region").(string),
		S3Endpoint:            data.Get("endpoint").(string),
		SecretKey:             data.Get("secret_key").(string),
		Token:                 data.Get("token").(string),
	}

	client, err := cfg.Client()
	if err != nil {
		return err
	}

	b.s3Client = client.(*terraformAWS.AWSClient).S3()
	b.dynClient = client.(*terraformAWS.AWSClient).DynamoDB()

	return nil
}
