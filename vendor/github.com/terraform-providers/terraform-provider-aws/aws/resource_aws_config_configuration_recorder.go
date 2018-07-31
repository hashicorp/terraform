package aws

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/configservice"
)

func resourceAwsConfigConfigurationRecorder() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsConfigConfigurationRecorderPut,
		Read:   resourceAwsConfigConfigurationRecorderRead,
		Update: resourceAwsConfigConfigurationRecorderPut,
		Delete: resourceAwsConfigConfigurationRecorderDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Default:      "default",
				ValidateFunc: validateMaxLength(256),
			},
			"role_arn": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateArn,
			},
			"recording_group": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"all_supported": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"include_global_resource_types": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"resource_types": {
							Type:     schema.TypeSet,
							Set:      schema.HashString,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
		},
	}
}

func resourceAwsConfigConfigurationRecorderPut(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).configconn

	name := d.Get("name").(string)
	recorder := configservice.ConfigurationRecorder{
		Name:    aws.String(name),
		RoleARN: aws.String(d.Get("role_arn").(string)),
	}

	if g, ok := d.GetOk("recording_group"); ok {
		recorder.RecordingGroup = expandConfigRecordingGroup(g.([]interface{}))
	}

	input := configservice.PutConfigurationRecorderInput{
		ConfigurationRecorder: &recorder,
	}
	_, err := conn.PutConfigurationRecorder(&input)
	if err != nil {
		return fmt.Errorf("Creating Configuration Recorder failed: %s", err)
	}

	d.SetId(name)

	return resourceAwsConfigConfigurationRecorderRead(d, meta)
}

func resourceAwsConfigConfigurationRecorderRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).configconn

	input := configservice.DescribeConfigurationRecordersInput{
		ConfigurationRecorderNames: []*string{aws.String(d.Id())},
	}
	out, err := conn.DescribeConfigurationRecorders(&input)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NoSuchConfigurationRecorderException" {
			log.Printf("[WARN] Configuration Recorder %q is gone (NoSuchConfigurationRecorderException)", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Getting Configuration Recorder failed: %s", err)
	}

	numberOfRecorders := len(out.ConfigurationRecorders)
	if numberOfRecorders < 1 {
		log.Printf("[WARN] Configuration Recorder %q is gone (no recorders found)", d.Id())
		d.SetId("")
		return nil
	}

	if numberOfRecorders > 1 {
		return fmt.Errorf("Expected exactly 1 Configuration Recorder, received %d: %#v",
			numberOfRecorders, out.ConfigurationRecorders)
	}

	recorder := out.ConfigurationRecorders[0]

	d.Set("name", recorder.Name)
	d.Set("role_arn", recorder.RoleARN)

	if recorder.RecordingGroup != nil {
		flattened := flattenConfigRecordingGroup(recorder.RecordingGroup)
		err = d.Set("recording_group", flattened)
		if err != nil {
			return fmt.Errorf("Failed to set recording_group: %s", err)
		}
	}

	return nil
}

func resourceAwsConfigConfigurationRecorderDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).configconn
	input := configservice.DeleteConfigurationRecorderInput{
		ConfigurationRecorderName: aws.String(d.Id()),
	}
	_, err := conn.DeleteConfigurationRecorder(&input)
	if err != nil {
		return fmt.Errorf("Deleting Configuration Recorder failed: %s", err)
	}

	return nil
}
