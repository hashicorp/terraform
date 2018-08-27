package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/pinpoint"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsPinpointBaiduChannel() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsPinpointBaiduChannelUpsert,
		Read:   resourceAwsPinpointBaiduChannelRead,
		Update: resourceAwsPinpointBaiduChannelUpsert,
		Delete: resourceAwsPinpointBaiduChannelDelete,
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
			"api_key": {
				Type:      schema.TypeString,
				Required:  true,
				Sensitive: true,
			},
			"secret_key": {
				Type:      schema.TypeString,
				Required:  true,
				Sensitive: true,
			},
		},
	}
}

func resourceAwsPinpointBaiduChannelUpsert(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).pinpointconn

	applicationId := d.Get("application_id").(string)

	params := &pinpoint.BaiduChannelRequest{}

	params.Enabled = aws.Bool(d.Get("enabled").(bool))
	params.ApiKey = aws.String(d.Get("api_key").(string))
	params.SecretKey = aws.String(d.Get("secret_key").(string))

	req := pinpoint.UpdateBaiduChannelInput{
		ApplicationId:       aws.String(applicationId),
		BaiduChannelRequest: params,
	}

	_, err := conn.UpdateBaiduChannel(&req)
	if err != nil {
		return fmt.Errorf("error updating Pinpoint Baidu Channel for application %s: %s", applicationId, err)
	}

	d.SetId(applicationId)

	return resourceAwsPinpointBaiduChannelRead(d, meta)
}

func resourceAwsPinpointBaiduChannelRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).pinpointconn

	log.Printf("[INFO] Reading Pinpoint Baidu Channel for application %s", d.Id())

	output, err := conn.GetBaiduChannel(&pinpoint.GetBaiduChannelInput{
		ApplicationId: aws.String(d.Id()),
	})
	if err != nil {
		if isAWSErr(err, pinpoint.ErrCodeNotFoundException, "") {
			log.Printf("[WARN] Pinpoint Baidu Channel for application %s not found, error code (404)", d.Id())
			d.SetId("")
			return nil
		}

		return fmt.Errorf("error getting Pinpoint Baidu Channel for application %s: %s", d.Id(), err)
	}

	d.Set("application_id", output.BaiduChannelResponse.ApplicationId)
	d.Set("enabled", output.BaiduChannelResponse.Enabled)
	// ApiKey and SecretKey are never returned

	return nil
}

func resourceAwsPinpointBaiduChannelDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).pinpointconn

	log.Printf("[DEBUG] Deleting Pinpoint Baidu Channel for application %s", d.Id())
	_, err := conn.DeleteBaiduChannel(&pinpoint.DeleteBaiduChannelInput{
		ApplicationId: aws.String(d.Id()),
	})

	if isAWSErr(err, pinpoint.ErrCodeNotFoundException, "") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("error deleting Pinpoint Baidu Channel for application %s: %s", d.Id(), err)
	}
	return nil
}
