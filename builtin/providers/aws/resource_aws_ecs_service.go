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
	"github.com/aws/aws-sdk-go/service/iam"
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
				Optional: true,
			},

			"load_balancer": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"elb_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"container_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"container_port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
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
		DesiredCount:   aws.Long(int64(d.Get("desired_count").(int))),
	}

	if v, ok := d.GetOk("cluster"); ok {
		input.Cluster = aws.String(v.(string))
	}

	loadBalancers := expandEcsLoadBalancers(d.Get("load_balancer").(*schema.Set).List())
	if len(loadBalancers) > 0 {
		log.Printf("[DEBUG] Adding ECS load balancers: %#v", loadBalancers)
		input.LoadBalancers = loadBalancers
	}
	if v, ok := d.GetOk("iam_role"); ok {
		input.Role = aws.String(v.(string))
	}

	log.Printf("[DEBUG] Creating ECS service: %#v", input)
	out, err := conn.CreateService(&input)
	if err != nil {
		return err
	}

	service := *out.Service

	log.Printf("[DEBUG] ECS service created: %s", *service.ServiceARN)
	d.SetId(*service.ServiceARN)
	d.Set("cluster", *service.ClusterARN)

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

	service := out.Services[0]
	log.Printf("[DEBUG] Received ECS service %#v", service)

	d.SetId(*service.ServiceARN)
	d.Set("name", *service.ServiceName)

	// Save task definition in the same format
	if strings.HasPrefix(d.Get("task_definition").(string), "arn:aws:ecs:") {
		d.Set("task_definition", *service.TaskDefinition)
	} else {
		taskDefinition := buildFamilyAndRevisionFromARN(*service.TaskDefinition)
		d.Set("task_definition", taskDefinition)
	}

	d.Set("desired_count", *service.DesiredCount)
	d.Set("cluster", *service.ClusterARN)

	if service.RoleARN != nil {
		d.Set("iam_role", *service.RoleARN)
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
		input.DesiredCount = aws.Long(int64(n.(int)))
	}
	if d.HasChange("task_definition") {
		_, n := d.GetChange("task_definition")
		input.TaskDefinition = aws.String(n.(string))
	}

	out, err := conn.UpdateService(&input)
	if err != nil {
		return err
	}
	service := out.Service
	log.Printf("[DEBUG] Updated ECS service %#v", service)

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
			DesiredCount: aws.Long(int64(0)),
		})
		if err != nil {
			return err
		}
	}

	input := ecs.DeleteServiceInput{
		Service: aws.String(d.Id()),
		Cluster: aws.String(d.Get("cluster").(string)),
	}

	log.Printf("[DEBUG] Deleting ECS service %#v", input)
	out, err := conn.DeleteService(&input)
	if err != nil {
		return err
	}

	// Wait until it's deleted
	wait := resource.StateChangeConf{
		Pending:    []string{"DRAINING"},
		Target:     "INACTIVE",
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

			return resp, *resp.Services[0].Status, nil
		},
	}

	_, err = wait.WaitForState()
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] ECS service %s deleted.", *out.Service.ServiceARN)
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

func buildTaskDefinitionARN(taskDefinition string, meta interface{}) (string, error) {
	// If it's already an ARN, just return it
	if strings.HasPrefix(taskDefinition, "arn:aws:ecs:") {
		return taskDefinition, nil
	}

	// Parse out family & revision
	family, revision, err := parseTaskDefinition(taskDefinition)
	if err != nil {
		return "", err
	}

	iamconn := meta.(*AWSClient).iamconn
	region := meta.(*AWSClient).region

	// An zero value GetUserInput{} defers to the currently logged in user
	resp, err := iamconn.GetUser(&iam.GetUserInput{})
	if err != nil {
		return "", fmt.Errorf("GetUser ERROR: %#v", err)
	}

	// arn:aws:iam::0123456789:user/username
	userARN := *resp.User.ARN
	accountID := strings.Split(userARN, ":")[4]

	// arn:aws:ecs:us-west-2:01234567890:task-definition/mongodb:3
	arn := fmt.Sprintf("arn:aws:ecs:%s:%s:task-definition/%s:%s",
		region, accountID, family, revision)
	log.Printf("[DEBUG] Built task definition ARN: %s", arn)
	return arn, nil
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
