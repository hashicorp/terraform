package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/pinpoint"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsPinpointEventStream() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsPinpointEventStreamUpsert,
		Read:   resourceAwsPinpointEventStreamRead,
		Update: resourceAwsPinpointEventStreamUpsert,
		Delete: resourceAwsPinpointEventStreamDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"application_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"destination_stream_arn": {
				Type:     schema.TypeString,
				Required: true,
			},
			"role_arn": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceAwsPinpointEventStreamUpsert(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).pinpointconn

	applicationId := d.Get("application_id").(string)

	params := &pinpoint.WriteEventStream{}

	params.DestinationStreamArn = aws.String(d.Get("destination_stream_arn").(string))
	params.RoleArn = aws.String(d.Get("role_arn").(string))

	req := pinpoint.PutEventStreamInput{
		ApplicationId:    aws.String(applicationId),
		WriteEventStream: params,
	}

	_, err := conn.PutEventStream(&req)
	if err != nil {
		return fmt.Errorf("error putting Pinpoint Event Stream for application %s: %s", applicationId, err)
	}

	d.SetId(applicationId)

	return resourceAwsPinpointEventStreamRead(d, meta)
}

func resourceAwsPinpointEventStreamRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).pinpointconn

	log.Printf("[INFO] Reading Pinpoint Event Stream for application %s", d.Id())

	output, err := conn.GetEventStream(&pinpoint.GetEventStreamInput{
		ApplicationId: aws.String(d.Id()),
	})
	if err != nil {
		if isAWSErr(err, pinpoint.ErrCodeNotFoundException, "") {
			log.Printf("[WARN] Pinpoint Event Stream for application %s not found, error code (404)", d.Id())
			d.SetId("")
			return nil
		}

		return fmt.Errorf("error getting Pinpoint Event Stream for application %s: %s", d.Id(), err)
	}

	d.Set("application_id", output.EventStream.ApplicationId)
	d.Set("destination_stream_arn", output.EventStream.DestinationStreamArn)
	d.Set("role_arn", output.EventStream.RoleArn)

	return nil
}

func resourceAwsPinpointEventStreamDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).pinpointconn

	log.Printf("[DEBUG] Pinpoint Delete Event Stream: %s", d.Id())
	_, err := conn.DeleteEventStream(&pinpoint.DeleteEventStreamInput{
		ApplicationId: aws.String(d.Id()),
	})

	if isAWSErr(err, pinpoint.ErrCodeNotFoundException, "") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("error deleting Pinpoint Event Stream for application %s: %s", d.Id(), err)
	}
	return nil
}
