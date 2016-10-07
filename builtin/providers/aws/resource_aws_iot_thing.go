package aws

import (
	"log"

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
			"principals": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Set: schema.HashString,
			},
			"attributes": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
			},
			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsIotThingCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iotconn

	thingName := d.Get("name").(string)

	attributes := make(map[string]*string)

	if attrs, ok := d.GetOk("attributes"); ok {
		for k, v := range attrs.(map[string]interface{}) {
			attributes[k] = new(string)
			*attributes[k] = v.(string)
		}
	}

	params := &iot.CreateThingInput{
		ThingName: aws.String(thingName), // Required
		AttributePayload: &iot.AttributePayload{
			Attributes: attributes,
		},
	}

	log.Printf("[DEBUG] Creating IoT thing %s", thingName)
	out, err := conn.CreateThing(params)

	if err != nil {
		log.Printf("[ERROR] %s", err)
		return err
	}

	log.Printf("[DEBUG] IoT thing %s created", *out.ThingArn)

	if principals, ok := d.GetOk("principals"); ok {
		for _, p := range principals.(*schema.Set).List() {
			_, err := conn.AttachThingPrincipal(&iot.AttachThingPrincipalInput{
				ThingName: aws.String(thingName),
				Principal: aws.String(p.(string)),
			})
			if err != nil {
				log.Printf("[ERROR] %s", err)
				return err
			}
		}
	}

	d.SetId(*out.ThingName)
	d.Set("name", *out.ThingName)
	d.Set("arn", *out.ThingArn)

	return nil
}

func resourceAwsIotThingRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iotconn

	thingName := d.Get("name").(string)
	params := &iot.DescribeThingInput{
		ThingName: aws.String(thingName), // Required
	}
	log.Printf("[DEBUG] Reading IoT thing %s", thingName)
	out, err := conn.DescribeThing(params)

	if err != nil {
		return err
	}
	log.Printf("[DEBUG] Received IoT thing: %s", out.ThingName)

	d.SetId(*out.ThingName)
	d.Set("attributes", aws.StringValueMap(out.Attributes))

	return nil
}

func resourceAwsIotThingUpdate(d *schema.ResourceData, meta interface{}) error {

	conn := meta.(*AWSClient).iotconn

	thingName := d.Get("name").(string)

	if d.HasChange("attributes") {
		attributes := make(map[string]*string)

		if attrs, ok := d.GetOk("attributes"); ok {
			for k, v := range attrs.(map[string]interface{}) {
				attributes[k] = new(string)
				*attributes[k] = v.(string)
			}
		}
		params := &iot.UpdateThingInput{
			AttributePayload: &iot.AttributePayload{
				Attributes: attributes,
			},
			ThingName: aws.String(thingName),
		}
		_, err := conn.UpdateThing(params)

		if err != nil {
			log.Printf("[ERROR] %v", err)
			return err
		}
	}

	if d.HasChange("principals") {
		err := updatePrincipals(conn, d)
		if err != nil {
			log.Printf("[ERROR] %v", err)
			return err
		}
	}

	return resourceAwsIotThingRead(d, meta)
}

func updatePrincipals(conn *iot.IoT, d *schema.ResourceData) error {
	o, n := d.GetChange("principals")
	if o == nil {
		o = new(schema.Set)
	}
	if n == nil {
		n = new(schema.Set)
	}
	os := o.(*schema.Set)
	ns := n.(*schema.Set)

	toBeDetached := expandStringList(os.Difference(ns).List())
	toBeAttached := expandStringList(ns.Difference(os).List())

	thingName := d.Get("name").(string)
	for _, p := range toBeDetached {
		_, err := conn.DetachThingPrincipal(&iot.DetachThingPrincipalInput{
			Principal: aws.String(*p),
			ThingName: aws.String(thingName),
		})
		if err != nil {
			return err
		}
	}

	for _, p := range toBeAttached {
		_, err := conn.AttachThingPrincipal(&iot.AttachThingPrincipalInput{
			Principal: aws.String(*p),
			ThingName: aws.String(thingName),
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func resourceAwsIotThingDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iotconn

	thingName := aws.String(d.Get("name").(string))

	if principals, ok := d.GetOk("principals"); ok {
		for _, p := range principals.(*schema.Set).List() {
			_, err := conn.DetachThingPrincipal(&iot.DetachThingPrincipalInput{
				ThingName: thingName,
				Principal: aws.String(p.(string)),
			})
			if err != nil {
				return err
			}
		}
	}

	params := &iot.DeleteThingInput{
		ThingName: thingName, // Required
	}
	_, err := conn.DeleteThing(params)

	if err != nil {
		return err
	}

	return nil
}
