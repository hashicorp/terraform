package dynamodb

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	awsbase "github.com/hashicorp/aws-sdk-go-base"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/helper/logging"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/version"
)

type State struct {
	StateID     string
	SegmentID   int64
	Body        string
	NextStateID string
	TTL         int64
}

// New creates a new backend for DynamoDB remote state.
func New() backend.Backend {
	s := &schema.Backend{
		Schema: map[string]*schema.Schema{
			"state_table": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the DynamoDB Table used for state.",
				ValidateFunc: func(v interface{}, s string) ([]string, []error) {
					if strings.Contains(v.(string), "/") || strings.Contains(v.(string), "=") {
						return nil, []error{errors.New("State table name must not contain '/' nor '='")}
					}
					return nil, nil
				},
			},

			"hash": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the hashKey of state file inside the dynamodb table.",
				ValidateFunc: func(v interface{}, s string) ([]string, []error) {
					if strings.Contains(v.(string), "/") || strings.Contains(v.(string), "=") {
						return nil, []error{errors.New("Hash must not contain '/' nor '='")}
					}
					return nil, nil
				},
			},

			"region": {
				Type:        schema.TypeString,
				Optional:    true,
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

			"lock_table": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "DynamoDB table for state locking and consistency",
				Default:     "",
				ValidateFunc: func(v interface{}, s string) ([]string, []error) {
					if strings.Contains(v.(string), "/") || strings.Contains(v.(string), "=") {
						return nil, []error{errors.New("Lock table name must not contain '/' nor '='")}
					}
					return nil, nil
				},
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
					if strings.Contains(prefix, "=") || strings.Contains(prefix, "/") {
						return nil, []error{errors.New("Workspace Key Prefix must not contains '=' nor '/'")}
					}
					return nil, nil
				},
			},

			"state_days_ttl": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The Number of days used for old states time to live.",
				Default:     0,
				ValidateFunc: func(v interface{}, s string) ([]string, []error) {
					value := v.(int)
					if value < 0 {
						return nil, []error{errors.New("state_days_ttl value must be greater than 0")}
					}
					return nil, nil
				},
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
	dynClient        *dynamodb.DynamoDB
	dynGlobalClients []*dynamodb.DynamoDB

	tableName          string
	hashName           string
	lockTable          string
	workspaceKeyPrefix string
	state_days_ttl     int
}

func (b *Backend) validateTablesSchema() error {

	if b.lockTable != "" {
		lockTableParam := &dynamodb.DescribeTableInput{
			TableName: aws.String(b.lockTable),
		}
		lockTableDes, err := b.dynClient.DescribeTable(lockTableParam)
		if err != nil {
			return nil
		}
		lockAttDef := lockTableDes.Table.AttributeDefinitions
		lockKeyDef := lockTableDes.Table.KeySchema
		lockBool := len(lockAttDef) == 1 && len(lockKeyDef) == 1
		for _, l := range lockAttDef {
			lockBool = lockBool && *l.AttributeName == "LockID" && *l.AttributeType == "S"
		}
		for _, l := range lockKeyDef {
			if *l.AttributeName == "LockID" {
				lockBool = lockBool && *l.KeyType == "HASH"
			}
		}
		if !lockBool {
			return fmt.Errorf(errDynamoDBLockTable, b.lockTable, b.lockTable, b.lockTable)
		}

	}

	if b.tableName != "" {
		stateTableParam := &dynamodb.DescribeTableInput{
			TableName: aws.String(b.tableName),
		}
		stateTableDes, err := b.dynClient.DescribeTable(stateTableParam)
		if err != nil {
			return nil
		}
		stateAttDef := stateTableDes.Table.AttributeDefinitions
		stateKeyDef := stateTableDes.Table.KeySchema
		stateBool := len(stateAttDef) == 2 && len(stateKeyDef) == 2
		for _, s := range stateAttDef {
			stateBool = stateBool && (*s.AttributeName == "StateID" && *s.AttributeType == "S" || *s.AttributeName == "SegmentID" && *s.AttributeType == "N")
		}
		for _, s := range stateKeyDef {
			switch att := *s.AttributeName; att {
			case "StateID":
				stateBool = stateBool && *s.KeyType == "HASH"
			case "SegmentID":
				stateBool = stateBool && *s.KeyType == "RANGE"
			}
		}
		if !stateBool {
			return fmt.Errorf(errDynamoDBStateTable, b.tableName, b.tableName, b.tableName)
		}
	}

	return nil
}

func (b *Backend) getGlobalClients(endpoint string, sess *session.Session) ([]*dynamodb.DynamoDB, error) {

	dyClients := make([]*dynamodb.DynamoDB, 0)
	if b.lockTable != "" {
		globalTableParams := &dynamodb.DescribeGlobalTableInput{
			GlobalTableName: aws.String(b.lockTable),
		}

		res, err := b.dynClient.DescribeGlobalTable(globalTableParams)
		if err != nil {
			return nil, nil
		}

		regions := res.GlobalTableDescription.ReplicationGroup
		if len(regions) == 0 {
			return dyClients, nil
		}

		for _, region := range regions {
			dyClients = append(dyClients, dynamodb.New(sess.Copy(&aws.Config{
				Endpoint: aws.String(endpoint),
				Region:   aws.String(*region.RegionName),
			})))
		}

		lockTableParam := &dynamodb.DescribeTableInput{
			TableName: aws.String(b.tableName),
		}

		for _, dyClient := range dyClients {
			_, err := dyClient.DescribeTable(lockTableParam)
			if err != nil {
				return nil, err
			}
		}
	}

	return dyClients, nil
}

func (b *Backend) configure(ctx context.Context) error {
	if b.dynClient != nil {
		return nil
	}

	// Grab the resource data
	data := schema.FromContextBackendConfig(ctx)

	if data.Get("region").(string) == "" {
		return fmt.Errorf("Please set env AWS_REGION or AWS_DEFAULT_REGION, otherwise set region in backend configuration.")
	}

	if !data.Get("skip_region_validation").(bool) {
		if err := awsbase.ValidateRegion(data.Get("region").(string)); err != nil {
			return err
		}
	}

	b.tableName = data.Get("state_table").(string)
	b.hashName = data.Get("hash").(string)
	b.workspaceKeyPrefix = data.Get("workspace_key_prefix").(string)
	b.lockTable = data.Get("lock_table").(string)
	b.state_days_ttl = data.Get("state_days_ttl").(int)

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

	err = b.validateTablesSchema()
	if err != nil {
		return err
	}

	b.dynGlobalClients, err = b.getGlobalClients(data.Get("endpoint").(string), sess)
	if err != nil {
		return err
	}

	return nil
}

const errDynamoDBStateTable = `DynamoDB state table schema check error.

Please create DynamoDB table using the following command:

aws dynamodb delete-table --table-name %s && \
aws dynamodb wait table-not-exists --table-name %s && \
aws dynamodb create-table \
--table-name %s \
--attribute-definitions AttributeName=StateID,AttributeType=S AttributeName=SegmentID,AttributeType=N \
--key-schema AttributeName=StateID,KeyType=HASH AttributeName=SegmentID,KeyType=RANGE \
--provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5
`

const errDynamoDBLockTable = `DynamoDB lock table schema check error.

Please create DynamoDB table using the following command:

aws dynamodb delete-table --table-name %s && \
aws dynamodb wait table-not-exists --table-name %s && \
aws dynamodb create-table \
--table-name %s \
--attribute-definitions AttributeName=LockID,AttributeType=S \
--key-schema AttributeName=LockID,KeyType=HASH \
--provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5
`
