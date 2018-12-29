package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/pinpoint"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsPinpointGCMChannel() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsPinpointGCMChannelUpsert,
		Read:   resourceAwsPinpointGCMChannelRead,
		Update: resourceAwsPinpointGCMChannelUpsert,
		Delete: resourceAwsPinpointGCMChannelDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"application_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"api_key": {
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

func resourceAwsPinpointGCMChannelUpsert(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).pinpointconn

	applicationId := d.Get("application_id").(string)

	params := &pinpoint.GCMChannelRequest{}

	params.ApiKey = aws.String(d.Get("api_key").(string))
	params.Enabled = aws.Bool(d.Get("enabled").(bool))

	req := pinpoint.UpdateGcmChannelInput{
		ApplicationId:     aws.String(applicationId),
		GCMChannelRequest: params,
	}

	_, err := conn.UpdateGcmChannel(&req)
	if err != nil {
		return fmt.Errorf("error putting Pinpoint GCM Channel for application %s: %s", applicationId, err)
	}

	d.SetId(applicationId)

	return resourceAwsPinpointGCMChannelRead(d, meta)
}

func resourceAwsPinpointGCMChannelRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).pinpointconn

	log.Printf("[INFO] Reading Pinpoint GCM Channel for application %s", d.Id())

	output, err := conn.GetGcmChannel(&pinpoint.GetGcmChannelInput{
		ApplicationId: aws.String(d.Id()),
	})
	if err != nil {
		if isAWSErr(err, pinpoint.ErrCodeNotFoundException, "") {
			log.Printf("[WARN] Pinpoint GCM Channel for application %s not found, error code (404)", d.Id())
			d.SetId("")
			return nil
		}

		return fmt.Errorf("error getting Pinpoint GCM Channel for application %s: %s", d.Id(), err)
	}

	d.Set("application_id", output.GCMChannelResponse.ApplicationId)
	d.Set("enabled", output.GCMChannelResponse.Enabled)
	// api_key is never returned

	return nil
}

func resourceAwsPinpointGCMChannelDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).pinpointconn

	log.Printf("[DEBUG] Deleting Pinpoint GCM Channel for application %s", d.Id())
	_, err := conn.DeleteGcmChannel(&pinpoint.DeleteGcmChannelInput{
		ApplicationId: aws.String(d.Id()),
	})

	if isAWSErr(err, pinpoint.ErrCodeNotFoundException, "") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("error deleting Pinpoint GCM Channel for application %s: %s", d.Id(), err)
	}
	return nil
}
