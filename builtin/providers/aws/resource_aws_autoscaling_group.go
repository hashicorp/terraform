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
				Optional: true,
				Computed: true,
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
				Default:  false,
			},

			"health_check_grace_period": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  300,
			},

			"health_check_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"availability_zones": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"placement_group": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
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
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"wait_for_capacity_timeout": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "10m",
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					duration, err := time.ParseDuration(value)
					if err != nil {
						errors = append(errors, fmt.Errorf(
							"%q cannot be parsed as a duration: %s", k, err))
					}
					if duration < 0 {
						errors = append(errors, fmt.Errorf(
							"%q must be greater than zero", k))
					}
					return
				},
			},

			"wait_for_elb_capacity": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},

			"enabled_metrics": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"metrics_granularity": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "1Minute",
			},

			"tag": autoscalingTagsSchema(),
		},
	}
}

func resourceAwsAutoscalingGroupCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).autoscalingconn

	var autoScalingGroupOpts autoscaling.CreateAutoScalingGroupInput

	var asgName string
	if v, ok := d.GetOk("name"); ok {
		asgName = v.(string)
	} else {
		asgName = resource.PrefixedUniqueId("tf-asg-")
		d.Set("name", asgName)
	}

	autoScalingGroupOpts.AutoScalingGroupName = aws.String(asgName)
	autoScalingGroupOpts.LaunchConfigurationName = aws.String(d.Get("launch_configuration").(string))
	autoScalingGroupOpts.MinSize = aws.Int64(int64(d.Get("min_size").(int)))
	autoScalingGroupOpts.MaxSize = aws.Int64(int64(d.Get("max_size").(int)))

	// Availability Zones are optional if VPC Zone Identifer(s) are specified
	if v, ok := d.GetOk("availability_zones"); ok && v.(*schema.Set).Len() > 0 {
		autoScalingGroupOpts.AvailabilityZones = expandStringList(v.(*schema.Set).List())
	}

	if v, ok := d.GetOk("tag"); ok {
		autoScalingGroupOpts.Tags = autoscalingTagsFromMap(
			setToMapByKey(v.(*schema.Set), "key"), d.Get("name").(string))
	}

	if v, ok := d.GetOk("default_cooldown"); ok {
		autoScalingGroupOpts.DefaultCooldown = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("health_check_type"); ok && v.(string) != "" {
		autoScalingGroupOpts.HealthCheckType = aws.String(v.(string))
	}

	if v, ok := d.GetOk("desired_capacity"); ok {
		autoScalingGroupOpts.DesiredCapacity = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("health_check_grace_period"); ok {
		autoScalingGroupOpts.HealthCheckGracePeriod = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("placement_group"); ok {
		autoScalingGroupOpts.PlacementGroup = aws.String(v.(string))
	}

	if v, ok := d.GetOk("load_balancers"); ok && v.(*schema.Set).Len() > 0 {
		autoScalingGroupOpts.LoadBalancerNames = expandStringList(
			v.(*schema.Set).List())
	}

	if v, ok := d.GetOk("vpc_zone_identifier"); ok && v.(*schema.Set).Len() > 0 {
		autoScalingGroupOpts.VPCZoneIdentifier = expandVpcZoneIdentifiers(v.(*schema.Set).List())
	}

	if v, ok := d.GetOk("termination_policies"); ok && len(v.([]interface{})) > 0 {
		autoScalingGroupOpts.TerminationPolicies = expandStringList(v.([]interface{}))
	}

	log.Printf("[DEBUG] AutoScaling Group create configuration: %#v", autoScalingGroupOpts)
	_, err := conn.CreateAutoScalingGroup(&autoScalingGroupOpts)
	if err != nil {
		return fmt.Errorf("Error creating Autoscaling Group: %s", err)
	}

	d.SetId(d.Get("name").(string))
	log.Printf("[INFO] AutoScaling Group ID: %s", d.Id())

	if err := waitForASGCapacity(d, meta, capacitySatifiedCreate); err != nil {
		return err
	}

	if _, ok := d.GetOk("enabled_metrics"); ok {
		metricsErr := enableASGMetricsCollection(d, conn)
		if metricsErr != nil {
			return metricsErr
		}
	}

	return resourceAwsAutoscalingGroupRead(d, meta)
}

func resourceAwsAutoscalingGroupRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).autoscalingconn

	g, err := getAwsAutoscalingGroup(d.Id(), conn)
	if err != nil {
		return err
	}
	if g == nil {
		log.Printf("[INFO] Autoscaling Group %q not found", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("availability_zones", flattenStringList(g.AvailabilityZones))
	d.Set("default_cooldown", g.DefaultCooldown)
	d.Set("desired_capacity", g.DesiredCapacity)
	d.Set("health_check_grace_period", g.HealthCheckGracePeriod)
	d.Set("health_check_type", g.HealthCheckType)
	d.Set("launch_configuration", g.LaunchConfigurationName)
	d.Set("load_balancers", flattenStringList(g.LoadBalancerNames))
	d.Set("min_size", g.MinSize)
	d.Set("max_size", g.MaxSize)
	d.Set("placement_group", g.PlacementGroup)
	d.Set("name", g.AutoScalingGroupName)
	d.Set("tag", autoscalingTagDescriptionsToSlice(g.Tags))
	d.Set("vpc_zone_identifier", strings.Split(*g.VPCZoneIdentifier, ","))

	// If no termination polices are explicitly configured and the upstream state
	// is only using the "Default" policy, clear the state to make it consistent
	// with the default AWS create API behavior.
	_, ok := d.GetOk("termination_policies")
	if !ok && len(g.TerminationPolicies) == 1 && *g.TerminationPolicies[0] == "Default" {
		d.Set("termination_policies", []interface{}{})
	} else {
		d.Set("termination_policies", flattenStringList(g.TerminationPolicies))
	}

	if g.EnabledMetrics != nil {
		if err := d.Set("enabled_metrics", flattenAsgEnabledMetrics(g.EnabledMetrics)); err != nil {
			log.Printf("[WARN] Error setting metrics for (%s): %s", d.Id(), err)
		}
		d.Set("metrics_granularity", g.EnabledMetrics[0].Granularity)
	}

	return nil
}

func resourceAwsAutoscalingGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).autoscalingconn
	shouldWaitForCapacity := false

	opts := autoscaling.UpdateAutoScalingGroupInput{
		AutoScalingGroupName: aws.String(d.Id()),
	}

	if d.HasChange("default_cooldown") {
		opts.DefaultCooldown = aws.Int64(int64(d.Get("default_cooldown").(int)))
	}

	if d.HasChange("desired_capacity") {
		opts.DesiredCapacity = aws.Int64(int64(d.Get("desired_capacity").(int)))
		shouldWaitForCapacity = true
	}

	if d.HasChange("launch_configuration") {
		opts.LaunchConfigurationName = aws.String(d.Get("launch_configuration").(string))
	}

	if d.HasChange("min_size") {
		opts.MinSize = aws.Int64(int64(d.Get("min_size").(int)))
		shouldWaitForCapacity = true
	}

	if d.HasChange("max_size") {
		opts.MaxSize = aws.Int64(int64(d.Get("max_size").(int)))
	}

	if d.HasChange("health_check_grace_period") {
		opts.HealthCheckGracePeriod = aws.Int64(int64(d.Get("health_check_grace_period").(int)))
	}

	if d.HasChange("health_check_type") {
		opts.HealthCheckGracePeriod = aws.Int64(int64(d.Get("health_check_grace_period").(int)))
		opts.HealthCheckType = aws.String(d.Get("health_check_type").(string))
	}

	if d.HasChange("vpc_zone_identifier") {
		opts.VPCZoneIdentifier = expandVpcZoneIdentifiers(d.Get("vpc_zone_identifier").(*schema.Set).List())
	}

	if d.HasChange("availability_zones") {
		if v, ok := d.GetOk("availability_zones"); ok && v.(*schema.Set).Len() > 0 {
			opts.AvailabilityZones = expandStringList(v.(*schema.Set).List())
		}
	}

	if d.HasChange("placement_group") {
		opts.PlacementGroup = aws.String(d.Get("placement_group").(string))
	}

	if d.HasChange("termination_policies") {
		// If the termination policy is set to null, we need to explicitly set
		// it back to "Default", or the API won't reset it for us.
		if v, ok := d.GetOk("termination_policies"); ok && len(v.([]interface{})) > 0 {
			opts.TerminationPolicies = expandStringList(v.([]interface{}))
		} else {
			log.Printf("[DEBUG] Explictly setting null termination policy to 'Default'")
			opts.TerminationPolicies = aws.StringSlice([]string{"Default"})
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

	if shouldWaitForCapacity {
		waitForASGCapacity(d, meta, capacitySatifiedUpdate)
	}

	if d.HasChange("enabled_metrics") {
		updateASGMetricsCollection(d, conn)
	}

	return resourceAwsAutoscalingGroupRead(d, meta)
}

func resourceAwsAutoscalingGroupDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).autoscalingconn

	// Read the autoscaling group first. If it doesn't exist, we're done.
	// We need the group in order to check if there are instances attached.
	// If so, we need to remove those first.
	g, err := getAwsAutoscalingGroup(d.Id(), conn)
	if err != nil {
		return err
	}
	if g == nil {
		log.Printf("[INFO] Autoscaling Group %q not found", d.Id())
		d.SetId("")
		return nil
	}
	if len(g.Instances) > 0 || *g.DesiredCapacity > 0 {
		if err := resourceAwsAutoscalingGroupDrain(d, meta); err != nil {
			return err
		}
	}

	log.Printf("[DEBUG] AutoScaling Group destroy: %v", d.Id())
	deleteopts := autoscaling.DeleteAutoScalingGroupInput{
		AutoScalingGroupName: aws.String(d.Id()),
		ForceDelete:          aws.Bool(d.Get("force_delete").(bool)),
	}

	// We retry the delete operation to handle InUse/InProgress errors coming
	// from scaling operations. We should be able to sneak in a delete in between
	// scaling operations within 5m.
	err = resource.Retry(5*time.Minute, func() *resource.RetryError {
		if _, err := conn.DeleteAutoScalingGroup(&deleteopts); err != nil {
			if awserr, ok := err.(awserr.Error); ok {
				switch awserr.Code() {
				case "InvalidGroup.NotFound":
					// Already gone? Sure!
					return nil
				case "ResourceInUse", "ScalingActivityInProgress":
					// These are retryable
					return resource.RetryableError(awserr)
				}
			}
			// Didn't recognize the error, so shouldn't retry.
			return resource.NonRetryableError(err)
		}
		// Successful delete
		return nil
	})
	if err != nil {
		return err
	}

	return resource.Retry(5*time.Minute, func() *resource.RetryError {
		if g, _ = getAwsAutoscalingGroup(d.Id(), conn); g != nil {
			return resource.RetryableError(
				fmt.Errorf("Auto Scaling Group still exists"))
		}
		return nil
	})
}

func getAwsAutoscalingGroup(
	asgName string,
	conn *autoscaling.AutoScaling) (*autoscaling.Group, error) {

	describeOpts := autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{aws.String(asgName)},
	}

	log.Printf("[DEBUG] AutoScaling Group describe configuration: %#v", describeOpts)
	describeGroups, err := conn.DescribeAutoScalingGroups(&describeOpts)
	if err != nil {
		autoscalingerr, ok := err.(awserr.Error)
		if ok && autoscalingerr.Code() == "InvalidGroup.NotFound" {
			return nil, nil
		}

		return nil, fmt.Errorf("Error retrieving AutoScaling groups: %s", err)
	}

	// Search for the autoscaling group
	for idx, asc := range describeGroups.AutoScalingGroups {
		if *asc.AutoScalingGroupName == asgName {
			return describeGroups.AutoScalingGroups[idx], nil
		}
	}

	return nil, nil
}

func resourceAwsAutoscalingGroupDrain(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).autoscalingconn

	if d.Get("force_delete").(bool) {
		log.Printf("[DEBUG] Skipping ASG drain, force_delete was set.")
		return nil
	}

	// First, set the capacity to zero so the group will drain
	log.Printf("[DEBUG] Reducing autoscaling group capacity to zero")
	opts := autoscaling.UpdateAutoScalingGroupInput{
		AutoScalingGroupName: aws.String(d.Id()),
		DesiredCapacity:      aws.Int64(0),
		MinSize:              aws.Int64(0),
		MaxSize:              aws.Int64(0),
	}
	if _, err := conn.UpdateAutoScalingGroup(&opts); err != nil {
		return fmt.Errorf("Error setting capacity to zero to drain: %s", err)
	}

	// Next, wait for the autoscale group to drain
	log.Printf("[DEBUG] Waiting for group to have zero instances")
	return resource.Retry(10*time.Minute, func() *resource.RetryError {
		g, err := getAwsAutoscalingGroup(d.Id(), conn)
		if err != nil {
			return resource.NonRetryableError(err)
		}
		if g == nil {
			log.Printf("[INFO] Autoscaling Group %q not found", d.Id())
			d.SetId("")
			return nil
		}

		if len(g.Instances) == 0 {
			return nil
		}

		return resource.RetryableError(
			fmt.Errorf("group still has %d instances", len(g.Instances)))
	})
}

func enableASGMetricsCollection(d *schema.ResourceData, conn *autoscaling.AutoScaling) error {
	props := &autoscaling.EnableMetricsCollectionInput{
		AutoScalingGroupName: aws.String(d.Id()),
		Granularity:          aws.String(d.Get("metrics_granularity").(string)),
		Metrics:              expandStringList(d.Get("enabled_metrics").(*schema.Set).List()),
	}

	log.Printf("[INFO] Enabling metrics collection for the ASG: %s", d.Id())
	_, metricsErr := conn.EnableMetricsCollection(props)
	if metricsErr != nil {
		return metricsErr
	}

	return nil
}

func updateASGMetricsCollection(d *schema.ResourceData, conn *autoscaling.AutoScaling) error {

	o, n := d.GetChange("enabled_metrics")
	if o == nil {
		o = new(schema.Set)
	}
	if n == nil {
		n = new(schema.Set)
	}

	os := o.(*schema.Set)
	ns := n.(*schema.Set)

	disableMetrics := os.Difference(ns)
	if disableMetrics.Len() != 0 {
		props := &autoscaling.DisableMetricsCollectionInput{
			AutoScalingGroupName: aws.String(d.Id()),
			Metrics:              expandStringList(disableMetrics.List()),
		}

		_, err := conn.DisableMetricsCollection(props)
		if err != nil {
			return fmt.Errorf("Failure to Disable metrics collection types for ASG %s: %s", d.Id(), err)
		}
	}

	enabledMetrics := ns.Difference(os)
	if enabledMetrics.Len() != 0 {
		props := &autoscaling.EnableMetricsCollectionInput{
			AutoScalingGroupName: aws.String(d.Id()),
			Metrics:              expandStringList(enabledMetrics.List()),
		}

		_, err := conn.EnableMetricsCollection(props)
		if err != nil {
			return fmt.Errorf("Failure to Enable metrics collection types for ASG %s: %s", d.Id(), err)
		}
	}

	return nil
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

func expandVpcZoneIdentifiers(list []interface{}) *string {
	strs := make([]string, len(list))
	for _, s := range list {
		strs = append(strs, s.(string))
	}
	return aws.String(strings.Join(strs, ","))
}
