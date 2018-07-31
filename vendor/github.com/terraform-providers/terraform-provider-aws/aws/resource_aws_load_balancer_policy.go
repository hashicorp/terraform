package aws

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsLoadBalancerPolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsLoadBalancerPolicyCreate,
		Read:   resourceAwsLoadBalancerPolicyRead,
		Update: resourceAwsLoadBalancerPolicyUpdate,
		Delete: resourceAwsLoadBalancerPolicyDelete,

		Schema: map[string]*schema.Schema{
			"load_balancer_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
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
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"value": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func resourceAwsLoadBalancerPolicyCreate(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbconn

	attributes := []*elb.PolicyAttribute{}
	if attributedata, ok := d.GetOk("policy_attribute"); ok {
		attributeSet := attributedata.(*schema.Set).List()
		for _, attribute := range attributeSet {
			data := attribute.(map[string]interface{})
			attributes = append(attributes, &elb.PolicyAttribute{
				AttributeName:  aws.String(data["name"].(string)),
				AttributeValue: aws.String(data["value"].(string)),
			})
		}
	}

	lbspOpts := &elb.CreateLoadBalancerPolicyInput{
		LoadBalancerName: aws.String(d.Get("load_balancer_name").(string)),
		PolicyName:       aws.String(d.Get("policy_name").(string)),
		PolicyTypeName:   aws.String(d.Get("policy_type_name").(string)),
		PolicyAttributes: attributes,
	}

	if _, err := elbconn.CreateLoadBalancerPolicy(lbspOpts); err != nil {
		return fmt.Errorf("Error creating LoadBalancerPolicy: %s", err)
	}

	d.SetId(fmt.Sprintf("%s:%s",
		*lbspOpts.LoadBalancerName,
		*lbspOpts.PolicyName))
	return resourceAwsLoadBalancerPolicyRead(d, meta)
}

func resourceAwsLoadBalancerPolicyRead(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbconn

	loadBalancerName, policyName := resourceAwsLoadBalancerPolicyParseId(d.Id())

	request := &elb.DescribeLoadBalancerPoliciesInput{
		LoadBalancerName: aws.String(loadBalancerName),
		PolicyNames:      []*string{aws.String(policyName)},
	}

	getResp, err := elbconn.DescribeLoadBalancerPolicies(request)
	if err != nil {
		if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "PolicyNotFound" {
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
	policyAttributes := policyDesc.PolicyAttributeDescriptions

	attributes := []map[string]string{}
	for _, a := range policyAttributes {
		pair := make(map[string]string)
		pair["name"] = *a.AttributeName
		pair["value"] = *a.AttributeValue
		if (*policyTypeName == "SSLNegotiationPolicyType") && (*a.AttributeValue == "false") {
			continue
		}
		attributes = append(attributes, pair)
	}

	d.Set("policy_name", policyName)
	d.Set("policy_type_name", policyTypeName)
	d.Set("load_balancer_name", loadBalancerName)
	d.Set("policy_attribute", attributes)

	return nil
}

func resourceAwsLoadBalancerPolicyUpdate(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbconn
	reassignments := Reassignment{}

	loadBalancerName, policyName := resourceAwsLoadBalancerPolicyParseId(d.Id())

	assigned, err := resourceAwsLoadBalancerPolicyAssigned(policyName, loadBalancerName, elbconn)
	if err != nil {
		return fmt.Errorf("Error determining assignment status of Load Balancer Policy %s: %s", policyName, err)
	}

	if assigned {
		reassignments, err = resourceAwsLoadBalancerPolicyUnassign(policyName, loadBalancerName, elbconn)
		if err != nil {
			return fmt.Errorf("Error unassigning Load Balancer Policy %s: %s", policyName, err)
		}
	}

	request := &elb.DeleteLoadBalancerPolicyInput{
		LoadBalancerName: aws.String(loadBalancerName),
		PolicyName:       aws.String(policyName),
	}

	if _, err := elbconn.DeleteLoadBalancerPolicy(request); err != nil {
		return fmt.Errorf("Error deleting Load Balancer Policy %s: %s", d.Id(), err)
	}

	err = resourceAwsLoadBalancerPolicyCreate(d, meta)

	for _, listenerAssignment := range reassignments.listenerPolicies {
		if _, err := elbconn.SetLoadBalancerPoliciesOfListener(listenerAssignment); err != nil {
			return fmt.Errorf("Error setting LoadBalancerPoliciesOfListener: %s", err)
		}
	}

	for _, backendServerAssignment := range reassignments.backendServerPolicies {
		if _, err := elbconn.SetLoadBalancerPoliciesForBackendServer(backendServerAssignment); err != nil {
			return fmt.Errorf("Error setting LoadBalancerPoliciesForBackendServer: %s", err)
		}
	}

	return resourceAwsLoadBalancerPolicyRead(d, meta)
}

func resourceAwsLoadBalancerPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbconn

	loadBalancerName, policyName := resourceAwsLoadBalancerPolicyParseId(d.Id())

	assigned, err := resourceAwsLoadBalancerPolicyAssigned(policyName, loadBalancerName, elbconn)
	if err != nil {
		return fmt.Errorf("Error determining assignment status of Load Balancer Policy %s: %s", policyName, err)
	}

	if assigned {
		_, err := resourceAwsLoadBalancerPolicyUnassign(policyName, loadBalancerName, elbconn)
		if err != nil {
			return fmt.Errorf("Error unassigning Load Balancer Policy %s: %s", policyName, err)
		}
	}

	request := &elb.DeleteLoadBalancerPolicyInput{
		LoadBalancerName: aws.String(loadBalancerName),
		PolicyName:       aws.String(policyName),
	}

	if _, err := elbconn.DeleteLoadBalancerPolicy(request); err != nil {
		return fmt.Errorf("Error deleting Load Balancer Policy %s: %s", d.Id(), err)
	}

	return nil
}

func resourceAwsLoadBalancerPolicyParseId(id string) (string, string) {
	parts := strings.SplitN(id, ":", 2)
	return parts[0], parts[1]
}

func resourceAwsLoadBalancerPolicyAssigned(policyName, loadBalancerName string, elbconn *elb.ELB) (bool, error) {
	describeElbOpts := &elb.DescribeLoadBalancersInput{
		LoadBalancerNames: []*string{aws.String(loadBalancerName)},
	}

	describeResp, err := elbconn.DescribeLoadBalancers(describeElbOpts)

	if err != nil {
		if ec2err, ok := err.(awserr.Error); ok {
			if ec2err.Code() == "LoadBalancerNotFound" {
				return false, nil
			}
		}
		return false, fmt.Errorf("Error retrieving ELB description: %s", err)
	}

	if len(describeResp.LoadBalancerDescriptions) != 1 {
		return false, fmt.Errorf("Unable to find ELB: %#v", describeResp.LoadBalancerDescriptions)
	}

	lb := describeResp.LoadBalancerDescriptions[0]
	assigned := false
	for _, backendServer := range lb.BackendServerDescriptions {
		for _, name := range backendServer.PolicyNames {
			if policyName == *name {
				assigned = true
				break
			}
		}
	}

	for _, listener := range lb.ListenerDescriptions {
		for _, name := range listener.PolicyNames {
			if policyName == *name {
				assigned = true
				break
			}
		}
	}

	return assigned, nil
}

type Reassignment struct {
	backendServerPolicies []*elb.SetLoadBalancerPoliciesForBackendServerInput
	listenerPolicies      []*elb.SetLoadBalancerPoliciesOfListenerInput
}

func resourceAwsLoadBalancerPolicyUnassign(policyName, loadBalancerName string, elbconn *elb.ELB) (Reassignment, error) {
	reassignments := Reassignment{}

	describeElbOpts := &elb.DescribeLoadBalancersInput{
		LoadBalancerNames: []*string{aws.String(loadBalancerName)},
	}

	describeResp, err := elbconn.DescribeLoadBalancers(describeElbOpts)

	if err != nil {
		if ec2err, ok := err.(awserr.Error); ok {
			if ec2err.Code() == "LoadBalancerNotFound" {
				return reassignments, nil
			}
		}
		return reassignments, fmt.Errorf("Error retrieving ELB description: %s", err)
	}

	if len(describeResp.LoadBalancerDescriptions) != 1 {
		return reassignments, fmt.Errorf("Unable to find ELB: %#v", describeResp.LoadBalancerDescriptions)
	}

	lb := describeResp.LoadBalancerDescriptions[0]

	for _, backendServer := range lb.BackendServerDescriptions {
		policies := []*string{}

		for _, name := range backendServer.PolicyNames {
			if policyName != *name {
				policies = append(policies, name)
			}
		}

		if len(backendServer.PolicyNames) != len(policies) {
			setOpts := &elb.SetLoadBalancerPoliciesForBackendServerInput{
				LoadBalancerName: aws.String(loadBalancerName),
				InstancePort:     aws.Int64(*backendServer.InstancePort),
				PolicyNames:      policies,
			}

			reassignOpts := &elb.SetLoadBalancerPoliciesForBackendServerInput{
				LoadBalancerName: aws.String(loadBalancerName),
				InstancePort:     aws.Int64(*backendServer.InstancePort),
				PolicyNames:      backendServer.PolicyNames,
			}

			reassignments.backendServerPolicies = append(reassignments.backendServerPolicies, reassignOpts)

			_, err = elbconn.SetLoadBalancerPoliciesForBackendServer(setOpts)
			if err != nil {
				return reassignments, fmt.Errorf("Error Setting Load Balancer Policies for Backend Server: %s", err)
			}
		}
	}

	for _, listener := range lb.ListenerDescriptions {
		policies := []*string{}

		for _, name := range listener.PolicyNames {
			if policyName != *name {
				policies = append(policies, name)
			}
		}

		if len(listener.PolicyNames) != len(policies) {
			setOpts := &elb.SetLoadBalancerPoliciesOfListenerInput{
				LoadBalancerName: aws.String(loadBalancerName),
				LoadBalancerPort: aws.Int64(*listener.Listener.LoadBalancerPort),
				PolicyNames:      policies,
			}

			reassignOpts := &elb.SetLoadBalancerPoliciesOfListenerInput{
				LoadBalancerName: aws.String(loadBalancerName),
				LoadBalancerPort: aws.Int64(*listener.Listener.LoadBalancerPort),
				PolicyNames:      listener.PolicyNames,
			}

			reassignments.listenerPolicies = append(reassignments.listenerPolicies, reassignOpts)

			_, err = elbconn.SetLoadBalancerPoliciesOfListener(setOpts)
			if err != nil {
				return reassignments, fmt.Errorf("Error Setting Load Balancer Policies of Listener: %s", err)
			}
		}
	}

	return reassignments, nil
}
