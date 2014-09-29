package aws

import (
	"log"

	"github.com/hashicorp/terraform/helper/config"
	"github.com/hashicorp/terraform/helper/multierror"
	"github.com/hashicorp/terraform/helper/schema"
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

	// This is the schema.Provider. Eventually this will replace much
	// of this structure. For now it is an element of it for compatiblity.
	p *schema.Provider
}

func (p *ResourceProvider) Input(
	input terraform.UIInput,
	c *terraform.ResourceConfig) (*terraform.ResourceConfig, error) {
	return Provider().Input(input, c)
}

func (p *ResourceProvider) Validate(c *terraform.ResourceConfig) ([]string, []error) {
	return Provider().Validate(c)
}

func (p *ResourceProvider) ValidateResource(
	t string, c *terraform.ResourceConfig) ([]string, []error) {
	prov := Provider()
	if _, ok := prov.ResourcesMap[t]; ok {
		return prov.ValidateResource(t, c)
	}

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

	// Create the provider, set the meta
	p.p = Provider()
	p.p.SetMeta(p)

	return nil
}

func (p *ResourceProvider) Apply(
	info *terraform.InstanceInfo,
	s *terraform.InstanceState,
	d *terraform.InstanceDiff) (*terraform.InstanceState, error) {
	if _, ok := p.p.ResourcesMap[info.Type]; ok {
		return p.p.Apply(info, s, d)
	}

	return resourceMap.Apply(info, s, d, p)
}

func (p *ResourceProvider) Diff(
	info *terraform.InstanceInfo,
	s *terraform.InstanceState,
	c *terraform.ResourceConfig) (*terraform.InstanceDiff, error) {
	if _, ok := p.p.ResourcesMap[info.Type]; ok {
		return p.p.Diff(info, s, c)
	}

	return resourceMap.Diff(info, s, c, p)
}

func (p *ResourceProvider) Refresh(
	info *terraform.InstanceInfo,
	s *terraform.InstanceState) (*terraform.InstanceState, error) {
	if _, ok := p.p.ResourcesMap[info.Type]; ok {
		return p.p.Refresh(info, s)
	}

	return resourceMap.Refresh(info, s, p)
}

func (p *ResourceProvider) Resources() []terraform.ResourceType {
	result := resourceMap.Resources()
	result = append(result, Provider().Resources()...)
	return result
}
