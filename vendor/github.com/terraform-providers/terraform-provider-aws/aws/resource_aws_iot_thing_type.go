package aws

import (
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iot"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

// https://docs.aws.amazon.com/iot/latest/apireference/API_CreateThingType.html
func resourceAwsIotThingType() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsIotThingTypeCreate,
		Read:   resourceAwsIotThingTypeRead,
		Update: resourceAwsIotThingTypeUpdate,
		Delete: resourceAwsIotThingTypeDelete,

		Importer: &schema.ResourceImporter{
			State: func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				d.Set("name", d.Id())
				return []*schema.ResourceData{d}, nil
			},
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateIotThingTypeName,
			},
			"properties": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"description": {
							Type:         schema.TypeString,
							Optional:     true,
							ForceNew:     true,
							ValidateFunc: validateIotThingTypeDescription,
						},
						"searchable_attributes": {
							Type:     schema.TypeSet,
							Optional: true,
							Computed: true,
							ForceNew: true,
							MaxItems: 3,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validateIotThingTypeSearchableAttribute,
							},
						},
					},
				},
			},
			"deprecated": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsIotThingTypeCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iotconn

	params := &iot.CreateThingTypeInput{
		ThingTypeName: aws.String(d.Get("name").(string)),
	}

	if v, ok := d.GetOk("properties"); ok {
		configs := v.([]interface{})
		config, ok := configs[0].(map[string]interface{})

		if ok && config != nil {
			params.ThingTypeProperties = expandIotThingTypeProperties(config)
		}
	}

	log.Printf("[DEBUG] Creating IoT Thing Type: %s", params)
	out, err := conn.CreateThingType(params)

	if err != nil {
		return err
	}

	d.SetId(*out.ThingTypeName)

	if v := d.Get("deprecated").(bool); v {
		params := &iot.DeprecateThingTypeInput{
			ThingTypeName: aws.String(d.Id()),
			UndoDeprecate: aws.Bool(false),
		}

		log.Printf("[DEBUG] Deprecating IoT Thing Type: %s", params)
		_, err := conn.DeprecateThingType(params)

		if err != nil {
			return err
		}
	}

	return resourceAwsIotThingTypeRead(d, meta)
}

func resourceAwsIotThingTypeRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iotconn

	params := &iot.DescribeThingTypeInput{
		ThingTypeName: aws.String(d.Id()),
	}
	log.Printf("[DEBUG] Reading IoT Thing Type: %s", params)
	out, err := conn.DescribeThingType(params)

	if err != nil {
		if isAWSErr(err, iot.ErrCodeResourceNotFoundException, "") {
			log.Printf("[WARN] IoT Thing Type %q not found, removing from state", d.Id())
			d.SetId("")
		}
		return err
	}

	if out.ThingTypeMetadata != nil {
		d.Set("deprecated", out.ThingTypeMetadata.Deprecated)
	}

	d.Set("arn", out.ThingTypeArn)
	d.Set("properties", flattenIotThingTypeProperties(out.ThingTypeProperties))

	return nil
}

func resourceAwsIotThingTypeUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iotconn

	if d.HasChange("deprecated") {
		params := &iot.DeprecateThingTypeInput{
			ThingTypeName: aws.String(d.Id()),
			UndoDeprecate: aws.Bool(!d.Get("deprecated").(bool)),
		}

		log.Printf("[DEBUG] Updating IoT Thing Type: %s", params)
		_, err := conn.DeprecateThingType(params)

		if err != nil {
			return err
		}
	}

	return resourceAwsIotThingTypeRead(d, meta)
}

func resourceAwsIotThingTypeDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iotconn

	// In order to delete an IoT Thing Type, you must deprecate it first and wait
	// at least 5 minutes.
	deprecateParams := &iot.DeprecateThingTypeInput{
		ThingTypeName: aws.String(d.Id()),
	}
	log.Printf("[DEBUG] Deprecating IoT Thing Type: %s", deprecateParams)
	_, err := conn.DeprecateThingType(deprecateParams)

	if err != nil {
		return err
	}

	deleteParams := &iot.DeleteThingTypeInput{
		ThingTypeName: aws.String(d.Id()),
	}
	log.Printf("[DEBUG] Deleting IoT Thing Type: %s", deleteParams)

	return resource.Retry(6*time.Minute, func() *resource.RetryError {
		_, err := conn.DeleteThingType(deleteParams)

		if err != nil {
			if isAWSErr(err, iot.ErrCodeInvalidRequestException, "Please wait for 5 minutes after deprecation and then retry") {
				return resource.RetryableError(err)
			}

			// As the delay post-deprecation is about 5 minutes, it may have been
			// deleted in between, thus getting a Not Found Exception.
			if isAWSErr(err, iot.ErrCodeResourceNotFoundException, "") {
				return nil
			}

			return resource.NonRetryableError(err)
		}

		return nil
	})
}
