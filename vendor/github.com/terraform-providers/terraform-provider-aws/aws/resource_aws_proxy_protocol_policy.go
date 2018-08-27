package aws

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsProxyProtocolPolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsProxyProtocolPolicyCreate,
		Read:   resourceAwsProxyProtocolPolicyRead,
		Update: resourceAwsProxyProtocolPolicyUpdate,
		Delete: resourceAwsProxyProtocolPolicyDelete,

		Schema: map[string]*schema.Schema{
			"load_balancer": {
				Type:     schema.TypeString,
				Required: true,
			},

			"instance_ports": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Required: true,
				Set:      schema.HashString,
			},
		},
	}
}

func resourceAwsProxyProtocolPolicyCreate(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbconn
	elbname := aws.String(d.Get("load_balancer").(string))

	input := &elb.CreateLoadBalancerPolicyInput{
		LoadBalancerName: elbname,
		PolicyAttributes: []*elb.PolicyAttribute{
			{
				AttributeName:  aws.String("ProxyProtocol"),
				AttributeValue: aws.String("True"),
			},
		},
		PolicyName:     aws.String("TFEnableProxyProtocol"),
		PolicyTypeName: aws.String("ProxyProtocolPolicyType"),
	}

	// Create a policy
	log.Printf("[DEBUG] ELB create a policy %s from policy type %s",
		*input.PolicyName, *input.PolicyTypeName)

	if _, err := elbconn.CreateLoadBalancerPolicy(input); err != nil {
		return fmt.Errorf("Error creating a policy %s: %s",
			*input.PolicyName, err)
	}

	// Assign the policy name for use later
	d.Partial(true)
	d.SetId(fmt.Sprintf("%s:%s", *elbname, *input.PolicyName))
	d.SetPartial("load_balancer")
	log.Printf("[INFO] ELB PolicyName: %s", *input.PolicyName)

	return resourceAwsProxyProtocolPolicyUpdate(d, meta)
}

func resourceAwsProxyProtocolPolicyRead(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbconn
	elbname := aws.String(d.Get("load_balancer").(string))

	// Retrieve the current ELB policies for updating the state
	req := &elb.DescribeLoadBalancersInput{
		LoadBalancerNames: []*string{elbname},
	}
	resp, err := elbconn.DescribeLoadBalancers(req)
	if err != nil {
		if isLoadBalancerNotFound(err) {
			// The ELB is gone now, so just remove it from the state
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error retrieving ELB attributes: %s", err)
	}

	backends := flattenBackendPolicies(resp.LoadBalancerDescriptions[0].BackendServerDescriptions)

	ports := []*string{}
	for ip := range backends {
		ipstr := strconv.Itoa(int(ip))
		ports = append(ports, &ipstr)
	}
	d.Set("instance_ports", ports)
	d.Set("load_balancer", *elbname)
	return nil
}

func resourceAwsProxyProtocolPolicyUpdate(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbconn
	elbname := aws.String(d.Get("load_balancer").(string))

	// Retrieve the current ELB policies for updating the state
	req := &elb.DescribeLoadBalancersInput{
		LoadBalancerNames: []*string{elbname},
	}
	resp, err := elbconn.DescribeLoadBalancers(req)
	if err != nil {
		if isLoadBalancerNotFound(err) {
			// The ELB is gone now, so just remove it from the state
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error retrieving ELB attributes: %s", err)
	}

	backends := flattenBackendPolicies(resp.LoadBalancerDescriptions[0].BackendServerDescriptions)
	_, policyName := resourceAwsProxyProtocolPolicyParseId(d.Id())

	d.Partial(true)
	if d.HasChange("instance_ports") {
		o, n := d.GetChange("instance_ports")
		os := o.(*schema.Set)
		ns := n.(*schema.Set)
		remove := os.Difference(ns).List()
		add := ns.Difference(os).List()

		inputs := []*elb.SetLoadBalancerPoliciesForBackendServerInput{}

		i, err := resourceAwsProxyProtocolPolicyRemove(policyName, remove, backends)
		if err != nil {
			return err
		}
		inputs = append(inputs, i...)

		i, err = resourceAwsProxyProtocolPolicyAdd(policyName, add, backends)
		if err != nil {
			return err
		}
		inputs = append(inputs, i...)

		for _, input := range inputs {
			input.LoadBalancerName = elbname
			if _, err := elbconn.SetLoadBalancerPoliciesForBackendServer(input); err != nil {
				return fmt.Errorf("Error setting policy for backend: %s", err)
			}
		}

		d.SetPartial("instance_ports")
	}

	return resourceAwsProxyProtocolPolicyRead(d, meta)
}

func resourceAwsProxyProtocolPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbconn
	elbname := aws.String(d.Get("load_balancer").(string))

	// Retrieve the current ELB policies for updating the state
	req := &elb.DescribeLoadBalancersInput{
		LoadBalancerNames: []*string{elbname},
	}
	var err error
	resp, err := elbconn.DescribeLoadBalancers(req)
	if err != nil {
		if isLoadBalancerNotFound(err) {
			return nil
		}
		return fmt.Errorf("Error retrieving ELB attributes: %s", err)
	}

	backends := flattenBackendPolicies(resp.LoadBalancerDescriptions[0].BackendServerDescriptions)
	ports := d.Get("instance_ports").(*schema.Set).List()
	_, policyName := resourceAwsProxyProtocolPolicyParseId(d.Id())

	inputs, err := resourceAwsProxyProtocolPolicyRemove(policyName, ports, backends)
	if err != nil {
		return fmt.Errorf("Error detaching a policy from backend: %s", err)
	}
	for _, input := range inputs {
		input.LoadBalancerName = elbname
		if _, err := elbconn.SetLoadBalancerPoliciesForBackendServer(input); err != nil {
			return fmt.Errorf("Error setting policy for backend: %s", err)
		}
	}

	pOpt := &elb.DeleteLoadBalancerPolicyInput{
		LoadBalancerName: elbname,
		PolicyName:       aws.String(policyName),
	}
	if _, err := elbconn.DeleteLoadBalancerPolicy(pOpt); err != nil {
		return fmt.Errorf("Error removing a policy from load balancer: %s", err)
	}

	return nil
}

func resourceAwsProxyProtocolPolicyRemove(policyName string, ports []interface{}, backends map[int64][]string) ([]*elb.SetLoadBalancerPoliciesForBackendServerInput, error) {
	inputs := make([]*elb.SetLoadBalancerPoliciesForBackendServerInput, 0, len(ports))
	for _, p := range ports {
		ip, err := strconv.ParseInt(p.(string), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("Error detaching the policy: %s", err)
		}

		newPolicies := []*string{}
		curPolicies, found := backends[ip]
		if !found {
			// No policy for this instance port found, just skip it.
			continue
		}

		for _, policy := range curPolicies {
			if policy == policyName {
				// remove the policy
				continue
			}
			newPolicies = append(newPolicies, &policy)
		}

		inputs = append(inputs, &elb.SetLoadBalancerPoliciesForBackendServerInput{
			InstancePort: &ip,
			PolicyNames:  newPolicies,
		})
	}
	return inputs, nil
}

func resourceAwsProxyProtocolPolicyAdd(policyName string, ports []interface{}, backends map[int64][]string) ([]*elb.SetLoadBalancerPoliciesForBackendServerInput, error) {
	inputs := make([]*elb.SetLoadBalancerPoliciesForBackendServerInput, 0, len(ports))
	for _, p := range ports {
		ip, err := strconv.ParseInt(p.(string), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("Error attaching the policy: %s", err)
		}

		newPolicies := []*string{}
		curPolicies := backends[ip]
		for _, p := range curPolicies {
			if p == policyName {
				// Just remove it for now. It will be back later.
				continue
			} else {
				newPolicies = append(newPolicies, &p)
			}
		}
		newPolicies = append(newPolicies, aws.String(policyName))

		inputs = append(inputs, &elb.SetLoadBalancerPoliciesForBackendServerInput{
			InstancePort: &ip,
			PolicyNames:  newPolicies,
		})
	}
	return inputs, nil
}

// resourceAwsProxyProtocolPolicyParseId takes an ID and parses it into
// it's constituent parts. You need two axes (LB name, policy name)
// to create or identify a proxy protocol policy in AWS's API.
func resourceAwsProxyProtocolPolicyParseId(id string) (string, string) {
	parts := strings.SplitN(id, ":", 2)
	return parts[0], parts[1]
}
