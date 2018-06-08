package aws

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/configservice"
)

func resourceAwsConfigConfigurationRecorderStatus() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsConfigConfigurationRecorderStatusPut,
		Read:   resourceAwsConfigConfigurationRecorderStatusRead,
		Update: resourceAwsConfigConfigurationRecorderStatusPut,
		Delete: resourceAwsConfigConfigurationRecorderStatusDelete,

		Importer: &schema.ResourceImporter{
			State: func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				d.Set("name", d.Id())
				return []*schema.ResourceData{d}, nil
			},
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"is_enabled": {
				Type:     schema.TypeBool,
				Required: true,
			},
		},
	}
}

func resourceAwsConfigConfigurationRecorderStatusPut(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).configconn

	name := d.Get("name").(string)
	d.SetId(name)

	if d.HasChange("is_enabled") {
		isEnabled := d.Get("is_enabled").(bool)
		if isEnabled {
			log.Printf("[DEBUG] Starting AWSConfig Configuration recorder %q", name)
			startInput := configservice.StartConfigurationRecorderInput{
				ConfigurationRecorderName: aws.String(name),
			}
			_, err := conn.StartConfigurationRecorder(&startInput)
			if err != nil {
				return fmt.Errorf("Failed to start Configuration Recorder: %s", err)
			}
		} else {
			log.Printf("[DEBUG] Stopping AWSConfig Configuration recorder %q", name)
			stopInput := configservice.StopConfigurationRecorderInput{
				ConfigurationRecorderName: aws.String(name),
			}
			_, err := conn.StopConfigurationRecorder(&stopInput)
			if err != nil {
				return fmt.Errorf("Failed to stop Configuration Recorder: %s", err)
			}
		}
	}

	return resourceAwsConfigConfigurationRecorderStatusRead(d, meta)
}

func resourceAwsConfigConfigurationRecorderStatusRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).configconn

	name := d.Id()
	statusInput := configservice.DescribeConfigurationRecorderStatusInput{
		ConfigurationRecorderNames: []*string{aws.String(name)},
	}
	statusOut, err := conn.DescribeConfigurationRecorderStatus(&statusInput)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "NoSuchConfigurationRecorderException" {
				log.Printf("[WARN] Configuration Recorder (status) %q is gone (NoSuchConfigurationRecorderException)", name)
				d.SetId("")
				return nil
			}
		}
		return fmt.Errorf("Failed describing Configuration Recorder %q status: %s",
			name, err)
	}

	numberOfStatuses := len(statusOut.ConfigurationRecordersStatus)
	if numberOfStatuses < 1 {
		log.Printf("[WARN] Configuration Recorder (status) %q is gone (no recorders found)", name)
		d.SetId("")
		return nil
	}

	if numberOfStatuses > 1 {
		return fmt.Errorf("Expected exactly 1 Configuration Recorder (status), received %d: %#v",
			numberOfStatuses, statusOut.ConfigurationRecordersStatus)
	}

	d.Set("is_enabled", statusOut.ConfigurationRecordersStatus[0].Recording)

	return nil
}

func resourceAwsConfigConfigurationRecorderStatusDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).configconn
	input := configservice.StopConfigurationRecorderInput{
		ConfigurationRecorderName: aws.String(d.Get("name").(string)),
	}
	_, err := conn.StopConfigurationRecorder(&input)
	if err != nil {
		return fmt.Errorf("Stopping Configuration Recorder failed: %s", err)
	}

	d.SetId("")
	return nil
}
