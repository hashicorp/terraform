package aws

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/flatmap"
	"github.com/hashicorp/terraform/helper/config"
	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/autoscaling"
)

func resource_aws_launch_configuration_create(
	s *terraform.InstanceState,
	d *terraform.InstanceDiff,
	meta interface{}) (*terraform.InstanceState, error) {
	p := meta.(*ResourceProvider)
	autoscalingconn := p.autoscalingconn

	// Merge the diff into the state so that we have all the attributes
	// properly.
	rs := s.MergeDiff(d)

	var err error
	createLaunchConfigurationOpts := autoscaling.CreateLaunchConfiguration{}

	if rs.Attributes["image_id"] != "" {
		createLaunchConfigurationOpts.ImageId = rs.Attributes["image_id"]
	}

	if rs.Attributes["instance_type"] != "" {
		createLaunchConfigurationOpts.InstanceType = rs.Attributes["instance_type"]
	}

	if rs.Attributes["instance_id"] != "" {
		createLaunchConfigurationOpts.InstanceId = rs.Attributes["instance_id"]
	}

	if rs.Attributes["key_name"] != "" {
		createLaunchConfigurationOpts.KeyName = rs.Attributes["key_name"]
	}

	if err != nil {
		return nil, fmt.Errorf("Error parsing configuration: %s", err)
	}

	if _, ok := rs.Attributes["security_groups.#"]; ok {
		createLaunchConfigurationOpts.SecurityGroups = expandStringList(flatmap.Expand(
			rs.Attributes, "security_groups").([]interface{}))
	}

	if rs.Attributes["user_data"] != "" {
		createLaunchConfigurationOpts.UserData = rs.Attributes["user_data"]
	}

	createLaunchConfigurationOpts.Name = rs.Attributes["name"]

	log.Printf("[DEBUG] autoscaling create launch configuration: %#v", createLaunchConfigurationOpts)
	_, err = autoscalingconn.CreateLaunchConfiguration(&createLaunchConfigurationOpts)
	if err != nil {
		return nil, fmt.Errorf("Error creating launch configuration: %s", err)
	}

	rs.ID = rs.Attributes["name"]

	log.Printf("[INFO] launch configuration ID: %s", rs.ID)

	g, err := resource_aws_launch_configuration_retrieve(rs.ID, autoscalingconn)
	if err != nil {
		return rs, err
	}

	return resource_aws_launch_configuration_update_state(rs, g)
}

func resource_aws_launch_configuration_update(
	s *terraform.InstanceState,
	d *terraform.InstanceDiff,
	meta interface{}) (*terraform.InstanceState, error) {
	panic("Update for AWS Launch Configuration is not supported")
}

func resource_aws_launch_configuration_destroy(
	s *terraform.InstanceState,
	meta interface{}) error {
	p := meta.(*ResourceProvider)
	autoscalingconn := p.autoscalingconn

	log.Printf("[DEBUG] Launch Configuration destroy: %v", s.ID)

	_, err := autoscalingconn.DeleteLaunchConfiguration(&autoscaling.DeleteLaunchConfiguration{Name: s.ID})

	if err != nil {
		autoscalingerr, ok := err.(*autoscaling.Error)
		if ok && autoscalingerr.Code == "InvalidConfiguration.NotFound" {
			return nil
		}
		return err
	}

	return nil
}

func resource_aws_launch_configuration_refresh(
	s *terraform.InstanceState,
	meta interface{}) (*terraform.InstanceState, error) {
	p := meta.(*ResourceProvider)
	autoscalingconn := p.autoscalingconn

	g, err := resource_aws_launch_configuration_retrieve(s.ID, autoscalingconn)

	if err != nil {
		return s, err
	}

	return resource_aws_launch_configuration_update_state(s, g)
}

func resource_aws_launch_configuration_diff(
	s *terraform.InstanceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.InstanceDiff, error) {

	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"image_id":        diff.AttrTypeCreate,
			"instance_id":     diff.AttrTypeCreate,
			"instance_type":   diff.AttrTypeCreate,
			"key_name":        diff.AttrTypeCreate,
			"name":            diff.AttrTypeCreate,
			"security_groups": diff.AttrTypeCreate,
			"user_data":       diff.AttrTypeCreate,
		},

		ComputedAttrs: []string{
			"key_name",
		},
	}

	return b.Diff(s, c)
}

func resource_aws_launch_configuration_update_state(
	s *terraform.InstanceState,
	lc *autoscaling.LaunchConfiguration) (*terraform.InstanceState, error) {

	s.Attributes["image_id"] = lc.ImageId
	s.Attributes["instance_type"] = lc.InstanceType
	s.Attributes["key_name"] = lc.KeyName
	s.Attributes["name"] = lc.Name

	// Flatten our group values
	toFlatten := make(map[string]interface{})

	if len(lc.SecurityGroups) > 0 && lc.SecurityGroups[0].SecurityGroup != "" {
		toFlatten["security_groups"] = flattenAutoscalingSecurityGroups(lc.SecurityGroups)
	}

	for k, v := range flatmap.Flatten(toFlatten) {
		s.Attributes[k] = v
	}

	return s, nil
}

// Returns a single group by its ID
func resource_aws_launch_configuration_retrieve(id string, autoscalingconn *autoscaling.AutoScaling) (*autoscaling.LaunchConfiguration, error) {
	describeOpts := autoscaling.DescribeLaunchConfigurations{
		Names: []string{id},
	}

	log.Printf("[DEBUG] launch configuration describe configuration: %#v", describeOpts)

	describConfs, err := autoscalingconn.DescribeLaunchConfigurations(&describeOpts)

	if err != nil {
		return nil, fmt.Errorf("Error retrieving launch configuration: %s", err)
	}

	// Verify AWS returned our launch configuration
	if len(describConfs.LaunchConfigurations) != 1 ||
		describConfs.LaunchConfigurations[0].Name != id {
		if err != nil {
			return nil, fmt.Errorf("Unable to find launch configuration: %#v", describConfs.LaunchConfigurations)
		}
	}

	l := describConfs.LaunchConfigurations[0]

	return &l, nil
}

func resource_aws_launch_configuration_validation() *config.Validator {
	return &config.Validator{
		Required: []string{
			"name",
			"image_id",
			"instance_type",
		},
		Optional: []string{
			"key_name",
			"security_groups.*",
			"user_data",
		},
	}
}
