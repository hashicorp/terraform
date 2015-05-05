package aws

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/autoscaling"
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
				ForceNew: true,
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
				ForceNew: true,
			},

			"availability_zones": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"load_balancers": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"vpc_zone_identifier": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				ForceNew: true,
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
	autoscalingconn := meta.(*AWSClient).autoscalingconn

	var autoScalingGroupOpts autoscaling.CreateAutoScalingGroupInput
	autoScalingGroupOpts.AutoScalingGroupName = aws.String(d.Get("name").(string))
	autoScalingGroupOpts.LaunchConfigurationName = aws.String(d.Get("launch_configuration").(string))
	autoScalingGroupOpts.MinSize = aws.Long(int64(d.Get("min_size").(int)))
	autoScalingGroupOpts.MaxSize = aws.Long(int64(d.Get("max_size").(int)))
	autoScalingGroupOpts.AvailabilityZones = expandStringList(
		d.Get("availability_zones").(*schema.Set).List())

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
		exp := expandStringList(v.(*schema.Set).List())
		strs := make([]string, len(exp))
		for _, s := range exp {
			strs = append(strs, *s)
		}
		autoScalingGroupOpts.VPCZoneIdentifier = aws.String(strings.Join(strs, ","))
	}

	if v, ok := d.GetOk("termination_policies"); ok && v.(*schema.Set).Len() > 0 {
		autoScalingGroupOpts.TerminationPolicies = expandStringList(
			v.(*schema.Set).List())
	}

	log.Printf("[DEBUG] AutoScaling Group create configuration: %#v", autoScalingGroupOpts)
	_, err := autoscalingconn.CreateAutoScalingGroup(&autoScalingGroupOpts)
	if err != nil {
		return fmt.Errorf("Error creating Autoscaling Group: %s", err)
	}

	d.SetId(d.Get("name").(string))
	log.Printf("[INFO] AutoScaling Group ID: %s", d.Id())

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
	autoscalingconn := meta.(*AWSClient).autoscalingconn

	opts := autoscaling.UpdateAutoScalingGroupInput{
		AutoScalingGroupName: aws.String(d.Id()),
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

	if err := setAutoscalingTags(autoscalingconn, d); err != nil {
		return err
	} else {
		d.SetPartial("tag")
	}

	log.Printf("[DEBUG] AutoScaling Group update configuration: %#v", opts)
	_, err := autoscalingconn.UpdateAutoScalingGroup(&opts)
	if err != nil {
		d.Partial(true)
		return fmt.Errorf("Error updating Autoscaling group: %s", err)
	}

	return resourceAwsAutoscalingGroupRead(d, meta)
}

func resourceAwsAutoscalingGroupDelete(d *schema.ResourceData, meta interface{}) error {
	autoscalingconn := meta.(*AWSClient).autoscalingconn

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

	if _, err := autoscalingconn.DeleteAutoScalingGroup(&deleteopts); err != nil {
		autoscalingerr, ok := err.(aws.APIError)
		if ok && autoscalingerr.Code == "InvalidGroup.NotFound" {
			return nil
		}
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
	meta interface{}) (*autoscaling.AutoScalingGroup, error) {
	autoscalingconn := meta.(*AWSClient).autoscalingconn

	describeOpts := autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{aws.String(d.Id())},
	}

	log.Printf("[DEBUG] AutoScaling Group describe configuration: %#v", describeOpts)
	describeGroups, err := autoscalingconn.DescribeAutoScalingGroups(&describeOpts)
	if err != nil {
		autoscalingerr, ok := err.(aws.APIError)
		if ok && autoscalingerr.Code == "InvalidGroup.NotFound" {
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
	autoscalingconn := meta.(*AWSClient).autoscalingconn

	// First, set the capacity to zero so the group will drain
	log.Printf("[DEBUG] Reducing autoscaling group capacity to zero")
	opts := autoscaling.UpdateAutoScalingGroupInput{
		AutoScalingGroupName: aws.String(d.Id()),
		DesiredCapacity:      aws.Long(0),
		MinSize:              aws.Long(0),
		MaxSize:              aws.Long(0),
	}
	if _, err := autoscalingconn.UpdateAutoScalingGroup(&opts); err != nil {
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
