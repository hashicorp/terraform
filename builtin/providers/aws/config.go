package aws

import (
	"fmt"
	"log"
	"strings"
	"unicode"

	"github.com/hashicorp/terraform/helper/multierror"
	"github.com/mitchellh/goamz/autoscaling"
	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/ec2"
	"github.com/mitchellh/goamz/elb"
	"github.com/mitchellh/goamz/rds"
	"github.com/mitchellh/goamz/route53"
	"github.com/mitchellh/goamz/s3"
)

type Config struct {
	AccessKey string
	SecretKey string
	Region    string
}

type AWSClient struct {
	ec2conn         *ec2.EC2
	elbconn         *elb.ELB
	autoscalingconn *autoscaling.AutoScaling
	s3conn          *s3.S3
	rdsconn         *rds.Rds
	route53         *route53.Route53
}

// Client configures and returns a fully initailized AWSClient
func (c *Config) Client() (interface{}, error) {
	var client AWSClient

	// Get the auth and region. This can fail if keys/regions were not
	// specified and we're attempting to use the environment.
	var errs []error
	log.Println("[INFO] Building AWS auth structure")
	auth, err := c.AWSAuth()
	if err != nil {
		errs = append(errs, err)
	}

	log.Println("[INFO] Building AWS region structure")
	region, err := c.AWSRegion()
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) == 0 {
		log.Println("[INFO] Initializing EC2 connection")
		client.ec2conn = ec2.New(auth, region)
		log.Println("[INFO] Initializing ELB connection")
		client.elbconn = elb.New(auth, region)
		log.Println("[INFO] Initializing AutoScaling connection")
		client.autoscalingconn = autoscaling.New(auth, region)
		log.Println("[INFO] Initializing S3 connection")
		client.s3conn = s3.New(auth, region)
		log.Println("[INFO] Initializing RDS connection")
		client.rdsconn = rds.New(auth, region)
		log.Println("[INFO] Initializing Route53 connection")
		client.route53 = route53.New(auth, region)
	}

	if len(errs) > 0 {
		return nil, &multierror.Error{Errors: errs}
	}

	return &client, nil
}

// AWSAuth returns a valid aws.Auth object for access to AWS services, or
// an error if the authentication couldn't be resolved.
//
// TODO(mitchellh): Test in some way.
func (c *Config) AWSAuth() (aws.Auth, error) {
	auth, err := aws.GetAuth(c.AccessKey, c.SecretKey)
	if err == nil {
		// Store the accesskey and secret that we got...
		c.AccessKey = auth.AccessKey
		c.SecretKey = auth.SecretKey
	}

	return auth, err
}

// IsValidRegion returns true if the configured region is a valid AWS
// region and false if it's not
func (c *Config) IsValidRegion() bool {
	var regions = [11]string{"us-east-1", "us-west-2", "us-west-1", "eu-west-1",
		"eu-central-1", "ap-southeast-1", "ap-southeast-2", "ap-northeast-1",
		"sa-east-1", "cn-north-1", "us-gov-west-1"}

	for _, valid := range regions {
		if c.Region == valid {
			return true
		}
	}
	return false
}

// AWSRegion returns the configured region.
//
// TODO(mitchellh): Test in some way.
func (c *Config) AWSRegion() (aws.Region, error) {
	if c.Region != "" {
		if c.IsValidRegion() {
			return aws.Regions[c.Region], nil
		} else {
			return aws.Region{}, fmt.Errorf("Not a valid region: %s", c.Region)
		}
	}

	md, err := aws.GetMetaData("placement/availability-zone")
	if err != nil {
		return aws.Region{}, err
	}

	region := strings.TrimRightFunc(string(md), unicode.IsLetter)
	return aws.Regions[region], nil
}
