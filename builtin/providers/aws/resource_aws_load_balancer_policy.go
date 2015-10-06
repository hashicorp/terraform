package aws

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsLoadBalancerPolicy() *schema.Resource {
	return &schema.Resource{
		// There is no concept of "updating" a Load Balancer Policy in
		// the AWS API.
		Create: resourceAwsLoadBalancerPolicyCreate,

		Read:   resourceAwsLoadBalancerPolicyRead,
		Delete: resourceAwsLoadBalancerPolicyDelete,

		Schema: map[string]*schema.Schema{
			"load_balancer_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"instance_port": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
				ForceNew: true,
			},

			"policy_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"policy_type_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"policy_attribute": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
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
				ForceNew: true,
				Set:      resourceAwsLoadBalancerPolicyAttributeHash,
			},
		},
	}
}

func resourceAwsLoadBalancerPolicyAttributeHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	if v, ok := m["name"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}

	if v, ok := m["value"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}

	return hashcode.String(buf.String())
}

func resourceAwsLoadBalancerPolicyCreate(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbconn

	attributes := []*elb.PolicyAttribute{}
	if attributedata, ok := d.GetOk("policy_attribute"); ok {
		attributeSet := attributedata.(*schema.Set)
		for _, attribute := range attributeSet.List() {
			attr := attribute.(map[string]interface{})
			attributes = append(attributes, &elb.PolicyAttribute{
				AttributeName:  aws.String(attr["name"].(string)),
				AttributeValue: aws.String(attr["value"].(string)),
			})
		}
	}

	// Provision the LoadBalancerPolicy
	lbspOpts := &elb.CreateLoadBalancerPolicyInput{
		LoadBalancerName: aws.String(d.Get("load_balancer_name").(string)),
		PolicyName:       aws.String(d.Get("policy_name").(string)),
		PolicyTypeName:   aws.String(d.Get("policy_type_name").(string)),
		PolicyAttributes: attributes,
	}

	if _, err := elbconn.CreateLoadBalancerPolicy(lbspOpts); err != nil {
		return fmt.Errorf("Error creating LoadBalancerPolicy: %s", err)
	}

	instancePort := int64(0)
	if v, ok := d.GetOk("instance_port"); ok {
		instancePort = int64(v.(int))
	}
	policyTypeName := lbspOpts.PolicyTypeName

	if (instancePort > 0) && (*policyTypeName == "BackendServerAuthenticationPolicyType") {
		dlbOpts := &elb.DescribeLoadBalancersInput{
			LoadBalancerNames: []*string{lbspOpts.LoadBalancerName},
		}

		lbDesc, err := elbconn.DescribeLoadBalancers(dlbOpts)
		if err != nil {
			return fmt.Errorf("Error retrieving ELB: %s", err)
		}

		policyNames := []*string{}

		for _, backendServerDesc := range lbDesc.LoadBalancerDescriptions[0].BackendServerDescriptions {
			foundInstancePort := backendServerDesc.InstancePort
			if instancePort == *foundInstancePort {
				policyNames = append(backendServerDesc.PolicyNames, lbspOpts.PolicyName)
			}
		}

		if len(policyNames) == 0 {
			policyNames = append(policyNames, lbspOpts.PolicyName)
		}

		lbpbsOpts := &elb.SetLoadBalancerPoliciesForBackendServerInput{
			InstancePort:     aws.Int64(int64(instancePort)),
			LoadBalancerName: lbspOpts.LoadBalancerName,
			PolicyNames:      policyNames,
		}

		if _, err := elbconn.SetLoadBalancerPoliciesForBackendServer(lbpbsOpts); err != nil {
			return fmt.Errorf("Error setting BackendServerAuthenticationPolicyType: %s", err)
		}
	}

	d.SetId(fmt.Sprintf("%s:%d:%s",
		*lbspOpts.LoadBalancerName,
		instancePort,
		*lbspOpts.PolicyName))
	return nil
}

func resourceAwsLoadBalancerPolicyRead(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbconn

	lbName, _, policyName := resourceAwsLoadBalancerPolicyParseId(d.Id())

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
		}
		return fmt.Errorf("Error retrieving policy: %s", err)
	}

	if len(getResp.PolicyDescriptions) != 1 {
		return fmt.Errorf("Unable to find policy %#v", getResp.PolicyDescriptions)
	}

	policyDesc := getResp.PolicyDescriptions[0]
	policyTypeName := policyDesc.PolicyTypeName

	d.Set("policy_name", policyName)
	d.Set("policy_type_name", policyTypeName)
	d.Set("load_balancer_name", lbName)

	return nil
}

func resourceAwsLoadBalancerPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbconn

	lbName, _, policyName := resourceAwsLoadBalancerPolicyParseId(d.Id())

	instancePortInt := d.Get("instance_port")
	if instancePortInt == 0 {
		instancePortInt = 1
	}

	policyTypeName := d.Get("policy_type_name")

	if policyTypeName == "BackendServerAuthenticationPolicyType" {
		policyNames := []*string{}

		lbpbsOpts := &elb.SetLoadBalancerPoliciesForBackendServerInput{
			InstancePort:     aws.Int64(int64(instancePortInt.(int))),
			LoadBalancerName: aws.String(lbName),
			PolicyNames:      policyNames,
		}

		if _, err := elbconn.SetLoadBalancerPoliciesForBackendServer(lbpbsOpts); err != nil {
			return fmt.Errorf("Error clearing BackendServerAuthenticationPolicyType: %s", err)
		}
	}

	request := &elb.DeleteLoadBalancerPolicyInput{
		LoadBalancerName: aws.String(lbName),
		PolicyName:       aws.String(policyName),
	}

	if _, err := elbconn.DeleteLoadBalancerPolicy(request); err != nil {
		return fmt.Errorf("Error deleting Load Balancer Policy %s: %s", d.Id(), err)
	}
	return nil
}

// resourceAwsLoadBalancerPolicyParseId takes an ID and parses it into
// it's constituent parts. You need three axes (LB name, instance port,
// and policy name) to create or identify a stickiness policy in AWS's API.
func resourceAwsLoadBalancerPolicyParseId(id string) (string, string, string) {
	parts := strings.SplitN(id, ":", 3)
	return parts[0], parts[1], parts[2]
}
