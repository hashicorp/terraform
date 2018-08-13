package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsLBCookieStickinessPolicy() *schema.Resource {
	return &schema.Resource{
		// There is no concept of "updating" an LB Stickiness policy in
		// the AWS API.
		Create: resourceAwsLBCookieStickinessPolicyCreate,
		Read:   resourceAwsLBCookieStickinessPolicyRead,
		Delete: resourceAwsLBCookieStickinessPolicyDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"load_balancer": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"lb_port": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},

			"cookie_expiration_period": &schema.Schema{
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.IntAtLeast(1),
			},
		},
	}
}

func resourceAwsLBCookieStickinessPolicyCreate(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbconn

	// Provision the LBStickinessPolicy
	lbspOpts := &elb.CreateLBCookieStickinessPolicyInput{
		LoadBalancerName: aws.String(d.Get("load_balancer").(string)),
		PolicyName:       aws.String(d.Get("name").(string)),
	}

	if v := d.Get("cookie_expiration_period").(int); v > 0 {
		lbspOpts.CookieExpirationPeriod = aws.Int64(int64(v))
	}

	log.Printf("[DEBUG] LB Cookie Stickiness Policy opts: %#v", lbspOpts)
	if _, err := elbconn.CreateLBCookieStickinessPolicy(lbspOpts); err != nil {
		return fmt.Errorf("Error creating LBCookieStickinessPolicy: %s", err)
	}

	setLoadBalancerOpts := &elb.SetLoadBalancerPoliciesOfListenerInput{
		LoadBalancerName: aws.String(d.Get("load_balancer").(string)),
		LoadBalancerPort: aws.Int64(int64(d.Get("lb_port").(int))),
		PolicyNames:      []*string{aws.String(d.Get("name").(string))},
	}

	log.Printf("[DEBUG] LB Cookie Stickiness create configuration: %#v", setLoadBalancerOpts)
	if _, err := elbconn.SetLoadBalancerPoliciesOfListener(setLoadBalancerOpts); err != nil {
		return fmt.Errorf("Error setting LBCookieStickinessPolicy: %s", err)
	}

	d.SetId(fmt.Sprintf("%s:%d:%s",
		*lbspOpts.LoadBalancerName,
		*setLoadBalancerOpts.LoadBalancerPort,
		*lbspOpts.PolicyName))
	return nil
}

func resourceAwsLBCookieStickinessPolicyRead(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbconn

	lbName, lbPort, policyName := resourceAwsLBCookieStickinessPolicyParseId(d.Id())

	request := &elb.DescribeLoadBalancerPoliciesInput{
		LoadBalancerName: aws.String(lbName),
		PolicyNames:      []*string{aws.String(policyName)},
	}

	getResp, err := elbconn.DescribeLoadBalancerPolicies(request)
	if err != nil {
		if ec2err, ok := err.(awserr.Error); ok {
			if ec2err.Code() == "PolicyNotFound" || ec2err.Code() == "LoadBalancerNotFound" {
				d.SetId("")
			}
			return nil
		}
		return fmt.Errorf("Error retrieving policy: %s", err)
	}

	if len(getResp.PolicyDescriptions) != 1 {
		return fmt.Errorf("Unable to find policy %#v", getResp.PolicyDescriptions)
	}

	// we know the policy exists now, but we have to check if it's assigned to a listener
	assigned, err := resourceAwsELBSticknessPolicyAssigned(policyName, lbName, lbPort, elbconn)
	if err != nil {
		return err
	}
	if !assigned {
		// policy exists, but isn't assigned to a listener
		log.Printf("[DEBUG] policy '%s' exists, but isn't assigned to a listener", policyName)
		d.SetId("")
		return nil
	}

	// We can get away with this because there's only one attribute, the
	// cookie expiration, in these descriptions.
	policyDesc := getResp.PolicyDescriptions[0]
	cookieAttr := policyDesc.PolicyAttributeDescriptions[0]
	if *cookieAttr.AttributeName != "CookieExpirationPeriod" {
		return fmt.Errorf("Unable to find cookie expiration period.")
	}
	d.Set("cookie_expiration_period", cookieAttr.AttributeValue)

	d.Set("name", policyName)
	d.Set("load_balancer", lbName)
	d.Set("lb_port", lbPort)

	return nil
}

func resourceAwsLBCookieStickinessPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbconn

	lbName, _, policyName := resourceAwsLBCookieStickinessPolicyParseId(d.Id())

	// Perversely, if we Set an empty list of PolicyNames, we detach the
	// policies attached to a listener, which is required to delete the
	// policy itself.
	setLoadBalancerOpts := &elb.SetLoadBalancerPoliciesOfListenerInput{
		LoadBalancerName: aws.String(d.Get("load_balancer").(string)),
		LoadBalancerPort: aws.Int64(int64(d.Get("lb_port").(int))),
		PolicyNames:      []*string{},
	}

	if _, err := elbconn.SetLoadBalancerPoliciesOfListener(setLoadBalancerOpts); err != nil {
		return fmt.Errorf("Error removing LBCookieStickinessPolicy: %s", err)
	}

	request := &elb.DeleteLoadBalancerPolicyInput{
		LoadBalancerName: aws.String(lbName),
		PolicyName:       aws.String(policyName),
	}

	if _, err := elbconn.DeleteLoadBalancerPolicy(request); err != nil {
		return fmt.Errorf("Error deleting LB stickiness policy %s: %s", d.Id(), err)
	}
	return nil
}

// resourceAwsLBCookieStickinessPolicyParseId takes an ID and parses it into
// it's constituent parts. You need three axes (LB name, policy name, and LB
// port) to create or identify a stickiness policy in AWS's API.
func resourceAwsLBCookieStickinessPolicyParseId(id string) (string, string, string) {
	parts := strings.SplitN(id, ":", 3)
	return parts[0], parts[1], parts[2]
}
