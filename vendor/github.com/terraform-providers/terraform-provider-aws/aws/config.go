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
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/aws/aws-sdk-go/service/acmpca"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/aws/aws-sdk-go/service/apigatewayv2"
	"github.com/aws/aws-sdk-go/service/applicationautoscaling"
	"github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/aws/aws-sdk-go/service/appsync"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/backup"
	"github.com/aws/aws-sdk-go/service/batch"
	"github.com/aws/aws-sdk-go/service/budgets"
	"github.com/aws/aws-sdk-go/service/cloud9"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/aws/aws-sdk-go/service/cloudhsmv2"
	"github.com/aws/aws-sdk-go/service/cloudsearch"
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
	"github.com/aws/aws-sdk-go/service/costandusagereportservice"
	"github.com/aws/aws-sdk-go/service/databasemigrationservice"
	"github.com/aws/aws-sdk-go/service/datapipeline"
	"github.com/aws/aws-sdk-go/service/datasync"
	"github.com/aws/aws-sdk-go/service/dax"
	"github.com/aws/aws-sdk-go/service/devicefarm"
	"github.com/aws/aws-sdk-go/service/directconnect"
	"github.com/aws/aws-sdk-go/service/directoryservice"
	"github.com/aws/aws-sdk-go/service/dlm"
	"github.com/aws/aws-sdk-go/service/docdb"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/efs"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/aws/aws-sdk-go/service/elasticbeanstalk"
	elasticsearch "github.com/aws/aws-sdk-go/service/elasticsearchservice"
	"github.com/aws/aws-sdk-go/service/elastictranscoder"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/emr"
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/aws/aws-sdk-go/service/fms"
	"github.com/aws/aws-sdk-go/service/fsx"
	"github.com/aws/aws-sdk-go/service/gamelift"
	"github.com/aws/aws-sdk-go/service/glacier"
	"github.com/aws/aws-sdk-go/service/globalaccelerator"
	"github.com/aws/aws-sdk-go/service/glue"
	"github.com/aws/aws-sdk-go/service/guardduty"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/inspector"
	"github.com/aws/aws-sdk-go/service/iot"
	"github.com/aws/aws-sdk-go/service/kafka"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/kinesisanalytics"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/lexmodelbuildingservice"
	"github.com/aws/aws-sdk-go/service/licensemanager"
	"github.com/aws/aws-sdk-go/service/lightsail"
	"github.com/aws/aws-sdk-go/service/macie"
	"github.com/aws/aws-sdk-go/service/mediaconnect"
	"github.com/aws/aws-sdk-go/service/mediaconvert"
	"github.com/aws/aws-sdk-go/service/medialive"
	"github.com/aws/aws-sdk-go/service/mediapackage"
	"github.com/aws/aws-sdk-go/service/mediastore"
	"github.com/aws/aws-sdk-go/service/mediastoredata"
	"github.com/aws/aws-sdk-go/service/mq"
	"github.com/aws/aws-sdk-go/service/neptune"
	"github.com/aws/aws-sdk-go/service/opsworks"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/aws/aws-sdk-go/service/pinpoint"
	"github.com/aws/aws-sdk-go/service/pricing"
	"github.com/aws/aws-sdk-go/service/ram"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/redshift"
	"github.com/aws/aws-sdk-go/service/resourcegroups"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/route53resolver"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3control"
	"github.com/aws/aws-sdk-go/service/sagemaker"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/securityhub"
	"github.com/aws/aws-sdk-go/service/serverlessapplicationrepository"
	"github.com/aws/aws-sdk-go/service/servicecatalog"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/aws/aws-sdk-go/service/sfn"
	"github.com/aws/aws-sdk-go/service/simpledb"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/storagegateway"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/swf"
	"github.com/aws/aws-sdk-go/service/transfer"
	"github.com/aws/aws-sdk-go/service/waf"
	"github.com/aws/aws-sdk-go/service/wafregional"
	"github.com/aws/aws-sdk-go/service/worklink"
	"github.com/aws/aws-sdk-go/service/workspaces"
	"github.com/davecgh/go-spew/spew"
	cleanhttp "github.com/hashicorp/go-cleanhttp"
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
	AutoscalingEndpoint      string
	EcrEndpoint              string
	EfsEndpoint              string
	EsEndpoint               string
	ElbEndpoint              string
	IamEndpoint              string
	KinesisEndpoint          string
	KinesisAnalyticsEndpoint string
	KmsEndpoint              string
	LambdaEndpoint           string
	RdsEndpoint              string
	R53Endpoint              string
	S3Endpoint               string
	S3ControlEndpoint        string
	SnsEndpoint              string
	SqsEndpoint              string
	StsEndpoint              string
	SsmEndpoint              string
	Insecure                 bool

	SkipCredsValidation     bool
	SkipGetEC2Platforms     bool
	SkipRegionValidation    bool
	SkipRequestingAccountId bool
	SkipMetadataApiCheck    bool
	S3ForcePathStyle        bool
}

type AWSClient struct {
	accountid                           string
	acmconn                             *acm.ACM
	acmpcaconn                          *acmpca.ACMPCA
	apigateway                          *apigateway.APIGateway
	apigatewayv2conn                    *apigatewayv2.ApiGatewayV2
	appautoscalingconn                  *applicationautoscaling.ApplicationAutoScaling
	appmeshconn                         *appmesh.AppMesh
	appsyncconn                         *appsync.AppSync
	athenaconn                          *athena.Athena
	autoscalingconn                     *autoscaling.AutoScaling
	backupconn                          *backup.Backup
	batchconn                           *batch.Batch
	budgetconn                          *budgets.Budgets
	cfconn                              *cloudformation.CloudFormation
	cloud9conn                          *cloud9.Cloud9
	cloudfrontconn                      *cloudfront.CloudFront
	cloudhsmv2conn                      *cloudhsmv2.CloudHSMV2
	cloudsearchconn                     *cloudsearch.CloudSearch
	cloudtrailconn                      *cloudtrail.CloudTrail
	cloudwatchconn                      *cloudwatch.CloudWatch
	cloudwatcheventsconn                *cloudwatchevents.CloudWatchEvents
	cloudwatchlogsconn                  *cloudwatchlogs.CloudWatchLogs
	codebuildconn                       *codebuild.CodeBuild
	codecommitconn                      *codecommit.CodeCommit
	codedeployconn                      *codedeploy.CodeDeploy
	codepipelineconn                    *codepipeline.CodePipeline
	cognitoconn                         *cognitoidentity.CognitoIdentity
	cognitoidpconn                      *cognitoidentityprovider.CognitoIdentityProvider
	configconn                          *configservice.ConfigService
	costandusagereportconn              *costandusagereportservice.CostandUsageReportService
	datapipelineconn                    *datapipeline.DataPipeline
	datasyncconn                        *datasync.DataSync
	daxconn                             *dax.DAX
	devicefarmconn                      *devicefarm.DeviceFarm
	dlmconn                             *dlm.DLM
	dmsconn                             *databasemigrationservice.DatabaseMigrationService
	docdbconn                           *docdb.DocDB
	dsconn                              *directoryservice.DirectoryService
	dxconn                              *directconnect.DirectConnect
	dynamodbconn                        *dynamodb.DynamoDB
	ec2conn                             *ec2.EC2
	ecrconn                             *ecr.ECR
	ecsconn                             *ecs.ECS
	efsconn                             *efs.EFS
	eksconn                             *eks.EKS
	elasticacheconn                     *elasticache.ElastiCache
	elasticbeanstalkconn                *elasticbeanstalk.ElasticBeanstalk
	elastictranscoderconn               *elastictranscoder.ElasticTranscoder
	elbconn                             *elb.ELB
	elbv2conn                           *elbv2.ELBV2
	emrconn                             *emr.EMR
	esconn                              *elasticsearch.ElasticsearchService
	firehoseconn                        *firehose.Firehose
	fmsconn                             *fms.FMS
	fsxconn                             *fsx.FSx
	gameliftconn                        *gamelift.GameLift
	glacierconn                         *glacier.Glacier
	globalacceleratorconn               *globalaccelerator.GlobalAccelerator
	glueconn                            *glue.Glue
	guarddutyconn                       *guardduty.GuardDuty
	iamconn                             *iam.IAM
	inspectorconn                       *inspector.Inspector
	iotconn                             *iot.IoT
	kafkaconn                           *kafka.Kafka
	kinesisanalyticsconn                *kinesisanalytics.KinesisAnalytics
	kinesisconn                         *kinesis.Kinesis
	kmsconn                             *kms.KMS
	lambdaconn                          *lambda.Lambda
	lexmodelconn                        *lexmodelbuildingservice.LexModelBuildingService
	licensemanagerconn                  *licensemanager.LicenseManager
	lightsailconn                       *lightsail.Lightsail
	macieconn                           *macie.Macie
	mediaconnectconn                    *mediaconnect.MediaConnect
	mediaconvertconn                    *mediaconvert.MediaConvert
	medialiveconn                       *medialive.MediaLive
	mediapackageconn                    *mediapackage.MediaPackage
	mediastoreconn                      *mediastore.MediaStore
	mediastoredataconn                  *mediastoredata.MediaStoreData
	mqconn                              *mq.MQ
	neptuneconn                         *neptune.Neptune
	opsworksconn                        *opsworks.OpsWorks
	organizationsconn                   *organizations.Organizations
	partition                           string
	pinpointconn                        *pinpoint.Pinpoint
	pricingconn                         *pricing.Pricing
	r53conn                             *route53.Route53
	ramconn                             *ram.RAM
	rdsconn                             *rds.RDS
	redshiftconn                        *redshift.Redshift
	region                              string
	resourcegroupsconn                  *resourcegroups.ResourceGroups
	route53resolverconn                 *route53resolver.Route53Resolver
	s3conn                              *s3.S3
	s3controlconn                       *s3control.S3Control
	sagemakerconn                       *sagemaker.SageMaker
	scconn                              *servicecatalog.ServiceCatalog
	sdconn                              *servicediscovery.ServiceDiscovery
	secretsmanagerconn                  *secretsmanager.SecretsManager
	securityhubconn                     *securityhub.SecurityHub
	serverlessapplicationrepositoryconn *serverlessapplicationrepository.ServerlessApplicationRepository
	sesConn                             *ses.SES
	sfnconn                             *sfn.SFN
	simpledbconn                        *simpledb.SimpleDB
	snsconn                             *sns.SNS
	sqsconn                             *sqs.SQS
	ssmconn                             *ssm.SSM
	storagegatewayconn                  *storagegateway.StorageGateway
	stsconn                             *sts.STS
	supportedplatforms                  []string
	swfconn                             *swf.SWF
	transferconn                        *transfer.Transfer
	wafconn                             *waf.WAF
	wafregionalconn                     *wafregional.WAFRegional
	worklinkconn                        *worklink.WorkLink
	workspacesconn                      *workspaces.WorkSpaces
}

func (c *AWSClient) S3() *s3.S3 {
	return c.s3conn
}

func (c *AWSClient) DynamoDB() *dynamodb.DynamoDB {
	return c.dynamodbconn
}

func (c *AWSClient) IsChinaCloud() bool {
	_, isChinaCloud := endpoints.PartitionForRegion([]endpoints.Partition{endpoints.AwsCnPartition()}, c.region)
	return isChinaCloud
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
			// If a profile wasn't specified, the session may still be able to resolve credentials from shared config.
			if c.Profile == "" {
				sess, err := session.NewSession()
				if err != nil {
					return nil, errors.New(`No valid credential sources found for AWS Provider.
	Please see https://terraform.io/docs/providers/aws/index.html for more information on
	providing credentials for the AWS Provider`)
				}
				_, err = sess.Config.Credentials.Get()
				if err != nil {
					return nil, errors.New(`No valid credential sources found for AWS Provider.
	Please see https://terraform.io/docs/providers/aws/index.html for more information on
	providing credentials for the AWS Provider`)
				}
				log.Printf("[INFO] Using session-derived AWS Auth")
				opt.Config.Credentials = sess.Config.Credentials
			} else {
				log.Printf("[INFO] AWS Auth using Profile: %q", c.Profile)
				opt.Profile = c.Profile
				opt.SharedConfigState = session.SharedConfigEnable
			}
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
		return nil, fmt.Errorf("Error creating AWS session: %s", err)
	}

	sess.Handlers.Build.PushBackNamed(addTerraformVersionToUserAgent)

	if extraDebug := os.Getenv("TERRAFORM_AWS_AUTHFAILURE_DEBUG"); extraDebug != "" {
		sess.Handlers.UnmarshalError.PushFrontNamed(debugAuthFailure)
	}

	// if the desired number of retries is non-zero, update the session
	if c.MaxRetries > 0 {
		sess = sess.Copy(&aws.Config{MaxRetries: aws.Int(c.MaxRetries)})
	}

	// Generally, we want to configure a lower retry theshold for networking issues
	// as the session retry threshold is very high by default and can mask permanent
	// networking failures, such as a non-existent service endpoint.
	// MaxRetries will override this logic if it has a lower retry threshold.
	// NOTE: This logic can be fooled by other request errors raising the retry count
	//       before any networking error occurs
	sess.Handlers.Retry.PushBack(func(r *request.Request) {
		// We currently depend on the DefaultRetryer exponential backoff here.
		// ~10 retries gives a fair backoff of a few seconds.
		if r.RetryCount < 9 {
			return
		}
		// RequestError: send request failed
		// caused by: Post https://FQDN/: dial tcp: lookup FQDN: no such host
		if IsAWSErrExtended(r.Error, "RequestError", "send request failed", "no such host") {
			log.Printf("[WARN] Disabling retries after next request due to networking issue")
			r.Retryable = aws.Bool(false)
		}
		// RequestError: send request failed
		// caused by: Post https://FQDN/: dial tcp IPADDRESS:443: connect: connection refused
		if IsAWSErrExtended(r.Error, "RequestError", "send request failed", "connection refused") {
			log.Printf("[WARN] Disabling retries after next request due to networking issue")
			r.Retryable = aws.Bool(false)
		}
	})

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
	awsAutoscalingSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.AutoscalingEndpoint)})
	awsEcrSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.EcrEndpoint)})
	awsEcsSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.EcsEndpoint)})
	awsEfsSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.EfsEndpoint)})
	awsElbSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.ElbEndpoint)})
	awsEsSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.EsEndpoint)})
	awsIamSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.IamEndpoint)})
	awsLambdaSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.LambdaEndpoint)})
	awsKinesisSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.KinesisEndpoint)})
	awsKinesisAnalyticsSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.KinesisAnalyticsEndpoint)})
	awsKmsSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.KmsEndpoint)})
	awsRdsSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.RdsEndpoint)})
	awsS3Sess := sess.Copy(&aws.Config{Endpoint: aws.String(c.S3Endpoint)})
	awsS3ControlSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.S3ControlEndpoint)})
	awsSnsSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.SnsEndpoint)})
	awsSqsSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.SqsEndpoint)})
	awsStsSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.StsEndpoint)})
	awsDeviceFarmSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.DeviceFarmEndpoint)})
	awsSsmSess := sess.Copy(&aws.Config{Endpoint: aws.String(c.SsmEndpoint)})

	log.Println("[INFO] Initializing DeviceFarm SDK connection")
	client.devicefarmconn = devicefarm.New(awsDeviceFarmSess)

	// Beyond verifying credentials (if enabled), we use the next set of logic
	// to determine two pieces of information required for manually assembling
	// resource ARNs when they are not available in the service API:
	//  * client.accountid
	//  * client.partition
	client.iamconn = iam.New(awsIamSess)
	client.stsconn = sts.New(awsStsSess)

	if c.AssumeRoleARN != "" {
		client.accountid, client.partition, _ = parseAccountIDAndPartitionFromARN(c.AssumeRoleARN)
	}

	// Validate credentials early and fail before we do any graph walking.
	if !c.SkipCredsValidation {
		var err error
		client.accountid, client.partition, err = GetAccountIDAndPartitionFromSTSGetCallerIdentity(client.stsconn)
		if err != nil {
			return nil, fmt.Errorf("error validating provider credentials: %s", err)
		}
	}

	if client.accountid == "" && !c.SkipRequestingAccountId {
		var err error
		client.accountid, client.partition, err = GetAccountIDAndPartition(client.iamconn, client.stsconn, cp.ProviderName)
		if err != nil {
			// DEPRECATED: Next major version of the provider should return the error instead of logging
			//             if skip_request_account_id is not enabled.
			log.Printf("[WARN] %s", fmt.Sprintf(
				"AWS account ID not previously found and failed retrieving via all available methods. "+
					"This will return an error in the next major version of the AWS provider. "+
					"See https://www.terraform.io/docs/providers/aws/index.html#skip_requesting_account_id for workaround and implications. "+
					"Errors: %s", err))
		}
	}

	if client.accountid == "" {
		log.Printf("[WARN] AWS account ID not found for provider. See https://www.terraform.io/docs/providers/aws/index.html#skip_requesting_account_id for implications.")
	}

	authErr := c.ValidateAccountId(client.accountid)
	if authErr != nil {
		return nil, authErr
	}

	// Infer AWS partition from configured region if we still need it
	if client.partition == "" {
		if partition, ok := endpoints.PartitionForRegion(endpoints.DefaultPartitions(), client.region); ok {
			client.partition = partition.ID()
		}
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
	client.acmpcaconn = acmpca.New(sess)
	client.apigateway = apigateway.New(awsApigatewaySess)
	client.apigatewayv2conn = apigatewayv2.New(sess)
	client.appautoscalingconn = applicationautoscaling.New(sess)
	client.appmeshconn = appmesh.New(sess)
	client.appsyncconn = appsync.New(sess)
	client.athenaconn = athena.New(sess)
	client.autoscalingconn = autoscaling.New(awsAutoscalingSess)
	client.backupconn = backup.New(sess)
	client.batchconn = batch.New(sess)
	client.budgetconn = budgets.New(sess)
	client.cfconn = cloudformation.New(awsCfSess)
	client.cloud9conn = cloud9.New(sess)
	client.cloudfrontconn = cloudfront.New(sess)
	client.cloudhsmv2conn = cloudhsmv2.New(sess)
	client.cloudsearchconn = cloudsearch.New(sess)
	client.cloudtrailconn = cloudtrail.New(sess)
	client.cloudwatchconn = cloudwatch.New(awsCwSess)
	client.cloudwatcheventsconn = cloudwatchevents.New(awsCweSess)
	client.cloudwatchlogsconn = cloudwatchlogs.New(awsCwlSess)
	client.codebuildconn = codebuild.New(sess)
	client.codecommitconn = codecommit.New(sess)
	client.codedeployconn = codedeploy.New(sess)
	client.codepipelineconn = codepipeline.New(sess)
	client.cognitoconn = cognitoidentity.New(sess)
	client.cognitoidpconn = cognitoidentityprovider.New(sess)
	client.configconn = configservice.New(sess)
	client.costandusagereportconn = costandusagereportservice.New(sess)
	client.datapipelineconn = datapipeline.New(sess)
	client.datasyncconn = datasync.New(sess)
	client.daxconn = dax.New(awsDynamoSess)
	client.dlmconn = dlm.New(sess)
	client.dmsconn = databasemigrationservice.New(sess)
	client.docdbconn = docdb.New(sess)
	client.dsconn = directoryservice.New(sess)
	client.dxconn = directconnect.New(sess)
	client.dynamodbconn = dynamodb.New(awsDynamoSess)
	client.ecrconn = ecr.New(awsEcrSess)
	client.ecsconn = ecs.New(awsEcsSess)
	client.efsconn = efs.New(awsEfsSess)
	client.eksconn = eks.New(sess)
	client.elasticacheconn = elasticache.New(sess)
	client.elasticbeanstalkconn = elasticbeanstalk.New(sess)
	client.elastictranscoderconn = elastictranscoder.New(sess)
	client.elbconn = elb.New(awsElbSess)
	client.elbv2conn = elbv2.New(awsElbSess)
	client.emrconn = emr.New(sess)
	client.esconn = elasticsearch.New(awsEsSess)
	client.firehoseconn = firehose.New(sess)
	client.fmsconn = fms.New(sess)
	client.fsxconn = fsx.New(sess)
	client.gameliftconn = gamelift.New(sess)
	client.glacierconn = glacier.New(sess)
	client.globalacceleratorconn = globalaccelerator.New(sess)
	client.glueconn = glue.New(sess)
	client.guarddutyconn = guardduty.New(sess)
	client.inspectorconn = inspector.New(sess)
	client.iotconn = iot.New(sess)
	client.kafkaconn = kafka.New(sess)
	client.kinesisanalyticsconn = kinesisanalytics.New(awsKinesisAnalyticsSess)
	client.kinesisconn = kinesis.New(awsKinesisSess)
	client.kmsconn = kms.New(awsKmsSess)
	client.lambdaconn = lambda.New(awsLambdaSess)
	client.lexmodelconn = lexmodelbuildingservice.New(sess)
	client.licensemanagerconn = licensemanager.New(sess)
	client.lightsailconn = lightsail.New(sess)
	client.macieconn = macie.New(sess)
	client.mediaconnectconn = mediaconnect.New(sess)
	client.mediaconvertconn = mediaconvert.New(sess)
	client.medialiveconn = medialive.New(sess)
	client.mediapackageconn = mediapackage.New(sess)
	client.mediastoreconn = mediastore.New(sess)
	client.mediastoredataconn = mediastoredata.New(sess)
	client.mqconn = mq.New(sess)
	client.neptuneconn = neptune.New(sess)
	client.neptuneconn = neptune.New(sess)
	client.opsworksconn = opsworks.New(sess)
	client.organizationsconn = organizations.New(sess)
	client.pinpointconn = pinpoint.New(sess)
	client.pricingconn = pricing.New(sess)
	client.r53conn = route53.New(r53Sess)
	client.ramconn = ram.New(sess)
	client.rdsconn = rds.New(awsRdsSess)
	client.redshiftconn = redshift.New(sess)
	client.resourcegroupsconn = resourcegroups.New(sess)
	client.route53resolverconn = route53resolver.New(sess)
	client.s3conn = s3.New(awsS3Sess)
	client.s3controlconn = s3control.New(awsS3ControlSess)
	client.sagemakerconn = sagemaker.New(sess)
	client.scconn = servicecatalog.New(sess)
	client.sdconn = servicediscovery.New(sess)
	client.secretsmanagerconn = secretsmanager.New(sess)
	client.securityhubconn = securityhub.New(sess)
	client.serverlessapplicationrepositoryconn = serverlessapplicationrepository.New(sess)
	client.sesConn = ses.New(sess)
	client.sfnconn = sfn.New(sess)
	client.simpledbconn = simpledb.New(sess)
	client.snsconn = sns.New(awsSnsSess)
	client.sqsconn = sqs.New(awsSqsSess)
	client.ssmconn = ssm.New(awsSsmSess)
	client.storagegatewayconn = storagegateway.New(sess)
	client.swfconn = swf.New(sess)
	client.transferconn = transfer.New(sess)
	client.wafconn = waf.New(sess)
	client.wafregionalconn = wafregional.New(sess)
	client.worklinkconn = worklink.New(sess)
	client.workspacesconn = workspaces.New(sess)

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

	// See https://github.com/aws/aws-sdk-go/pull/1276
	client.dynamodbconn.Handlers.Retry.PushBack(func(r *request.Request) {
		if r.Operation.Name != "PutItem" && r.Operation.Name != "UpdateItem" && r.Operation.Name != "DeleteItem" {
			return
		}
		if isAWSErr(r.Error, dynamodb.ErrCodeLimitExceededException, "Subscriber limit exceeded:") {
			r.Retryable = aws.Bool(true)
		}
	})

	client.kinesisconn.Handlers.Retry.PushBack(func(r *request.Request) {
		if r.Operation.Name == "CreateStream" {
			if isAWSErr(r.Error, kinesis.ErrCodeLimitExceededException, "simultaneously be in CREATING or DELETING") {
				r.Retryable = aws.Bool(true)
			}
		}
		if r.Operation.Name == "CreateStream" || r.Operation.Name == "DeleteStream" {
			if isAWSErr(r.Error, kinesis.ErrCodeLimitExceededException, "Rate exceeded for stream") {
				r.Retryable = aws.Bool(true)
			}
		}
	})

	client.storagegatewayconn.Handlers.Retry.PushBack(func(r *request.Request) {
		// InvalidGatewayRequestException: The specified gateway proxy network connection is busy.
		if isAWSErr(r.Error, storagegateway.ErrCodeInvalidGatewayRequestException, "The specified gateway proxy network connection is busy") {
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
	for _, partition := range endpoints.DefaultPartitions() {
		for _, region := range partition.Regions() {
			if c.Region == region.ID() {
				return nil
			}
		}
	}

	return fmt.Errorf("Not a valid region: %s", c.Region)
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
