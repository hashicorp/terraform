package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/codedeploy"
)

func resourceAwsCodeDeployApp() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCodeDeployAppCreate,
		Read:   resourceAwsCodeDeployAppRead,
		Update: resourceAwsCodeDeployUpdate,
		Delete: resourceAwsCodeDeployAppDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			// The unique ID is set by AWS on create.
			"unique_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceAwsCodeDeployAppCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codedeployconn

	application := d.Get("name").(string)
	log.Printf("[DEBUG] Creating CodeDeploy application %s", application)

	resp, err := conn.CreateApplication(&codedeploy.CreateApplicationInput{
		ApplicationName: aws.String(application),
	})
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] CodeDeploy application %s created", *resp.ApplicationId)

	// Despite giving the application a unique ID, AWS doesn't actually use
	// it in API calls. Use it and the app name to identify the resource in
	// the state file. This allows us to reliably detect both when the TF
	// config file changes and when the user deletes the app without removing
	// it first from the TF config.
	d.SetId(fmt.Sprintf("%s:%s", *resp.ApplicationId, application))

	return resourceAwsCodeDeployAppRead(d, meta)
}

func resourceAwsCodeDeployAppRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codedeployconn

	_, application := resourceAwsCodeDeployAppParseId(d.Id())
	log.Printf("[DEBUG] Reading CodeDeploy application %s", application)
	resp, err := conn.GetApplication(&codedeploy.GetApplicationInput{
		ApplicationName: aws.String(application),
	})
	if err != nil {
		if codedeployerr, ok := err.(awserr.Error); ok && codedeployerr.Code() == "ApplicationDoesNotExistException" {
			d.SetId("")
			return nil
		} else {
			log.Printf("[ERROR] Error finding CodeDeploy application: %s", err)
			return err
		}
	}

	d.Set("name", resp.Application.ApplicationName)

	return nil
}

func resourceAwsCodeDeployUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codedeployconn

	o, n := d.GetChange("name")

	_, err := conn.UpdateApplication(&codedeploy.UpdateApplicationInput{
		ApplicationName:    aws.String(o.(string)),
		NewApplicationName: aws.String(n.(string)),
	})
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] CodeDeploy application %s updated", n)

	d.Set("name", n)

	return nil
}

func resourceAwsCodeDeployAppDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codedeployconn

	_, err := conn.DeleteApplication(&codedeploy.DeleteApplicationInput{
		ApplicationName: aws.String(d.Get("name").(string)),
	})
	if err != nil {
		if cderr, ok := err.(awserr.Error); ok && cderr.Code() == "InvalidApplicationNameException" {
			d.SetId("")
			return nil
		} else {
			log.Printf("[ERROR] Error deleting CodeDeploy application: %s", err)
			return err
		}
	}

	return nil
}

func resourceAwsCodeDeployAppParseId(id string) (string, string) {
	parts := strings.SplitN(id, ":", 2)
	return parts[0], parts[1]
}
