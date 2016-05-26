package aws

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/helper/logging"
	"github.com/hashicorp/terraform/terraform"

	"crypto/tls"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/aws/aws-sdk-go/service/cloudtrail"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatchevents"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/codecommit"
	"github.com/aws/aws-sdk-go/service/codedeploy"
	"github.com/aws/aws-sdk-go/service/directoryservice"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/efs"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/aws/aws-sdk-go/service/elasticbeanstalk"
	elasticsearch "github.com/aws/aws-sdk-go/service/elasticsearchservice"
	"github.com/aws/aws-sdk-go/service/elastictranscoder"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/emr"
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/aws/aws-sdk-go/service/glacier"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/opsworks"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/redshift"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sts"
)

type Config struct {
	AccessKey     string
	SecretKey     string
	CredsFilename string
	Profile       string
	Token         string
	Region        string
	MaxRetries    int

	AllowedAccountIds   []interface{}
	ForbiddenAccountIds []interface{}

	DynamoDBEndpoint string
	KinesisEndpoint  string
	Ec2Endpoint      string
	IamEndpoint      string
	ElbEndpoint      string
	Insecure         bool
}

type AWSClient struct {
	cfconn                *cloudformation.CloudFormation
	cloudfrontconn        *cloudfront.CloudFront
	cloudtrailconn        *cloudtrail.CloudTrail
	cloudwatchconn        *cloudwatch.CloudWatch
	cloudwatchlogsconn    *cloudwatchlogs.CloudWatchLogs
	cloudwatcheventsconn  *cloudwatchevents.CloudWatchEvents
	dsconn                *directoryservice.DirectoryService
	dynamodbconn          *dynamodb.DynamoDB
	ec2conn               *ec2.EC2
	ecrconn               *ecr.ECR
	ecsconn               *ecs.ECS
	efsconn               *efs.EFS
	elbconn               *elb.ELB
	emrconn               *emr.EMR
	esconn                *elasticsearch.ElasticsearchService
	apigateway            *apigateway.APIGateway
	autoscalingconn       *autoscaling.AutoScaling
	s3conn                *s3.S3
	sqsconn               *sqs.SQS
	snsconn               *sns.SNS
	stsconn               *sts.STS
	redshiftconn          *redshift.Redshift
	r53conn               *route53.Route53
	accountid             string
	region                string
	rdsconn               *rds.RDS
	iamconn               *iam.IAM
	kinesisconn           *kinesis.Kinesis
	kmsconn               *kms.KMS
	firehoseconn          *firehose.Firehose
	elasticacheconn       *elasticache.ElastiCache
	elasticbeanstalkconn  *elasticbeanstalk.ElasticBeanstalk
	elastictranscoderconn *elastictranscoder.ElasticTranscoder
	lambdaconn            *lambda.Lambda
	opsworksconn          *opsworks.OpsWorks
	glacierconn           *glacier.Glacier
	codedeployconn        *codedeploy.CodeDeploy
	codecommitconn        *codecommit.CodeCommit
}

// Client configures and returns a fully initialized AWSClient
func (c *Config) Client() (interface{}, error) {
	// Get the auth and region. This can fail if keys/regions were not
	// specified and we're attempting to use the environment.
	var errs []error

	log.Println("[INFO] Building AWS region structure")
	err := c.ValidateRegion()
	if err != nil {
		errs = append(errs, err)
	}

	var client AWSClient
	if len(errs) == 0 {
		// store AWS region in client struct, for region specific operations such as
		// bucket storage in S3
		client.region = c.Region

		log.Println("[INFO] Building AWS auth structure")
		creds := GetCredentials(c.AccessKey, c.SecretKey, c.Token, c.Profile, c.CredsFilename)
		// Call Get to check for credential provider. If nothing found, we'll get an
		// error, and we can present it nicely to the user
		cp, err := creds.Get()
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NoCredentialProviders" {
				errs = append(errs, fmt.Errorf(`No valid credential sources found for AWS Provider.
  Please see https://terraform.io/docs/providers/aws/index.html for more information on
  providing credentials for the AWS Provider`))
			} else {
				errs = append(errs, fmt.Errorf("Error loading credentials for AWS Provider: %s", err))
			}
			return nil, &multierror.Error{Errors: errs}
		}

		log.Printf("[INFO] AWS Auth provider used: %q", cp.ProviderName)

		awsConfig := &aws.Config{
			Credentials: creds,
			Region:      aws.String(c.Region),
			MaxRetries:  aws.Int(c.MaxRetries),
			HTTPClient:  cleanhttp.DefaultClient(),
		}

		if logging.IsDebugOrHigher() {
			awsConfig.LogLevel = aws.LogLevel(aws.LogDebugWithHTTPBody)
			awsConfig.Logger = awsLogger{}
		}

		if c.Insecure {
			transport := awsConfig.HTTPClient.Transport.(*http.Transport)
			transport.TLSClientConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
		}

		// Set up base session
		sess := session.New(awsConfig)
		sess.Handlers.Build.PushFrontNamed(addTerraformVersionToUserAgent)

		log.Println("[INFO] Initializing IAM Connection")
		awsIamSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.IamEndpoint)})
		client.iamconn = iam.New(awsIamSess)

		log.Println("[INFO] Initializing STS connection")
		client.stsconn = sts.New(sess)

		err = c.ValidateCredentials(client.iamconn)
		if err != nil {
			errs = append(errs, err)
			return nil, &multierror.Error{Errors: errs}
		}

		// Some services exist only in us-east-1, e.g. because they manage
		// resources that can span across multiple regions, or because
		// signature format v4 requires region to be us-east-1 for global
		// endpoints:
		// http://docs.aws.amazon.com/general/latest/gr/sigv4_changes.html
		usEast1Sess := sess.Copy(&aws.Config{Region: aws.String("us-east-1")})

		accountId, err := GetAccountId(client.iamconn, client.stsconn, cp.ProviderName)
		if err == nil {
			client.accountid = accountId
		}

		log.Println("[INFO] Initializing DynamoDB connection")
		dynamoSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.DynamoDBEndpoint)})
		client.dynamodbconn = dynamodb.New(dynamoSess)

		log.Println("[INFO] Initializing Cloudfront connection")
		client.cloudfrontconn = cloudfront.New(sess)

		log.Println("[INFO] Initializing ELB connection")
		awsElbSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.ElbEndpoint)})
		client.elbconn = elb.New(awsElbSess)

		log.Println("[INFO] Initializing S3 connection")
		client.s3conn = s3.New(sess)

		log.Println("[INFO] Initializing SQS connection")
		client.sqsconn = sqs.New(sess)

		log.Println("[INFO] Initializing SNS connection")
		client.snsconn = sns.New(sess)

		log.Println("[INFO] Initializing RDS Connection")
		client.rdsconn = rds.New(sess)

		log.Println("[INFO] Initializing Kinesis Connection")
		kinesisSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.KinesisEndpoint)})
		client.kinesisconn = kinesis.New(kinesisSess)

		log.Println("[INFO] Initializing Elastic Beanstalk Connection")
		client.elasticbeanstalkconn = elasticbeanstalk.New(sess)

		log.Println("[INFO] Initializing Elastic Transcoder Connection")
		client.elastictranscoderconn = elastictranscoder.New(sess)

		authErr := c.ValidateAccountId(client.accountid)
		if authErr != nil {
			errs = append(errs, authErr)
		}

		log.Println("[INFO] Initializing Kinesis Firehose Connection")
		client.firehoseconn = firehose.New(sess)

		log.Println("[INFO] Initializing AutoScaling connection")
		client.autoscalingconn = autoscaling.New(sess)

		log.Println("[INFO] Initializing EC2 Connection")

		awsEc2Sess := sess.Copy(&aws.Config{Endpoint: aws.String(c.Ec2Endpoint)})
		client.ec2conn = ec2.New(awsEc2Sess)

		log.Println("[INFO] Initializing ECR Connection")
		client.ecrconn = ecr.New(sess)

		log.Println("[INFO] Initializing API Gateway")
		client.apigateway = apigateway.New(sess)

		log.Println("[INFO] Initializing ECS Connection")
		client.ecsconn = ecs.New(sess)

		log.Println("[INFO] Initializing EFS Connection")
		client.efsconn = efs.New(sess)

		log.Println("[INFO] Initializing ElasticSearch Connection")
		client.esconn = elasticsearch.New(sess)

		log.Println("[INFO] Initializing EMR Connection")
		client.emrconn = emr.New(sess)

		log.Println("[INFO] Initializing Route 53 connection")
		client.r53conn = route53.New(usEast1Sess)

		log.Println("[INFO] Initializing Elasticache Connection")
		client.elasticacheconn = elasticache.New(sess)

		log.Println("[INFO] Initializing Lambda Connection")
		client.lambdaconn = lambda.New(sess)

		log.Println("[INFO] Initializing Cloudformation Connection")
		client.cfconn = cloudformation.New(sess)

		log.Println("[INFO] Initializing CloudWatch SDK connection")
		client.cloudwatchconn = cloudwatch.New(sess)

		log.Println("[INFO] Initializing CloudWatch Events connection")
		client.cloudwatcheventsconn = cloudwatchevents.New(sess)

		log.Println("[INFO] Initializing CloudTrail connection")
		client.cloudtrailconn = cloudtrail.New(sess)

		log.Println("[INFO] Initializing CloudWatch Logs connection")
		client.cloudwatchlogsconn = cloudwatchlogs.New(sess)

		log.Println("[INFO] Initializing OpsWorks Connection")
		client.opsworksconn = opsworks.New(usEast1Sess)

		log.Println("[INFO] Initializing Directory Service connection")
		client.dsconn = directoryservice.New(sess)

		log.Println("[INFO] Initializing Glacier connection")
		client.glacierconn = glacier.New(sess)

		log.Println("[INFO] Initializing CodeDeploy Connection")
		client.codedeployconn = codedeploy.New(sess)

		log.Println("[INFO] Initializing CodeCommit SDK connection")
		client.codecommitconn = codecommit.New(usEast1Sess)

		log.Println("[INFO] Initializing Redshift SDK connection")
		client.redshiftconn = redshift.New(sess)

		log.Println("[INFO] Initializing KMS connection")
		client.kmsconn = kms.New(sess)
	}

	if len(errs) > 0 {
		return nil, &multierror.Error{Errors: errs}
	}

	return &client, nil
}

// ValidateRegion returns an error if the configured region is not a
// valid aws region and nil otherwise.
func (c *Config) ValidateRegion() error {
	var regions = [12]string{"us-east-1", "us-west-2", "us-west-1", "eu-west-1",
		"eu-central-1", "ap-southeast-1", "ap-southeast-2", "ap-northeast-1",
		"ap-northeast-2", "sa-east-1", "cn-north-1", "us-gov-west-1"}

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
			log.Printf("[WARN] AccessDenied Error with iam.GetUser, assuming IAM role")
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
func (c *Config) ValidateAccountId(accountId string) error {
	if c.AllowedAccountIds == nil && c.ForbiddenAccountIds == nil {
		return nil
	}

	log.Printf("[INFO] Validating account ID")

	if c.ForbiddenAccountIds != nil {
		for _, id := range c.ForbiddenAccountIds {
			if id == accountId {
				return fmt.Errorf("Forbidden account ID (%s)", id)
			}
		}
	}

	if c.AllowedAccountIds != nil {
		for _, id := range c.AllowedAccountIds {
			if id == accountId {
				return nil
			}
		}
		return fmt.Errorf("Account ID not allowed (%s)", accountId)
	}

	return nil
}

// addTerraformVersionToUserAgent is a named handler that will add Terraform's
// version information to requests made by the AWS SDK.
var addTerraformVersionToUserAgent = request.NamedHandler{
	Name: "terraform.TerraformVersionUserAgentHandler",
	Fn: request.MakeAddToUserAgentHandler(
		"terraform", terraform.Version, terraform.VersionPrerelease),
}

type awsLogger struct{}

func (l awsLogger) Log(args ...interface{}) {
	tokens := make([]string, 0, len(args))
	for _, arg := range args {
		if token, ok := arg.(string); ok {
			tokens = append(tokens, token)
		}
	}
	log.Printf("[DEBUG] [aws-sdk-go] %s", strings.Join(tokens, " "))
}
