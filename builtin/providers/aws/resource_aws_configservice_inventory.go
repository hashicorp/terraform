package aws

import (
	"fmt"

	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/configservice"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsConfigServiceInventory() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsConfigServiceInventoryCreate,
		Read:   resourceAwsConfigServiceInventoryRead,
		Update: resourceAwsConfigServiceInventoryUpdate,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"role_arn": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"delivery_channel": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"sns_topic_arn": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"s3_bucket_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"s3_bucket_prefix": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},

			"configuration_recorder": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"resource_types": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
						"all_supported": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
					},
				},
			},

			"start_recording": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
	}
}

func resourceAwsConfigServiceInventoryCreate(d *schema.ResourceData, meta interface{}) error {
	configserviceconn := meta.(*AWSClient).configserviceconn

	configurationRecorder := &configservice.ConfigurationRecorder{
		Name:    aws.String(d.Get("name").(string)),
		RoleARN: aws.String(d.Get("role_arn").(string)),
		RecordingGroup: &configservice.RecordingGroup{
			AllSupported: aws.Bool(true),
		},
	}

	input := &configservice.PutConfigurationRecorderInput{
		ConfigurationRecorder: configurationRecorder,
	}

	_, err := configserviceconn.PutConfigurationRecorder(input)
	if err != nil {
		return fmt.Errorf("Error Creating ConfigService Configuration Recorder: %s", err.Error())
	}

	d.SetId(d.Get("name").(string))

	return resourceAwsConfigServiceInventoryUpdate(d, meta)
}

func resourceAwsConfigServiceInventoryRead(d *schema.ResourceData, meta interface{}) error {
	configserviceconn := meta.(*AWSClient).configserviceconn

	out, err := configserviceconn.DescribeConfigurationRecorders(nil)
	if err != nil {
		log.Printf("Error")
	}
	log.Printf("Got the ARN %s", *out.ConfigurationRecorders[0].RoleARN)
	log.Printf("Got the Name %s", *out.ConfigurationRecorders[0].Name)

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
