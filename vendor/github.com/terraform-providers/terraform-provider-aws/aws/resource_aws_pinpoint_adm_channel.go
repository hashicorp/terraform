package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/pinpoint"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsPinpointADMChannel() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsPinpointADMChannelUpsert,
		Read:   resourceAwsPinpointADMChannelRead,
		Update: resourceAwsPinpointADMChannelUpsert,
		Delete: resourceAwsPinpointADMChannelDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"application_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"client_id": {
				Type:      schema.TypeString,
				Required:  true,
				Sensitive: true,
			},
			"client_secret": {
				Type:      schema.TypeString,
				Required:  true,
				Sensitive: true,
			},
			"enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
		},
	}
}

func resourceAwsPinpointADMChannelUpsert(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).pinpointconn

	applicationId := d.Get("application_id").(string)

	params := &pinpoint.ADMChannelRequest{}

	params.ClientId = aws.String(d.Get("client_id").(string))
	params.ClientSecret = aws.String(d.Get("client_secret").(string))
	params.Enabled = aws.Bool(d.Get("enabled").(bool))

	req := pinpoint.UpdateAdmChannelInput{
		ApplicationId:     aws.String(applicationId),
		ADMChannelRequest: params,
	}

	_, err := conn.UpdateAdmChannel(&req)
	if err != nil {
		return err
	}

	d.SetId(applicationId)

	return resourceAwsPinpointADMChannelRead(d, meta)
}

func resourceAwsPinpointADMChannelRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).pinpointconn

	log.Printf("[INFO] Reading Pinpoint ADM Channel for application %s", d.Id())

	channel, err := conn.GetAdmChannel(&pinpoint.GetAdmChannelInput{
		ApplicationId: aws.String(d.Id()),
	})
	if err != nil {
		if isAWSErr(err, pinpoint.ErrCodeNotFoundException, "") {
			log.Printf("[WARN] Pinpoint ADM Channel for application %s not found, error code (404)", d.Id())
			d.SetId("")
			return nil
		}

		return fmt.Errorf("error getting Pinpoint ADM Channel for application %s: %s", d.Id(), err)
	}

	d.Set("application_id", channel.ADMChannelResponse.ApplicationId)
	d.Set("enabled", channel.ADMChannelResponse.Enabled)
	// client_id and client_secret are never returned

	return nil
}

func resourceAwsPinpointADMChannelDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).pinpointconn

	log.Printf("[DEBUG] Pinpoint Delete ADM Channel: %s", d.Id())
	_, err := conn.DeleteAdmChannel(&pinpoint.DeleteAdmChannelInput{
		ApplicationId: aws.String(d.Id()),
	})

	if isAWSErr(err, pinpoint.ErrCodeNotFoundException, "") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("error deleting Pinpoint ADM Channel for application %s: %s", d.Id(), err)
	}
	return nil
}
