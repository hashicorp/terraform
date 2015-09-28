package aws

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
)

// Mutable attributes
// http://docs.aws.amazon.com/sns/latest/api/API_SetPlatformApplicationAttributes.html
var SNSPlatformAppAttributeMap = map[string]string{
	"principal":           "PlatformPrincipal",
	"credential":          "PlatformCredential",
	"created_topic":       "EventEndpointCreated",
	"deleted_topic":       "EventEndpointDeleted",
	"updated_topic":       "EventEndpointUpdated",
	"failure_topic":       "EventDeliveryFailure",
	"success_iam_arn":     "SuccessFeedbackRoleArn",
	"failure_iam_arn":     "FailureFeedbackRoleArn",
	"success_sample_rate": "SuccessFeedbackSampleRate",
}

func resourceAwsSnsApplication() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSnsApplicationCreate,
		Read:   resourceAwsSnsApplicationRead,
		Update: resourceAwsSnsApplicationUpdate,
		Delete: resourceAwsSnsApplicationDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"platform": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"principal": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"credential": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
			"created_topic": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"deleted_topic": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"updated_topic": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"failure_topic": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"success_iam_role": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"failure_iam_role": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"success_sample_rate": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsSnsApplicationCreate(d *schema.ResourceData, meta interface{}) error {
	snsconn := meta.(*AWSClient).snsconn

	name := d.Get("name").(string)

	log.Printf("[DEBUG] SNS create application: %s", name)

	attributes := make(map[string]*string)

	attributes["PlatformCredential"] = d.Get("credential").(*string)

	if v := d.Get("principal").(*string); v != nil {
		attributes["PlatformPrincipal"] = v
	}

	req := &sns.CreatePlatformApplicationInput{
		Name:       aws.String(name),
		Platform:   aws.String(d.Get("platform").(string)),
		Attributes: attributes,
	}

	output, err := snsconn.CreatePlatformApplication(req)
	if err != nil {
		return fmt.Errorf("Error creating SNS application: %s", err)
	}

	d.SetId(*output.PlatformApplicationArn)

	// Write the ARN to the 'arn' field for export
	d.Set("arn", *output.PlatformApplicationArn)

	return resourceAwsSnsApplicationUpdate(d, meta)
}

func resourceAwsSnsApplicationUpdate(d *schema.ResourceData, meta interface{}) error {
	snsconn := meta.(*AWSClient).snsconn

	resource := *resourceAwsSnsApplication()

	attributes := make(map[string]*string)

	for k, _ := range resource.Schema {
		if attrKey, ok := SNSPlatformAppAttributeMap[k]; ok {
			if d.HasChange(k) {
				log.Printf("[DEBUG] Updating %s", attrKey)
				_, n := d.GetChange(k)
				attributes[attrKey] = n.(*string)
			}
		}
	}

	// Make API call to update attributes
	req := &sns.SetPlatformApplicationAttributesInput{
		PlatformApplicationArn: aws.String(d.Id()),
		Attributes:             attributes,
	}
	_, err := snsconn.SetPlatformApplicationAttributes(req)

	if err != nil {
		return fmt.Errorf("Error updating SNS application: %s", err)
	}

	return resourceAwsSnsApplicationRead(d, meta)
}

func resourceAwsSnsApplicationRead(d *schema.ResourceData, meta interface{}) error {
	snsconn := meta.(*AWSClient).snsconn

	attributeOutput, err := snsconn.GetPlatformApplicationAttributes(&sns.GetPlatformApplicationAttributesInput{
		PlatformApplicationArn: aws.String(d.Id()),
	})

	if err != nil {
		return err
	}

	if attributeOutput.Attributes != nil && len(attributeOutput.Attributes) > 0 {
		attrmap := attributeOutput.Attributes
		resource := *resourceAwsSnsApplication()
		// iKey = internal struct key, oKey = AWS Attribute Map key
		for iKey, oKey := range SNSPlatformAppAttributeMap {
			log.Printf("[DEBUG] Updating %s => %s", iKey, oKey)

			if attrmap[oKey] != nil {
				// Some of the fetched attributes are stateful properties such as
				// the number of subscriptions, the owner, etc. skip those
				if resource.Schema[iKey] != nil {
					value := *attrmap[oKey]
					log.Printf("[DEBUG] Updating %s => %s -> %s", iKey, oKey, value)
					d.Set(iKey, *attrmap[oKey])
				}
			}
		}
	}

	return nil
}

func resourceAwsSnsApplicationDelete(d *schema.ResourceData, meta interface{}) error {
	snsconn := meta.(*AWSClient).snsconn

	log.Printf("[DEBUG] SNS Delete Application: %s", d.Id())
	_, err := snsconn.DeletePlatformApplication(&sns.DeletePlatformApplicationInput{
		PlatformApplicationArn: aws.String(d.Id()),
	})
	if err != nil {
		return err
	}
	return nil
}
