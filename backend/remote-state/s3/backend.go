package s3

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/s3"
	cleanhttp "github.com/hashicorp/go-cleanhttp"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/helper/schema"

	terraformAWS "github.com/hashicorp/terraform/builtin/providers/aws"
)

// New creates a new backend for S3 remote state.
func New() backend.Backend {
	s := &schema.Backend{
		Schema: map[string]*schema.Schema{
			"bucket": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the S3 bucket",
			},

			"key": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The path to the state file inside the bucket",
			},

			"region": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The region of the S3 bucket.",
				DefaultFunc: schema.EnvDefaultFunc("AWS_DEFAULT_REGION", nil),
			},

			"endpoint": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A custom endpoint for the S3 API",
				DefaultFunc: schema.EnvDefaultFunc("AWS_S3_ENDPOINT", ""),
			},

			"encrypt": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Whether to enable server side encryption of the state file",
				Default:     false,
			},

			"acl": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Canned ACL to be applied to the state file",
				Default:     "",
			},

			"access_key": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "AWS access key",
				Default:     "",
			},

			"secret_key": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "AWS secret key",
				Default:     "",
			},

			"kms_key_id": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The ARN of a KMS Key to use for encrypting the state",
				Default:     "",
			},

			"lock_table": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "DynamoDB table for state locking",
				Default:     "",
			},

			"profile": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "AWS profile name",
				Default:     "",
			},

			"shared_credentials_file": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Path to a shared credentials file",
				Default:     "",
			},

			"token": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "MFA token",
				Default:     "",
			},

			"role_arn": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The role to be assumed",
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
	lockTable            string
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
	b.lockTable = data.Get("lock_table").(string)

	var errs []error
	creds, err := terraformAWS.GetCredentials(&terraformAWS.Config{
		AccessKey:     data.Get("access_key").(string),
		SecretKey:     data.Get("secret_key").(string),
		Token:         data.Get("token").(string),
		Profile:       data.Get("profile").(string),
		CredsFilename: data.Get("shared_credentials_file").(string),
		AssumeRoleARN: data.Get("role_arn").(string),
	})
	if err != nil {
		return err
	}

	// Call Get to check for credential provider. If nothing found, we'll get an
	// error, and we can present it nicely to the user
	_, err = creds.Get()
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NoCredentialProviders" {
			errs = append(errs, fmt.Errorf(`No valid credential sources found for AWS S3 remote.
Please see https://www.terraform.io/docs/state/remote/s3.html for more information on
providing credentials for the AWS S3 remote`))
		} else {
			errs = append(errs, fmt.Errorf("Error loading credentials for AWS S3 remote: %s", err))
		}
		return &multierror.Error{Errors: errs}
	}

	endpoint := data.Get("endpoint").(string)
	region := data.Get("region").(string)

	awsConfig := &aws.Config{
		Credentials: creds,
		Endpoint:    aws.String(endpoint),
		Region:      aws.String(region),
		HTTPClient:  cleanhttp.DefaultClient(),
	}
	sess := session.New(awsConfig)
	b.s3Client = s3.New(sess)
	b.dynClient = dynamodb.New(sess)

	return nil
}
