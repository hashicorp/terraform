package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsAutoscalingGroup() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsAutoscalingGroupRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"availability_zones": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"default_cooldown": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"desired_capacity": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"health_check_grace_period": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"health_check_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"launch_configuration": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"load_balancers": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"new_instances_protected_from_scale_in": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"max_size": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"min_size": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"placement_group": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"service_linked_role_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"target_group_arns": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"termination_policies": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"vpc_zone_identifier": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsAutoscalingGroupRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).autoscalingconn
	d.SetId(time.Now().UTC().String())

	groupName := d.Get("name").(string)

	input := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{
			aws.String(groupName),
		},
	}

	log.Printf("[DEBUG] Reading Autoscaling Group: %s", input)

	result, err := conn.DescribeAutoScalingGroups(input)

	log.Printf("[DEBUG] Checking for error: %s", err)

	if err != nil {
		return fmt.Errorf("error describing AutoScaling Groups: %s", err)
	}

	log.Printf("[DEBUG] Found Autoscaling Group: %s", result)

	if len(result.AutoScalingGroups) < 1 {
		return fmt.Errorf("Your query did not return any results. Please try a different search criteria.")
	}

	if len(result.AutoScalingGroups) > 1 {
		return fmt.Errorf("Your query returned more than one result. Please try a more " +
			"specific search criteria.")
	}

	// If execution made it to this point, we have exactly one 1 group returned
	// and this is a safe operation
	group := result.AutoScalingGroups[0]

	log.Printf("[DEBUG] aws_autoscaling_group - Single Auto Scaling Group found: %s", *group.AutoScalingGroupName)

	err1 := groupDescriptionAttributes(d, group)
	return err1
}

// Populate group attribute fields with the returned group
func groupDescriptionAttributes(d *schema.ResourceData, group *autoscaling.Group) error {
	log.Printf("[DEBUG] Setting attributes: %s", group)
	d.Set("name", group.AutoScalingGroupName)
	d.Set("arn", group.AutoScalingGroupARN)
	if err := d.Set("availability_zones", aws.StringValueSlice(group.AvailabilityZones)); err != nil {
		return err
	}
	d.Set("default_cooldown", group.DefaultCooldown)
	d.Set("desired_capacity", group.DesiredCapacity)
	d.Set("health_check_grace_period", group.HealthCheckGracePeriod)
	d.Set("health_check_type", group.HealthCheckType)
	d.Set("launch_configuration", group.LaunchConfigurationName)
	if err := d.Set("load_balancers", aws.StringValueSlice(group.LoadBalancerNames)); err != nil {
		return err
	}
	d.Set("new_instances_protected_from_scale_in", group.NewInstancesProtectedFromScaleIn)
	d.Set("max_size", group.MaxSize)
	d.Set("min_size", group.MinSize)
	d.Set("placement_group", group.PlacementGroup)
	d.Set("service_linked_role_arn", group.ServiceLinkedRoleARN)
	d.Set("status", group.Status)
	if err := d.Set("target_group_arns", aws.StringValueSlice(group.TargetGroupARNs)); err != nil {
		return err
	}
	if err := d.Set("termination_policies", aws.StringValueSlice(group.TerminationPolicies)); err != nil {
		return err
	}
	d.Set("vpc_zone_identifier", group.VPCZoneIdentifier)

	return nil
}
