package aws

import (
	"log"
	"os"

	"github.com/hashicorp/terraform/helper/config"
	"github.com/hashicorp/terraform/helper/multierror"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/autoscaling"
	"github.com/mitchellh/goamz/ec2"
	"github.com/mitchellh/goamz/elb"
	"github.com/mitchellh/goamz/rds"
	"github.com/mitchellh/goamz/route53"
	"github.com/mitchellh/goamz/s3"
)

type ResourceProvider struct {
	Config Config

	ec2conn         *ec2.EC2
	elbconn         *elb.ELB
	autoscalingconn *autoscaling.AutoScaling
	s3conn          *s3.S3
	rdsconn         *rds.Rds
	route53         *route53.Route53
}

func (p *ResourceProvider) Validate(c *terraform.ResourceConfig) ([]string, []error) {
	type param struct {
		env string
		key string
	}
	params := []param{
		{"AWS_REGION", "region"},
		{"AWS_ACCESS_KEY", "access_key"},
		{"AWS_SECRET_KEY", "secret_key"},
	}

	var optional []string
	var required []string
	for _, p := range params {
		if v := os.Getenv(p.env); v != "" {
			optional = append(optional, p.key)
		} else {
			required = append(required, p.key)
		}
	}

	v := &config.Validator{
		Required: required,
		Optional: optional,
	}
	return v.Validate(c)
}

func (p *ResourceProvider) ValidateResource(
	t string, c *terraform.ResourceConfig) ([]string, []error) {
	return resourceMap.Validate(t, c)
}

func (p *ResourceProvider) Configure(c *terraform.ResourceConfig) error {
	if _, err := config.Decode(&p.Config, c.Config); err != nil {
		return err
	}

	// Get the auth and region. This can fail if keys/regions were not
	// specified and we're attempting to use the environment.
	var errs []error
	log.Println("[INFO] Building AWS auth structure")
	auth, err := p.Config.AWSAuth()
	if err != nil {
		errs = append(errs, err)
	}

	log.Println("[INFO] Building AWS region structure")
	region, err := p.Config.AWSRegion()
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) == 0 {
		log.Println("[INFO] Initializing EC2 connection")
		p.ec2conn = ec2.New(auth, region)
		log.Println("[INFO] Initializing ELB connection")
		p.elbconn = elb.New(auth, region)
		log.Println("[INFO] Initializing AutoScaling connection")
		p.autoscalingconn = autoscaling.New(auth, region)
		log.Println("[INFO] Initializing S3 connection")
		p.s3conn = s3.New(auth, region)
		log.Println("[INFO] Initializing RDS connection")
		p.rdsconn = rds.New(auth, region)
		log.Println("[INFO] Initializing Route53 connection")
		p.route53 = route53.New(auth, region)
	}

	if len(errs) > 0 {
		return &multierror.Error{Errors: errs}
	}

	return nil
}

func (p *ResourceProvider) Apply(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff) (*terraform.ResourceState, error) {
	return resourceMap.Apply(s, d, p)
}

func (p *ResourceProvider) Diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig) (*terraform.ResourceDiff, error) {
	return resourceMap.Diff(s, c, p)
}

func (p *ResourceProvider) Refresh(
	s *terraform.ResourceState) (*terraform.ResourceState, error) {
	return resourceMap.Refresh(s, p)
}

func (p *ResourceProvider) Resources() []terraform.ResourceType {
	return resourceMap.Resources()
}
