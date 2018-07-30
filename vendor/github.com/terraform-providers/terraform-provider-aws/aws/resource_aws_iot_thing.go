package aws

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iot"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsIotThing() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsIotThingCreate,
		Read:   resourceAwsIotThingRead,
		Update: resourceAwsIotThingUpdate,
		Delete: resourceAwsIotThingDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringLenBetween(1, 128),
			},
			"attributes": {
				Type:     schema.TypeMap,
				Optional: true,
			},
			"thing_type_name": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringLenBetween(1, 128),
			},
			"default_client_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"version": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsIotThingCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iotconn

	params := &iot.CreateThingInput{
		ThingName: aws.String(d.Get("name").(string)),
	}

	if v, ok := d.GetOk("thing_type_name"); ok {
		params.ThingTypeName = aws.String(v.(string))
	}
	if v, ok := d.GetOk("attributes"); ok {
		params.AttributePayload = &iot.AttributePayload{
			Attributes: stringMapToPointers(v.(map[string]interface{})),
		}
	}

	log.Printf("[DEBUG] Creating IoT Thing: %s", params)
	out, err := conn.CreateThing(params)
	if err != nil {
		return err
	}

	d.SetId(*out.ThingName)

	return resourceAwsIotThingRead(d, meta)
}

func resourceAwsIotThingRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iotconn

	params := &iot.DescribeThingInput{
		ThingName: aws.String(d.Id()),
	}
	log.Printf("[DEBUG] Reading IoT Thing: %s", params)
	out, err := conn.DescribeThing(params)

	if err != nil {
		if isAWSErr(err, iot.ErrCodeResourceNotFoundException, "") {
			log.Printf("[WARN] IoT Thing %q not found, removing from state", d.Id())
			d.SetId("")
		}
		return err
	}

	log.Printf("[DEBUG] Received IoT Thing: %s", out)

	d.Set("arn", out.ThingArn)
	d.Set("name", out.ThingName)
	d.Set("attributes", aws.StringValueMap(out.Attributes))
	d.Set("default_client_id", out.DefaultClientId)
	d.Set("thing_type_name", out.ThingTypeName)
	d.Set("version", out.Version)

	return nil
}

func resourceAwsIotThingUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iotconn

	params := &iot.UpdateThingInput{
		ThingName: aws.String(d.Get("name").(string)),
	}
	if d.HasChange("thing_type_name") {
		if v, ok := d.GetOk("thing_type_name"); ok {
			params.ThingTypeName = aws.String(v.(string))
		} else {
			params.RemoveThingType = aws.Bool(true)
		}
	}
	if d.HasChange("attributes") {
		attributes := map[string]*string{}

		if v, ok := d.GetOk("attributes"); ok {
			if m, ok := v.(map[string]interface{}); ok {
				attributes = stringMapToPointers(m)
			}
		}
		params.AttributePayload = &iot.AttributePayload{
			Attributes: attributes,
		}
	}

	_, err := conn.UpdateThing(params)
	if err != nil {
		return err
	}

	return resourceAwsIotThingRead(d, meta)
}

func resourceAwsIotThingDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iotconn

	params := &iot.DeleteThingInput{
		ThingName: aws.String(d.Id()),
	}
	log.Printf("[DEBUG] Deleting IoT Thing: %s", params)

	_, err := conn.DeleteThing(params)
	if err != nil {
		if isAWSErr(err, iot.ErrCodeResourceNotFoundException, "") {
			return nil
		}
		return err
	}

	return nil
}
