package aws

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

var taskDefinitionRE = regexp.MustCompile("^([a-zA-Z0-9_-]+):([0-9]+)$")

func resourceAwsEcsService() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEcsServiceCreate,
		Read:   resourceAwsEcsServiceRead,
		Update: resourceAwsEcsServiceUpdate,
		Delete: resourceAwsEcsServiceDelete,
		Importer: &schema.ResourceImporter{
			State: resourceAwsEcsServiceImport,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"cluster": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"task_definition": {
				Type:     schema.TypeString,
				Required: true,
			},

			"desired_count": {
				Type:     schema.TypeInt,
				Optional: true,
			},

			"health_check_grace_period_seconds": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validateAwsEcsServiceHealthCheckGracePeriodSeconds,
			},

			"launch_type": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Default:  "EC2",
			},

			"iam_role": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Computed: true,
			},

			"deployment_maximum_percent": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  200,
			},

			"deployment_minimum_healthy_percent": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  100,
			},

			"load_balancer": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"elb_name": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"target_group_arn": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"container_name": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},

						"container_port": {
							Type:     schema.TypeInt,
							Required: true,
							ForceNew: true,
						},
					},
				},
				Set: resourceAwsEcsLoadBalancerHash,
			},
			"network_configuration": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"security_groups": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
						"subnets": {
							Type:     schema.TypeSet,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
						"assign_public_ip": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
					},
				},
			},
			"placement_strategy": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				MaxItems: 5,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:     schema.TypeString,
							ForceNew: true,
							Required: true,
						},
						"field": {
							Type:     schema.TypeString,
							ForceNew: true,
							Optional: true,
							DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
								if strings.ToLower(old) == strings.ToLower(new) {
									return true
								}
								return false
							},
						},
					},
				},
				Set: func(v interface{}) int {
					var buf bytes.Buffer
					m := v.(map[string]interface{})
					buf.WriteString(fmt.Sprintf("%s-", m["type"].(string)))
					if m["field"] != nil {
						field := m["field"].(string)
						if field == "host" {
							buf.WriteString("instanceId-")
						} else {
							buf.WriteString(fmt.Sprintf("%s-", field))
						}
					}
					return hashcode.String(buf.String())
				},
			},

			"placement_constraints": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				MaxItems: 10,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:     schema.TypeString,
							ForceNew: true,
							Required: true,
						},
						"expression": {
							Type:     schema.TypeString,
							ForceNew: true,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func resourceAwsEcsServiceImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	if len(strings.Split(d.Id(), "/")) != 2 {
		return []*schema.ResourceData{}, fmt.Errorf("[ERR] Wrong format of resource: %s. Please follow 'cluster-name/service-name'", d.Id())
	}
	cluster := strings.Split(d.Id(), "/")[0]
	name := strings.Split(d.Id(), "/")[1]
	log.Printf("[DEBUG] Importing ECS service %s from cluster %s", name, cluster)

	d.SetId(name)
	clusterArn := arnString(
		meta.(*AWSClient).partition,
		meta.(*AWSClient).region,
		"ecs",
		meta.(*AWSClient).accountid,
		fmt.Sprintf("cluster/%s", cluster),
	)
	d.Set("cluster", clusterArn)
	return []*schema.ResourceData{d}, nil
}

func resourceAwsEcsServiceCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecsconn

	input := ecs.CreateServiceInput{
		ServiceName:    aws.String(d.Get("name").(string)),
		TaskDefinition: aws.String(d.Get("task_definition").(string)),
		DesiredCount:   aws.Int64(int64(d.Get("desired_count").(int))),
		ClientToken:    aws.String(resource.UniqueId()),
		DeploymentConfiguration: &ecs.DeploymentConfiguration{
			MaximumPercent:        aws.Int64(int64(d.Get("deployment_maximum_percent").(int))),
			MinimumHealthyPercent: aws.Int64(int64(d.Get("deployment_minimum_healthy_percent").(int))),
		},
	}

	if v, ok := d.GetOk("cluster"); ok {
		input.Cluster = aws.String(v.(string))
	}

	if v, ok := d.GetOk("health_check_grace_period_seconds"); ok {
		input.HealthCheckGracePeriodSeconds = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("launch_type"); ok {
		input.LaunchType = aws.String(v.(string))
	}

	loadBalancers := expandEcsLoadBalancers(d.Get("load_balancer").(*schema.Set).List())
	if len(loadBalancers) > 0 {
		log.Printf("[DEBUG] Adding ECS load balancers: %s", loadBalancers)
		input.LoadBalancers = loadBalancers
	}
	if v, ok := d.GetOk("iam_role"); ok {
		input.Role = aws.String(v.(string))
	}

	input.NetworkConfiguration = expandEcsNetworkConfiguration(d.Get("network_configuration").([]interface{}))

	strategies := d.Get("placement_strategy").(*schema.Set).List()
	if len(strategies) > 0 {
		var ps []*ecs.PlacementStrategy
		for _, raw := range strategies {
			p := raw.(map[string]interface{})
			t := p["type"].(string)
			f := p["field"].(string)
			if err := validateAwsEcsPlacementStrategy(t, f); err != nil {
				return err
			}
			ps = append(ps, &ecs.PlacementStrategy{
				Type:  aws.String(p["type"].(string)),
				Field: aws.String(p["field"].(string)),
			})
		}
		input.PlacementStrategy = ps
	}

	constraints := d.Get("placement_constraints").(*schema.Set).List()
	if len(constraints) > 0 {
		var pc []*ecs.PlacementConstraint
		for _, raw := range constraints {
			p := raw.(map[string]interface{})
			t := p["type"].(string)
			e := p["expression"].(string)
			if err := validateAwsEcsPlacementConstraint(t, e); err != nil {
				return err
			}
			constraint := &ecs.PlacementConstraint{
				Type: aws.String(t),
			}
			if e != "" {
				constraint.Expression = aws.String(e)
			}

			pc = append(pc, constraint)
		}
		input.PlacementConstraints = pc
	}

	log.Printf("[DEBUG] Creating ECS service: %s", input)

	// Retry due to AWS IAM & ECS eventual consistency
	var out *ecs.CreateServiceOutput
	var err error
	err = resource.Retry(2*time.Minute, func() *resource.RetryError {
		out, err = conn.CreateService(&input)

		if err != nil {
			if isAWSErr(err, ecs.ErrCodeClusterNotFoundException, "") {
				return resource.RetryableError(err)
			}
			if isAWSErr(err, ecs.ErrCodeInvalidParameterException, "Please verify that the ECS service role being passed has the proper permissions.") {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("%s %q", err, d.Get("name").(string))
	}

	service := *out.Service

	log.Printf("[DEBUG] ECS service created: %s", *service.ServiceArn)
	d.SetId(*service.ServiceArn)

	return resourceAwsEcsServiceRead(d, meta)
}

func resourceAwsEcsServiceRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecsconn

	log.Printf("[DEBUG] Reading ECS service %s", d.Id())
	input := ecs.DescribeServicesInput{
		Services: []*string{aws.String(d.Id())},
		Cluster:  aws.String(d.Get("cluster").(string)),
	}

	var out *ecs.DescribeServicesOutput
	err := resource.Retry(2*time.Minute, func() *resource.RetryError {
		var err error
		out, err = conn.DescribeServices(&input)
		if err != nil {
			if d.IsNewResource() && isAWSErr(err, ecs.ErrCodeServiceNotFoundException, "") {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}

		if d.IsNewResource() && len(out.Services) < 1 {
			return resource.RetryableError(fmt.Errorf("No ECS service found: %q", d.Id()))
		}

		service := out.Services[0]
		if d.IsNewResource() && *service.Status == "INACTIVE" {
			return resource.RetryableError(fmt.Errorf("ECS service currently INACTIVE: %q", d.Id()))
		}

		return nil
	})
	if err != nil {
		return err
	}

	if len(out.Services) < 1 {
		log.Printf("[WARN] Removing ECS service %s (%s) because it's gone", d.Get("name").(string), d.Id())
		d.SetId("")
		return nil
	}

	service := out.Services[0]

	// Status==INACTIVE means deleted service
	if *service.Status == "INACTIVE" {
		log.Printf("[WARN] Removing ECS service %q because it's INACTIVE", *service.ServiceArn)
		d.SetId("")
		return nil
	}

	log.Printf("[DEBUG] Received ECS service %s", service)

	d.SetId(*service.ServiceArn)
	d.Set("name", service.ServiceName)

	// Save task definition in the same format
	if strings.HasPrefix(d.Get("task_definition").(string), "arn:"+meta.(*AWSClient).partition+":ecs:") {
		d.Set("task_definition", service.TaskDefinition)
	} else {
		taskDefinition := buildFamilyAndRevisionFromARN(*service.TaskDefinition)
		d.Set("task_definition", taskDefinition)
	}

	d.Set("desired_count", service.DesiredCount)
	d.Set("health_check_grace_period_seconds", service.HealthCheckGracePeriodSeconds)
	d.Set("launch_type", service.LaunchType)

	// Save cluster in the same format
	if strings.HasPrefix(d.Get("cluster").(string), "arn:"+meta.(*AWSClient).partition+":ecs:") {
		d.Set("cluster", service.ClusterArn)
	} else {
		clusterARN := getNameFromARN(*service.ClusterArn)
		d.Set("cluster", clusterARN)
	}

	// Save IAM role in the same format
	if service.RoleArn != nil {
		if strings.HasPrefix(d.Get("iam_role").(string), "arn:"+meta.(*AWSClient).partition+":iam:") {
			d.Set("iam_role", service.RoleArn)
		} else {
			roleARN := getNameFromARN(*service.RoleArn)
			d.Set("iam_role", roleARN)
		}
	}

	if service.DeploymentConfiguration != nil {
		d.Set("deployment_maximum_percent", service.DeploymentConfiguration.MaximumPercent)
		d.Set("deployment_minimum_healthy_percent", service.DeploymentConfiguration.MinimumHealthyPercent)
	}

	if service.LoadBalancers != nil {
		d.Set("load_balancer", flattenEcsLoadBalancers(service.LoadBalancers))
	}

	if err := d.Set("placement_strategy", flattenPlacementStrategy(service.PlacementStrategy)); err != nil {
		log.Printf("[ERR] Error setting placement_strategy for (%s): %s", d.Id(), err)
	}
	if err := d.Set("placement_constraints", flattenServicePlacementConstraints(service.PlacementConstraints)); err != nil {
		log.Printf("[ERR] Error setting placement_constraints for (%s): %s", d.Id(), err)
	}

	if err := d.Set("network_configuration", flattenEcsNetworkConfiguration(service.NetworkConfiguration)); err != nil {
		return fmt.Errorf("[ERR] Error setting network_configuration for (%s): %s", d.Id(), err)
	}

	return nil
}

func flattenEcsNetworkConfiguration(nc *ecs.NetworkConfiguration) []interface{} {
	if nc == nil {
		return nil
	}

	result := make(map[string]interface{})
	result["security_groups"] = schema.NewSet(schema.HashString, flattenStringList(nc.AwsvpcConfiguration.SecurityGroups))
	result["subnets"] = schema.NewSet(schema.HashString, flattenStringList(nc.AwsvpcConfiguration.Subnets))

	if nc.AwsvpcConfiguration.AssignPublicIp != nil {
		result["assign_public_ip"] = *nc.AwsvpcConfiguration.AssignPublicIp == ecs.AssignPublicIpEnabled
	}

	return []interface{}{result}
}

func expandEcsNetworkConfiguration(nc []interface{}) *ecs.NetworkConfiguration {
	if len(nc) == 0 {
		return nil
	}
	awsVpcConfig := &ecs.AwsVpcConfiguration{}
	raw := nc[0].(map[string]interface{})
	if val, ok := raw["security_groups"]; ok {
		awsVpcConfig.SecurityGroups = expandStringSet(val.(*schema.Set))
	}
	awsVpcConfig.Subnets = expandStringSet(raw["subnets"].(*schema.Set))
	if val, ok := raw["assign_public_ip"].(bool); ok {
		awsVpcConfig.AssignPublicIp = aws.String(ecs.AssignPublicIpDisabled)
		if val {
			awsVpcConfig.AssignPublicIp = aws.String(ecs.AssignPublicIpEnabled)
		}
	}

	return &ecs.NetworkConfiguration{AwsvpcConfiguration: awsVpcConfig}
}

func flattenServicePlacementConstraints(pcs []*ecs.PlacementConstraint) []map[string]interface{} {
	if len(pcs) == 0 {
		return nil
	}
	results := make([]map[string]interface{}, 0)
	for _, pc := range pcs {
		c := make(map[string]interface{})
		c["type"] = *pc.Type
		if pc.Expression != nil {
			c["expression"] = *pc.Expression
		}

		results = append(results, c)
	}
	return results
}

func flattenPlacementStrategy(pss []*ecs.PlacementStrategy) []map[string]interface{} {
	if len(pss) == 0 {
		return nil
	}
	results := make([]map[string]interface{}, 0)
	for _, ps := range pss {
		c := make(map[string]interface{})
		c["type"] = *ps.Type
		c["field"] = *ps.Field

		// for some fields the API requires lowercase for creation but will return uppercase on query
		if *ps.Field == "MEMORY" || *ps.Field == "CPU" {
			c["field"] = strings.ToLower(*ps.Field)
		}

		results = append(results, c)
	}
	return results
}

func resourceAwsEcsServiceUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecsconn

	log.Printf("[DEBUG] Updating ECS service %s", d.Id())
	input := ecs.UpdateServiceInput{
		Service: aws.String(d.Id()),
		Cluster: aws.String(d.Get("cluster").(string)),
	}

	if d.HasChange("desired_count") {
		_, n := d.GetChange("desired_count")
		input.DesiredCount = aws.Int64(int64(n.(int)))
	}
	if d.HasChange("health_check_grace_period_seconds") {
		_, n := d.GetChange("health_check_grace_period_seconds")
		input.HealthCheckGracePeriodSeconds = aws.Int64(int64(n.(int)))
	}
	if d.HasChange("task_definition") {
		_, n := d.GetChange("task_definition")
		input.TaskDefinition = aws.String(n.(string))
	}

	if d.HasChange("deployment_maximum_percent") || d.HasChange("deployment_minimum_healthy_percent") {
		input.DeploymentConfiguration = &ecs.DeploymentConfiguration{
			MaximumPercent:        aws.Int64(int64(d.Get("deployment_maximum_percent").(int))),
			MinimumHealthyPercent: aws.Int64(int64(d.Get("deployment_minimum_healthy_percent").(int))),
		}
	}

	if d.HasChange("network_configuration") {
		input.NetworkConfiguration = expandEcsNetworkConfiguration(d.Get("network_configuration").([]interface{}))
	}

	// Retry due to IAM eventual consistency
	err := resource.Retry(2*time.Minute, func() *resource.RetryError {
		out, err := conn.UpdateService(&input)
		if err != nil {
			if isAWSErr(err, ecs.ErrCodeInvalidParameterException, "Please verify that the ECS service role being passed has the proper permissions.") {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}

		log.Printf("[DEBUG] Updated ECS service %s", out.Service)
		return nil
	})
	if err != nil {
		return err
	}

	return resourceAwsEcsServiceRead(d, meta)
}

func resourceAwsEcsServiceDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecsconn

	// Check if it's not already gone
	resp, err := conn.DescribeServices(&ecs.DescribeServicesInput{
		Services: []*string{aws.String(d.Id())},
		Cluster:  aws.String(d.Get("cluster").(string)),
	})
	if err != nil {
		if isAWSErr(err, ecs.ErrCodeServiceNotFoundException, "") {
			log.Printf("[DEBUG] Removing ECS Service from state, %q is already gone", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	if len(resp.Services) == 0 {
		log.Printf("[DEBUG] Removing ECS Service from state, %q is already gone", d.Id())
		d.SetId("")
		return nil
	}

	log.Printf("[DEBUG] ECS service %s is currently %s", d.Id(), *resp.Services[0].Status)

	if *resp.Services[0].Status == "INACTIVE" {
		return nil
	}

	// Drain the ECS service
	if *resp.Services[0].Status != "DRAINING" {
		log.Printf("[DEBUG] Draining ECS service %s", d.Id())
		_, err = conn.UpdateService(&ecs.UpdateServiceInput{
			Service:      aws.String(d.Id()),
			Cluster:      aws.String(d.Get("cluster").(string)),
			DesiredCount: aws.Int64(int64(0)),
		})
		if err != nil {
			return err
		}
	}

	input := ecs.DeleteServiceInput{
		Service: aws.String(d.Id()),
		Cluster: aws.String(d.Get("cluster").(string)),
	}
	// Wait until the ECS service is drained
	err = resource.Retry(5*time.Minute, func() *resource.RetryError {
		log.Printf("[DEBUG] Trying to delete ECS service %s", input)
		_, err := conn.DeleteService(&input)
		if err != nil {
			if isAWSErr(err, ecs.ErrCodeInvalidParameterException, "The service cannot be stopped while deployments are active.") {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})

	if err != nil {
		return err
	}

	// Wait until it's deleted
	wait := resource.StateChangeConf{
		Pending:    []string{"ACTIVE", "DRAINING"},
		Target:     []string{"INACTIVE"},
		Timeout:    10 * time.Minute,
		MinTimeout: 1 * time.Second,
		Refresh: func() (interface{}, string, error) {
			log.Printf("[DEBUG] Checking if ECS service %s is INACTIVE", d.Id())
			resp, err := conn.DescribeServices(&ecs.DescribeServicesInput{
				Services: []*string{aws.String(d.Id())},
				Cluster:  aws.String(d.Get("cluster").(string)),
			})
			if err != nil {
				return resp, "FAILED", err
			}

			log.Printf("[DEBUG] ECS service (%s) is currently %q", d.Id(), *resp.Services[0].Status)
			return resp, *resp.Services[0].Status, nil
		},
	}

	_, err = wait.WaitForState()
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] ECS service %s deleted.", d.Id())
	return nil
}

func resourceAwsEcsLoadBalancerHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%s-", m["elb_name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["container_name"].(string)))
	buf.WriteString(fmt.Sprintf("%d-", m["container_port"].(int)))

	if s := m["target_group_arn"].(string); s != "" {
		buf.WriteString(fmt.Sprintf("%s-", s))
	}

	return hashcode.String(buf.String())
}

func buildFamilyAndRevisionFromARN(arn string) string {
	return strings.Split(arn, "/")[1]
}

// Expects the following ARNs:
// arn:aws:iam::0123456789:role/EcsService
// arn:aws:ecs:us-west-2:0123456789:cluster/radek-cluster
func getNameFromARN(arn string) string {
	return strings.Split(arn, "/")[1]
}

func parseTaskDefinition(taskDefinition string) (string, string, error) {
	matches := taskDefinitionRE.FindAllStringSubmatch(taskDefinition, 2)

	if len(matches) == 0 || len(matches[0]) != 3 {
		return "", "", fmt.Errorf(
			"Invalid task definition format, family:rev or ARN expected (%#v)",
			taskDefinition)
	}

	return matches[0][1], matches[0][2], nil
}

func validateAwsEcsServiceHealthCheckGracePeriodSeconds(v interface{}, k string) (ws []string, errors []error) {
	value := v.(int)
	if (value < 0) || (value > 1800) {
		errors = append(errors, fmt.Errorf("%q must be between 0 and 1800", k))
	}
	return
}
