package aws

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
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

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"cluster": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"task_definition": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"desired_count": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},

			"iam_role": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
			},

			"deployment_maximum_percent": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  200,
			},

			"deployment_minimum_healthy_percent": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  100,
			},

			"load_balancer": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"elb_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},

						"container_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},

						"container_port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
							ForceNew: true,
						},
					},
				},
				Set: resourceAwsEcsLoadBalancerHash,
			},
		},
	}
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

	loadBalancers := expandEcsLoadBalancers(d.Get("load_balancer").(*schema.Set).List())
	if len(loadBalancers) > 0 {
		log.Printf("[DEBUG] Adding ECS load balancers: %s", loadBalancers)
		input.LoadBalancers = loadBalancers
	}
	if v, ok := d.GetOk("iam_role"); ok {
		input.Role = aws.String(v.(string))
	}

	log.Printf("[DEBUG] Creating ECS service: %s", input)

	// Retry due to AWS IAM policy eventual consistency
	// See https://github.com/hashicorp/terraform/issues/2869
	var out *ecs.CreateServiceOutput
	var err error
	err = resource.Retry(2*time.Minute, func() *resource.RetryError {
		out, err = conn.CreateService(&input)

		if err != nil {
			ec2err, ok := err.(awserr.Error)
			if !ok {
				return resource.NonRetryableError(err)
			}
			if ec2err.Code() == "InvalidParameterException" {
				log.Printf("[DEBUG] Trying to create ECS service again: %q",
					ec2err.Message())
				return resource.RetryableError(err)
			}

			return resource.NonRetryableError(err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	service := *out.Service

	log.Printf("[DEBUG] ECS service created: %s", *service.ServiceArn)
	d.SetId(*service.ServiceArn)

	return resourceAwsEcsServiceUpdate(d, meta)
}

func resourceAwsEcsServiceRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecsconn

	log.Printf("[DEBUG] Reading ECS service %s", d.Id())
	input := ecs.DescribeServicesInput{
		Services: []*string{aws.String(d.Id())},
		Cluster:  aws.String(d.Get("cluster").(string)),
	}

	out, err := conn.DescribeServices(&input)
	if err != nil {
		return err
	}

	if len(out.Services) < 1 {
		log.Printf("[DEBUG] Removing ECS service %s (%s) because it's gone", d.Get("name").(string), d.Id())
		d.SetId("")
		return nil
	}

	service := out.Services[0]

	// Status==INACTIVE means deleted service
	if *service.Status == "INACTIVE" {
		log.Printf("[DEBUG] Removing ECS service %q because it's INACTIVE", *service.ServiceArn)
		d.SetId("")
		return nil
	}

	log.Printf("[DEBUG] Received ECS service %s", service)

	d.SetId(*service.ServiceArn)
	d.Set("name", *service.ServiceName)

	// Save task definition in the same format
	if strings.HasPrefix(d.Get("task_definition").(string), "arn:aws:ecs:") {
		d.Set("task_definition", *service.TaskDefinition)
	} else {
		taskDefinition := buildFamilyAndRevisionFromARN(*service.TaskDefinition)
		d.Set("task_definition", taskDefinition)
	}

	d.Set("desired_count", *service.DesiredCount)

	// Save cluster in the same format
	if strings.HasPrefix(d.Get("cluster").(string), "arn:aws:ecs:") {
		d.Set("cluster", *service.ClusterArn)
	} else {
		clusterARN := getNameFromARN(*service.ClusterArn)
		d.Set("cluster", clusterARN)
	}

	// Save IAM role in the same format
	if service.RoleArn != nil {
		if strings.HasPrefix(d.Get("iam_role").(string), "arn:aws:iam:") {
			d.Set("iam_role", *service.RoleArn)
		} else {
			roleARN := getNameFromARN(*service.RoleArn)
			d.Set("iam_role", roleARN)
		}
	}

	if service.DeploymentConfiguration != nil {
		d.Set("deployment_maximum_percent", *service.DeploymentConfiguration.MaximumPercent)
		d.Set("deployment_minimum_healthy_percent", *service.DeploymentConfiguration.MinimumHealthyPercent)
	}

	if service.LoadBalancers != nil {
		d.Set("load_balancers", flattenEcsLoadBalancers(service.LoadBalancers))
	}

	return nil
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

	out, err := conn.UpdateService(&input)
	if err != nil {
		return err
	}
	service := out.Service
	log.Printf("[DEBUG] Updated ECS service %s", service)

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
		return err
	}

	if len(resp.Services) == 0 {
		log.Printf("[DEBUG] ECS Service %q is already gone", d.Id())
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

	// Wait until the ECS service is drained
	err = resource.Retry(5*time.Minute, func() *resource.RetryError {
		input := ecs.DeleteServiceInput{
			Service: aws.String(d.Id()),
			Cluster: aws.String(d.Get("cluster").(string)),
		}

		log.Printf("[DEBUG] Trying to delete ECS service %s", input)
		_, err := conn.DeleteService(&input)
		if err == nil {
			return nil
		}

		ec2err, ok := err.(awserr.Error)
		if !ok {
			return resource.NonRetryableError(err)
		}
		if ec2err.Code() == "InvalidParameterException" {
			// Prevent "The service cannot be stopped while deployments are active."
			log.Printf("[DEBUG] Trying to delete ECS service again: %q",
				ec2err.Message())
			return resource.RetryableError(err)
		}

		return resource.NonRetryableError(err)

	})
	if err != nil {
		return err
	}

	// Wait until it's deleted
	wait := resource.StateChangeConf{
		Pending:    []string{"DRAINING"},
		Target:     []string{"INACTIVE"},
		Timeout:    5 * time.Minute,
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
