package aws

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"time"

	"bytes"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsLb() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsLbCreate,
		Read:   resoureAwsLbRead,
		Update: resourceAwsLbUpdate,
		Delete: resourceAwsLbDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"arn_suffix": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name_prefix"},
				ValidateFunc:  validateElbName,
			},

			"name_prefix": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateElbNamePrefix,
			},

			"internal": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"load_balancer_type": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Default:  "application",
			},

			"security_groups": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
				Optional: true,
				Set:      schema.HashString,
			},

			"subnets": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
				Computed: true,
				Set:      schema.HashString,
			},

			"subnet_mapping": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"subnet_id": {
							Type:     schema.TypeString,
							Required: true,
						},
						"allocation_id": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
				Set: func(v interface{}) int {
					var buf bytes.Buffer
					m := v.(map[string]interface{})
					buf.WriteString(fmt.Sprintf("%s-", m["subnet_id"].(string)))
					if m["allocation_id"] != "" {
						buf.WriteString(fmt.Sprintf("%s-", m["allocation_id"].(string)))
					}
					return hashcode.String(buf.String())
				},
			},

			"access_logs": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"bucket": {
							Type:     schema.TypeString,
							Required: true,
						},
						"prefix": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Computed: true,
						},
					},
				},
			},

			"enable_deletion_protection": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"idle_timeout": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  60,
			},

			"ip_address_type": {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
			},

			"vpc_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"zone_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"dns_name": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsLbCreate(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbv2conn

	var name string
	if v, ok := d.GetOk("name"); ok {
		name = v.(string)
	} else if v, ok := d.GetOk("name_prefix"); ok {
		name = resource.PrefixedUniqueId(v.(string))
	} else {
		name = resource.PrefixedUniqueId("tf-lb-")
	}
	d.Set("name", name)

	elbOpts := &elbv2.CreateLoadBalancerInput{
		Name: aws.String(name),
		Type: aws.String(d.Get("load_balancer_type").(string)),
		Tags: tagsFromMapELBv2(d.Get("tags").(map[string]interface{})),
	}

	if scheme, ok := d.GetOk("internal"); ok && scheme.(bool) {
		elbOpts.Scheme = aws.String("internal")
	}

	if v, ok := d.GetOk("security_groups"); ok {
		elbOpts.SecurityGroups = expandStringList(v.(*schema.Set).List())
	}

	if v, ok := d.GetOk("subnets"); ok {
		elbOpts.Subnets = expandStringList(v.(*schema.Set).List())
	}

	if v, ok := d.GetOk("subnet_mapping"); ok {
		rawMappings := v.(*schema.Set).List()
		elbOpts.SubnetMappings = make([]*elbv2.SubnetMapping, len(rawMappings))
		for i, mapping := range rawMappings {
			subnetMap := mapping.(map[string]interface{})

			elbOpts.SubnetMappings[i] = &elbv2.SubnetMapping{
				SubnetId: aws.String(subnetMap["subnet_id"].(string)),
			}

			if subnetMap["allocation_id"].(string) != "" {
				elbOpts.SubnetMappings[i].AllocationId = aws.String(subnetMap["allocation_id"].(string))
			}
		}
	}

	if v, ok := d.GetOk("ip_address_type"); ok {
		elbOpts.IpAddressType = aws.String(v.(string))
	}

	log.Printf("[DEBUG] ALB create configuration: %#v", elbOpts)

	resp, err := elbconn.CreateLoadBalancer(elbOpts)
	if err != nil {
		return errwrap.Wrapf("Error creating Application Load Balancer: {{err}}", err)
	}

	if len(resp.LoadBalancers) != 1 {
		return fmt.Errorf("No load balancers returned following creation of %s", d.Get("name").(string))
	}

	lb := resp.LoadBalancers[0]
	d.SetId(*lb.LoadBalancerArn)
	log.Printf("[INFO] ALB ID: %s", d.Id())

	stateConf := &resource.StateChangeConf{
		Pending: []string{"provisioning", "failed"},
		Target:  []string{"active"},
		Refresh: func() (interface{}, string, error) {
			describeResp, err := elbconn.DescribeLoadBalancers(&elbv2.DescribeLoadBalancersInput{
				LoadBalancerArns: []*string{lb.LoadBalancerArn},
			})
			if err != nil {
				return nil, "", err
			}

			if len(describeResp.LoadBalancers) != 1 {
				return nil, "", fmt.Errorf("No load balancers returned for %s", *lb.LoadBalancerArn)
			}
			dLb := describeResp.LoadBalancers[0]

			log.Printf("[INFO] ALB state: %s", *dLb.State.Code)

			return describeResp, *dLb.State.Code, nil
		},
		Timeout:    d.Timeout(schema.TimeoutCreate),
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second, // Wait 30 secs before starting
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}

	return resourceAwsLbUpdate(d, meta)
}

func resoureAwsLbRead(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbv2conn
	lbArn := d.Id()

	describeLbOpts := &elbv2.DescribeLoadBalancersInput{
		LoadBalancerArns: []*string{aws.String(lbArn)},
	}

	describeResp, err := elbconn.DescribeLoadBalancers(describeLbOpts)
	if err != nil {
		if isLoadBalancerNotFound(err) {
			// The ALB is gone now, so just remove it from the state
			log.Printf("[WARN] ALB %s not found in AWS, removing from state", d.Id())
			d.SetId("")
			return nil
		}

		return errwrap.Wrapf("Error retrieving ALB: {{err}}", err)
	}
	if len(describeResp.LoadBalancers) != 1 {
		return fmt.Errorf("Unable to find ALB: %#v", describeResp.LoadBalancers)
	}

	return flattenAwsLbResource(d, meta, describeResp.LoadBalancers[0])
}

func resourceAwsLbUpdate(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbv2conn

	if !d.IsNewResource() {
		if err := setElbV2Tags(elbconn, d); err != nil {
			return errwrap.Wrapf("Error Modifying Tags on ALB: {{err}}", err)
		}
	}

	attributes := make([]*elbv2.LoadBalancerAttribute, 0)

	if d.HasChange("access_logs") {
		logs := d.Get("access_logs").([]interface{})
		if len(logs) == 1 {
			log := logs[0].(map[string]interface{})

			attributes = append(attributes,
				&elbv2.LoadBalancerAttribute{
					Key:   aws.String("access_logs.s3.enabled"),
					Value: aws.String(strconv.FormatBool(log["enabled"].(bool))),
				},
				&elbv2.LoadBalancerAttribute{
					Key:   aws.String("access_logs.s3.bucket"),
					Value: aws.String(log["bucket"].(string)),
				})

			if prefix, ok := log["prefix"]; ok {
				attributes = append(attributes, &elbv2.LoadBalancerAttribute{
					Key:   aws.String("access_logs.s3.prefix"),
					Value: aws.String(prefix.(string)),
				})
			}
		} else if len(logs) == 0 {
			attributes = append(attributes, &elbv2.LoadBalancerAttribute{
				Key:   aws.String("access_logs.s3.enabled"),
				Value: aws.String("false"),
			})
		}
	}

	if d.HasChange("enable_deletion_protection") {
		attributes = append(attributes, &elbv2.LoadBalancerAttribute{
			Key:   aws.String("deletion_protection.enabled"),
			Value: aws.String(fmt.Sprintf("%t", d.Get("enable_deletion_protection").(bool))),
		})
	}

	// It's important to know that Idle timeout is not supported for Network Loadbalancers
	if d.Get("load_balancer_type").(string) != "network" && d.HasChange("idle_timeout") {
		attributes = append(attributes, &elbv2.LoadBalancerAttribute{
			Key:   aws.String("idle_timeout.timeout_seconds"),
			Value: aws.String(fmt.Sprintf("%d", d.Get("idle_timeout").(int))),
		})
	}

	if len(attributes) != 0 {
		input := &elbv2.ModifyLoadBalancerAttributesInput{
			LoadBalancerArn: aws.String(d.Id()),
			Attributes:      attributes,
		}

		log.Printf("[DEBUG] ALB Modify Load Balancer Attributes Request: %#v", input)
		_, err := elbconn.ModifyLoadBalancerAttributes(input)
		if err != nil {
			return fmt.Errorf("Failure configuring ALB attributes: %s", err)
		}
	}

	if d.HasChange("security_groups") {
		sgs := expandStringList(d.Get("security_groups").(*schema.Set).List())

		params := &elbv2.SetSecurityGroupsInput{
			LoadBalancerArn: aws.String(d.Id()),
			SecurityGroups:  sgs,
		}
		_, err := elbconn.SetSecurityGroups(params)
		if err != nil {
			return fmt.Errorf("Failure Setting ALB Security Groups: %s", err)
		}

	}

	if d.HasChange("subnets") {
		subnets := expandStringList(d.Get("subnets").(*schema.Set).List())

		params := &elbv2.SetSubnetsInput{
			LoadBalancerArn: aws.String(d.Id()),
			Subnets:         subnets,
		}

		_, err := elbconn.SetSubnets(params)
		if err != nil {
			return fmt.Errorf("Failure Setting ALB Subnets: %s", err)
		}
	}

	if d.HasChange("ip_address_type") {

		params := &elbv2.SetIpAddressTypeInput{
			LoadBalancerArn: aws.String(d.Id()),
			IpAddressType:   aws.String(d.Get("ip_address_type").(string)),
		}

		_, err := elbconn.SetIpAddressType(params)
		if err != nil {
			return fmt.Errorf("Failure Setting ALB IP Address Type: %s", err)
		}

	}

	stateConf := &resource.StateChangeConf{
		Pending: []string{"active", "provisioning", "failed"},
		Target:  []string{"active"},
		Refresh: func() (interface{}, string, error) {
			describeResp, err := elbconn.DescribeLoadBalancers(&elbv2.DescribeLoadBalancersInput{
				LoadBalancerArns: []*string{aws.String(d.Id())},
			})
			if err != nil {
				return nil, "", err
			}

			if len(describeResp.LoadBalancers) != 1 {
				return nil, "", fmt.Errorf("No load balancers returned for %s", d.Id())
			}
			dLb := describeResp.LoadBalancers[0]

			log.Printf("[INFO] ALB state: %s", *dLb.State.Code)

			return describeResp, *dLb.State.Code, nil
		},
		Timeout:    d.Timeout(schema.TimeoutUpdate),
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second, // Wait 30 secs before starting
	}
	_, err := stateConf.WaitForState()
	if err != nil {
		return err
	}

	return resoureAwsLbRead(d, meta)
}

func resourceAwsLbDelete(d *schema.ResourceData, meta interface{}) error {
	lbconn := meta.(*AWSClient).elbv2conn

	log.Printf("[INFO] Deleting ALB: %s", d.Id())

	// Destroy the load balancer
	deleteElbOpts := elbv2.DeleteLoadBalancerInput{
		LoadBalancerArn: aws.String(d.Id()),
	}
	if _, err := lbconn.DeleteLoadBalancer(&deleteElbOpts); err != nil {
		return fmt.Errorf("Error deleting ALB: %s", err)
	}

	conn := meta.(*AWSClient).ec2conn

	err := cleanupLBNetworkInterfaces(conn, d.Id())
	if err != nil {
		log.Printf("[WARN] Failed to cleanup ENIs for ALB %q: %#v", d.Id(), err)
	}

	err = waitForNLBNetworkInterfacesToDetach(conn, d.Id())
	if err != nil {
		log.Printf("[WARN] Failed to wait for ENIs to disappear for NLB %q: %#v", d.Id(), err)
	}

	return nil
}

// ALB automatically creates ENI(s) on creation
// but the cleanup is asynchronous and may take time
// which then blocks IGW, SG or VPC on deletion
// So we make the cleanup "synchronous" here
func cleanupLBNetworkInterfaces(conn *ec2.EC2, lbArn string) error {
	name, err := getLbNameFromArn(lbArn)
	if err != nil {
		return err
	}

	out, err := conn.DescribeNetworkInterfaces(&ec2.DescribeNetworkInterfacesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("attachment.instance-owner-id"),
				Values: []*string{aws.String("amazon-elb")},
			},
			{
				Name:   aws.String("description"),
				Values: []*string{aws.String("ELB " + name)},
			},
		},
	})
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Found %d ENIs to cleanup for LB %q",
		len(out.NetworkInterfaces), name)

	if len(out.NetworkInterfaces) == 0 {
		// Nothing to cleanup
		return nil
	}

	err = detachNetworkInterfaces(conn, out.NetworkInterfaces)
	if err != nil {
		return err
	}

	err = deleteNetworkInterfaces(conn, out.NetworkInterfaces)
	if err != nil {
		return err
	}

	return nil
}

func waitForNLBNetworkInterfacesToDetach(conn *ec2.EC2, lbArn string) error {
	name, err := getLbNameFromArn(lbArn)
	if err != nil {
		return err
	}

	// We cannot cleanup these ENIs ourselves as that would result in
	// OperationNotPermitted: You are not allowed to manage 'ela-attach' attachments.
	// yet presence of these ENIs may prevent us from deleting EIPs associated w/ the NLB

	return resource.Retry(1*time.Minute, func() *resource.RetryError {
		out, err := conn.DescribeNetworkInterfaces(&ec2.DescribeNetworkInterfacesInput{
			Filters: []*ec2.Filter{
				{
					Name:   aws.String("attachment.instance-owner-id"),
					Values: []*string{aws.String("amazon-aws")},
				},
				{
					Name:   aws.String("attachment.attachment-id"),
					Values: []*string{aws.String("ela-attach-*")},
				},
				{
					Name:   aws.String("description"),
					Values: []*string{aws.String("ELB " + name)},
				},
			},
		})
		if err != nil {
			return resource.NonRetryableError(err)
		}

		niCount := len(out.NetworkInterfaces)
		if niCount > 0 {
			log.Printf("[DEBUG] Found %d ENIs to cleanup for NLB %q", niCount, lbArn)
			return resource.RetryableError(fmt.Errorf("Waiting for %d ENIs of %q to clean up", niCount, lbArn))
		}
		log.Printf("[DEBUG] ENIs gone for NLB %q", lbArn)

		return nil
	})
}

func getLbNameFromArn(arn string) (string, error) {
	re := regexp.MustCompile("([^/]+/[^/]+/[^/]+)$")
	matches := re.FindStringSubmatch(arn)
	if len(matches) != 2 {
		return "", fmt.Errorf("Unexpected ARN format: %q", arn)
	}

	// e.g. app/example-alb/b26e625cdde161e6
	return matches[1], nil
}

// flattenSubnetsFromAvailabilityZones creates a slice of strings containing the subnet IDs
// for the ALB based on the AvailabilityZones structure returned by the API.
func flattenSubnetsFromAvailabilityZones(availabilityZones []*elbv2.AvailabilityZone) []string {
	var result []string
	for _, az := range availabilityZones {
		result = append(result, *az.SubnetId)
	}
	return result
}

func lbSuffixFromARN(arn *string) string {
	if arn == nil {
		return ""
	}

	if arnComponents := regexp.MustCompile(`arn:.*:loadbalancer/(.*)`).FindAllStringSubmatch(*arn, -1); len(arnComponents) == 1 {
		if len(arnComponents[0]) == 2 {
			return arnComponents[0][1]
		}
	}

	return ""
}

// flattenAwsLbResource takes a *elbv2.LoadBalancer and populates all respective resource fields.
func flattenAwsLbResource(d *schema.ResourceData, meta interface{}, lb *elbv2.LoadBalancer) error {
	elbconn := meta.(*AWSClient).elbv2conn

	d.Set("arn", lb.LoadBalancerArn)
	d.Set("arn_suffix", lbSuffixFromARN(lb.LoadBalancerArn))
	d.Set("name", lb.LoadBalancerName)
	d.Set("internal", (lb.Scheme != nil && *lb.Scheme == "internal"))
	d.Set("security_groups", flattenStringList(lb.SecurityGroups))
	d.Set("subnets", flattenSubnetsFromAvailabilityZones(lb.AvailabilityZones))
	d.Set("vpc_id", lb.VpcId)
	d.Set("zone_id", lb.CanonicalHostedZoneId)
	d.Set("dns_name", lb.DNSName)
	d.Set("ip_address_type", lb.IpAddressType)
	d.Set("load_balancer_type", lb.Type)

	subnetMappings := make([]interface{}, 0)
	for _, az := range lb.AvailabilityZones {
		subnetMappingRaw := make([]map[string]interface{}, len(az.LoadBalancerAddresses))
		for _, subnet := range az.LoadBalancerAddresses {
			subnetMap := make(map[string]interface{}, 0)
			subnetMap["subnet_id"] = *az.SubnetId

			if subnet.AllocationId != nil {
				subnetMap["allocation_id"] = *subnet.AllocationId
			}

			subnetMappingRaw = append(subnetMappingRaw, subnetMap)
		}
		subnetMappings = append(subnetMappings, subnetMappingRaw)
	}
	d.Set("subnet_mapping", subnetMappings)

	respTags, err := elbconn.DescribeTags(&elbv2.DescribeTagsInput{
		ResourceArns: []*string{lb.LoadBalancerArn},
	})
	if err != nil {
		return errwrap.Wrapf("Error retrieving ALB Tags: {{err}}", err)
	}

	var et []*elbv2.Tag
	if len(respTags.TagDescriptions) > 0 {
		et = respTags.TagDescriptions[0].Tags
	}

	if err := d.Set("tags", tagsToMapELBv2(et)); err != nil {
		log.Printf("[WARN] Error setting tags for AWS LB (%s): %s", d.Id(), err)
	}

	attributesResp, err := elbconn.DescribeLoadBalancerAttributes(&elbv2.DescribeLoadBalancerAttributesInput{
		LoadBalancerArn: aws.String(d.Id()),
	})
	if err != nil {
		return errwrap.Wrapf("Error retrieving ALB Attributes: {{err}}", err)
	}

	accessLogMap := map[string]interface{}{}
	for _, attr := range attributesResp.Attributes {
		switch *attr.Key {
		case "access_logs.s3.enabled":
			accessLogMap["enabled"] = *attr.Value
		case "access_logs.s3.bucket":
			accessLogMap["bucket"] = *attr.Value
		case "access_logs.s3.prefix":
			accessLogMap["prefix"] = *attr.Value
		case "idle_timeout.timeout_seconds":
			timeout, err := strconv.Atoi(*attr.Value)
			if err != nil {
				return errwrap.Wrapf("Error parsing ALB timeout: {{err}}", err)
			}
			log.Printf("[DEBUG] Setting ALB Timeout Seconds: %d", timeout)
			d.Set("idle_timeout", timeout)
		case "deletion_protection.enabled":
			protectionEnabled := (*attr.Value) == "true"
			log.Printf("[DEBUG] Setting ALB Deletion Protection Enabled: %t", protectionEnabled)
			d.Set("enable_deletion_protection", protectionEnabled)
		}
	}

	log.Printf("[DEBUG] Setting ALB Access Logs: %#v", accessLogMap)
	if accessLogMap["bucket"] != "" || accessLogMap["prefix"] != "" {
		d.Set("access_logs", []interface{}{accessLogMap})
	} else {
		d.Set("access_logs", []interface{}{})
	}

	return nil
}
