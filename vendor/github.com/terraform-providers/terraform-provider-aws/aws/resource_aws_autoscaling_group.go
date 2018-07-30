package aws

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/elbv2"
)

func resourceAwsAutoscalingGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsAutoscalingGroupCreate,
		Read:   resourceAwsAutoscalingGroupRead,
		Update: resourceAwsAutoscalingGroupUpdate,
		Delete: resourceAwsAutoscalingGroupDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name_prefix"},
				ValidateFunc:  validateMaxLength(255),
			},
			"name_prefix": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateMaxLength(255 - resource.UniqueIDSuffixLength),
			},

			"launch_configuration": {
				Type:     schema.TypeString,
				Required: true,
			},

			"desired_capacity": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},

			"min_elb_capacity": {
				Type:     schema.TypeInt,
				Optional: true,
			},

			"min_size": {
				Type:     schema.TypeInt,
				Required: true,
			},

			"max_size": {
				Type:     schema.TypeInt,
				Required: true,
			},

			"default_cooldown": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},

			"force_delete": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"health_check_grace_period": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  300,
			},

			"health_check_type": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"availability_zones": {
				Type:             schema.TypeSet,
				Optional:         true,
				Computed:         true,
				Elem:             &schema.Schema{Type: schema.TypeString},
				Set:              schema.HashString,
				DiffSuppressFunc: suppressAutoscalingGroupAvailabilityZoneDiffs,
			},

			"placement_group": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"load_balancers": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"vpc_zone_identifier": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"termination_policies": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"wait_for_capacity_timeout": {
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

			"wait_for_elb_capacity": {
				Type:     schema.TypeInt,
				Optional: true,
			},

			"enabled_metrics": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"suspended_processes": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"metrics_granularity": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "1Minute",
			},

			"protect_from_scale_in": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"target_group_arns": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"initial_lifecycle_hook": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"default_result": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"heartbeat_timeout": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"lifecycle_transition": {
							Type:     schema.TypeString,
							Required: true,
						},
						"notification_metadata": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"notification_target_arn": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"role_arn": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},

			"tag": autoscalingTagSchema(),

			"tags": &schema.Schema{
				Type:          schema.TypeList,
				Optional:      true,
				Elem:          &schema.Schema{Type: schema.TypeMap},
				ConflictsWith: []string{"tag"},
			},

			"service_linked_role_arn": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func generatePutLifecycleHookInputs(asgName string, cfgs []interface{}) []autoscaling.PutLifecycleHookInput {
	res := make([]autoscaling.PutLifecycleHookInput, 0, len(cfgs))

	for _, raw := range cfgs {
		cfg := raw.(map[string]interface{})

		input := autoscaling.PutLifecycleHookInput{
			AutoScalingGroupName: &asgName,
			LifecycleHookName:    aws.String(cfg["name"].(string)),
		}

		if v, ok := cfg["default_result"]; ok && v.(string) != "" {
			input.DefaultResult = aws.String(v.(string))
		}

		if v, ok := cfg["heartbeat_timeout"]; ok && v.(int) > 0 {
			input.HeartbeatTimeout = aws.Int64(int64(v.(int)))
		}

		if v, ok := cfg["lifecycle_transition"]; ok && v.(string) != "" {
			input.LifecycleTransition = aws.String(v.(string))
		}

		if v, ok := cfg["notification_metadata"]; ok && v.(string) != "" {
			input.NotificationMetadata = aws.String(v.(string))
		}

		if v, ok := cfg["notification_target_arn"]; ok && v.(string) != "" {
			input.NotificationTargetARN = aws.String(v.(string))
		}

		if v, ok := cfg["role_arn"]; ok && v.(string) != "" {
			input.RoleARN = aws.String(v.(string))
		}

		res = append(res, input)
	}

	return res
}

func resourceAwsAutoscalingGroupCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).autoscalingconn

	var asgName string
	if v, ok := d.GetOk("name"); ok {
		asgName = v.(string)
	} else {
		if v, ok := d.GetOk("name_prefix"); ok {
			asgName = resource.PrefixedUniqueId(v.(string))
		} else {
			asgName = resource.PrefixedUniqueId("tf-asg-")
		}
		d.Set("name", asgName)
	}

	createOpts := autoscaling.CreateAutoScalingGroupInput{
		AutoScalingGroupName:             aws.String(asgName),
		LaunchConfigurationName:          aws.String(d.Get("launch_configuration").(string)),
		NewInstancesProtectedFromScaleIn: aws.Bool(d.Get("protect_from_scale_in").(bool)),
	}
	updateOpts := autoscaling.UpdateAutoScalingGroupInput{
		AutoScalingGroupName: aws.String(asgName),
	}

	initialLifecycleHooks := d.Get("initial_lifecycle_hook").(*schema.Set).List()
	twoPhases := len(initialLifecycleHooks) > 0

	minSize := aws.Int64(int64(d.Get("min_size").(int)))
	maxSize := aws.Int64(int64(d.Get("max_size").(int)))

	if twoPhases {
		createOpts.MinSize = aws.Int64(int64(0))
		createOpts.MaxSize = aws.Int64(int64(0))

		updateOpts.MinSize = minSize
		updateOpts.MaxSize = maxSize

		if v, ok := d.GetOk("desired_capacity"); ok {
			updateOpts.DesiredCapacity = aws.Int64(int64(v.(int)))
		}
	} else {
		createOpts.MinSize = minSize
		createOpts.MaxSize = maxSize

		if v, ok := d.GetOk("desired_capacity"); ok {
			createOpts.DesiredCapacity = aws.Int64(int64(v.(int)))
		}
	}

	// Availability Zones are optional if VPC Zone Identifer(s) are specified
	if v, ok := d.GetOk("availability_zones"); ok && v.(*schema.Set).Len() > 0 {
		createOpts.AvailabilityZones = expandStringList(v.(*schema.Set).List())
	}

	resourceID := d.Get("name").(string)
	if v, ok := d.GetOk("tag"); ok {
		var err error
		createOpts.Tags, err = autoscalingTagsFromMap(
			setToMapByKey(v.(*schema.Set), "key"), resourceID)
		if err != nil {
			return err
		}
	}

	if v, ok := d.GetOk("tags"); ok {
		tags, err := autoscalingTagsFromList(v.([]interface{}), resourceID)
		if err != nil {
			return err
		}

		createOpts.Tags = append(createOpts.Tags, tags...)
	}

	if v, ok := d.GetOk("default_cooldown"); ok {
		createOpts.DefaultCooldown = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("health_check_type"); ok && v.(string) != "" {
		createOpts.HealthCheckType = aws.String(v.(string))
	}

	if v, ok := d.GetOk("health_check_grace_period"); ok {
		createOpts.HealthCheckGracePeriod = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("placement_group"); ok {
		createOpts.PlacementGroup = aws.String(v.(string))
	}

	if v, ok := d.GetOk("load_balancers"); ok && v.(*schema.Set).Len() > 0 {
		createOpts.LoadBalancerNames = expandStringList(
			v.(*schema.Set).List())
	}

	if v, ok := d.GetOk("vpc_zone_identifier"); ok && v.(*schema.Set).Len() > 0 {
		createOpts.VPCZoneIdentifier = expandVpcZoneIdentifiers(v.(*schema.Set).List())
	}

	if v, ok := d.GetOk("termination_policies"); ok && len(v.([]interface{})) > 0 {
		createOpts.TerminationPolicies = expandStringList(v.([]interface{}))
	}

	if v, ok := d.GetOk("target_group_arns"); ok && len(v.(*schema.Set).List()) > 0 {
		createOpts.TargetGroupARNs = expandStringList(v.(*schema.Set).List())
	}

	if v, ok := d.GetOk("service_linked_role_arn"); ok {
		createOpts.ServiceLinkedRoleARN = aws.String(v.(string))
	}

	log.Printf("[DEBUG] AutoScaling Group create configuration: %#v", createOpts)
	_, err := conn.CreateAutoScalingGroup(&createOpts)
	if err != nil {
		return fmt.Errorf("Error creating AutoScaling Group: %s", err)
	}

	d.SetId(d.Get("name").(string))
	log.Printf("[INFO] AutoScaling Group ID: %s", d.Id())

	if twoPhases {
		for _, hook := range generatePutLifecycleHookInputs(asgName, initialLifecycleHooks) {
			if err = resourceAwsAutoscalingLifecycleHookPutOp(conn, &hook); err != nil {
				return fmt.Errorf("Error creating initial lifecycle hooks: %s", err)
			}
		}

		_, err = conn.UpdateAutoScalingGroup(&updateOpts)
		if err != nil {
			return fmt.Errorf("Error setting AutoScaling Group initial capacity: %s", err)
		}
	}

	if err := waitForASGCapacity(d, meta, capacitySatisfiedCreate); err != nil {
		return err
	}

	if _, ok := d.GetOk("suspended_processes"); ok {
		suspendedProcessesErr := enableASGSuspendedProcesses(d, conn)
		if suspendedProcessesErr != nil {
			return suspendedProcessesErr
		}
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
		log.Printf("[WARN] Autoscaling Group (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("availability_zones", flattenStringList(g.AvailabilityZones))
	d.Set("default_cooldown", g.DefaultCooldown)
	d.Set("arn", g.AutoScalingGroupARN)
	d.Set("desired_capacity", g.DesiredCapacity)
	d.Set("health_check_grace_period", g.HealthCheckGracePeriod)
	d.Set("health_check_type", g.HealthCheckType)
	d.Set("launch_configuration", g.LaunchConfigurationName)
	d.Set("load_balancers", flattenStringList(g.LoadBalancerNames))

	if err := d.Set("suspended_processes", flattenAsgSuspendedProcesses(g.SuspendedProcesses)); err != nil {
		log.Printf("[WARN] Error setting suspended_processes for %q: %s", d.Id(), err)
	}
	if err := d.Set("target_group_arns", flattenStringList(g.TargetGroupARNs)); err != nil {
		log.Printf("[ERR] Error setting target groups: %s", err)
	}
	d.Set("min_size", g.MinSize)
	d.Set("max_size", g.MaxSize)
	d.Set("placement_group", g.PlacementGroup)
	d.Set("name", g.AutoScalingGroupName)
	d.Set("service_linked_role_arn", g.ServiceLinkedRoleARN)

	var tagList, tagsList []*autoscaling.TagDescription
	var tagOk, tagsOk bool
	var v interface{}

	if v, tagOk = d.GetOk("tag"); tagOk {
		tags := setToMapByKey(v.(*schema.Set), "key")
		for _, t := range g.Tags {
			if _, ok := tags[*t.Key]; ok {
				tagList = append(tagList, t)
			}
		}
		d.Set("tag", autoscalingTagDescriptionsToSlice(tagList))
	}

	if v, tagsOk = d.GetOk("tags"); tagsOk {
		tags := map[string]struct{}{}
		for _, tag := range v.([]interface{}) {
			attr, ok := tag.(map[string]interface{})
			if !ok {
				continue
			}

			key, ok := attr["key"].(string)
			if !ok {
				continue
			}

			tags[key] = struct{}{}
		}

		for _, t := range g.Tags {
			if _, ok := tags[*t.Key]; ok {
				tagsList = append(tagsList, t)
			}
		}
		d.Set("tags", autoscalingTagDescriptionsToSlice(tagsList))
	}

	if !tagOk && !tagsOk {
		d.Set("tag", autoscalingTagDescriptionsToSlice(g.Tags))
	}

	if len(*g.VPCZoneIdentifier) > 0 {
		d.Set("vpc_zone_identifier", strings.Split(*g.VPCZoneIdentifier, ","))
	} else {
		d.Set("vpc_zone_identifier", []string{})
	}

	d.Set("protect_from_scale_in", g.NewInstancesProtectedFromScaleIn)

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
	} else {
		d.Set("enabled_metrics", nil)
	}

	return nil
}

func resourceAwsAutoscalingGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).autoscalingconn
	shouldWaitForCapacity := false

	opts := autoscaling.UpdateAutoScalingGroupInput{
		AutoScalingGroupName: aws.String(d.Id()),
	}

	opts.NewInstancesProtectedFromScaleIn = aws.Bool(d.Get("protect_from_scale_in").(bool))

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
			log.Printf("[DEBUG] Explicitly setting null termination policy to 'Default'")
			opts.TerminationPolicies = aws.StringSlice([]string{"Default"})
		}
	}

	if d.HasChange("service_linked_role_arn") {
		opts.ServiceLinkedRoleARN = aws.String(d.Get("service_linked_role_arn").(string))
	}

	if err := setAutoscalingTags(conn, d); err != nil {
		return err
	}

	if d.HasChange("tag") {
		d.SetPartial("tag")
	}

	if d.HasChange("tags") {
		d.SetPartial("tags")
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

	if d.HasChange("target_group_arns") {

		o, n := d.GetChange("target_group_arns")
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
			_, err := conn.DetachLoadBalancerTargetGroups(&autoscaling.DetachLoadBalancerTargetGroupsInput{
				AutoScalingGroupName: aws.String(d.Id()),
				TargetGroupARNs:      remove,
			})
			if err != nil {
				return fmt.Errorf("[WARN] Error updating Load Balancers Target Groups for AutoScaling Group (%s), error: %s", d.Id(), err)
			}
		}

		if len(add) > 0 {
			_, err := conn.AttachLoadBalancerTargetGroups(&autoscaling.AttachLoadBalancerTargetGroupsInput{
				AutoScalingGroupName: aws.String(d.Id()),
				TargetGroupARNs:      add,
			})
			if err != nil {
				return fmt.Errorf("[WARN] Error updating Load Balancers Target Groups for AutoScaling Group (%s), error: %s", d.Id(), err)
			}
		}
	}

	if shouldWaitForCapacity {
		if err := waitForASGCapacity(d, meta, capacitySatisfiedUpdate); err != nil {
			return errwrap.Wrapf("Error waiting for AutoScaling Group Capacity: {{err}}", err)
		}
	}

	if d.HasChange("enabled_metrics") {
		if err := updateASGMetricsCollection(d, conn); err != nil {
			return errwrap.Wrapf("Error updating AutoScaling Group Metrics collection: {{err}}", err)
		}
	}

	if d.HasChange("suspended_processes") {
		if err := updateASGSuspendedProcesses(d, conn); err != nil {
			return errwrap.Wrapf("Error updating AutoScaling Group Suspended Processes: {{err}}", err)
		}
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
		log.Printf("[WARN] Autoscaling Group (%s) not found, removing from state", d.Id())
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
	err = resource.Retry(d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
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

	return resource.Retry(d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
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
	return resource.Retry(d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		g, err := getAwsAutoscalingGroup(d.Id(), conn)
		if err != nil {
			return resource.NonRetryableError(err)
		}
		if g == nil {
			log.Printf("[WARN] Autoscaling Group (%s) not found, removing from state", d.Id())
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

func enableASGSuspendedProcesses(d *schema.ResourceData, conn *autoscaling.AutoScaling) error {
	props := &autoscaling.ScalingProcessQuery{
		AutoScalingGroupName: aws.String(d.Id()),
		ScalingProcesses:     expandStringList(d.Get("suspended_processes").(*schema.Set).List()),
	}

	_, err := conn.SuspendProcesses(props)
	if err != nil {
		return err
	}

	return nil
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

func updateASGSuspendedProcesses(d *schema.ResourceData, conn *autoscaling.AutoScaling) error {
	o, n := d.GetChange("suspended_processes")
	if o == nil {
		o = new(schema.Set)
	}
	if n == nil {
		n = new(schema.Set)
	}

	os := o.(*schema.Set)
	ns := n.(*schema.Set)

	resumeProcesses := os.Difference(ns)
	if resumeProcesses.Len() != 0 {
		props := &autoscaling.ScalingProcessQuery{
			AutoScalingGroupName: aws.String(d.Id()),
			ScalingProcesses:     expandStringList(resumeProcesses.List()),
		}

		_, err := conn.ResumeProcesses(props)
		if err != nil {
			return fmt.Errorf("Error Resuming Processes for ASG %q: %s", d.Id(), err)
		}
	}

	suspendedProcesses := ns.Difference(os)
	if suspendedProcesses.Len() != 0 {
		props := &autoscaling.ScalingProcessQuery{
			AutoScalingGroupName: aws.String(d.Id()),
			ScalingProcesses:     expandStringList(suspendedProcesses.List()),
		}

		_, err := conn.SuspendProcesses(props)
		if err != nil {
			return fmt.Errorf("Error Suspending Processes for ASG %q: %s", d.Id(), err)
		}
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
			Granularity:          aws.String(d.Get("metrics_granularity").(string)),
		}

		_, err := conn.EnableMetricsCollection(props)
		if err != nil {
			return fmt.Errorf("Failure to Enable metrics collection types for ASG %s: %s", d.Id(), err)
		}
	}

	return nil
}

// getELBInstanceStates returns a mapping of the instance states of all the ELBs attached to the
// provided ASG.
//
// Note that this is the instance state function for ELB Classic.
//
// Nested like: lbName -> instanceId -> instanceState
func getELBInstanceStates(g *autoscaling.Group, meta interface{}) (map[string]map[string]string, error) {
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

// getTargetGroupInstanceStates returns a mapping of the instance states of
// all the ALB target groups attached to the provided ASG.
//
// Note that this is the instance state function for Application Load
// Balancing (aka ELBv2).
//
// Nested like: targetGroupARN -> instanceId -> instanceState
func getTargetGroupInstanceStates(g *autoscaling.Group, meta interface{}) (map[string]map[string]string, error) {
	targetInstanceStates := make(map[string]map[string]string)
	elbv2conn := meta.(*AWSClient).elbv2conn

	for _, targetGroupARN := range g.TargetGroupARNs {
		targetInstanceStates[*targetGroupARN] = make(map[string]string)
		opts := &elbv2.DescribeTargetHealthInput{TargetGroupArn: targetGroupARN}
		r, err := elbv2conn.DescribeTargetHealth(opts)
		if err != nil {
			return nil, err
		}
		for _, desc := range r.TargetHealthDescriptions {
			if desc.Target == nil || desc.Target.Id == nil || desc.TargetHealth == nil || desc.TargetHealth.State == nil {
				continue
			}
			targetInstanceStates[*targetGroupARN][*desc.Target.Id] = *desc.TargetHealth.State
		}
	}

	return targetInstanceStates, nil
}

func expandVpcZoneIdentifiers(list []interface{}) *string {
	strs := make([]string, len(list))
	for _, s := range list {
		strs = append(strs, s.(string))
	}
	return aws.String(strings.Join(strs, ","))
}
