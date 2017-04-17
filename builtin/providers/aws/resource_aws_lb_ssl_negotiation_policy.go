package aws

import (
	"bytes"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsLBSSLNegotiationPolicy() *schema.Resource {
	return &schema.Resource{
		// There is no concept of "updating" an LB policy in
		// the AWS API.
		Create: resourceAwsLBSSLNegotiationPolicyCreate,
		Read:   resourceAwsLBSSLNegotiationPolicyRead,
		Delete: resourceAwsLBSSLNegotiationPolicyDelete,

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

			"attribute": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"value": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Set: func(v interface{}) int {
					var buf bytes.Buffer
					m := v.(map[string]interface{})
					buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
					return hashcode.String(buf.String())
				},
			},
		},
	}
}

func resourceAwsLBSSLNegotiationPolicyCreate(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbconn

	// Provision the SSLNegotiationPolicy
	lbspOpts := &elb.CreateLoadBalancerPolicyInput{
		LoadBalancerName: aws.String(d.Get("load_balancer").(string)),
		PolicyName:       aws.String(d.Get("name").(string)),
		PolicyTypeName:   aws.String("SSLNegotiationPolicyType"),
	}

	// Check for Policy Attributes
	if v, ok := d.GetOk("attribute"); ok {
		var err error
		// Expand the "attribute" set to aws-sdk-go compat []*elb.PolicyAttribute
		lbspOpts.PolicyAttributes, err = expandPolicyAttributes(v.(*schema.Set).List())
		if err != nil {
			return err
		}
	}

	log.Printf("[DEBUG] Load Balancer Policy opts: %#v", lbspOpts)
	if _, err := elbconn.CreateLoadBalancerPolicy(lbspOpts); err != nil {
		return fmt.Errorf("Error creating Load Balancer Policy: %s", err)
	}

	setLoadBalancerOpts := &elb.SetLoadBalancerPoliciesOfListenerInput{
		LoadBalancerName: aws.String(d.Get("load_balancer").(string)),
		LoadBalancerPort: aws.Int64(int64(d.Get("lb_port").(int))),
		PolicyNames:      []*string{aws.String(d.Get("name").(string))},
	}

	log.Printf("[DEBUG] SSL Negotiation create configuration: %#v", setLoadBalancerOpts)
	if _, err := elbconn.SetLoadBalancerPoliciesOfListener(setLoadBalancerOpts); err != nil {
		return fmt.Errorf("Error setting SSLNegotiationPolicy: %s", err)
	}

	d.SetId(fmt.Sprintf("%s:%d:%s",
		*lbspOpts.LoadBalancerName,
		*setLoadBalancerOpts.LoadBalancerPort,
		*lbspOpts.PolicyName))
	return nil
}

func resourceAwsLBSSLNegotiationPolicyRead(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbconn

	lbName, lbPort, policyName := resourceAwsLBSSLNegotiationPolicyParseId(d.Id())

	request := &elb.DescribeLoadBalancerPoliciesInput{
		LoadBalancerName: aws.String(lbName),
		PolicyNames:      []*string{aws.String(policyName)},
	}

	getResp, err := elbconn.DescribeLoadBalancerPolicies(request)
	if err != nil {
		if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "PolicyNotFound" {
			// The policy is gone.
			d.SetId("")
			return nil
		} else if isLoadBalancerNotFound(err) {
			// The ELB is gone now, so just remove it from the state
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error retrieving policy: %s", err)
	}

	if len(getResp.PolicyDescriptions) != 1 {
		return fmt.Errorf("Unable to find policy %#v", getResp.PolicyDescriptions)
	}

	// We can get away with this because there's only one policy returned
	policyDesc := getResp.PolicyDescriptions[0]
	attributes := flattenPolicyAttributes(policyDesc.PolicyAttributeDescriptions)
	d.Set("attributes", attributes)

	d.Set("name", policyName)
	d.Set("load_balancer", lbName)
	d.Set("lb_port", lbPort)

	return nil
}

func resourceAwsLBSSLNegotiationPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbconn

	lbName, _, policyName := resourceAwsLBSSLNegotiationPolicyParseId(d.Id())

	// Perversely, if we Set an empty list of PolicyNames, we detach the
	// policies attached to a listener, which is required to delete the
	// policy itself.
	setLoadBalancerOpts := &elb.SetLoadBalancerPoliciesOfListenerInput{
		LoadBalancerName: aws.String(d.Get("load_balancer").(string)),
		LoadBalancerPort: aws.Int64(int64(d.Get("lb_port").(int))),
		PolicyNames:      []*string{},
	}

	if _, err := elbconn.SetLoadBalancerPoliciesOfListener(setLoadBalancerOpts); err != nil {
		return fmt.Errorf("Error removing SSLNegotiationPolicy: %s", err)
	}

	request := &elb.DeleteLoadBalancerPolicyInput{
		LoadBalancerName: aws.String(lbName),
		PolicyName:       aws.String(policyName),
	}

	if _, err := elbconn.DeleteLoadBalancerPolicy(request); err != nil {
		return fmt.Errorf("Error deleting SSL negotiation policy %s: %s", d.Id(), err)
	}
	return nil
}

// resourceAwsLBSSLNegotiationPolicyParseId takes an ID and parses it into
// it's constituent parts. You need three axes (LB name, policy name, and LB
// port) to create or identify an SSL negotiation policy in AWS's API.
func resourceAwsLBSSLNegotiationPolicyParseId(id string) (string, string, string) {
	parts := strings.SplitN(id, ":", 3)
	return parts[0], parts[1], parts[2]
}
