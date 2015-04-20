package aws

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/autoscaling"
	"github.com/awslabs/aws-sdk-go/service/ec2"
	"github.com/awslabs/aws-sdk-go/service/elb"
	"github.com/awslabs/aws-sdk-go/service/iam"
	"github.com/awslabs/aws-sdk-go/service/rds"
	"github.com/awslabs/aws-sdk-go/service/route53"
	"github.com/awslabs/aws-sdk-go/service/s3"

	"github.com/hashicorp/terraform/helper/multierror"
)

type Config struct {
	AccessKey              string
	SecretKey              string
	Token                  string
	CredentialsFilePath    string
	CredentialsFileProfile string
	Region                 string
	Provider               aws.CredentialsProvider
}

type AWSClient struct {
	ec2conn         *ec2.EC2
	elbconn         *elb.ELB
	autoscalingconn *autoscaling.AutoScaling
	s3conn          *s3.S3
	r53conn         *route53.Route53
	region          string
	rdsconn         *rds.RDS
	iamconn         *iam.IAM
}

func (c *Config) loadAndValidate(providerCode string) (interface{}, error) {
	c.tryLoadingDeprecatedEnvVars()
	credsProvider, err := c.getCredsProvider(providerCode)
	if err != nil {
		return nil, err
	}

	if _, err := credsProvider.Credentials(); err != nil {
		return nil, err
	}

	c.Provider = credsProvider

	return c.Client()
}

func (c *Config) tryLoadingDeprecatedEnvVars() {
	// Backward compatibility
	if c.Token == "" {
		c.Token = os.Getenv("AWS_SECURITY_TOKEN")
	}
	if c.CredentialsFilePath == "" {
		c.CredentialsFilePath = os.Getenv("AWS_CREDENTIAL_FILE")
	}
	if c.CredentialsFileProfile == "" {
		c.CredentialsFileProfile = os.Getenv("AWS_PROFILE")
	}
}

func (c *Config) getCredsProvider(providerCode string) (aws.CredentialsProvider, error) {
	switch providerCode {
	case "static":
		log.Println("[INFO] Loading static credentials")
		return aws.Creds(c.AccessKey, c.SecretKey, c.Token), nil
	case "iam":
		log.Println("[INFO] Loading credentials via IAM")
		return aws.IAMCreds(), nil
	case "env":
		log.Println("[INFO] Loading credentials from ENV variables")
		return aws.EnvCreds()
	case "file":
		log.Printf("[INFO] Loading credentials from config file at %s",
			c.CredentialsFilePath)
		// TODO: Could be a variable but there's no standardized name for it
		// More importantly, what is really the point of this variable??
		expiry := 10 * time.Minute

		return aws.ProfileCreds(
			c.CredentialsFilePath, c.CredentialsFileProfile, expiry)
	}

	log.Println("[INFO] Loading credentials automagically via AWS library")
	return aws.DetectCreds(c.AccessKey, c.SecretKey, c.Token), nil
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
		awsConfig := &aws.Config{
			Credentials: c.Provider,
			Region:      c.Region,
		}

		log.Println("[INFO] Initializing ELB connection")
		client.elbconn = elb.New(awsConfig)

		log.Println("[INFO] Initializing S3 connection")
		client.s3conn = s3.New(awsConfig)

		log.Println("[INFO] Initializing RDS Connection")
		client.rdsconn = rds.New(awsConfig)

		log.Println("[INFO] Initializing IAM Connection")
		client.iamconn = iam.New(awsConfig)
		log.Println("[INFO] Initializing AutoScaling connection")
		client.autoscalingconn = autoscaling.New(awsConfig)

		log.Println("[INFO] Initializing EC2 Connection")
		client.ec2conn = ec2.New(awsConfig)

		// aws-sdk-go uses v4 for signing requests, which requires all global
		// endpoints to use 'us-east-1'.
		// See http://docs.aws.amazon.com/general/latest/gr/sigv4_changes.html
		log.Println("[INFO] Initializing Route 53 connection")
		client.r53conn = route53.New(&aws.Config{
			Credentials: c.Provider,
			Region:      "us-east-1",
		})
	}

	if len(errs) > 0 {
		return nil, &multierror.Error{Errors: errs}
	}

	return &client, nil
}

// IsValidRegion returns true if the configured region is a valid AWS
// region and false if it's not
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
