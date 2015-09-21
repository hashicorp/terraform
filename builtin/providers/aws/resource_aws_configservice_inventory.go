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
		Delete: resourceAwsConfigServiceInventoryDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"role_arn": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"configuration_recorder": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"resource_types": &schema.Schema{
							Type:     schema.TypeSet,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
						"all_supported": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func resourceAwsConfigServiceInventoryCreate(d *schema.ResourceData, meta interface{}) error {
	configserviceconn := meta.(*AWSClient).configserviceconn

	configurationRecorder := &configservice.ConfigurationRecorder{
		Name:    aws.String("name"),
		RoleARN: aws.String("role_arn"),
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

	return resourceAwsConfigServiceInventoryUpdate(d, meta)
}

func resourceAwsConfigServiceInventoryRead(d *schema.ResourceData, meta interface{}) error {
	configserviceconn := meta.(*AWSClient).configserviceconn

	out, err := configserviceconn.DescribeConfigurationRecorders(nil)
	if err != nil {
		log.Printf("Error")
	}
	log.Printf("Got the ARN %s", out.ConfigurationRecorders[0].RoleARN)
	log.Printf("Got the Name %s", out.ConfigurationRecorders[0].Name)

	return nil
}

func resourceAwsConfigServiceInventoryUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceAwsConfigServiceInventoryDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}
