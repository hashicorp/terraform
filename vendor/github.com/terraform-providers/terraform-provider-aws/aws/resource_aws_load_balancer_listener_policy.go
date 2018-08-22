package aws

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsLoadBalancerListenerPolicies() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsLoadBalancerListenerPoliciesCreate,
		Read:   resourceAwsLoadBalancerListenerPoliciesRead,
		Update: resourceAwsLoadBalancerListenerPoliciesCreate,
		Delete: resourceAwsLoadBalancerListenerPoliciesDelete,

		Schema: map[string]*schema.Schema{
			"load_balancer_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"policy_names": &schema.Schema{
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
				Set:      schema.HashString,
			},

			"load_balancer_port": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
		},
	}
}

func resourceAwsLoadBalancerListenerPoliciesCreate(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbconn

	loadBalancerName := d.Get("load_balancer_name")

	policyNames := []*string{}
	if v, ok := d.GetOk("policy_names"); ok {
		policyNames = expandStringList(v.(*schema.Set).List())
	}

	setOpts := &elb.SetLoadBalancerPoliciesOfListenerInput{
		LoadBalancerName: aws.String(loadBalancerName.(string)),
		LoadBalancerPort: aws.Int64(int64(d.Get("load_balancer_port").(int))),
		PolicyNames:      policyNames,
	}

	if _, err := elbconn.SetLoadBalancerPoliciesOfListener(setOpts); err != nil {
		return fmt.Errorf("Error setting LoadBalancerPoliciesOfListener: %s", err)
	}

	d.SetId(fmt.Sprintf("%s:%s", *setOpts.LoadBalancerName, strconv.FormatInt(*setOpts.LoadBalancerPort, 10)))
	return resourceAwsLoadBalancerListenerPoliciesRead(d, meta)
}

func resourceAwsLoadBalancerListenerPoliciesRead(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbconn

	loadBalancerName, loadBalancerPort := resourceAwsLoadBalancerListenerPoliciesParseId(d.Id())

	describeElbOpts := &elb.DescribeLoadBalancersInput{
		LoadBalancerNames: []*string{aws.String(loadBalancerName)},
	}

	describeResp, err := elbconn.DescribeLoadBalancers(describeElbOpts)

	if err != nil {
		if ec2err, ok := err.(awserr.Error); ok {
			if ec2err.Code() == "LoadBalancerNotFound" {
				d.SetId("")
				return fmt.Errorf("LoadBalancerNotFound: %s", err)
			}
		}
		return fmt.Errorf("Error retrieving ELB description: %s", err)
	}

	if len(describeResp.LoadBalancerDescriptions) != 1 {
		return fmt.Errorf("Unable to find ELB: %#v", describeResp.LoadBalancerDescriptions)
	}

	lb := describeResp.LoadBalancerDescriptions[0]

	policyNames := []*string{}
	for _, listener := range lb.ListenerDescriptions {
		if loadBalancerPort != strconv.Itoa(int(*listener.Listener.LoadBalancerPort)) {
			continue
		}

		for _, name := range listener.PolicyNames {
			policyNames = append(policyNames, name)
		}
	}

	d.Set("load_balancer_name", loadBalancerName)
	d.Set("load_balancer_port", loadBalancerPort)
	d.Set("policy_names", flattenStringList(policyNames))

	return nil
}

func resourceAwsLoadBalancerListenerPoliciesDelete(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbconn

	loadBalancerName, loadBalancerPort := resourceAwsLoadBalancerListenerPoliciesParseId(d.Id())

	loadBalancerPortInt, err := strconv.ParseInt(loadBalancerPort, 10, 64)
	if err != nil {
		return fmt.Errorf("Error parsing loadBalancerPort as integer: %s", err)
	}

	setOpts := &elb.SetLoadBalancerPoliciesOfListenerInput{
		LoadBalancerName: aws.String(loadBalancerName),
		LoadBalancerPort: aws.Int64(loadBalancerPortInt),
		PolicyNames:      []*string{},
	}

	if _, err := elbconn.SetLoadBalancerPoliciesOfListener(setOpts); err != nil {
		return fmt.Errorf("Error setting LoadBalancerPoliciesOfListener: %s", err)
	}

	return nil
}

func resourceAwsLoadBalancerListenerPoliciesParseId(id string) (string, string) {
	parts := strings.SplitN(id, ":", 2)
	return parts[0], parts[1]
}
