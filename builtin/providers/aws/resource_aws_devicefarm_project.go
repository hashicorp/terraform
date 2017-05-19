package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/devicefarm"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsDevicefarmProject() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDevicefarmProjectCreate,
		Read:   resourceAwsDevicefarmProjectRead,
		Update: resourceAwsDevicefarmProjectUpdate,
		Delete: resourceAwsDevicefarmProjectDelete,

		Schema: map[string]*schema.Schema{
			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceAwsDevicefarmProjectCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).devicefarmconn
	region := meta.(*AWSClient).region

	//	We need to ensure that DeviceFarm is only being run against us-west-2
	//	As this is the only place that AWS currently supports it
	if region != "us-west-2" {
		return fmt.Errorf("DeviceFarm can only be used with us-west-2. You are trying to use it on %s", region)
	}

	input := &devicefarm.CreateProjectInput{
		Name: aws.String(d.Get("name").(string)),
	}

	log.Printf("[DEBUG] Creating DeviceFarm Project: %s", d.Get("name").(string))
	out, err := conn.CreateProject(input)
	if err != nil {
		return fmt.Errorf("Error creating DeviceFarm Project: %s", err)
	}

	log.Printf("[DEBUG] Successsfully Created DeviceFarm Project: %s", *out.Project.Arn)
	d.SetId(*out.Project.Arn)

	return resourceAwsDevicefarmProjectRead(d, meta)
}

func resourceAwsDevicefarmProjectRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).devicefarmconn

	input := &devicefarm.GetProjectInput{
		Arn: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Reading DeviceFarm Project: %s", d.Id())
	out, err := conn.GetProject(input)
	if err != nil {
		return fmt.Errorf("Error reading DeviceFarm Project: %s", err)
	}

	d.Set("name", out.Project.Name)
	d.Set("arn", out.Project.Arn)

	return nil
}

func resourceAwsDevicefarmProjectUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).devicefarmconn

	if d.HasChange("name") {
		input := &devicefarm.UpdateProjectInput{
			Arn:  aws.String(d.Id()),
			Name: aws.String(d.Get("name").(string)),
		}

		log.Printf("[DEBUG] Updating DeviceFarm Project: %s", d.Id())
		_, err := conn.UpdateProject(input)
		if err != nil {
			return fmt.Errorf("Error Updating DeviceFarm Project: %s", err)
		}

	}

	return resourceAwsDevicefarmProjectRead(d, meta)
}

func resourceAwsDevicefarmProjectDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).devicefarmconn

	input := &devicefarm.DeleteProjectInput{
		Arn: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Deleting DeviceFarm Project: %s", d.Id())
	_, err := conn.DeleteProject(input)
	if err != nil {
		return fmt.Errorf("Error deleting DeviceFarm Project: %s", err)
	}

	return nil
}
