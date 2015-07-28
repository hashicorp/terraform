package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/multierror"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/lambda"
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
}

type AWSClient struct {
	cloudwatchconn  *cloudwatch.CloudWatch
	dynamodbconn    *dynamodb.DynamoDB
	ec2conn         *ec2.EC2
	ecsconn         *ecs.ECS
	elbconn         *elb.ELB
	autoscalingconn *autoscaling.AutoScaling
	s3conn          *s3.S3
	sqsconn         *sqs.SQS
	snsconn         *sns.SNS
	r53conn         *route53.Route53
	region          string
	rdsconn         *rds.RDS
	iamconn         *iam.IAM
	kinesisconn     *kinesis.Kinesis
	elasticacheconn *elasticache.ElastiCache
	lambdaconn      *lambda.Lambda
}

// Client configures and returns a fully initailized AWSClient
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
			Region:      c.Region,
			MaxRetries:  c.MaxRetries,
		}

		log.Println("[INFO] Initializing IAM Connection")
		client.iamconn = iam.New(awsConfig)

		err := c.ValidateCredentials(client.iamconn)
		if err != nil {
			errs = append(errs, err)
		}

		log.Println("[INFO] Initializing DynamoDB connection")
		client.dynamodbconn = dynamodb.New(awsConfig)

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

		// aws-sdk-go uses v4 for signing requests, which requires all global
		// endpoints to use 'us-east-1'.
		// See http://docs.aws.amazon.com/general/latest/gr/sigv4_changes.html
		log.Println("[INFO] Initializing Route 53 connection")
		client.r53conn = route53.New(&aws.Config{
			Credentials: creds,
			Region:      "us-east-1",
			MaxRetries:  c.MaxRetries,
		})

		log.Println("[INFO] Initializing Elasticache Connection")
		client.elasticacheconn = elasticache.New(awsConfig)

		log.Println("[INFO] Initializing Lambda Connection")
		client.lambdaconn = lambda.New(awsConfig)

		log.Println("[INFO] Initializing CloudWatch SDK connection")
		client.cloudwatchconn = cloudwatch.New(awsConfig)
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

// Validate credentials early and fail before we do any graph walking
func (c *Config) ValidateCredentials(iamconn *iam.IAM) error {
	_, err := iamconn.GetUser(nil)

	if awsErr, ok := err.(awserr.Error); ok {
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
		return fmt.Errorf("Failed getting account ID from IAM: %s", err)
	}

	account_id := strings.Split(*out.User.ARN, ":")[4]

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
