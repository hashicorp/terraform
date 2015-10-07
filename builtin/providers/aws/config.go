package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/go-multierror"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/efs"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/opsworks"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
)

type Config struct {
	AccessKey  string
	SecretKey  string
	Token      string
	Region     string
	MaxRetries int

	AllowedAccountIds   []interface{}
	ForbiddenAccountIds []interface{}

	DynamoDBEndpoint string
}

type AWSClient struct {
	cloudwatchconn     *cloudwatch.CloudWatch
	cloudwatchlogsconn *cloudwatchlogs.CloudWatchLogs
	dynamodbconn       *dynamodb.DynamoDB
	ec2conn            *ec2.EC2
	ecsconn            *ecs.ECS
	efsconn            *efs.EFS
	elbconn            *elb.ELB
	autoscalingconn    *autoscaling.AutoScaling
	s3conn             *s3.S3
	sqsconn            *sqs.SQS
	snsconn            *sns.SNS
	r53conn            *route53.Route53
	region             string
	rdsconn            *rds.RDS
	iamconn            *iam.IAM
	kinesisconn        *kinesis.Kinesis
	elasticacheconn    *elasticache.ElastiCache
	lambdaconn         *lambda.Lambda
	opsworksconn    *opsworks.OpsWorks
}

// Client configures and returns a fully initialized AWSClient
func (c *Config) Client() (interface{}, error) {
	var client AWSClient

	// Get the auth and region. This can fail if keys/regions were not
	// specified and we're attempting to use the environment.
	var errs []error

	log.Println("[INFO] Building AWS region structure")
	err := c.ValidateRegion()
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) == 0 {
		// store AWS region in client struct, for region specific operations such as
		// bucket storage in S3
		client.region = c.Region

		log.Println("[INFO] Building AWS auth structure")
		// We fetched all credential sources in Provider. If they are
		// available, they'll already be in c. See Provider definition.
		creds := credentials.NewStaticCredentials(c.AccessKey, c.SecretKey, c.Token)
		awsConfig := &aws.Config{
			Credentials: creds,
			Region:      aws.String(c.Region),
			MaxRetries:  aws.Int(c.MaxRetries),
		}

		log.Println("[INFO] Initializing IAM Connection")
		client.iamconn = iam.New(awsConfig)

		err := c.ValidateCredentials(client.iamconn)
		if err != nil {
			errs = append(errs, err)
		}

		awsDynamoDBConfig := &aws.Config{
			Credentials: creds,
			Region:      aws.String(c.Region),
			MaxRetries:  aws.Int(c.MaxRetries),
			Endpoint:    aws.String(c.DynamoDBEndpoint),
		}
		// Some services exist only in us-east-1, e.g. because they manage
		// resources that can span across multiple regions, or because
		// signature format v4 requires region to be us-east-1 for global
		// endpoints:
		// http://docs.aws.amazon.com/general/latest/gr/sigv4_changes.html
		usEast1AwsConfig := &aws.Config{
			Credentials: creds,
			Region:      aws.String("us-east-1"),
			MaxRetries:  aws.Int(c.MaxRetries),
		}

		log.Println("[INFO] Initializing DynamoDB connection")
		client.dynamodbconn = dynamodb.New(awsDynamoDBConfig)

		log.Println("[INFO] Initializing ELB connection")
		client.elbconn = elb.New(awsConfig)

		log.Println("[INFO] Initializing S3 connection")
		client.s3conn = s3.New(awsConfig)

		log.Println("[INFO] Initializing SQS connection")
		client.sqsconn = sqs.New(awsConfig)

		log.Println("[INFO] Initializing SNS connection")
		client.snsconn = sns.New(awsConfig)

		log.Println("[INFO] Initializing RDS Connection")
		client.rdsconn = rds.New(awsConfig)

		log.Println("[INFO] Initializing Kinesis Connection")
		client.kinesisconn = kinesis.New(awsConfig)

		authErr := c.ValidateAccountId(client.iamconn)
		if authErr != nil {
			errs = append(errs, authErr)
		}

		log.Println("[INFO] Initializing AutoScaling connection")
		client.autoscalingconn = autoscaling.New(awsConfig)

		log.Println("[INFO] Initializing EC2 Connection")
		client.ec2conn = ec2.New(awsConfig)

		log.Println("[INFO] Initializing ECS Connection")
		client.ecsconn = ecs.New(awsConfig)

		log.Println("[INFO] Initializing EFS Connection")
		client.efsconn = efs.New(awsConfig)

		log.Println("[INFO] Initializing Route 53 connection")
		client.r53conn = route53.New(usEast1AwsConfig)

		log.Println("[INFO] Initializing Elasticache Connection")
		client.elasticacheconn = elasticache.New(awsConfig)

		log.Println("[INFO] Initializing Lambda Connection")
		client.lambdaconn = lambda.New(awsConfig)

		log.Println("[INFO] Initializing CloudWatch SDK connection")
		client.cloudwatchconn = cloudwatch.New(awsConfig)

		log.Println("[INFO] Initializing CloudWatch Logs connection")
		client.cloudwatchlogsconn = cloudwatchlogs.New(awsConfig)

		log.Println("[INFO] Initializing OpsWorks Connection")
		client.opsworksconn = opsworks.New(usEast1AwsConfig)
	}

	if len(errs) > 0 {
		return nil, &multierror.Error{Errors: errs}
	}

	return &client, nil
}

// ValidateRegion returns an error if the configured region is not a
// valid aws region and nil otherwise.
func (c *Config) ValidateRegion() error {
	var regions = [11]string{"us-east-1", "us-west-2", "us-west-1", "eu-west-1",
		"eu-central-1", "ap-southeast-1", "ap-southeast-2", "ap-northeast-1",
		"sa-east-1", "cn-north-1", "us-gov-west-1"}

	for _, valid := range regions {
		if c.Region == valid {
			return nil
		}
	}
	return fmt.Errorf("Not a valid region: %s", c.Region)
}

// Validate credentials early and fail before we do any graph walking.
// In the case of an IAM role/profile with insuffecient privileges, fail
// silently
func (c *Config) ValidateCredentials(iamconn *iam.IAM) error {
	_, err := iamconn.GetUser(nil)

	if awsErr, ok := err.(awserr.Error); ok {

		if awsErr.Code() == "AccessDenied" || awsErr.Code() == "ValidationError" {
			log.Printf("[WARN] AccessDenied Error with iam.GetUser, assuming IAM profile")
			// User may be an IAM instance profile, or otherwise IAM role without the
			// GetUser permissions, so fail silently
			return nil
		}

		if awsErr.Code() == "SignatureDoesNotMatch" {
			return fmt.Errorf("Failed authenticating with AWS: please verify credentials")
		}
	}

	return err
}

// ValidateAccountId returns a context-specific error if the configured account
// id is explicitly forbidden or not authorised; and nil if it is authorised.
func (c *Config) ValidateAccountId(iamconn *iam.IAM) error {
	if c.AllowedAccountIds == nil && c.ForbiddenAccountIds == nil {
		return nil
	}

	log.Printf("[INFO] Validating account ID")

	out, err := iamconn.GetUser(nil)

	if err != nil {
		awsErr, _ := err.(awserr.Error)
		if awsErr.Code() == "ValidationError" {
			log.Printf("[WARN] ValidationError with iam.GetUser, assuming its an IAM profile")
			// User may be an IAM instance profile, so fail silently.
			// If it is an IAM instance profile
			// validating account might be superfluous
			return nil
		} else {
			return fmt.Errorf("Failed getting account ID from IAM: %s", err)
			// return error if the account id is explicitly not authorised
		}
	}

	account_id := strings.Split(*out.User.Arn, ":")[4]

	if c.ForbiddenAccountIds != nil {
		for _, id := range c.ForbiddenAccountIds {
			if id == account_id {
				return fmt.Errorf("Forbidden account ID (%s)", id)
			}
		}
	}

	if c.AllowedAccountIds != nil {
		for _, id := range c.AllowedAccountIds {
			if id == account_id {
				return nil
			}
		}
		return fmt.Errorf("Account ID not allowed (%s)", account_id)
	}

	return nil
}
