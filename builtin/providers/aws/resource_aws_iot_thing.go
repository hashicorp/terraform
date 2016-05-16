package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iot"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsIotThing() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsIotThingCreate,
		Read:   resourceAwsIotThingRead,
		Update: resourceAwsIotThingUpdate,
		Delete: resourceAwsIotThingDelete,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"attributes": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
			},
		},
	}
}

func resourceAwsIotThingCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iotconn

	params := &iot.CreateThingInput{
		ThingName: aws.String(d.Get("name").(string)), // Required
		AttributePayload: &iot.AttributePayload{
			Attributes: d.Get("attributes").(map[string]*string),
		},
	}
	_, err := conn.CreateThing(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return err
	}

	return nil
}

func resourceAwsIotThingRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iotconn

	params := &iot.DescribeThingInput{
		ThingName: aws.String(d.Get("name").(string)), // Required
	}
	describeThingResp, err := conn.DescribeThing(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return err
	}

	d.Set("name", describeThingResp.ThingName)
	d.Set("default_client_id", describeThingResp.DefaultClientId)
	d.Set("attributes", describeThingResp.Attributes)

	return nil
}

func resourceAwsIotThingUpdate(d *schema.ResourceData, meta interface{}) error {

	conn := meta.(*AWSClient).iotconn

	if d.HasChange("attributes") {
		params := &iot.UpdateThingInput{
			AttributePayload: &iot.AttributePayload{ // Required
				Attributes: d.Get("attributes").(map[string]*string),
			},
			ThingName: aws.String(d.Get("name").(string)), // Required
		}
		_, err := conn.UpdateThing(params)

		if err != nil {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
			return err
		}
	}
	return nil
}

func resourceAwsIotThingDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iotconn

	params := &iot.DeleteThingInput{
		ThingName: aws.String(d.Get("name").(string)), // Required
	}
	_, err := conn.DeleteThing(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return err
	}

	return nil
}
