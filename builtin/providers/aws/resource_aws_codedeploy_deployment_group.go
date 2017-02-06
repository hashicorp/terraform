package aws

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"sort"
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

			"deployment_style": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"deployment_option": &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateDeploymentOption,
						},

						"deployment_type": &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateDeploymentType,
						},
					},
				},
			},

			"service_role_arn": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"alarm_configuration": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"alarms": &schema.Schema{
							Type:     schema.TypeSet,
							MaxItems: 10,
							Optional: true,
							Set:      schema.HashString,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},

						"enabled": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},

						"ignore_poll_alarm_failure": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
					},
				},
			},

			"load_balancer_info": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"elb_info": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							Set:      elbInfoHash,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
					},
				},
			},

			"auto_rollback_configuration": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},

						"events": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							Set:      schema.HashString,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
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

			"trigger_configuration": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"trigger_events": &schema.Schema{
							Type:     schema.TypeSet,
							Required: true,
							Set:      schema.HashString,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validateTriggerEvent,
							},
						},

						"trigger_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"trigger_target_arn": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Set: resourceAwsCodeDeployTriggerConfigHash,
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

	if attr, ok := d.GetOk("deployment_style"); ok {
		input.DeploymentStyle = buildDeploymentStyle(attr.([]interface{}))
	}

	if attr, ok := d.GetOk("deployment_config_name"); ok {
		input.DeploymentConfigName = aws.String(attr.(string))
	}

	if attr, ok := d.GetOk("autoscaling_groups"); ok {
		input.AutoScalingGroups = expandStringList(attr.(*schema.Set).List())
	}

	if attr, ok := d.GetOk("on_premises_instance_tag_filter"); ok {
		onPremFilters := buildOnPremTagFilters(attr.(*schema.Set).List())
		input.OnPremisesInstanceTagFilters = onPremFilters
	}

	if attr, ok := d.GetOk("ec2_tag_filter"); ok {
		ec2TagFilters := buildEC2TagFilters(attr.(*schema.Set).List())
		input.Ec2TagFilters = ec2TagFilters
	}

	if attr, ok := d.GetOk("trigger_configuration"); ok {
		triggerConfigs := buildTriggerConfigs(attr.(*schema.Set).List())
		input.TriggerConfigurations = triggerConfigs
	}

	if attr, ok := d.GetOk("auto_rollback_configuration"); ok {
		input.AutoRollbackConfiguration = buildAutoRollbackConfig(attr.([]interface{}))
	}

	if attr, ok := d.GetOk("alarm_configuration"); ok {
		input.AlarmConfiguration = buildAlarmConfig(attr.([]interface{}))
	}

	if attr, ok := d.GetOk("load_balancer_info"); ok {
		input.LoadBalancerInfo = buildLoadBalancerInfo(attr.([]interface{}))
	}

	// Retry to handle IAM role eventual consistency.
	var resp *codedeploy.CreateDeploymentGroupOutput
	var err error
	err = resource.Retry(5*time.Minute, func() *resource.RetryError {
		resp, err = conn.CreateDeploymentGroup(&input)
		if err != nil {
			retry := false
			codedeployErr, ok := err.(awserr.Error)
			if !ok {
				return resource.NonRetryableError(err)
			}
			if codedeployErr.Code() == "InvalidRoleException" {
				retry = true
			}
			if codedeployErr.Code() == "InvalidTriggerConfigException" {
				r := regexp.MustCompile("^Topic ARN .+ is not valid$")
				if r.MatchString(codedeployErr.Message()) {
					retry = true
				}
			}
			if retry {
				log.Printf("[DEBUG] Trying to create deployment group again: %q",
					codedeployErr.Message())
				return resource.RetryableError(err)
			}

			return resource.NonRetryableError(err)
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
		if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "DeploymentGroupDoesNotExistException" {
			log.Printf("[INFO] CodeDeployment DeploymentGroup %s not found", d.Get("deployment_group_name").(string))
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("app_name", resp.DeploymentGroupInfo.ApplicationName)
	d.Set("autoscaling_groups", resp.DeploymentGroupInfo.AutoScalingGroups)
	d.Set("deployment_config_name", resp.DeploymentGroupInfo.DeploymentConfigName)
	d.Set("deployment_group_name", resp.DeploymentGroupInfo.DeploymentGroupName)
	d.Set("service_role_arn", resp.DeploymentGroupInfo.ServiceRoleArn)

	if err := d.Set("deployment_style", deploymentStyleToMap(resp.DeploymentGroupInfo.DeploymentStyle)); err != nil {
		return err
	}

	if err := d.Set("ec2_tag_filter", ec2TagFiltersToMap(resp.DeploymentGroupInfo.Ec2TagFilters)); err != nil {
		return err
	}

	if err := d.Set("on_premises_instance_tag_filter", onPremisesTagFiltersToMap(resp.DeploymentGroupInfo.OnPremisesInstanceTagFilters)); err != nil {
		return err
	}

	if err := d.Set("trigger_configuration", triggerConfigsToMap(resp.DeploymentGroupInfo.TriggerConfigurations)); err != nil {
		return err
	}

	if err := d.Set("auto_rollback_configuration", autoRollbackConfigToMap(resp.DeploymentGroupInfo.AutoRollbackConfiguration)); err != nil {
		return err
	}

	if err := d.Set("alarm_configuration", alarmConfigToMap(resp.DeploymentGroupInfo.AlarmConfiguration)); err != nil {
		return err
	}

	if err := d.Set("load_balancer_info", loadBalancerInfoToMap(resp.DeploymentGroupInfo.LoadBalancerInfo)); err != nil {
		return err
	}

	return nil
}

func resourceAwsCodeDeployDeploymentGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codedeployconn

	input := codedeploy.UpdateDeploymentGroupInput{
		ApplicationName:            aws.String(d.Get("app_name").(string)),
		CurrentDeploymentGroupName: aws.String(d.Get("deployment_group_name").(string)),
		ServiceRoleArn:             aws.String(d.Get("service_role_arn").(string)),
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

	if d.HasChange("deployment_style") {
		_, n := d.GetChange("deployment_style")
		input.DeploymentStyle = buildDeploymentStyle(n.([]interface{}))
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

	if d.HasChange("trigger_configuration") {
		_, n := d.GetChange("trigger_configuration")
		triggerConfigs := buildTriggerConfigs(n.(*schema.Set).List())
		input.TriggerConfigurations = triggerConfigs
	}

	if d.HasChange("auto_rollback_configuration") {
		_, n := d.GetChange("auto_rollback_configuration")
		input.AutoRollbackConfiguration = buildAutoRollbackConfig(n.([]interface{}))
	}

	if d.HasChange("alarm_configuration") {
		_, n := d.GetChange("alarm_configuration")
		input.AlarmConfiguration = buildAlarmConfig(n.([]interface{}))
	}

	if d.HasChange("load_balancer_info") {
		_, n := d.GetChange("load_balancer_info")
		input.LoadBalancerInfo = buildLoadBalancerInfo(n.([]interface{}))
	}

	log.Printf("[DEBUG] Updating CodeDeploy DeploymentGroup %s", d.Id())
	// Retry to handle IAM role eventual consistency.
	err := resource.Retry(5*time.Minute, func() *resource.RetryError {
		_, err := conn.UpdateDeploymentGroup(&input)
		if err != nil {
			retry := false
			codedeployErr, ok := err.(awserr.Error)
			if !ok {
				return resource.NonRetryableError(err)
			}
			if codedeployErr.Code() == "InvalidRoleException" {
				retry = true
			}
			if codedeployErr.Code() == "InvalidTriggerConfigException" {
				r := regexp.MustCompile("^Topic ARN .+ is not valid$")
				if r.MatchString(codedeployErr.Message()) {
					retry = true
				}
			}
			if retry {
				log.Printf("[DEBUG] Retrying Code Deployment Group Update: %q",
					codedeployErr.Message())
				return resource.RetryableError(err)
			}

			return resource.NonRetryableError(err)
		}
		return nil
	})

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

		if v, ok := m["key"]; ok {
			filter.Key = aws.String(v.(string))
		}
		if v, ok := m["type"]; ok {
			filter.Type = aws.String(v.(string))
		}
		if v, ok := m["value"]; ok {
			filter.Value = aws.String(v.(string))
		}

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

// buildTriggerConfigs converts a raw schema list into a list of
// codedeploy.TriggerConfig.
func buildTriggerConfigs(configured []interface{}) []*codedeploy.TriggerConfig {
	configs := make([]*codedeploy.TriggerConfig, 0, len(configured))
	for _, raw := range configured {
		var config codedeploy.TriggerConfig
		m := raw.(map[string]interface{})

		config.TriggerEvents = expandStringSet(m["trigger_events"].(*schema.Set))
		config.TriggerName = aws.String(m["trigger_name"].(string))
		config.TriggerTargetArn = aws.String(m["trigger_target_arn"].(string))

		configs = append(configs, &config)
	}
	return configs
}

// buildAutoRollbackConfig converts a raw schema list containing a map[string]interface{}
// into a single codedeploy.AutoRollbackConfiguration
func buildAutoRollbackConfig(configured []interface{}) *codedeploy.AutoRollbackConfiguration {
	result := &codedeploy.AutoRollbackConfiguration{}

	if len(configured) == 1 {
		config := configured[0].(map[string]interface{})
		result.Enabled = aws.Bool(config["enabled"].(bool))
		result.Events = expandStringSet(config["events"].(*schema.Set))
	} else { // delete the configuration
		result.Enabled = aws.Bool(false)
		result.Events = make([]*string, 0)
	}

	return result
}

// buildAlarmConfig converts a raw schema list containing a map[string]interface{}
// into a single codedeploy.AlarmConfiguration
func buildAlarmConfig(configured []interface{}) *codedeploy.AlarmConfiguration {
	result := &codedeploy.AlarmConfiguration{}

	if len(configured) == 1 {
		config := configured[0].(map[string]interface{})
		names := expandStringSet(config["alarms"].(*schema.Set))
		alarms := make([]*codedeploy.Alarm, 0, len(names))

		for _, name := range names {
			alarm := &codedeploy.Alarm{
				Name: name,
			}
			alarms = append(alarms, alarm)
		}

		result.Alarms = alarms
		result.Enabled = aws.Bool(config["enabled"].(bool))
		result.IgnorePollAlarmFailure = aws.Bool(config["ignore_poll_alarm_failure"].(bool))
	} else { // delete the configuration
		result.Alarms = make([]*codedeploy.Alarm, 0)
		result.Enabled = aws.Bool(false)
		result.IgnorePollAlarmFailure = aws.Bool(false)
	}

	return result
}

// buildDeploymentStyle converts a raw schema list containing a map[string]interface{}
// into a single codedeploy.DeploymentStyle object
func buildDeploymentStyle(list []interface{}) *codedeploy.DeploymentStyle {
	result := &codedeploy.DeploymentStyle{}

	if len(list) == 1 {
		style := list[0].(map[string]interface{})
		result.DeploymentOption = aws.String(style["deployment_option"].(string))
		result.DeploymentType = aws.String(style["deployment_type"].(string))
	}

	return result
}

// buildLoadBalancerInfo converts a raw schema list containing a map[string]interface{}
// into a single codedeploy.LoadBalancerInfo object
func buildLoadBalancerInfo(list []interface{}) *codedeploy.LoadBalancerInfo {
	if len(list) == 0 || list[0] == nil {
		return nil
	}

	lbInfo := list[0].(map[string]interface{})
	elbs := lbInfo["elb_info"].(*schema.Set).List()

	loadBalancerInfo := &codedeploy.LoadBalancerInfo{
		ElbInfoList: make([]*codedeploy.ELBInfo, 0, len(elbs)),
	}

	for _, v := range elbs {
		elb := v.(map[string]interface{})
		loadBalancerInfo.ElbInfoList = append(loadBalancerInfo.ElbInfoList, &codedeploy.ELBInfo{
			Name: aws.String(elb["name"].(string)),
		})
	}

	return loadBalancerInfo
}

// ec2TagFiltersToMap converts lists of tag filters into a []map[string]string.
func ec2TagFiltersToMap(list []*codedeploy.EC2TagFilter) []map[string]string {
	result := make([]map[string]string, 0, len(list))
	for _, tf := range list {
		l := make(map[string]string)
		if tf.Key != nil && *tf.Key != "" {
			l["key"] = *tf.Key
		}
		if tf.Value != nil && *tf.Value != "" {
			l["value"] = *tf.Value
		}
		if tf.Type != nil && *tf.Type != "" {
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
		if tf.Key != nil && *tf.Key != "" {
			l["key"] = *tf.Key
		}
		if tf.Value != nil && *tf.Value != "" {
			l["value"] = *tf.Value
		}
		if tf.Type != nil && *tf.Type != "" {
			l["type"] = *tf.Type
		}
		result = append(result, l)
	}
	return result
}

// triggerConfigsToMap converts a list of []*codedeploy.TriggerConfig into a []map[string]interface{}
func triggerConfigsToMap(list []*codedeploy.TriggerConfig) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(list))
	for _, tc := range list {
		item := make(map[string]interface{})
		item["trigger_events"] = schema.NewSet(schema.HashString, flattenStringList(tc.TriggerEvents))
		item["trigger_name"] = *tc.TriggerName
		item["trigger_target_arn"] = *tc.TriggerTargetArn
		result = append(result, item)
	}
	return result
}

// autoRollbackConfigToMap converts a codedeploy.AutoRollbackConfiguration
// into a []map[string]interface{} list containing a single item
func autoRollbackConfigToMap(config *codedeploy.AutoRollbackConfiguration) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, 1)

	// only create configurations that are enabled or temporarily disabled (retaining events)
	// otherwise empty configurations will be created
	if config != nil && (*config.Enabled == true || len(config.Events) > 0) {
		item := make(map[string]interface{})
		item["enabled"] = *config.Enabled
		item["events"] = schema.NewSet(schema.HashString, flattenStringList(config.Events))
		result = append(result, item)
	}

	return result
}

// alarmConfigToMap converts a codedeploy.AlarmConfiguration
// into a []map[string]interface{} list containing a single item
func alarmConfigToMap(config *codedeploy.AlarmConfiguration) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, 1)

	// only create configurations that are enabled or temporarily disabled (retaining alarms)
	// otherwise empty configurations will be created
	if config != nil && (*config.Enabled == true || len(config.Alarms) > 0) {
		names := make([]*string, 0, len(config.Alarms))
		for _, alarm := range config.Alarms {
			names = append(names, alarm.Name)
		}

		item := make(map[string]interface{})
		item["alarms"] = schema.NewSet(schema.HashString, flattenStringList(names))
		item["enabled"] = *config.Enabled
		item["ignore_poll_alarm_failure"] = *config.IgnorePollAlarmFailure

		result = append(result, item)
	}

	return result
}

// deploymentStyleToMap converts a codedeploy.DeploymentStyle object
// into a []map[string]interface{} list containing a single item
func deploymentStyleToMap(style *codedeploy.DeploymentStyle) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, 1)

	if style == nil {
		return result
	}

	item := make(map[string]interface{})
	item["deployment_option"] = *style.DeploymentOption
	item["deployment_type"] = *style.DeploymentType
	result = append(result, item)
	return result
}

// loadBalancerInfoToMap converts a codedeploy.LoadBalancerInfo object
// into a []map[string]interface{} list containing a single item
func loadBalancerInfoToMap(loadBalancerInfo *codedeploy.LoadBalancerInfo) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, 1)

	if loadBalancerInfo == nil {
		return result
	}

	elbs := make([]interface{}, 0, len(loadBalancerInfo.ElbInfoList))
	for _, elb := range loadBalancerInfo.ElbInfoList {
		item := make(map[string]interface{})
		item["name"] = *elb.Name
		elbs = append(elbs, item)
	}

	lbInfo := make(map[string]interface{})
	lbInfo["elb_info"] = schema.NewSet(elbInfoHash, elbs)
	result = append(result, lbInfo)

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

func resourceAwsCodeDeployTriggerConfigHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["trigger_name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["trigger_target_arn"].(string)))

	if triggerEvents, ok := m["trigger_events"]; ok {
		names := triggerEvents.(*schema.Set).List()
		strings := make([]string, len(names))
		for i, raw := range names {
			strings[i] = raw.(string)
		}
		sort.Strings(strings)

		for _, s := range strings {
			buf.WriteString(fmt.Sprintf("%s-", s))
		}
	}
	return hashcode.String(buf.String())
}

func elbInfoHash(v interface{}) int {
	var buf bytes.Buffer

	if v == nil {
		return hashcode.String(buf.String())
	}

	m := v.(map[string]interface{})
	if v, ok := m["name"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}

	return hashcode.String(buf.String())
}

func validateTriggerEvent(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	triggerEvents := map[string]bool{
		"DeploymentStart":    true,
		"DeploymentStop":     true,
		"DeploymentSuccess":  true,
		"DeploymentFailure":  true,
		"DeploymentRollback": true,
		"InstanceStart":      true,
		"InstanceSuccess":    true,
		"InstanceFailure":    true,
	}

	if !triggerEvents[value] {
		errors = append(errors, fmt.Errorf("%q must be a valid event type value: %q", k, value))
	}
	return
}

func validateDeploymentOption(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	validOptions := map[string]bool{
		"WITH_TRAFFIC_CONTROL":    true,
		"WITHOUT_TRAFFIC_CONTROL": true,
	}

	if !validOptions[value] {
		errors = append(errors, fmt.Errorf("%q must be a valid deployment option: %q", k, value))
	}
	return
}

func validateDeploymentType(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	validTypes := map[string]bool{
		"IN_PLACE":   true,
		"BLUE_GREEN": true,
	}

	if !validTypes[value] {
		errors = append(errors, fmt.Errorf("%q must be a valid deployment type: %q", k, value))
	}
	return
}
