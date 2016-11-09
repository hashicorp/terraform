package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/configservice"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsConfigServiceInventory() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsConfigServiceInventoryCreate,
		Read:   resourceAwsConfigServiceInventoryRead,
		Update: resourceAwsConfigServiceInventoryUpdate,
		Delete: resourceAwsConfigServiceInventoryDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"role_arn": {
				Type:     schema.TypeString,
				Required: true,
			},

			"delivery_channel": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"sns_topic_arn": {
							Type:     schema.TypeString,
							Required: true,
						},
						"s3_bucket_name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"s3_bucket_prefix": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},

			"configuration_recorder": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"resource_types": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
						"all_supported": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
					},
				},
			},
			"start_recording": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
	}
}

func resourceAwsConfigServiceInventoryCreate(d *schema.ResourceData, meta interface{}) error {
	configserviceconn := meta.(*AWSClient).configserviceconn

	log.Printf("[INFO] Creating Cofig Inventory: %s", d.Get("name").(string))

	input := &configservice.PutConfigurationRecorderInput{
		ConfigurationRecorder: &configservice.ConfigurationRecorder{
			Name:    aws.String(d.Get("name").(string)),
			RoleARN: aws.String(d.Get("role_arn").(string)),
		},
	}

	_, err := configserviceconn.PutConfigurationRecorder(input)

	if err != nil {
		return errwrap.Wrapf("[ERROR] Error Creating ConfigService Configuration Recorder: {{err}}", err)
	}

	d.SetId(d.Get("name").(string))

	return resourceAwsConfigServiceInventoryUpdate(d, meta)
}

func resourceAwsConfigServiceInventoryRead(d *schema.ResourceData, meta interface{}) error {
	configserviceconn := meta.(*AWSClient).configserviceconn

	params := &configservice.DescribeConfigurationRecordersInput{
		ConfigurationRecorderNames: []*string{
			aws.String(d.Get("name").(string)),
		},
	}

	resp, err := configserviceconn.DescribeConfigurationRecorders(params)
	if err != nil {
		log.Printf("Error")
	}

	d.Set("role_arn", *resp.ConfigurationRecorders[0].RoleARN)
	d.Set("name", *resp.ConfigurationRecorders[0].Name)

	recorders := resp.ConfigurationRecorders[0]
	recorder := make(map[string]interface{})

	recorder["resource_types"] = recorders.RecordingGroup.ResourceTypes
	recorder["all_supported"] = recorders.RecordingGroup.AllSupported

	if err := d.Set("configuration_recorder", []interface{}{recorder}); err != nil {
		return err
	}

	return nil
}

func resourceAwsConfigServiceInventoryUpdate(d *schema.ResourceData, meta interface{}) error {
	configserviceconn := meta.(*AWSClient).configserviceconn

	if d.HasChange("start_recording") {
		if err := resourceAwsConfigServiceUpdateRecordingStatus(configserviceconn, d); err != nil {
			return nil
		}
	}

	if d.HasChange("configuration_recorder") {
		if err := resourceAwsConfigServiceUpdateRecordingResources(configserviceconn, d); err != nil {
			return err
		}
	}

	if d.HasChange("delivery_channel") {
		if err := resourceAwsConfigServiceUpdateDeliveryChannel(configserviceconn, d); err != nil {
			return err
		}
	}

	return nil
}

func resourceAwsConfigServiceUpdateDeliveryChannel(configserviceconn *configservice.ConfigService, d *schema.ResourceData) error {
	log.Printf("[DEBUG] Updating the ConfigService Delivery Channel for %s", d.Id())
	return nil
}

func resourceAwsConfigServiceUpdateRecordingResources(configserviceconn *configservice.ConfigService, d *schema.ResourceData) error {
	log.Printf("[DEBUG] Updating the ConfigService RecordingResources for %s", d.Id())
	return nil
}

func resourceAwsConfigServiceUpdateRecordingStatus(configserviceconn *configservice.ConfigService, d *schema.ResourceData) error {
	log.Printf("[DEBUG] Updating the ConfigService Recording Status for %s", d.Id())

	if d.Get("start_recording").(bool) {
		log.Printf("[DEBUG] Turning on the ConfigService Recording for %s", d.Id())

		_, err := configserviceconn.StartConfigurationRecorder(&configservice.StartConfigurationRecorderInput{
			ConfigurationRecorderName: aws.String(d.Id()),
		})
		if err != nil {
			return err
		}
	} else {
		log.Printf("[DEBUG] Turning off the ConfigService Recording for %s", d.Id())

		_, err := configserviceconn.StopConfigurationRecorder(&configservice.StopConfigurationRecorderInput{
			ConfigurationRecorderName: aws.String(d.Id()),
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func resourceAwsConfigServiceInventoryDelete(d *schema.ResourceData, meta interface{}) error {
	configserviceconn := meta.(*AWSClient).configserviceconn

	log.Printf("[INFO] Deleting ConfigService Inventory: %s", d.Id())

	params := &configservice.DeleteConfigurationRecorderInput{
		ConfigurationRecorderName: aws.String(d.Get("name").(string)),
	}

	_, err := configserviceconn.DeleteConfigurationRecorder(params)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Waiting for ConfigService Inventory %q to be deleted", d.Get("name").(string))
	err = resource.Retry(5*time.Minute, func() *resource.RetryError {
		_, err := configserviceconn.DescribeConfigurationRecorders(&configservice.DescribeConfigurationRecordersInput{
			ConfigurationRecorderNames: []*string{
				aws.String(d.Get("name").(string)),
			},
		})

		if err != nil {
			_, ok := err.(awserr.Error)
			if !ok {
				return resource.NonRetryableError(err)
			}

			return resource.NonRetryableError(err)
		}

		return resource.RetryableError(
			fmt.Errorf("%q: Timeout while waiting for the inventory to be deleted", d.Id()))
	})
	if err != nil {
		return err
	}

	d.SetId("")

	return nil
}
