package aws

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/multierror"

	"github.com/hashicorp/aws-sdk-go/aws"
	"github.com/hashicorp/aws-sdk-go/gen/autoscaling"
	"github.com/hashicorp/aws-sdk-go/gen/ec2"
	"github.com/hashicorp/aws-sdk-go/gen/elb"
	"github.com/hashicorp/aws-sdk-go/gen/rds"
	"github.com/hashicorp/aws-sdk-go/gen/route53"
	"github.com/hashicorp/aws-sdk-go/gen/s3"
)

type Config struct {
	AccessKey string
	SecretKey string
	Token     string
	Region    string
}

type AWSClient struct {
	ec2conn         *ec2.EC2
	elbconn         *elb.ELB
	autoscalingconn *autoscaling.AutoScaling
	s3conn          *s3.S3
	r53conn         *route53.Route53
	region          string
	rdsconn         *rds.RDS
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
		creds := aws.Creds(c.AccessKey, c.SecretKey, c.Token)

		log.Println("[INFO] Initializing ELB connection")
		client.elbconn = elb.New(creds, c.Region, nil)
		log.Println("[INFO] Initializing AutoScaling connection")
		client.autoscalingconn = autoscaling.New(creds, c.Region, nil)
		log.Println("[INFO] Initializing S3 connection")
		client.s3conn = s3.New(creds, c.Region, nil)
		log.Println("[INFO] Initializing RDS connection")
		client.rdsconn = rds.New(creds, c.Region, nil)

		// aws-sdk-go uses v4 for signing requests, which requires all global
		// endpoints to use 'us-east-1'.
		// See http://docs.aws.amazon.com/general/latest/gr/sigv4_changes.html
		log.Println("[INFO] Initializing Route53 connection")
		client.r53conn = route53.New(creds, "us-east-1", nil)
		log.Println("[INFO] Initializing EC2 Connection")
		client.ec2conn = ec2.New(creds, c.Region, nil)
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
