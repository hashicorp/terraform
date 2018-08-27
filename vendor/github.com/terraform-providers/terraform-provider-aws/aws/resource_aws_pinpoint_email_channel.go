package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/pinpoint"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsPinpointEmailChannel() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsPinpointEmailChannelUpsert,
		Read:   resourceAwsPinpointEmailChannelRead,
		Update: resourceAwsPinpointEmailChannelUpsert,
		Delete: resourceAwsPinpointEmailChannelDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"application_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"from_address": {
				Type:     schema.TypeString,
				Required: true,
			},
			"identity": {
				Type:     schema.TypeString,
				Required: true,
			},
			"role_arn": {
				Type:     schema.TypeString,
				Required: true,
			},
			"messages_per_second": {
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
	}
}

func resourceAwsPinpointEmailChannelUpsert(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).pinpointconn

	applicationId := d.Get("application_id").(string)

	params := &pinpoint.EmailChannelRequest{}

	params.Enabled = aws.Bool(d.Get("enabled").(bool))
	params.FromAddress = aws.String(d.Get("from_address").(string))
	params.Identity = aws.String(d.Get("identity").(string))
	params.RoleArn = aws.String(d.Get("role_arn").(string))

	req := pinpoint.UpdateEmailChannelInput{
		ApplicationId:       aws.String(applicationId),
		EmailChannelRequest: params,
	}

	_, err := conn.UpdateEmailChannel(&req)
	if err != nil {
		return fmt.Errorf("error updating Pinpoint Email Channel for application %s: %s", applicationId, err)
	}

	d.SetId(applicationId)

	return resourceAwsPinpointEmailChannelRead(d, meta)
}

func resourceAwsPinpointEmailChannelRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).pinpointconn

	log.Printf("[INFO] Reading Pinpoint Email Channel for application %s", d.Id())

	output, err := conn.GetEmailChannel(&pinpoint.GetEmailChannelInput{
		ApplicationId: aws.String(d.Id()),
	})
	if err != nil {
		if isAWSErr(err, pinpoint.ErrCodeNotFoundException, "") {
			log.Printf("[WARN] Pinpoint Email Channel for application %s not found, error code (404)", d.Id())
			d.SetId("")
			return nil
		}

		return fmt.Errorf("error getting Pinpoint Email Channel for application %s: %s", d.Id(), err)
	}

	d.Set("application_id", output.EmailChannelResponse.ApplicationId)
	d.Set("enabled", output.EmailChannelResponse.Enabled)
	d.Set("from_address", output.EmailChannelResponse.FromAddress)
	d.Set("identity", output.EmailChannelResponse.Identity)
	d.Set("role_arn", output.EmailChannelResponse.RoleArn)
	d.Set("messages_per_second", aws.Int64Value(output.EmailChannelResponse.MessagesPerSecond))
	return nil
}

func resourceAwsPinpointEmailChannelDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).pinpointconn

	log.Printf("[DEBUG] Deleting Pinpoint Email Channel for application %s", d.Id())
	_, err := conn.DeleteEmailChannel(&pinpoint.DeleteEmailChannelInput{
		ApplicationId: aws.String(d.Id()),
	})

	if isAWSErr(err, pinpoint.ErrCodeNotFoundException, "") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("error deleting Pinpoint Email Channel for application %s: %s", d.Id(), err)
	}
	return nil
}
