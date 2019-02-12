package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsSesEventDestination() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSesEventDestinationCreate,
		Read:   resourceAwsSesEventDestinationRead,
		Delete: resourceAwsSesEventDestinationDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"configuration_set_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},

			"matching_types": {
				Type:     schema.TypeSet,
				Required: true,
				ForceNew: true,
				Set:      schema.HashString,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{
						ses.EventTypeSend,
						ses.EventTypeReject,
						ses.EventTypeBounce,
						ses.EventTypeComplaint,
						ses.EventTypeDelivery,
						ses.EventTypeOpen,
						ses.EventTypeClick,
						ses.EventTypeRenderingFailure,
					}, false),
				},
			},

			"cloudwatch_destination": {
				Type:          schema.TypeSet,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"kinesis_destination", "sns_destination"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"default_value": {
							Type:     schema.TypeString,
							Required: true,
						},

						"dimension_name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"value_source": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								ses.DimensionValueSourceMessageTag,
								ses.DimensionValueSourceEmailHeader,
								ses.DimensionValueSourceLinkTag,
							}, false),
						},
					},
				},
			},

			"kinesis_destination": {
				Type:          schema.TypeSet,
				Optional:      true,
				ForceNew:      true,
				MaxItems:      1,
				ConflictsWith: []string{"cloudwatch_destination", "sns_destination"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"stream_arn": {
							Type:     schema.TypeString,
							Required: true,
						},

						"role_arn": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},

			"sns_destination": {
				Type:          schema.TypeSet,
				MaxItems:      1,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"cloudwatch_destination", "kinesis_destination"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"topic_arn": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func resourceAwsSesEventDestinationCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sesConn

	configurationSetName := d.Get("configuration_set_name").(string)
	eventDestinationName := d.Get("name").(string)
	enabled := d.Get("enabled").(bool)
	matchingEventTypes := d.Get("matching_types").(*schema.Set).List()

	createOpts := &ses.CreateConfigurationSetEventDestinationInput{
		ConfigurationSetName: aws.String(configurationSetName),
		EventDestination: &ses.EventDestination{
			Name:               aws.String(eventDestinationName),
			Enabled:            aws.Bool(enabled),
			MatchingEventTypes: expandStringList(matchingEventTypes),
		},
	}

	if v, ok := d.GetOk("cloudwatch_destination"); ok {
		destination := v.(*schema.Set).List()
		createOpts.EventDestination.CloudWatchDestination = &ses.CloudWatchDestination{
			DimensionConfigurations: generateCloudWatchDestination(destination),
		}
		log.Printf("[DEBUG] Creating cloudwatch destination: %#v", destination)
	}

	if v, ok := d.GetOk("kinesis_destination"); ok {
		destination := v.(*schema.Set).List()

		kinesis := destination[0].(map[string]interface{})
		createOpts.EventDestination.KinesisFirehoseDestination = &ses.KinesisFirehoseDestination{
			DeliveryStreamARN: aws.String(kinesis["stream_arn"].(string)),
			IAMRoleARN:        aws.String(kinesis["role_arn"].(string)),
		}
		log.Printf("[DEBUG] Creating kinesis destination: %#v", kinesis)
	}

	if v, ok := d.GetOk("sns_destination"); ok {
		destination := v.(*schema.Set).List()
		sns := destination[0].(map[string]interface{})
		createOpts.EventDestination.SNSDestination = &ses.SNSDestination{
			TopicARN: aws.String(sns["topic_arn"].(string)),
		}
		log.Printf("[DEBUG] Creating sns destination: %#v", sns)
	}

	_, err := conn.CreateConfigurationSetEventDestination(createOpts)
	if err != nil {
		return fmt.Errorf("Error creating SES configuration set event destination: %s", err)
	}

	d.SetId(eventDestinationName)

	log.Printf("[WARN] SES DONE")
	return resourceAwsSesEventDestinationRead(d, meta)
}

func resourceAwsSesEventDestinationRead(d *schema.ResourceData, meta interface{}) error {

	return nil
}

func resourceAwsSesEventDestinationDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sesConn

	log.Printf("[DEBUG] SES Delete Configuration Set Destination: %s", d.Id())
	_, err := conn.DeleteConfigurationSetEventDestination(&ses.DeleteConfigurationSetEventDestinationInput{
		ConfigurationSetName: aws.String(d.Get("configuration_set_name").(string)),
		EventDestinationName: aws.String(d.Id()),
	})

	return err
}

func generateCloudWatchDestination(v []interface{}) []*ses.CloudWatchDimensionConfiguration {

	b := make([]*ses.CloudWatchDimensionConfiguration, len(v))

	for i, vI := range v {
		cloudwatch := vI.(map[string]interface{})
		b[i] = &ses.CloudWatchDimensionConfiguration{
			DefaultDimensionValue: aws.String(cloudwatch["default_value"].(string)),
			DimensionName:         aws.String(cloudwatch["dimension_name"].(string)),
			DimensionValueSource:  aws.String(cloudwatch["value_source"].(string)),
		}
	}

	return b
}
