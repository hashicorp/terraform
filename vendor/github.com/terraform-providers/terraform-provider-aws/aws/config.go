package aws

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/aws/aws-sdk-go/service/applicationautoscaling"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/batch"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/aws/aws-sdk-go/service/cloudtrail"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatchevents"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/codebuild"
	"github.com/aws/aws-sdk-go/service/codecommit"
	"github.com/aws/aws-sdk-go/service/codedeploy"
	"github.com/aws/aws-sdk-go/service/codepipeline"
	"github.com/aws/aws-sdk-go/service/cognitoidentity"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go/service/configservice"
	"github.com/aws/aws-sdk-go/service/databasemigrationservice"
	"github.com/aws/aws-sdk-go/service/devicefarm"
	"github.com/aws/aws-sdk-go/service/directconnect"
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
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/emr"
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/aws/aws-sdk-go/service/gamelift"
	"github.com/aws/aws-sdk-go/service/glacier"
	"github.com/aws/aws-sdk-go/service/glue"
	"github.com/aws/aws-sdk-go/service/guardduty"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/inspector"
	"github.com/aws/aws-sdk-go/service/iot"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/lightsail"
	"github.com/aws/aws-sdk-go/service/mediastore"
	"github.com/aws/aws-sdk-go/service/mq"
	"github.com/aws/aws-sdk-go/service/opsworks"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/redshift"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/servicecatalog"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/aws/aws-sdk-go/service/sfn"
	"github.com/aws/aws-sdk-go/service/simpledb"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/waf"
	"github.com/aws/aws-sdk-go/service/wafregional"
	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/terraform/helper/logging"
	"github.com/hashicorp/terraform/terraform"
)

type Config struct {
	AccessKey     string
	SecretKey     string
	CredsFilename string
	Profile       string
	Token         string
	Region        string
	MaxRetries    int

	AssumeRoleARN         string
	AssumeRoleExternalID  string
	AssumeRoleSessionName string
	AssumeRolePolicy      string

	AllowedAccountIds   []interface{}
	ForbiddenAccountIds []interface{}

	AcmEndpoint              string
	ApigatewayEndpoint       string
	CloudFormationEndpoint   string
	CloudWatchEndpoint       string
	CloudWatchEventsEndpoint string
	CloudWatchLogsEndpoint   string
	DynamoDBEndpoint         string
	DeviceFarmEndpoint       string
	Ec2Endpoint              string
	EcsEndpoint              string
	EcrEndpoint              string
	ElbEndpoint              string
	IamEndpoint              string
	KinesisEndpoint          string
	KmsEndpoint              string
	LambdaEndpoint           string
	RdsEndpoint              string
	R53Endpoint              string
	S3Endpoint               string
	SnsEndpoint              string
	SqsEndpoint              string
	StsEndpoint              string
	Insecure                 bool

	SkipCredsValidation     bool
	SkipGetEC2Platforms     bool
	SkipRegionValidation    bool
	SkipRequestingAccountId bool
	SkipMetadataApiCheck    bool
	S3ForcePathStyle        bool
}

type AWSClient struct {
	cfconn                *cloudformation.CloudFormation
	cloudfrontconn        *cloudfront.CloudFront
	cloudtrailconn        *cloudtrail.CloudTrail
	cloudwatchconn        *cloudwatch.CloudWatch
	cloudwatchlogsconn    *cloudwatchlogs.CloudWatchLogs
	cloudwatcheventsconn  *cloudwatchevents.CloudWatchEvents
	cognitoconn           *cognitoidentity.CognitoIdentity
	cognitoidpconn        *cognitoidentityprovider.CognitoIdentityProvider
	configconn            *configservice.ConfigService
	devicefarmconn        *devicefarm.DeviceFarm
	dmsconn               *databasemigrationservice.DatabaseMigrationService
	dsconn                *directoryservice.DirectoryService
	dynamodbconn          *dynamodb.DynamoDB
	ec2conn               *ec2.EC2
	ecrconn               *ecr.ECR
	ecsconn               *ecs.ECS
	efsconn               *efs.EFS
	elbconn               *elb.ELB
	elbv2conn             *elbv2.ELBV2
	emrconn               *emr.EMR
	esconn                *elasticsearch.ElasticsearchService
	acmconn               *acm.ACM
	apigateway            *apigateway.APIGateway
	appautoscalingconn    *applicationautoscaling.ApplicationAutoScaling
	autoscalingconn       *autoscaling.AutoScaling
	s3conn                *s3.S3
	scconn                *servicecatalog.ServiceCatalog
	sesConn               *ses.SES
	simpledbconn          *simpledb.SimpleDB
	sqsconn               *sqs.SQS
	snsconn               *sns.SNS
	stsconn               *sts.STS
	redshiftconn          *redshift.Redshift
	r53conn               *route53.Route53
	partition             string
	accountid             string
	supportedplatforms    []string
	region                string
	rdsconn               *rds.RDS
	iamconn               *iam.IAM
	kinesisconn           *kinesis.Kinesis
	kmsconn               *kms.KMS
	gameliftconn          *gamelift.GameLift
	firehoseconn          *firehose.Firehose
	inspectorconn         *inspector.Inspector
	elasticacheconn       *elasticache.ElastiCache
	elasticbeanstalkconn  *elasticbeanstalk.ElasticBeanstalk
	elastictranscoderconn *elastictranscoder.ElasticTranscoder
	lambdaconn            *lambda.Lambda
	lightsailconn         *lightsail.Lightsail
	mqconn                *mq.MQ
	opsworksconn          *opsworks.OpsWorks
	glacierconn           *glacier.Glacier
	guarddutyconn         *guardduty.GuardDuty
	codebuildconn         *codebuild.CodeBuild
	codedeployconn        *codedeploy.CodeDeploy
	codecommitconn        *codecommit.CodeCommit
	codepipelineconn      *codepipeline.CodePipeline
	sdconn                *servicediscovery.ServiceDiscovery
	sfnconn               *sfn.SFN
	ssmconn               *ssm.SSM
	wafconn               *waf.WAF
	wafregionalconn       *wafregional.WAFRegional
	iotconn               *iot.IoT
	batchconn             *batch.Batch
	glueconn              *glue.Glue
	athenaconn            *athena.Athena
	dxconn                *directconnect.DirectConnect
	mediastoreconn        *mediastore.MediaStore
}

func (c *AWSClient) S3() *s3.S3 {
	return c.s3conn
}

func (c *AWSClient) DynamoDB() *dynamodb.DynamoDB {
	return c.dynamodbconn
}

func (c *AWSClient) IsGovCloud() bool {
	if c.region == "us-gov-west-1" {
		return true
	}
	return false
}

func (c *AWSClient) IsChinaCloud() bool {
	if c.region == "cn-north-1" {
		return true
	}
	if c.region == "cn-northwest-1" {
		return true
	}
	return false
}

// Client configures and returns a fully initialized AWSClient
func (c *Config) Client() (interface{}, error) {
	// Get the auth and region. This can fail if keys/regions were not
	// specified and we're attempting to use the environment.
	if c.SkipRegionValidation {
		log.Println("[INFO] Skipping region validation")
	} else {
		log.Println("[INFO] Building AWS region structure")
		err := c.ValidateRegion()
		if err != nil {
			return nil, err
		}
	}

	var client AWSClient
	// store AWS region in client struct, for region specific operations such as
	// bucket storage in S3
	client.region = c.Region

	log.Println("[INFO] Building AWS auth structure")
	creds, err := GetCredentials(c)
	if err != nil {
		return nil, err
	}

	// define the AWS Session options
	// Credentials or Profile will be set in the Options below
	// MaxRetries may be set once we validate credentials
	var opt = session.Options{
		Config: aws.Config{
			Region:           aws.String(c.Region),
			MaxRetries:       aws.Int(0),
			HTTPClient:       cleanhttp.DefaultClient(),
			S3ForcePathStyle: aws.Bool(c.S3ForcePathStyle),
		},
	}

	// Call Get to check for credential provider. If nothing found, we'll get an
	// error, and we can present it nicely to the user
	cp, err := creds.Get()
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NoCredentialProviders" {
			// If a profile wasn't specified then error out
			if c.Profile == "" {
				return nil, errors.New(`No valid credential sources found for AWS Provider.
  Please see https://terraform.io/docs/providers/aws/index.html for more information on
  providing credentials for the AWS Provider`)
			}
			// add the profile and enable share config file usage
			log.Printf("[INFO] AWS Auth using Profile: %q", c.Profile)
			opt.Profile = c.Profile
			opt.SharedConfigState = session.SharedConfigEnable
		} else {
			return nil, fmt.Errorf("Error loading credentials for AWS Provider: %s", err)
		}
	} else {
		// add the validated credentials to the session options
		log.Printf("[INFO] AWS Auth provider used: %q", cp.ProviderName)
		opt.Config.Credentials = creds
	}

	if logging.IsDebugOrHigher() {
		opt.Config.LogLevel = aws.LogLevel(aws.LogDebugWithHTTPBody | aws.LogDebugWithRequestRetries | aws.LogDebugWithRequestErrors)
		opt.Config.Logger = awsLogger{}
	}

	if c.Insecure {
		transport := opt.Config.HTTPClient.Transport.(*http.Transport)
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	// create base session with no retries. MaxRetries will be set later
	sess, err := session.NewSessionWithOptions(opt)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NoCredentialProviders" {
			return nil, errors.New(`No valid credential sources found for AWS Provider.
  Please see https://terraform.io/docs/providers/aws/index.html for more information on
  providing credentials for the AWS Provider`)
		}
		return nil, errwrap.Wrapf("Error creating AWS session: {{err}}", err)
	}

	sess.Handlers.Build.PushBackNamed(addTerraformVersionToUserAgent)

	if extraDebug := os.Getenv("TERRAFORM_AWS_AUTHFAILURE_DEBUG"); extraDebug != "" {
		sess.Handlers.UnmarshalError.PushFrontNamed(debugAuthFailure)
	}

	// if the desired number of retries is non-zero, update the session
	if c.MaxRetries > 0 {
		sess = sess.Copy(&aws.Config{MaxRetries: aws.Int(c.MaxRetries)})
	}

	// This restriction should only be used for Route53 sessions.
	// Other resources that have restrictions should allow the API to fail, rather
	// than Terraform abstracting the region for the user. This can lead to breaking
	// changes if that resource is ever opened up to more regions.
	r53Sess := sess.Copy(&aws.Config{Region: aws.String("us-east-1"), Endpoint: aws.String(c.R53Endpoint)})

	// Some services have user-configurable endpoints
	awsAcmSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.AcmEndpoint)})
	awsApigatewaySess := sess.Copy(&aws.Config{Endpoint: aws.String(c.ApigatewayEndpoint)})
	awsCfSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.CloudFormationEndpoint)})
	awsCwSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.CloudWatchEndpoint)})
	awsCweSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.CloudWatchEventsEndpoint)})
	awsCwlSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.CloudWatchLogsEndpoint)})
	awsDynamoSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.DynamoDBEndpoint)})
	awsEc2Sess := sess.Copy(&aws.Config{Endpoint: aws.String(c.Ec2Endpoint)})
	awsEcrSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.EcrEndpoint)})
	awsEcsSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.EcsEndpoint)})
	awsElbSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.ElbEndpoint)})
	awsIamSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.IamEndpoint)})
	awsLambdaSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.LambdaEndpoint)})
	awsKinesisSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.KinesisEndpoint)})
	awsKmsSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.KmsEndpoint)})
	awsRdsSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.RdsEndpoint)})
	awsS3Sess := sess.Copy(&aws.Config{Endpoint: aws.String(c.S3Endpoint)})
	awsSnsSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.SnsEndpoint)})
	awsSqsSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.SqsEndpoint)})
	awsStsSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.StsEndpoint)})
	awsDeviceFarmSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.DeviceFarmEndpoint)})

	log.Println("[INFO] Initializing DeviceFarm SDK connection")
	client.devicefarmconn = devicefarm.New(awsDeviceFarmSess)

	// These two services need to be set up early so we can check on AccountID
	client.iamconn = iam.New(awsIamSess)
	client.stsconn = sts.New(awsStsSess)

	if !c.SkipCredsValidation {
		err = c.ValidateCredentials(client.stsconn)
		if err != nil {
			return nil, err
		}
	}

	if !c.SkipRequestingAccountId {
		partition, accountId, err := GetAccountInfo(client.iamconn, client.stsconn, cp.ProviderName)
		if err == nil {
			client.partition = partition
			client.accountid = accountId
		}
	}

	authErr := c.ValidateAccountId(client.accountid)
	if authErr != nil {
		return nil, authErr
	}

	client.ec2conn = ec2.New(awsEc2Sess)

	if !c.SkipGetEC2Platforms {
		supportedPlatforms, err := GetSupportedEC2Platforms(client.ec2conn)
		if err != nil {
			// We intentionally fail *silently* because there's a chance
			// user just doesn't have ec2:DescribeAccountAttributes permissions
			log.Printf("[WARN] Unable to get supported EC2 platforms: %s", err)
		} else {
			client.supportedplatforms = supportedPlatforms
		}
	}

	client.acmconn = acm.New(awsAcmSess)
	client.apigateway = apigateway.New(awsApigatewaySess)
	client.appautoscalingconn = applicationautoscaling.New(sess)
	client.autoscalingconn = autoscaling.New(sess)
	client.cfconn = cloudformation.New(awsCfSess)
	client.cloudfrontconn = cloudfront.New(sess)
	client.cloudtrailconn = cloudtrail.New(sess)
	client.cloudwatchconn = cloudwatch.New(awsCwSess)
	client.cloudwatcheventsconn = cloudwatchevents.New(awsCweSess)
	client.cloudwatchlogsconn = cloudwatchlogs.New(awsCwlSess)
	client.codecommitconn = codecommit.New(sess)
	client.codebuildconn = codebuild.New(sess)
	client.codedeployconn = codedeploy.New(sess)
	client.configconn = configservice.New(sess)
	client.cognitoconn = cognitoidentity.New(sess)
	client.cognitoidpconn = cognitoidentityprovider.New(sess)
	client.dmsconn = databasemigrationservice.New(sess)
	client.codepipelineconn = codepipeline.New(sess)
	client.dsconn = directoryservice.New(sess)
	client.dynamodbconn = dynamodb.New(awsDynamoSess)
	client.ecrconn = ecr.New(awsEcrSess)
	client.ecsconn = ecs.New(awsEcsSess)
	client.efsconn = efs.New(sess)
	client.elasticacheconn = elasticache.New(sess)
	client.elasticbeanstalkconn = elasticbeanstalk.New(sess)
	client.elastictranscoderconn = elastictranscoder.New(sess)
	client.elbconn = elb.New(awsElbSess)
	client.elbv2conn = elbv2.New(awsElbSess)
	client.emrconn = emr.New(sess)
	client.esconn = elasticsearch.New(sess)
	client.firehoseconn = firehose.New(sess)
	client.inspectorconn = inspector.New(sess)
	client.gameliftconn = gamelift.New(sess)
	client.glacierconn = glacier.New(sess)
	client.guarddutyconn = guardduty.New(sess)
	client.iotconn = iot.New(sess)
	client.kinesisconn = kinesis.New(awsKinesisSess)
	client.kmsconn = kms.New(awsKmsSess)
	client.lambdaconn = lambda.New(awsLambdaSess)
	client.lightsailconn = lightsail.New(sess)
	client.mqconn = mq.New(sess)
	client.opsworksconn = opsworks.New(sess)
	client.r53conn = route53.New(r53Sess)
	client.rdsconn = rds.New(awsRdsSess)
	client.redshiftconn = redshift.New(sess)
	client.simpledbconn = simpledb.New(sess)
	client.s3conn = s3.New(awsS3Sess)
	client.scconn = servicecatalog.New(sess)
	client.sdconn = servicediscovery.New(sess)
	client.sesConn = ses.New(sess)
	client.sfnconn = sfn.New(sess)
	client.snsconn = sns.New(awsSnsSess)
	client.sqsconn = sqs.New(awsSqsSess)
	client.ssmconn = ssm.New(sess)
	client.wafconn = waf.New(sess)
	client.wafregionalconn = wafregional.New(sess)
	client.batchconn = batch.New(sess)
	client.glueconn = glue.New(sess)
	client.athenaconn = athena.New(sess)
	client.dxconn = directconnect.New(sess)
	client.mediastoreconn = mediastore.New(sess)

	// Workaround for https://github.com/aws/aws-sdk-go/issues/1376
	client.kinesisconn.Handlers.Retry.PushBack(func(r *request.Request) {
		if !strings.HasPrefix(r.Operation.Name, "Describe") && !strings.HasPrefix(r.Operation.Name, "List") {
			return
		}
		err, ok := r.Error.(awserr.Error)
		if !ok || err == nil {
			return
		}
		if err.Code() == kinesis.ErrCodeLimitExceededException {
			r.Retryable = aws.Bool(true)
		}
	})

	// Workaround for https://github.com/aws/aws-sdk-go/issues/1472
	client.appautoscalingconn.Handlers.Retry.PushBack(func(r *request.Request) {
		if !strings.HasPrefix(r.Operation.Name, "Describe") && !strings.HasPrefix(r.Operation.Name, "List") {
			return
		}
		err, ok := r.Error.(awserr.Error)
		if !ok || err == nil {
			return
		}
		if err.Code() == applicationautoscaling.ErrCodeFailedResourceAccessException {
			r.Retryable = aws.Bool(true)
		}
	})

	return &client, nil
}

func hasEc2Classic(platforms []string) bool {
	for _, p := range platforms {
		if p == "EC2" {
			return true
		}
	}
	return false
}

// ValidateRegion returns an error if the configured region is not a
// valid aws region and nil otherwise.
func (c *Config) ValidateRegion() error {
	var regions = []string{
		"ap-northeast-1",
		"ap-northeast-2",
		"ap-south-1",
		"ap-southeast-1",
		"ap-southeast-2",
		"ca-central-1",
		"cn-north-1",
		"cn-northwest-1",
		"eu-central-1",
		"eu-west-1",
		"eu-west-2",
		"eu-west-3",
		"sa-east-1",
		"us-east-1",
		"us-east-2",
		"us-gov-west-1",
		"us-west-1",
		"us-west-2",
	}

	for _, valid := range regions {
		if c.Region == valid {
			return nil
		}
	}
	return fmt.Errorf("Not a valid region: %s", c.Region)
}

// Validate credentials early and fail before we do any graph walking.
func (c *Config) ValidateCredentials(stsconn *sts.STS) error {
	_, err := stsconn.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	return err
}

// ValidateAccountId returns a context-specific error if the configured account
// id is explicitly forbidden or not authorised; and nil if it is authorised.
func (c *Config) ValidateAccountId(accountId string) error {
	if c.AllowedAccountIds == nil && c.ForbiddenAccountIds == nil {
		return nil
	}

	log.Println("[INFO] Validating account ID")

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

func GetSupportedEC2Platforms(conn *ec2.EC2) ([]string, error) {
	attrName := "supported-platforms"

	input := ec2.DescribeAccountAttributesInput{
		AttributeNames: []*string{aws.String(attrName)},
	}
	attributes, err := conn.DescribeAccountAttributes(&input)
	if err != nil {
		return nil, err
	}

	var platforms []string
	for _, attr := range attributes.AccountAttributes {
		if *attr.AttributeName == attrName {
			for _, v := range attr.AttributeValues {
				platforms = append(platforms, *v.AttributeValue)
			}
			break
		}
	}

	if len(platforms) == 0 {
		return nil, fmt.Errorf("No EC2 platforms detected")
	}

	return platforms, nil
}

// addTerraformVersionToUserAgent is a named handler that will add Terraform's
// version information to requests made by the AWS SDK.
var addTerraformVersionToUserAgent = request.NamedHandler{
	Name: "terraform.TerraformVersionUserAgentHandler",
	Fn: request.MakeAddToUserAgentHandler(
		"APN/1.0 HashiCorp/1.0 Terraform", terraform.VersionString()),
}

var debugAuthFailure = request.NamedHandler{
	Name: "terraform.AuthFailureAdditionalDebugHandler",
	Fn: func(req *request.Request) {
		if isAWSErr(req.Error, "AuthFailure", "AWS was not able to validate the provided access credentials") {
			log.Printf("[INFO] Additional AuthFailure Debugging Context")
			log.Printf("[INFO] Current system UTC time: %s", time.Now().UTC())
			log.Printf("[INFO] Request object: %s", spew.Sdump(req))
		}
	},
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
