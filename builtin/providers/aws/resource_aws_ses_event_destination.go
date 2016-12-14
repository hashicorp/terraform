package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/hashicorp/terraform/helper/schema"
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
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"configuration_set_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"enabled": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},

			"matching_types": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				ForceNew: true,
				Set:      schema.HashString,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validateMatchingTypes,
				},
			},

			"cloudwatch_destination": {
				Type:          schema.TypeSet,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"kinesis_destination"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"default_value": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"dimension_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"value_source": &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateDimensionValueSource,
						},
					},
				},
			},

			"kinesis_destination": {
				Type:          schema.TypeSet,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"cloudwatch_destination"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"stream_arn": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"role_arn": &schema.Schema{
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
		if len(destination) > 1 {
			return fmt.Errorf("You can only define a single kinesis destination per record")
		}
		kinesis := destination[0].(map[string]interface{})
		createOpts.EventDestination.KinesisFirehoseDestination = &ses.KinesisFirehoseDestination{
			DeliveryStreamARN: aws.String(kinesis["stream_arn"].(string)),
			IAMRoleARN:        aws.String(kinesis["role_arn"].(string)),
		}
		log.Printf("[DEBUG] Creating kinesis destination: %#v", kinesis)
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

	if err != nil {
		return err
	}

	return nil
}

func validateMatchingTypes(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	matchingTypes := map[string]bool{
		"send":      true,
		"reject":    true,
		"bounce":    true,
		"complaint": true,
		"delivery":  true,
	}

	if !matchingTypes[value] {
		errors = append(errors, fmt.Errorf("%q must be a valid matching event type value: %q", k, value))
	}
	return
}

func validateDimensionValueSource(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	matchingSource := map[string]bool{
		"messageTag":  true,
		"emailHeader": true,
	}

	if !matchingSource[value] {
		errors = append(errors, fmt.Errorf("%q must be a valid dimension value: %q", k, value))
	}
	return
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
