package aws

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elb"
)

func resourceAwsAutoscalingGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsAutoscalingGroupCreate,
		Read:   resourceAwsAutoscalingGroupRead,
		Update: resourceAwsAutoscalingGroupUpdate,
		Delete: resourceAwsAutoscalingGroupDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					// https://github.com/boto/botocore/blob/9f322b1/botocore/data/autoscaling/2011-01-01/service-2.json#L1862-L1873
					value := v.(string)
					if len(value) > 255 {
						errors = append(errors, fmt.Errorf(
							"%q cannot be longer than 255 characters", k))
					}
					return
				},
			},

			"launch_configuration": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"desired_capacity": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},

			"min_elb_capacity": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},

			"min_size": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},

			"max_size": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},

			"default_cooldown": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},

			"force_delete": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"health_check_grace_period": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},

			"health_check_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"availability_zones": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"load_balancers": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"vpc_zone_identifier": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"termination_policies": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"tag": autoscalingTagsSchema(),
		},
	}
}

func resourceAwsAutoscalingGroupCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).autoscalingconn

	var autoScalingGroupOpts autoscaling.CreateAutoScalingGroupInput
	autoScalingGroupOpts.AutoScalingGroupName = aws.String(d.Get("name").(string))
	autoScalingGroupOpts.LaunchConfigurationName = aws.String(d.Get("launch_configuration").(string))
	autoScalingGroupOpts.MinSize = aws.Long(int64(d.Get("min_size").(int)))
	autoScalingGroupOpts.MaxSize = aws.Long(int64(d.Get("max_size").(int)))

	// Availability Zones are optional if VPC Zone Identifer(s) are specified
	if v, ok := d.GetOk("availability_zones"); ok && v.(*schema.Set).Len() > 0 {
		autoScalingGroupOpts.AvailabilityZones = expandStringList(v.(*schema.Set).List())
	}

	if v, ok := d.GetOk("tag"); ok {
		autoScalingGroupOpts.Tags = autoscalingTagsFromMap(
			setToMapByKey(v.(*schema.Set), "key"), d.Get("name").(string))
	}

	if v, ok := d.GetOk("default_cooldown"); ok {
		autoScalingGroupOpts.DefaultCooldown = aws.Long(int64(v.(int)))
	}

	if v, ok := d.GetOk("health_check_type"); ok && v.(string) != "" {
		autoScalingGroupOpts.HealthCheckType = aws.String(v.(string))
	}

	if v, ok := d.GetOk("desired_capacity"); ok {
		autoScalingGroupOpts.DesiredCapacity = aws.Long(int64(v.(int)))
	}

	if v, ok := d.GetOk("health_check_grace_period"); ok {
		autoScalingGroupOpts.HealthCheckGracePeriod = aws.Long(int64(v.(int)))
	}

	if v, ok := d.GetOk("load_balancers"); ok && v.(*schema.Set).Len() > 0 {
		autoScalingGroupOpts.LoadBalancerNames = expandStringList(
			v.(*schema.Set).List())
	}

	if v, ok := d.GetOk("vpc_zone_identifier"); ok && v.(*schema.Set).Len() > 0 {
		autoScalingGroupOpts.VPCZoneIdentifier = expandVpcZoneIdentifiers(v.(*schema.Set).List())
	}

	if v, ok := d.GetOk("termination_policies"); ok && v.(*schema.Set).Len() > 0 {
		autoScalingGroupOpts.TerminationPolicies = expandStringList(
			v.(*schema.Set).List())
	}

	log.Printf("[DEBUG] AutoScaling Group create configuration: %#v", autoScalingGroupOpts)
	_, err := conn.CreateAutoScalingGroup(&autoScalingGroupOpts)
	if err != nil {
		return fmt.Errorf("Error creating Autoscaling Group: %s", err)
	}

	d.SetId(d.Get("name").(string))
	log.Printf("[INFO] AutoScaling Group Id: %s", d.Id())

	if err := waitForASGCapacity(d, meta); err != nil {
		return err
	}

	return resourceAwsAutoscalingGroupRead(d, meta)
}

func resourceAwsAutoscalingGroupRead(d *schema.ResourceData, meta interface{}) error {
	g, err := getAwsAutoscalingGroup(d, meta)
	if err != nil {
		return err
	}
	if g == nil {
		return nil
	}

	d.Set("availability_zones", g.AvailabilityZones)
	d.Set("default_cooldown", g.DefaultCooldown)
	d.Set("desired_capacity", g.DesiredCapacity)
	d.Set("health_check_grace_period", g.HealthCheckGracePeriod)
	d.Set("health_check_type", g.HealthCheckType)
	d.Set("launch_configuration", g.LaunchConfigurationName)
	d.Set("load_balancers", g.LoadBalancerNames)
	d.Set("min_size", g.MinSize)
	d.Set("max_size", g.MaxSize)
	d.Set("name", g.AutoScalingGroupName)
	d.Set("tag", g.Tags)
	d.Set("vpc_zone_identifier", strings.Split(*g.VPCZoneIdentifier, ","))
	d.Set("termination_policies", g.TerminationPolicies)

	return nil
}

func resourceAwsAutoscalingGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).autoscalingconn

	opts := autoscaling.UpdateAutoScalingGroupInput{
		AutoScalingGroupName: aws.String(d.Id()),
	}

	if d.HasChange("default_cooldown") {
		opts.DefaultCooldown = aws.Long(int64(d.Get("default_cooldown").(int)))
	}

	if d.HasChange("desired_capacity") {
		opts.DesiredCapacity = aws.Long(int64(d.Get("desired_capacity").(int)))
	}

	if d.HasChange("launch_configuration") {
		opts.LaunchConfigurationName = aws.String(d.Get("launch_configuration").(string))
	}

	if d.HasChange("min_size") {
		opts.MinSize = aws.Long(int64(d.Get("min_size").(int)))
	}

	if d.HasChange("max_size") {
		opts.MaxSize = aws.Long(int64(d.Get("max_size").(int)))
	}

	if d.HasChange("health_check_grace_period") {
		opts.HealthCheckGracePeriod = aws.Long(int64(d.Get("health_check_grace_period").(int)))
	}

	if d.HasChange("health_check_type") {
		opts.HealthCheckGracePeriod = aws.Long(int64(d.Get("health_check_grace_period").(int)))
		opts.HealthCheckType = aws.String(d.Get("health_check_type").(string))
	}

	if d.HasChange("vpc_zone_identifier") {
		opts.VPCZoneIdentifier = expandVpcZoneIdentifiers(d.Get("vpc_zone_identifier").(*schema.Set).List())
	}

	if d.HasChange("availability_zones") {
		if v, ok := d.GetOk("availability_zones"); ok && v.(*schema.Set).Len() > 0 {
			opts.AvailabilityZones = expandStringList(d.Get("availability_zones").(*schema.Set).List())
		}
	}

	if err := setAutoscalingTags(conn, d); err != nil {
		return err
	} else {
		d.SetPartial("tag")
	}

	log.Printf("[DEBUG] AutoScaling Group update configuration: %#v", opts)
	_, err := conn.UpdateAutoScalingGroup(&opts)
	if err != nil {
		d.Partial(true)
		return fmt.Errorf("Error updating Autoscaling group: %s", err)
	}

	if d.HasChange("load_balancers") {

		o, n := d.GetChange("load_balancers")
		if o == nil {
			o = new(schema.Set)
		}
		if n == nil {
			n = new(schema.Set)
		}

		os := o.(*schema.Set)
		ns := n.(*schema.Set)
		remove := expandStringList(os.Difference(ns).List())
		add := expandStringList(ns.Difference(os).List())

		if len(remove) > 0 {
			_, err := conn.DetachLoadBalancers(&autoscaling.DetachLoadBalancersInput{
				AutoScalingGroupName: aws.String(d.Id()),
				LoadBalancerNames:    remove,
			})
			if err != nil {
				return fmt.Errorf("[WARN] Error updating Load Balancers for AutoScaling Group (%s), error: %s", d.Id(), err)
			}
		}

		if len(add) > 0 {
			_, err := conn.AttachLoadBalancers(&autoscaling.AttachLoadBalancersInput{
				AutoScalingGroupName: aws.String(d.Id()),
				LoadBalancerNames:    add,
			})
			if err != nil {
				return fmt.Errorf("[WARN] Error updating Load Balancers for AutoScaling Group (%s), error: %s", d.Id(), err)
			}
		}
	}

	return resourceAwsAutoscalingGroupRead(d, meta)
}

func resourceAwsAutoscalingGroupDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).autoscalingconn

	// Read the autoscaling group first. If it doesn't exist, we're done.
	// We need the group in order to check if there are instances attached.
	// If so, we need to remove those first.
	g, err := getAwsAutoscalingGroup(d, meta)
	if err != nil {
		return err
	}
	if g == nil {
		return nil
	}
	if len(g.Instances) > 0 || *g.DesiredCapacity > 0 {
		if err := resourceAwsAutoscalingGroupDrain(d, meta); err != nil {
			return err
		}
	}

	log.Printf("[DEBUG] AutoScaling Group destroy: %v", d.Id())
	deleteopts := autoscaling.DeleteAutoScalingGroupInput{AutoScalingGroupName: aws.String(d.Id())}

	// You can force an autoscaling group to delete
	// even if it's in the process of scaling a resource.
	// Normally, you would set the min-size and max-size to 0,0
	// and then delete the group. This bypasses that and leaves
	// resources potentially dangling.
	if d.Get("force_delete").(bool) {
		deleteopts.ForceDelete = aws.Boolean(true)
	}

	// We retry the delete operation to handle InUse/InProgress errors coming
	// from scaling operations. We should be able to sneak in a delete in between
	// scaling operations within 5m.
	err = resource.Retry(5*time.Minute, func() error {
		if _, err := conn.DeleteAutoScalingGroup(&deleteopts); err != nil {
			if awserr, ok := err.(awserr.Error); ok {
				switch awserr.Code() {
				case "InvalidGroup.NotFound":
					// Already gone? Sure!
					return nil
				case "ResourceInUse", "ScalingActivityInProgress":
					// These are retryable
					return awserr
				}
			}
			// Didn't recognize the error, so shouldn't retry.
			return resource.RetryError{Err: err}
		}
		// Successful delete
		return nil
	})
	if err != nil {
		return err
	}

	return resource.Retry(5*time.Minute, func() error {
		if g, _ = getAwsAutoscalingGroup(d, meta); g != nil {
			return fmt.Errorf("Auto Scaling Group still exists")
		}
		return nil
	})
}

func getAwsAutoscalingGroup(
	d *schema.ResourceData,
	meta interface{}) (*autoscaling.Group, error) {
	conn := meta.(*AWSClient).autoscalingconn

	describeOpts := autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{aws.String(d.Id())},
	}

	log.Printf("[DEBUG] AutoScaling Group describe configuration: %#v", describeOpts)
	describeGroups, err := conn.DescribeAutoScalingGroups(&describeOpts)
	if err != nil {
		autoscalingerr, ok := err.(awserr.Error)
		if ok && autoscalingerr.Code() == "InvalidGroup.NotFound" {
			d.SetId("")
			return nil, nil
		}

		return nil, fmt.Errorf("Error retrieving AutoScaling groups: %s", err)
	}

	// Search for the autoscaling group
	for idx, asc := range describeGroups.AutoScalingGroups {
		if *asc.AutoScalingGroupName == d.Id() {
			return describeGroups.AutoScalingGroups[idx], nil
		}
	}

	// ASG not found
	d.SetId("")
	return nil, nil
}

func resourceAwsAutoscalingGroupDrain(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).autoscalingconn

	// First, set the capacity to zero so the group will drain
	log.Printf("[DEBUG] Reducing autoscaling group capacity to zero")
	opts := autoscaling.UpdateAutoScalingGroupInput{
		AutoScalingGroupName: aws.String(d.Id()),
		DesiredCapacity:      aws.Long(0),
		MinSize:              aws.Long(0),
		MaxSize:              aws.Long(0),
	}
	if _, err := conn.UpdateAutoScalingGroup(&opts); err != nil {
		return fmt.Errorf("Error setting capacity to zero to drain: %s", err)
	}

	// Next, wait for the autoscale group to drain
	log.Printf("[DEBUG] Waiting for group to have zero instances")
	return resource.Retry(10*time.Minute, func() error {
		g, err := getAwsAutoscalingGroup(d, meta)
		if err != nil {
			return resource.RetryError{Err: err}
		}
		if g == nil {
			return nil
		}

		if len(g.Instances) == 0 {
			return nil
		}

		return fmt.Errorf("group still has %d instances", len(g.Instances))
	})
}

var waitForASGCapacityTimeout = 10 * time.Minute

// Waits for a minimum number of healthy instances to show up as healthy in the
// ASG before continuing. Waits up to `waitForASGCapacityTimeout` for
// "desired_capacity", or "min_size" if desired capacity is not specified.
//
// If "min_elb_capacity" is specified, will also wait for that number of
// instances to show up InService in all attached ELBs. See "Waiting for
// Capacity" in docs for more discussion of the feature.
func waitForASGCapacity(d *schema.ResourceData, meta interface{}) error {
	wantASG := d.Get("min_size").(int)
	if v := d.Get("desired_capacity").(int); v > 0 {
		wantASG = v
	}
	wantELB := d.Get("min_elb_capacity").(int)

	log.Printf("[DEBUG] Waiting for capacity: %d ASG, %d ELB", wantASG, wantELB)

	return resource.Retry(waitForASGCapacityTimeout, func() error {
		g, err := getAwsAutoscalingGroup(d, meta)
		if err != nil {
			return resource.RetryError{Err: err}
		}
		if g == nil {
			return nil
		}
		lbis, err := getLBInstanceStates(g, meta)
		if err != nil {
			return resource.RetryError{Err: err}
		}

		haveASG := 0
		haveELB := 0

		for _, i := range g.Instances {
			if i.HealthStatus == nil || i.InstanceId == nil || i.LifecycleState == nil {
				continue
			}

			if !strings.EqualFold(*i.HealthStatus, "Healthy") {
				continue
			}

			if !strings.EqualFold(*i.LifecycleState, "InService") {
				continue
			}

			status, err := getInstanceSystemStatus(i.InstanceId, meta)
			if err != nil {
				log.Printf("[WARN] Error getting %s InstanceStatus: %s", *i.InstanceId, err)
				return resource.RetryError{Err: err}
			}

			log.Printf("[DEBUG] Instance [%s] Instance SystemStatus is %s", *i.InstanceId, status)
			if status != "ok" {
				continue
			}

			haveASG++

			if wantELB > 0 {
				inAllLbs := true
				for _, states := range lbis {
					state, ok := states[*i.InstanceId]
					if !ok || !strings.EqualFold(state, "InService") {
						inAllLbs = false
					}
				}
				if inAllLbs {
					haveELB++
				}
			}
		}

		log.Printf("[DEBUG] %q Capacity: %d/%d ASG, %d/%d ELB",
			d.Id(), haveASG, wantASG, haveELB, wantELB)

		if haveASG >= wantASG && haveELB >= wantELB {
			return nil
		}

		return fmt.Errorf("Still need to wait for more healthy instances. This could mean instances failed to launch. See Scaling History for more information.")
	})
}

// Returns a mapping of the instance states of all the ELBs attached to the
// provided ASG.
//
// Nested like: lbName -> instanceId -> instanceState
func getLBInstanceStates(g *autoscaling.Group, meta interface{}) (map[string]map[string]string, error) {
	lbInstanceStates := make(map[string]map[string]string)
	elbconn := meta.(*AWSClient).elbconn

	for _, lbName := range g.LoadBalancerNames {
		lbInstanceStates[*lbName] = make(map[string]string)
		opts := &elb.DescribeInstanceHealthInput{LoadBalancerName: lbName}
		r, err := elbconn.DescribeInstanceHealth(opts)
		if err != nil {
			return nil, err
		}
		for _, is := range r.InstanceStates {
			if is.InstanceId == nil || is.State == nil {
				continue
			}
			lbInstanceStates[*lbName][*is.InstanceId] = *is.State
		}
	}

	return lbInstanceStates, nil
}

func getInstanceSystemStatus(instance_id *string, meta interface{}) (string, error) {
	ec2conn := meta.(*AWSClient).ec2conn

	ids := []*string{instance_id}

	input := &ec2.DescribeInstanceStatusInput{
		InstanceIds:         ids,
		IncludeAllInstances: aws.Boolean(true),
	}

	output, err := ec2conn.DescribeInstanceStatus(input)
	if err != nil {
		return "", err
	}

	if len(output.InstanceStatuses) != 1 || output.InstanceStatuses[0].SystemStatus == nil {
		return "", fmt.Errorf("Unexpected response while trying to describe instance status: %s", output)
	}

	status := output.InstanceStatuses[0]
	return *status.SystemStatus.Status, nil
}

func expandVpcZoneIdentifiers(list []interface{}) *string {
	strs := make([]string, len(list))
	for _, s := range list {
		strs = append(strs, s.(string))
	}
	return aws.String(strings.Join(strs, ","))
}
