package aws

import (
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform/flatmap"
	"github.com/hashicorp/terraform/helper/config"
	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/autoscaling"
)

func resource_aws_autoscaling_group_create(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	autoscalingconn := p.autoscalingconn

	// Merge the diff into the state so that we have all the attributes
	// properly.
	rs := s.MergeDiff(d)

	var err error
	autoScalingGroupOpts := autoscaling.CreateAutoScalingGroup{}

	if rs.Attributes["min_size"] != "" {
		autoScalingGroupOpts.MinSize, err = strconv.Atoi(rs.Attributes["min_size"])
		autoScalingGroupOpts.SetMinSize = true
	}

	if rs.Attributes["max_size"] != "" {
		autoScalingGroupOpts.MaxSize, err = strconv.Atoi(rs.Attributes["max_size"])
		autoScalingGroupOpts.SetMaxSize = true
	}

	if rs.Attributes["default_cooldown"] != "" {
		autoScalingGroupOpts.DefaultCooldown, err = strconv.Atoi(rs.Attributes["default_cooldown"])
		autoScalingGroupOpts.SetDefaultCooldown = true
	}

	if rs.Attributes["desired_capacity"] != "" {
		autoScalingGroupOpts.DesiredCapacity, err = strconv.Atoi(rs.Attributes["desired_capacity"])
		autoScalingGroupOpts.SetDesiredCapacity = true
	}

	if rs.Attributes["health_check_grace_period"] != "" {
		autoScalingGroupOpts.HealthCheckGracePeriod, err = strconv.Atoi(rs.Attributes["health_check_grace_period"])
		autoScalingGroupOpts.SetHealthCheckGracePeriod = true
	}

	if err != nil {
		return nil, fmt.Errorf("Error parsing configuration: %s", err)
	}

	if _, ok := rs.Attributes["availability_zones.#"]; ok {
		autoScalingGroupOpts.AvailZone = expandStringList(flatmap.Expand(
			rs.Attributes, "availability_zones").([]interface{}))
	}

	if _, ok := rs.Attributes["load_balancers.#"]; ok {
		autoScalingGroupOpts.LoadBalancerNames = expandStringList(flatmap.Expand(
			rs.Attributes, "load_balancers").([]interface{}))
	}

	if _, ok := rs.Attributes["vpc_identifier.#"]; ok {
		autoScalingGroupOpts.VPCZoneIdentifier = expandStringList(flatmap.Expand(
			rs.Attributes, "vpc_identifier").([]interface{}))
	}

	autoScalingGroupOpts.Name = rs.Attributes["name"]
	autoScalingGroupOpts.HealthCheckType = rs.Attributes["health_check_type"]
	autoScalingGroupOpts.LaunchConfigurationName = rs.Attributes["launch_configuration"]

	log.Printf("[DEBUG] AutoScaling Group create configuration: %#v", autoScalingGroupOpts)
	_, err = autoscalingconn.CreateAutoScalingGroup(&autoScalingGroupOpts)
	if err != nil {
		return nil, fmt.Errorf("Error creating AutoScaling Group: %s", err)
	}

	rs.ID = rs.Attributes["name"]
	rs.Dependencies = []terraform.ResourceDependency{
		terraform.ResourceDependency{ID: rs.Attributes["launch_configuration"]},
	}

	log.Printf("[INFO] AutoScaling Group ID: %s", rs.ID)

	g, err := resource_aws_autoscaling_group_retrieve(rs.ID, autoscalingconn)
	if err != nil {
		return rs, err
	}

	return resource_aws_autoscaling_group_update_state(rs, g)
}

func resource_aws_autoscaling_group_update(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	autoscalingconn := p.autoscalingconn
	rs := s.MergeDiff(d)

	opts := autoscaling.UpdateAutoScalingGroup{
		Name: rs.ID,
	}

	var err error

	if _, ok := d.Attributes["min_size"]; ok {
		opts.MinSize, err = strconv.Atoi(rs.Attributes["min_size"])
		opts.SetMinSize = true
	}

	if _, ok := d.Attributes["max_size"]; ok {
		opts.MaxSize, err = strconv.Atoi(rs.Attributes["max_size"])
		opts.SetMaxSize = true
	}

	if err != nil {
		return s, fmt.Errorf("Error parsing configuration: %s", err)
	}

	log.Printf("[DEBUG] AutoScaling Group update configuration: %#v", opts)

	_, err = autoscalingconn.UpdateAutoScalingGroup(&opts)

	if err != nil {
		return rs, fmt.Errorf("Error updating AutoScaling group: %s", err)
	}

	g, err := resource_aws_autoscaling_group_retrieve(rs.ID, autoscalingconn)

	if err != nil {
		return rs, err
	}

	return resource_aws_autoscaling_group_update_state(rs, g)
}

func resource_aws_autoscaling_group_destroy(
	s *terraform.ResourceState,
	meta interface{}) error {
	p := meta.(*ResourceProvider)
	autoscalingconn := p.autoscalingconn

	log.Printf("[DEBUG] AutoScaling Group destroy: %v", s.ID)

	deleteopts := autoscaling.DeleteAutoScalingGroup{Name: s.ID}

	// You can force an autoscaling group to delete
	// even if it's in the process of scaling a resource.
	// Normally, you would set the min-size and max-size to 0,0
	// and then delete the group. This bypasses that and leaves
	// resources potentially dangling.
	if s.Attributes["force_delete"] != "" {
		deleteopts.ForceDelete = true
	}

	_, err := autoscalingconn.DeleteAutoScalingGroup(&deleteopts)

	if err != nil {
		autoscalingerr, ok := err.(*autoscaling.Error)
		if ok && autoscalingerr.Code == "InvalidGroup.NotFound" {
			return nil
		}
		return err
	}

	return nil
}

func resource_aws_autoscaling_group_refresh(
	s *terraform.ResourceState,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	autoscalingconn := p.autoscalingconn

	g, err := resource_aws_autoscaling_group_retrieve(s.ID, autoscalingconn)

	if err != nil {
		return s, err
	}

	return resource_aws_autoscaling_group_update_state(s, g)
}

func resource_aws_autoscaling_group_diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.ResourceDiff, error) {

	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"availability_zone":         diff.AttrTypeCreate,
			"default_cooldown":          diff.AttrTypeCreate,
			"desired_capacity":          diff.AttrTypeCreate,
			"force_delete":              diff.AttrTypeCreate,
			"health_check_grace_period": diff.AttrTypeCreate,
			"health_check_type":         diff.AttrTypeCreate,
			"launch_configuration":      diff.AttrTypeCreate,
			"load_balancers":            diff.AttrTypeCreate,
			"name":                      diff.AttrTypeCreate,
			"vpc_zone_identifier":       diff.AttrTypeCreate,

			"max_size": diff.AttrTypeUpdate,
			"min_size": diff.AttrTypeUpdate,
		},

		ComputedAttrs: []string{
			"health_check_grace_period",
			"health_check_type",
			"default_cooldown",
			"vpc_zone_identifier",
			"desired_capacity",
			"force_delete",
		},
	}

	return b.Diff(s, c)
}

func resource_aws_autoscaling_group_update_state(
	s *terraform.ResourceState,
	g *autoscaling.AutoScalingGroup) (*terraform.ResourceState, error) {

	s.Attributes["min_size"] = strconv.Itoa(g.MinSize)
	s.Attributes["max_size"] = strconv.Itoa(g.MaxSize)
	s.Attributes["default_cooldown"] = strconv.Itoa(g.DefaultCooldown)
	s.Attributes["name"] = g.Name
	s.Attributes["desired_capacity"] = strconv.Itoa(g.DesiredCapacity)
	s.Attributes["health_check_grace_period"] = strconv.Itoa(g.HealthCheckGracePeriod)
	s.Attributes["health_check_type"] = g.HealthCheckType
	s.Attributes["launch_configuration"] = g.LaunchConfigurationName
	s.Attributes["vpc_zone_identifier"] = g.VPCZoneIdentifier

	// Flatten our group values
	toFlatten := make(map[string]interface{})

	// Special case the return of amazons load balancers names in the XML having
	// a blank entry
	if len(g.LoadBalancerNames) > 0 && g.LoadBalancerNames[0].LoadBalancerName != "" {
		toFlatten["load_balancers"] = flattenLoadBalancers(g.LoadBalancerNames)
	}

	toFlatten["availability_zones"] = flattenAvailabilityZones(g.AvailabilityZones)

	for k, v := range flatmap.Flatten(toFlatten) {
		s.Attributes[k] = v
	}

	return s, nil
}

// Returns a single group by its ID
func resource_aws_autoscaling_group_retrieve(id string, autoscalingconn *autoscaling.AutoScaling) (*autoscaling.AutoScalingGroup, error) {
	describeOpts := autoscaling.DescribeAutoScalingGroups{
		Names: []string{id},
	}

	log.Printf("[DEBUG] AutoScaling Group describe configuration: %#v", describeOpts)

	describeGroups, err := autoscalingconn.DescribeAutoScalingGroups(&describeOpts)

	if err != nil {
		return nil, fmt.Errorf("Error retrieving AutoScaling groups: %s", err)
	}

	// Verify AWS returned our sg
	if len(describeGroups.AutoScalingGroups) != 1 ||
		describeGroups.AutoScalingGroups[0].Name != id {
		if err != nil {
			return nil, fmt.Errorf("Unable to find AutoScaling group: %#v", describeGroups.AutoScalingGroups)
		}
	}

	g := describeGroups.AutoScalingGroups[0]

	return &g, nil
}

func resource_aws_autoscaling_group_validation() *config.Validator {
	return &config.Validator{
		Required: []string{
			"name",
			"max_size",
			"min_size",
			"availability_zones.*",
			"launch_configuration",
		},
		Optional: []string{
			"health_check_grace_period",
			"health_check_type",
			"desired_capacity",
			"force_delete",
		},
	}
}
