package aws

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"

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
		Importer: &schema.ResourceImporter{
			State: func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				idParts := strings.Split(d.Id(), ":")

				if len(idParts) != 2 {
					return []*schema.ResourceData{}, fmt.Errorf("expected ID in format ApplicationName:DeploymentGroupName, received: %s", d.Id())
				}

				applicationName := idParts[0]
				deploymentGroupName := idParts[1]
				conn := meta.(*AWSClient).codedeployconn

				input := &codedeploy.GetDeploymentGroupInput{
					ApplicationName:     aws.String(applicationName),
					DeploymentGroupName: aws.String(deploymentGroupName),
				}

				log.Printf("[DEBUG] Reading CodeDeploy Application: %s", input)
				output, err := conn.GetDeploymentGroup(input)

				if err != nil {
					return []*schema.ResourceData{}, err
				}

				if output == nil || output.DeploymentGroupInfo == nil {
					return []*schema.ResourceData{}, fmt.Errorf("error reading CodeDeploy Application (%s): empty response", d.Id())
				}

				d.SetId(aws.StringValue(output.DeploymentGroupInfo.DeploymentGroupId))
				d.Set("app_name", applicationName)
				d.Set("deployment_group_name", deploymentGroupName)

				return []*schema.ResourceData{d}, nil
			},
		},

		Schema: map[string]*schema.Schema{
			"app_name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringLenBetween(0, 100),
			},

			"deployment_group_name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringLenBetween(0, 100),
			},

			"deployment_style": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"deployment_option": {
							Type:     schema.TypeString,
							Optional: true,
							ValidateFunc: validation.StringInSlice([]string{
								codedeploy.DeploymentOptionWithTrafficControl,
								codedeploy.DeploymentOptionWithoutTrafficControl,
							}, false),
						},
						"deployment_type": {
							Type:     schema.TypeString,
							Optional: true,
							ValidateFunc: validation.StringInSlice([]string{
								codedeploy.DeploymentTypeInPlace,
								codedeploy.DeploymentTypeBlueGreen,
							}, false),
						},
					},
				},
			},

			"blue_green_deployment_config": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"deployment_ready_option": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"action_on_timeout": {
										Type:     schema.TypeString,
										Optional: true,
										ValidateFunc: validation.StringInSlice([]string{
											codedeploy.DeploymentReadyActionContinueDeployment,
											codedeploy.DeploymentReadyActionStopDeployment,
										}, false),
									},
									"wait_time_in_minutes": {
										Type:     schema.TypeInt,
										Optional: true,
									},
								},
							},
						},

						"green_fleet_provisioning_option": {
							Type:     schema.TypeList,
							Optional: true,
							Computed: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"action": {
										Type:     schema.TypeString,
										Optional: true,
										ValidateFunc: validation.StringInSlice([]string{
											codedeploy.GreenFleetProvisioningActionDiscoverExisting,
											codedeploy.GreenFleetProvisioningActionCopyAutoScalingGroup,
										}, false),
									},
								},
							},
						},

						"terminate_blue_instances_on_deployment_success": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"action": {
										Type:     schema.TypeString,
										Optional: true,
										ValidateFunc: validation.StringInSlice([]string{
											codedeploy.InstanceActionTerminate,
											codedeploy.InstanceActionKeepAlive,
										}, false),
									},
									"termination_wait_time_in_minutes": {
										Type:     schema.TypeInt,
										Optional: true,
									},
								},
							},
						},
					},
				},
			},

			"service_role_arn": {
				Type:     schema.TypeString,
				Required: true,
			},

			"alarm_configuration": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"alarms": {
							Type:     schema.TypeSet,
							MaxItems: 10,
							Optional: true,
							Set:      schema.HashString,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},

						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
						},

						"ignore_poll_alarm_failure": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
					},
				},
			},

			"load_balancer_info": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"elb_info": {
							Type:     schema.TypeSet,
							Optional: true,
							Set:      loadBalancerInfoHash,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},

						"target_group_info": {
							Type:     schema.TypeSet,
							Optional: true,
							Set:      loadBalancerInfoHash,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},

						"target_group_pair_info": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"prod_traffic_route": {
										Type:     schema.TypeList,
										Required: true,
										MinItems: 1,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"listener_arns": {
													Type:     schema.TypeSet,
													Required: true,
													Elem:     &schema.Schema{Type: schema.TypeString},
												},
											},
										},
									},
									"target_group": {
										Type:     schema.TypeList,
										Required: true,
										MinItems: 1,
										MaxItems: 2,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"name": {
													Type:         schema.TypeString,
													Required:     true,
													ValidateFunc: validation.NoZeroValues,
												},
											},
										},
									},
									"test_traffic_route": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"listener_arns": {
													Type:     schema.TypeSet,
													Required: true,
													Elem:     &schema.Schema{Type: schema.TypeString},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},

			"auto_rollback_configuration": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
						},

						"events": {
							Type:     schema.TypeSet,
							Optional: true,
							Set:      schema.HashString,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},

			"autoscaling_groups": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"deployment_config_name": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "CodeDeployDefault.OneAtATime",
				ValidateFunc: validation.StringLenBetween(0, 100),
			},

			"ec2_tag_set": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ec2_tag_filter": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"key": {
										Type:     schema.TypeString,
										Optional: true,
									},

									"type": {
										Type:         schema.TypeString,
										Optional:     true,
										ValidateFunc: validateTagFilters,
									},

									"value": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
							Set: resourceAwsCodeDeployTagFilterHash,
						},
					},
				},
				Set: resourceAwsCodeDeployTagSetHash,
			},

			"ec2_tag_filter": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"type": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateTagFilters,
						},

						"value": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
				Set: resourceAwsCodeDeployTagFilterHash,
			},

			"ecs_service": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cluster_name": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.NoZeroValues,
						},
						"service_name": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.NoZeroValues,
						},
					},
				},
			},

			"on_premises_instance_tag_filter": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"type": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateTagFilters,
						},

						"value": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
				Set: resourceAwsCodeDeployTagFilterHash,
			},

			"trigger_configuration": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"trigger_events": {
							Type:     schema.TypeSet,
							Required: true,
							Set:      schema.HashString,
							Elem: &schema.Schema{
								Type: schema.TypeString,
								ValidateFunc: validation.StringInSlice([]string{
									codedeploy.TriggerEventTypeDeploymentStart,
									codedeploy.TriggerEventTypeDeploymentSuccess,
									codedeploy.TriggerEventTypeDeploymentFailure,
									codedeploy.TriggerEventTypeDeploymentStop,
									codedeploy.TriggerEventTypeDeploymentRollback,
									codedeploy.TriggerEventTypeDeploymentReady,
									codedeploy.TriggerEventTypeInstanceStart,
									codedeploy.TriggerEventTypeInstanceSuccess,
									codedeploy.TriggerEventTypeInstanceFailure,
									codedeploy.TriggerEventTypeInstanceReady,
								}, false),
							},
						},

						"trigger_name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"trigger_target_arn": {
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

	// required fields
	applicationName := d.Get("app_name").(string)
	deploymentGroupName := d.Get("deployment_group_name").(string)
	serviceRoleArn := d.Get("service_role_arn").(string)

	input := codedeploy.CreateDeploymentGroupInput{
		ApplicationName:     aws.String(applicationName),
		DeploymentGroupName: aws.String(deploymentGroupName),
		ServiceRoleArn:      aws.String(serviceRoleArn),
	}

	if attr, ok := d.GetOk("deployment_style"); ok {
		input.DeploymentStyle = expandDeploymentStyle(attr.([]interface{}))
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

	if attr, ok := d.GetOk("ec2_tag_set"); ok {
		input.Ec2TagSet = buildEC2TagSet(attr.(*schema.Set).List())
	}

	if attr, ok := d.GetOk("ec2_tag_filter"); ok {
		input.Ec2TagFilters = buildEC2TagFilters(attr.(*schema.Set).List())
	}

	if attr, ok := d.GetOk("ecs_service"); ok {
		input.EcsServices = expandCodeDeployEcsServices(attr.([]interface{}))
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
		input.LoadBalancerInfo = expandLoadBalancerInfo(attr.([]interface{}))
	}

	if attr, ok := d.GetOk("blue_green_deployment_config"); ok {
		input.BlueGreenDeploymentConfiguration = expandBlueGreenDeploymentConfig(attr.([]interface{}))
	}

	log.Printf("[DEBUG] Creating CodeDeploy DeploymentGroup %s", applicationName)

	var resp *codedeploy.CreateDeploymentGroupOutput
	var err error
	err = resource.Retry(5*time.Minute, func() *resource.RetryError {
		resp, err = conn.CreateDeploymentGroup(&input)
		return handleCreateError(err)
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
	d.Set("deployment_config_name", resp.DeploymentGroupInfo.DeploymentConfigName)
	d.Set("deployment_group_name", resp.DeploymentGroupInfo.DeploymentGroupName)
	d.Set("service_role_arn", resp.DeploymentGroupInfo.ServiceRoleArn)

	autoScalingGroups := make([]string, len(resp.DeploymentGroupInfo.AutoScalingGroups))
	for i, autoScalingGroup := range resp.DeploymentGroupInfo.AutoScalingGroups {
		autoScalingGroups[i] = aws.StringValue(autoScalingGroup.Name)
	}
	if err := d.Set("autoscaling_groups", autoScalingGroups); err != nil {
		return fmt.Errorf("error setting autoscaling_groups: %s", err)
	}

	if err := d.Set("deployment_style", flattenDeploymentStyle(resp.DeploymentGroupInfo.DeploymentStyle)); err != nil {
		return err
	}

	if err := d.Set("ec2_tag_set", ec2TagSetToMap(resp.DeploymentGroupInfo.Ec2TagSet)); err != nil {
		return err
	}

	if err := d.Set("ec2_tag_filter", ec2TagFiltersToMap(resp.DeploymentGroupInfo.Ec2TagFilters)); err != nil {
		return err
	}

	if err := d.Set("ecs_service", flattenCodeDeployEcsServices(resp.DeploymentGroupInfo.EcsServices)); err != nil {
		return fmt.Errorf("error setting ecs_service: %s", err)
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

	if err := d.Set("load_balancer_info", flattenLoadBalancerInfo(resp.DeploymentGroupInfo.LoadBalancerInfo)); err != nil {
		return err
	}

	if err := d.Set("blue_green_deployment_config", flattenBlueGreenDeploymentConfig(resp.DeploymentGroupInfo.BlueGreenDeploymentConfiguration)); err != nil {
		return err
	}

	return nil
}

func resourceAwsCodeDeployDeploymentGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codedeployconn

	// required fields
	applicationName := d.Get("app_name").(string)
	deploymentGroupName := d.Get("deployment_group_name").(string)
	serviceRoleArn := d.Get("service_role_arn").(string)

	input := codedeploy.UpdateDeploymentGroupInput{
		ApplicationName:            aws.String(applicationName),
		CurrentDeploymentGroupName: aws.String(deploymentGroupName),
		ServiceRoleArn:             aws.String(serviceRoleArn),
	}

	if d.HasChange("deployment_group_name") {
		_, n := d.GetChange("deployment_group_name")
		input.NewDeploymentGroupName = aws.String(n.(string))
	}

	if d.HasChange("deployment_style") {
		_, n := d.GetChange("deployment_style")
		input.DeploymentStyle = expandDeploymentStyle(n.([]interface{}))
	}

	if d.HasChange("deployment_config_name") {
		_, n := d.GetChange("deployment_config_name")
		input.DeploymentConfigName = aws.String(n.(string))
	}

	// include (original or new) autoscaling groups when blue_green_deployment_config changes
	if d.HasChange("autoscaling_groups") || d.HasChange("blue_green_deployment_config") {
		_, n := d.GetChange("autoscaling_groups")
		input.AutoScalingGroups = expandStringList(n.(*schema.Set).List())
	}

	// TagFilters aren't like tags. They don't append. They simply replace.
	if d.HasChange("on_premises_instance_tag_filter") {
		_, n := d.GetChange("on_premises_instance_tag_filter")
		onPremFilters := buildOnPremTagFilters(n.(*schema.Set).List())
		input.OnPremisesInstanceTagFilters = onPremFilters
	}

	if d.HasChange("ec2_tag_set") {
		_, n := d.GetChange("ec2_tag_set")
		ec2TagSet := buildEC2TagSet(n.(*schema.Set).List())
		input.Ec2TagSet = ec2TagSet
	}

	if d.HasChange("ec2_tag_filter") {
		_, n := d.GetChange("ec2_tag_filter")
		ec2Filters := buildEC2TagFilters(n.(*schema.Set).List())
		input.Ec2TagFilters = ec2Filters
	}

	if d.HasChange("ecs_service") {
		input.EcsServices = expandCodeDeployEcsServices(d.Get("ecs_service").([]interface{}))
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
		input.LoadBalancerInfo = expandLoadBalancerInfo(n.([]interface{}))
	}

	if d.HasChange("blue_green_deployment_config") {
		_, n := d.GetChange("blue_green_deployment_config")
		input.BlueGreenDeploymentConfiguration = expandBlueGreenDeploymentConfig(n.([]interface{}))
	}

	log.Printf("[DEBUG] Updating CodeDeploy DeploymentGroup %s", d.Id())

	var err error
	err = resource.Retry(5*time.Minute, func() *resource.RetryError {
		_, err = conn.UpdateDeploymentGroup(&input)
		return handleUpdateError(err)
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

	return nil
}

func handleCreateError(err error) *resource.RetryError {
	return handleCodeDeployApiError(err, "create")
}

func handleUpdateError(err error) *resource.RetryError {
	return handleCodeDeployApiError(err, "update")
}

func handleCodeDeployApiError(err error, operation string) *resource.RetryError {
	if err == nil {
		return nil
	}

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
		log.Printf("[DEBUG] Trying to %s DeploymentGroup again: %q", operation, codedeployErr.Message())
		return resource.RetryableError(err)
	}

	return resource.NonRetryableError(err)
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

// buildEC2TagSet converts raw schema lists into a codedeploy.EC2TagSet.
func buildEC2TagSet(configured []interface{}) *codedeploy.EC2TagSet {
	filterSets := make([][]*codedeploy.EC2TagFilter, 0)
	for _, raw := range configured {
		m := raw.(map[string]interface{})
		rawFilters := m["ec2_tag_filter"].(*schema.Set)
		filters := buildEC2TagFilters(rawFilters.List())
		filterSets = append(filterSets, filters)
	}
	return &codedeploy.EC2TagSet{Ec2TagSetList: filterSets}
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

func expandCodeDeployEcsServices(l []interface{}) []*codedeploy.ECSService {
	ecsServices := make([]*codedeploy.ECSService, 0)

	for _, mRaw := range l {
		if mRaw == nil {
			continue
		}

		m := mRaw.(map[string]interface{})

		ecsService := &codedeploy.ECSService{
			ClusterName: aws.String(m["cluster_name"].(string)),
			ServiceName: aws.String(m["service_name"].(string)),
		}

		ecsServices = append(ecsServices, ecsService)
	}

	return ecsServices
}

func expandCodeDeployElbInfo(l []interface{}) []*codedeploy.ELBInfo {
	elbInfos := []*codedeploy.ELBInfo{}

	for _, mRaw := range l {
		if mRaw == nil {
			continue
		}

		m := mRaw.(map[string]interface{})

		elbInfo := &codedeploy.ELBInfo{
			Name: aws.String(m["name"].(string)),
		}

		elbInfos = append(elbInfos, elbInfo)
	}

	return elbInfos
}

func expandCodeDeployTargetGroupInfo(l []interface{}) []*codedeploy.TargetGroupInfo {
	targetGroupInfos := []*codedeploy.TargetGroupInfo{}

	for _, mRaw := range l {
		if mRaw == nil {
			continue
		}

		m := mRaw.(map[string]interface{})

		targetGroupInfo := &codedeploy.TargetGroupInfo{
			Name: aws.String(m["name"].(string)),
		}

		targetGroupInfos = append(targetGroupInfos, targetGroupInfo)
	}

	return targetGroupInfos
}

func expandCodeDeployTargetGroupPairInfo(l []interface{}) []*codedeploy.TargetGroupPairInfo {
	targetGroupPairInfos := []*codedeploy.TargetGroupPairInfo{}

	for _, mRaw := range l {
		if mRaw == nil {
			continue
		}

		m := mRaw.(map[string]interface{})

		targetGroupPairInfo := &codedeploy.TargetGroupPairInfo{
			ProdTrafficRoute: expandCodeDeployTrafficRoute(m["prod_traffic_route"].([]interface{})),
			TargetGroups:     expandCodeDeployTargetGroupInfo(m["target_group"].([]interface{})),
			TestTrafficRoute: expandCodeDeployTrafficRoute(m["test_traffic_route"].([]interface{})),
		}

		targetGroupPairInfos = append(targetGroupPairInfos, targetGroupPairInfo)
	}

	return targetGroupPairInfos
}

func expandCodeDeployTrafficRoute(l []interface{}) *codedeploy.TrafficRoute {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	m := l[0].(map[string]interface{})

	trafficRoute := &codedeploy.TrafficRoute{
		ListenerArns: expandStringSet(m["listener_arns"].(*schema.Set)),
	}

	return trafficRoute
}

// expandDeploymentStyle converts a raw schema list containing a map[string]interface{}
// into a single codedeploy.DeploymentStyle object
func expandDeploymentStyle(list []interface{}) *codedeploy.DeploymentStyle {
	if len(list) == 0 || list[0] == nil {
		return nil
	}

	style := list[0].(map[string]interface{})
	result := &codedeploy.DeploymentStyle{}

	if v, ok := style["deployment_option"]; ok {
		result.DeploymentOption = aws.String(v.(string))
	}
	if v, ok := style["deployment_type"]; ok {
		result.DeploymentType = aws.String(v.(string))
	}

	return result
}

// expandLoadBalancerInfo converts a raw schema list containing a map[string]interface{}
// into a single codedeploy.LoadBalancerInfo object
func expandLoadBalancerInfo(list []interface{}) *codedeploy.LoadBalancerInfo {
	if len(list) == 0 || list[0] == nil {
		return nil
	}

	lbInfo := list[0].(map[string]interface{})
	loadBalancerInfo := &codedeploy.LoadBalancerInfo{}

	if attr, ok := lbInfo["elb_info"]; ok && attr.(*schema.Set).Len() > 0 {
		loadBalancerInfo.ElbInfoList = expandCodeDeployElbInfo(attr.(*schema.Set).List())
	}

	if attr, ok := lbInfo["target_group_info"]; ok && attr.(*schema.Set).Len() > 0 {
		loadBalancerInfo.TargetGroupInfoList = expandCodeDeployTargetGroupInfo(attr.(*schema.Set).List())
	}

	if attr, ok := lbInfo["target_group_pair_info"]; ok && len(attr.([]interface{})) > 0 {
		loadBalancerInfo.TargetGroupPairInfoList = expandCodeDeployTargetGroupPairInfo(attr.([]interface{}))
	}

	return loadBalancerInfo
}

// expandBlueGreenDeploymentConfig converts a raw schema list containing a map[string]interface{}
// into a single codedeploy.BlueGreenDeploymentConfiguration object
func expandBlueGreenDeploymentConfig(list []interface{}) *codedeploy.BlueGreenDeploymentConfiguration {
	if len(list) == 0 || list[0] == nil {
		return nil
	}

	config := list[0].(map[string]interface{})
	blueGreenDeploymentConfig := &codedeploy.BlueGreenDeploymentConfiguration{}

	if attr, ok := config["deployment_ready_option"]; ok {
		a := attr.([]interface{})

		if len(a) > 0 && a[0] != nil {
			m := a[0].(map[string]interface{})

			deploymentReadyOption := &codedeploy.DeploymentReadyOption{}
			if v, ok := m["action_on_timeout"]; ok {
				deploymentReadyOption.ActionOnTimeout = aws.String(v.(string))
			}
			if v, ok := m["wait_time_in_minutes"]; ok {
				deploymentReadyOption.WaitTimeInMinutes = aws.Int64(int64(v.(int)))
			}
			blueGreenDeploymentConfig.DeploymentReadyOption = deploymentReadyOption
		}
	}

	if attr, ok := config["green_fleet_provisioning_option"]; ok {
		a := attr.([]interface{})

		if len(a) > 0 && a[0] != nil {
			m := a[0].(map[string]interface{})

			greenFleetProvisioningOption := &codedeploy.GreenFleetProvisioningOption{}
			if v, ok := m["action"]; ok {
				greenFleetProvisioningOption.Action = aws.String(v.(string))
			}
			blueGreenDeploymentConfig.GreenFleetProvisioningOption = greenFleetProvisioningOption
		}
	}

	if attr, ok := config["terminate_blue_instances_on_deployment_success"]; ok {
		a := attr.([]interface{})

		if len(a) > 0 && a[0] != nil {
			m := a[0].(map[string]interface{})

			blueInstanceTerminationOption := &codedeploy.BlueInstanceTerminationOption{}
			if v, ok := m["action"]; ok {
				blueInstanceTerminationOption.Action = aws.String(v.(string))
			}
			if v, ok := m["termination_wait_time_in_minutes"]; ok {
				blueInstanceTerminationOption.TerminationWaitTimeInMinutes = aws.Int64(int64(v.(int)))
			}
			blueGreenDeploymentConfig.TerminateBlueInstancesOnDeploymentSuccess = blueInstanceTerminationOption
		}
	}

	return blueGreenDeploymentConfig
}

// ec2TagFiltersToMap converts lists of tag filters into a []map[string]interface{}.
func ec2TagFiltersToMap(list []*codedeploy.EC2TagFilter) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(list))
	for _, tf := range list {
		l := make(map[string]interface{})
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

// ec2TagSetToMap converts lists of tag filters into a [][]map[string]string.
func ec2TagSetToMap(tagSet *codedeploy.EC2TagSet) []map[string]interface{} {
	var result []map[string]interface{}
	if tagSet == nil {
		result = make([]map[string]interface{}, 0)
	} else {
		result = make([]map[string]interface{}, 0, len(tagSet.Ec2TagSetList))
		for _, filterSet := range tagSet.Ec2TagSetList {
			filters := ec2TagFiltersToMap(filterSet)
			filtersAsIntfSlice := make([]interface{}, 0, len(filters))
			for _, item := range filters {
				filtersAsIntfSlice = append(filtersAsIntfSlice, item)
			}
			tagFilters := map[string]interface{}{
				"ec2_tag_filter": schema.NewSet(resourceAwsCodeDeployTagFilterHash, filtersAsIntfSlice),
			}
			result = append(result, tagFilters)
		}
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

func flattenCodeDeployEcsServices(ecsServices []*codedeploy.ECSService) []interface{} {
	l := make([]interface{}, 0)

	for _, ecsService := range ecsServices {
		if ecsService == nil {
			continue
		}

		m := map[string]interface{}{
			"cluster_name": aws.StringValue(ecsService.ClusterName),
			"service_name": aws.StringValue(ecsService.ServiceName),
		}

		l = append(l, m)
	}

	return l
}

func flattenCodeDeployElbInfo(elbInfos []*codedeploy.ELBInfo) []interface{} {
	l := make([]interface{}, 0)

	for _, elbInfo := range elbInfos {
		if elbInfo == nil {
			continue
		}

		m := map[string]interface{}{
			"name": aws.StringValue(elbInfo.Name),
		}

		l = append(l, m)
	}

	return l
}

func flattenCodeDeployTargetGroupInfo(targetGroupInfos []*codedeploy.TargetGroupInfo) []interface{} {
	l := make([]interface{}, 0)

	for _, targetGroupInfo := range targetGroupInfos {
		if targetGroupInfo == nil {
			continue
		}

		m := map[string]interface{}{
			"name": aws.StringValue(targetGroupInfo.Name),
		}

		l = append(l, m)
	}

	return l
}

func flattenCodeDeployTargetGroupPairInfo(targetGroupPairInfos []*codedeploy.TargetGroupPairInfo) []interface{} {
	l := make([]interface{}, 0)

	for _, targetGroupPairInfo := range targetGroupPairInfos {
		if targetGroupPairInfo == nil {
			continue
		}

		m := map[string]interface{}{
			"prod_traffic_route": flattenCodeDeployTrafficRoute(targetGroupPairInfo.ProdTrafficRoute),
			"target_group":       flattenCodeDeployTargetGroupInfo(targetGroupPairInfo.TargetGroups),
			"test_traffic_route": flattenCodeDeployTrafficRoute(targetGroupPairInfo.TestTrafficRoute),
		}

		l = append(l, m)
	}

	return l
}

func flattenCodeDeployTrafficRoute(trafficRoute *codedeploy.TrafficRoute) []interface{} {
	if trafficRoute == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"listener_arns": schema.NewSet(schema.HashString, flattenStringList(trafficRoute.ListenerArns)),
	}

	return []interface{}{m}
}

// flattenDeploymentStyle converts a codedeploy.DeploymentStyle object
// into a []map[string]interface{} list containing a single item
func flattenDeploymentStyle(style *codedeploy.DeploymentStyle) []map[string]interface{} {
	if style == nil {
		return nil
	}

	item := make(map[string]interface{})
	if style.DeploymentOption != nil {
		item["deployment_option"] = *style.DeploymentOption
	}
	if style.DeploymentType != nil {
		item["deployment_type"] = *style.DeploymentType
	}

	result := make([]map[string]interface{}, 0, 1)
	result = append(result, item)
	return result
}

func flattenLoadBalancerInfo(loadBalancerInfo *codedeploy.LoadBalancerInfo) []interface{} {
	if loadBalancerInfo == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"elb_info":               schema.NewSet(loadBalancerInfoHash, flattenCodeDeployElbInfo(loadBalancerInfo.ElbInfoList)),
		"target_group_info":      schema.NewSet(loadBalancerInfoHash, flattenCodeDeployTargetGroupInfo(loadBalancerInfo.TargetGroupInfoList)),
		"target_group_pair_info": flattenCodeDeployTargetGroupPairInfo(loadBalancerInfo.TargetGroupPairInfoList),
	}

	return []interface{}{m}
}

// flattenBlueGreenDeploymentConfig converts a codedeploy.BlueGreenDeploymentConfiguration object
// into a []map[string]interface{} list containing a single item
func flattenBlueGreenDeploymentConfig(config *codedeploy.BlueGreenDeploymentConfiguration) []map[string]interface{} {

	if config == nil {
		return nil
	}

	m := make(map[string]interface{})

	if config.DeploymentReadyOption != nil {
		a := make([]map[string]interface{}, 0)
		deploymentReadyOption := make(map[string]interface{})

		if config.DeploymentReadyOption.ActionOnTimeout != nil {
			deploymentReadyOption["action_on_timeout"] = *config.DeploymentReadyOption.ActionOnTimeout
		}
		if config.DeploymentReadyOption.WaitTimeInMinutes != nil {
			deploymentReadyOption["wait_time_in_minutes"] = *config.DeploymentReadyOption.WaitTimeInMinutes
		}

		m["deployment_ready_option"] = append(a, deploymentReadyOption)
	}

	if config.GreenFleetProvisioningOption != nil {
		b := make([]map[string]interface{}, 0)
		greenFleetProvisioningOption := make(map[string]interface{})

		if config.GreenFleetProvisioningOption.Action != nil {
			greenFleetProvisioningOption["action"] = *config.GreenFleetProvisioningOption.Action
		}

		m["green_fleet_provisioning_option"] = append(b, greenFleetProvisioningOption)
	}

	if config.TerminateBlueInstancesOnDeploymentSuccess != nil {
		c := make([]map[string]interface{}, 0)
		blueInstanceTerminationOption := make(map[string]interface{})

		if config.TerminateBlueInstancesOnDeploymentSuccess.Action != nil {
			blueInstanceTerminationOption["action"] = *config.TerminateBlueInstancesOnDeploymentSuccess.Action
		}
		if config.TerminateBlueInstancesOnDeploymentSuccess.TerminationWaitTimeInMinutes != nil {
			blueInstanceTerminationOption["termination_wait_time_in_minutes"] = *config.TerminateBlueInstancesOnDeploymentSuccess.TerminationWaitTimeInMinutes
		}

		m["terminate_blue_instances_on_deployment_success"] = append(c, blueInstanceTerminationOption)
	}

	list := make([]map[string]interface{}, 0)
	list = append(list, m)
	return list
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

func resourceAwsCodeDeployTagSetHash(v interface{}) int {
	tagSetMap := v.(map[string]interface{})
	filterSet := tagSetMap["ec2_tag_filter"]
	filterSetSlice := filterSet.(*schema.Set).List()

	var x uint64
	x = 1
	for i, filter := range filterSetSlice {
		x = ((x << 7) | (x >> (64 - 7))) ^ uint64(i) ^ uint64(resourceAwsCodeDeployTagFilterHash(filter))
	}
	return int(x)
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

func loadBalancerInfoHash(v interface{}) int {
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
