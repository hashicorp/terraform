package aws

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/codedeploy"
)

func resourceAwsCodeDeployDeploymentGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCodeDeployDeploymentGroupCreate,
		Read:   resourceAwsCodeDeployDeploymentGroupRead,
		Update: resourceAwsCodeDeployDeploymentGroupUpdate,
		Delete: resourceAwsCodeDeployDeploymentGroupDelete,

		Schema: map[string]*schema.Schema{
			"app_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if len(value) > 100 {
						errors = append(errors, fmt.Errorf(
							"%q cannot exceed 100 characters", k))
					}
					return
				},
			},

			"deployment_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if len(value) > 100 {
						errors = append(errors, fmt.Errorf(
							"%q cannot exceed 100 characters", k))
					}
					return
				},
			},

			"service_role_arn": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"autoscaling_groups": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"deployment_config_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "CodeDeployDefault.OneAtATime",
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if len(value) > 100 {
						errors = append(errors, fmt.Errorf(
							"%q cannot exceed 100 characters", k))
					}
					return
				},
			},

			"ec2_tag_filter": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"type": &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateTagFilters,
						},

						"value": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
				Set: resourceAwsCodeDeployTagFilterHash,
			},

			"on_premises_instance_tag_filter": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"type": &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateTagFilters,
						},

						"value": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
				Set: resourceAwsCodeDeployTagFilterHash,
			},
		},
	}
}

func resourceAwsCodeDeployDeploymentGroupCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codedeployconn

	application := d.Get("app_name").(string)
	deploymentGroup := d.Get("deployment_group_name").(string)

	input := codedeploy.CreateDeploymentGroupInput{
		ApplicationName:     aws.String(application),
		DeploymentGroupName: aws.String(deploymentGroup),
		ServiceRoleArn:      aws.String(d.Get("service_role_arn").(string)),
	}
	if attr, ok := d.GetOk("deployment_config_name"); ok {
		input.DeploymentConfigName = aws.String(attr.(string))
	}
	if attr, ok := d.GetOk("autoscaling_groups"); ok {
		input.AutoScalingGroups = expandStringList(attr.(*schema.Set).List())
	}
	if attr, ok := d.GetOk("on_premises_instance_tag_filters"); ok {
		onPremFilters := buildOnPremTagFilters(attr.(*schema.Set).List())
		input.OnPremisesInstanceTagFilters = onPremFilters
	}
	if attr, ok := d.GetOk("ec2_tag_filter"); ok {
		ec2TagFilters := buildEC2TagFilters(attr.(*schema.Set).List())
		input.Ec2TagFilters = ec2TagFilters
	}

	// Retry to handle IAM role eventual consistency.
	var resp *codedeploy.CreateDeploymentGroupOutput
	var err error
	err = resource.Retry(2*time.Minute, func() error {
		resp, err = conn.CreateDeploymentGroup(&input)
		if err != nil {
			codedeployErr, ok := err.(awserr.Error)
			if !ok {
				return &resource.RetryError{Err: err}
			}
			if codedeployErr.Code() == "InvalidRoleException" {
				log.Printf("[DEBUG] Trying to create deployment group again: %q",
					codedeployErr.Message())
				return err
			}

			return &resource.RetryError{Err: err}
		}
		return nil
	})
	if err != nil {
		return err
	}

	d.SetId(*resp.DeploymentGroupId)

	return resourceAwsCodeDeployDeploymentGroupRead(d, meta)
}

func resourceAwsCodeDeployDeploymentGroupRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codedeployconn

	log.Printf("[DEBUG] Reading CodeDeploy DeploymentGroup %s", d.Id())
	resp, err := conn.GetDeploymentGroup(&codedeploy.GetDeploymentGroupInput{
		ApplicationName:     aws.String(d.Get("app_name").(string)),
		DeploymentGroupName: aws.String(d.Get("deployment_group_name").(string)),
	})
	if err != nil {
		return err
	}

	d.Set("app_name", *resp.DeploymentGroupInfo.ApplicationName)
	d.Set("autoscaling_groups", resp.DeploymentGroupInfo.AutoScalingGroups)
	d.Set("deployment_config_name", *resp.DeploymentGroupInfo.DeploymentConfigName)
	d.Set("deployment_group_name", *resp.DeploymentGroupInfo.DeploymentGroupName)
	d.Set("service_role_arn", *resp.DeploymentGroupInfo.ServiceRoleArn)
	if err := d.Set("ec2_tag_filter", ec2TagFiltersToMap(resp.DeploymentGroupInfo.Ec2TagFilters)); err != nil {
		return err
	}
	if err := d.Set("on_premises_instance_tag_filter", onPremisesTagFiltersToMap(resp.DeploymentGroupInfo.OnPremisesInstanceTagFilters)); err != nil {
		return err
	}

	return nil
}

func resourceAwsCodeDeployDeploymentGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codedeployconn

	input := codedeploy.UpdateDeploymentGroupInput{
		ApplicationName:            aws.String(d.Get("app_name").(string)),
		CurrentDeploymentGroupName: aws.String(d.Get("deployment_group_name").(string)),
	}

	if d.HasChange("autoscaling_groups") {
		_, n := d.GetChange("autoscaling_groups")
		input.AutoScalingGroups = expandStringList(n.(*schema.Set).List())
	}
	if d.HasChange("deployment_config_name") {
		_, n := d.GetChange("deployment_config_name")
		input.DeploymentConfigName = aws.String(n.(string))
	}
	if d.HasChange("deployment_group_name") {
		_, n := d.GetChange("deployment_group_name")
		input.NewDeploymentGroupName = aws.String(n.(string))
	}

	// TagFilters aren't like tags. They don't append. They simply replace.
	if d.HasChange("on_premises_instance_tag_filter") {
		_, n := d.GetChange("on_premises_instance_tag_filter")
		onPremFilters := buildOnPremTagFilters(n.(*schema.Set).List())
		input.OnPremisesInstanceTagFilters = onPremFilters
	}
	if d.HasChange("ec2_tag_filter") {
		_, n := d.GetChange("ec2_tag_filter")
		ec2Filters := buildEC2TagFilters(n.(*schema.Set).List())
		input.Ec2TagFilters = ec2Filters
	}

	log.Printf("[DEBUG] Updating CodeDeploy DeploymentGroup %s", d.Id())
	_, err := conn.UpdateDeploymentGroup(&input)
	if err != nil {
		return err
	}

	return resourceAwsCodeDeployDeploymentGroupRead(d, meta)
}

func resourceAwsCodeDeployDeploymentGroupDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codedeployconn

	log.Printf("[DEBUG] Deleting CodeDeploy DeploymentGroup %s", d.Id())
	_, err := conn.DeleteDeploymentGroup(&codedeploy.DeleteDeploymentGroupInput{
		ApplicationName:     aws.String(d.Get("app_name").(string)),
		DeploymentGroupName: aws.String(d.Get("deployment_group_name").(string)),
	})
	if err != nil {
		return err
	}

	d.SetId("")

	return nil
}

// buildOnPremTagFilters converts raw schema lists into a list of
// codedeploy.TagFilters.
func buildOnPremTagFilters(configured []interface{}) []*codedeploy.TagFilter {
	filters := make([]*codedeploy.TagFilter, 0)
	for _, raw := range configured {
		var filter codedeploy.TagFilter
		m := raw.(map[string]interface{})

		filter.Key = aws.String(m["key"].(string))
		filter.Type = aws.String(m["type"].(string))
		filter.Value = aws.String(m["value"].(string))

		filters = append(filters, &filter)
	}

	return filters
}

// buildEC2TagFilters converts raw schema lists into a list of
// codedeploy.EC2TagFilters.
func buildEC2TagFilters(configured []interface{}) []*codedeploy.EC2TagFilter {
	filters := make([]*codedeploy.EC2TagFilter, 0)
	for _, raw := range configured {
		var filter codedeploy.EC2TagFilter
		m := raw.(map[string]interface{})

		filter.Key = aws.String(m["key"].(string))
		filter.Type = aws.String(m["type"].(string))
		filter.Value = aws.String(m["value"].(string))

		filters = append(filters, &filter)
	}

	return filters
}

// ec2TagFiltersToMap converts lists of tag filters into a []map[string]string.
func ec2TagFiltersToMap(list []*codedeploy.EC2TagFilter) []map[string]string {
	result := make([]map[string]string, 0, len(list))
	for _, tf := range list {
		l := make(map[string]string)
		if *tf.Key != "" {
			l["key"] = *tf.Key
		}
		if *tf.Value != "" {
			l["value"] = *tf.Value
		}
		if *tf.Type != "" {
			l["type"] = *tf.Type
		}
		result = append(result, l)
	}
	return result
}

// onPremisesTagFiltersToMap converts lists of on-prem tag filters into a []map[string]string.
func onPremisesTagFiltersToMap(list []*codedeploy.TagFilter) []map[string]string {
	result := make([]map[string]string, 0, len(list))
	for _, tf := range list {
		l := make(map[string]string)
		if *tf.Key != "" {
			l["key"] = *tf.Key
		}
		if *tf.Value != "" {
			l["value"] = *tf.Value
		}
		if *tf.Type != "" {
			l["type"] = *tf.Type
		}
		result = append(result, l)
	}
	return result
}

func resourceAwsCodeDeployTagFilterHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	// Nothing's actually required in tag filters, so we must check the
	// presence of all values before attempting a hash.
	if v, ok := m["key"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	if v, ok := m["type"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	if v, ok := m["value"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}

	return hashcode.String(buf.String())
}
