package aws

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/configservice"
)

func resourceAwsConfigConfigurationRecorder() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsConfigConfigurationRecorderPut,
		Read:   resourceAwsConfigConfigurationRecorderRead,
		Update: resourceAwsConfigConfigurationRecorderPut,
		Delete: resourceAwsConfigConfigurationRecorderDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "default",
				ValidateFunc: validateMaxLength(256),
			},
			"role_arn": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateArn,
			},
			"recording_group": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"all_supported": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},
						"include_global_resource_types": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},
						"resource_types": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"is_enabled": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
		},
	}
}

func resourceAwsConfigConfigurationRecorderPut(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).configconn

	name := d.Get("name").(string)
	recorder := configservice.ConfigurationRecorder{
		Name: aws.String(name),
	}

	if v, ok := d.GetOk("role_arn"); ok {
		recorder.RoleARN = aws.String(v.(string))
	}

	if g, ok := d.GetOk("recording_group"); ok {
		groups := g.([]interface{})

		recordingGroup := configservice.RecordingGroup{}
		group := groups[0].(map[string]interface{})

		if v, ok := group["all_supported"]; ok {
			recordingGroup.AllSupported = aws.Bool(v.(bool))
		}

		if v, ok := group["include_global_resource_types"]; ok {
			recordingGroup.IncludeGlobalResourceTypes = aws.Bool(v.(bool))
		}

		if v, ok := group["resource_types"]; ok {
			recordingGroup.ResourceTypes = expandStringList(v.([]interface{}))
		}

		recorder.RecordingGroup = &recordingGroup
	}

	input := configservice.PutConfigurationRecorderInput{
		ConfigurationRecorder: &recorder,
	}
	_, err := conn.PutConfigurationRecorder(&input)
	if err != nil {
		return fmt.Errorf("Creating Configuration Recorder failed: %s", err)
	}

	d.SetId(name)

	if d.HasChange("is_enabled") {
		isEnabled := d.Get("is_enabled").(bool)
		if isEnabled {
			startInput := configservice.StartConfigurationRecorderInput{
				ConfigurationRecorderName: aws.String(d.Id()),
			}
			_, err := conn.StartConfigurationRecorder(&startInput)
			if err != nil {
				return fmt.Errorf("Failed to start Configuration Recorder: %s", err)
			}
		} else {
			stopInput := configservice.StopConfigurationRecorderInput{
				ConfigurationRecorderName: aws.String(d.Id()),
			}
			_, err := conn.StopConfigurationRecorder(&stopInput)
			if err != nil {
				return fmt.Errorf("Failed to stop Configuration Recorder: %s", err)
			}
		}
	}

	return resourceAwsConfigConfigurationRecorderRead(d, meta)
}

func resourceAwsConfigConfigurationRecorderRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).configconn

	input := configservice.DescribeConfigurationRecordersInput{
		ConfigurationRecorderNames: []*string{aws.String(d.Id())},
	}
	out, err := conn.DescribeConfigurationRecorders(&input)
	if err != nil {
		return fmt.Errorf("Getting Configuration Recorder failed: %s", err)
	}

	if len(out.ConfigurationRecorders) < 1 {
		log.Printf("[WARN] Configuration Recorder %q is gone", d.Id())
		d.SetId("")
		return nil
	}

	recorder := out.ConfigurationRecorders[0]

	d.Set("name", recorder.Name)
	d.Set("role_arn", recorder.RoleARN)

	if recorder.RecordingGroup != nil {
		d.Set("recording_group", flattenConfigRecordingGroup(recorder.RecordingGroup))
	}

	statusInput := configservice.DescribeConfigurationRecorderStatusInput{
		ConfigurationRecorderNames: []*string{aws.String(d.Id())},
	}
	statusOut, err := conn.DescribeConfigurationRecorderStatus(&statusInput)
	if err != nil {
		return fmt.Errorf("Failed describing Configuration Recorder %q status: %s",
			d.Id(), err)
	}
	if len(statusOut.ConfigurationRecordersStatus) < 1 {
		return fmt.Errorf("Failed describing Configuration Recorder %q status:"+
			" no recorders found", d.Id())
	}

	d.Set("is_enabled", statusOut.ConfigurationRecordersStatus[0].Recording)

	return nil
}

func resourceAwsConfigConfigurationRecorderDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).configconn
	input := configservice.StopConfigurationRecorderInput{
		ConfigurationRecorderName: aws.String(d.Id()),
	}
	_, err := conn.StopConfigurationRecorder(&input)
	if err != nil {
		return fmt.Errorf("Stopping Configuration Recorder failed: %s", err)
	}

	d.SetId("")
	return nil
}
