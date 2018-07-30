package aws

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsApiGatewayRestApi() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsApiGatewayRestApiCreate,
		Read:   resourceAwsApiGatewayRestApiRead,
		Update: resourceAwsApiGatewayRestApiUpdate,
		Delete: resourceAwsApiGatewayRestApiDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"binary_media_types": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"body": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"minimum_compression_size": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      -1,
				ValidateFunc: validateIntegerInRange(-1, 10485760),
			},

			"root_resource_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"created_date": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsApiGatewayRestApiCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway
	log.Printf("[DEBUG] Creating API Gateway")

	var description *string
	if d.Get("description").(string) != "" {
		description = aws.String(d.Get("description").(string))
	}

	params := &apigateway.CreateRestApiInput{
		Name:        aws.String(d.Get("name").(string)),
		Description: description,
	}

	binaryMediaTypes, binaryMediaTypesOk := d.GetOk("binary_media_types")
	if binaryMediaTypesOk {
		params.BinaryMediaTypes = expandStringList(binaryMediaTypes.([]interface{}))
	}

	minimumCompressionSize := d.Get("minimum_compression_size").(int)
	if minimumCompressionSize > -1 {
		params.MinimumCompressionSize = aws.Int64(int64(minimumCompressionSize))
	}

	gateway, err := conn.CreateRestApi(params)
	if err != nil {
		return fmt.Errorf("Error creating API Gateway: %s", err)
	}

	d.SetId(*gateway.Id)

	if body, ok := d.GetOk("body"); ok {
		log.Printf("[DEBUG] Initializing API Gateway from OpenAPI spec %s", d.Id())
		_, err := conn.PutRestApi(&apigateway.PutRestApiInput{
			RestApiId: gateway.Id,
			Mode:      aws.String(apigateway.PutModeOverwrite),
			Body:      []byte(body.(string)),
		})
		if err != nil {
			return errwrap.Wrapf("Error creating API Gateway specification: {{err}}", err)
		}
	}

	if err = resourceAwsApiGatewayRestApiRefreshResources(d, meta); err != nil {
		return err
	}

	return resourceAwsApiGatewayRestApiRead(d, meta)
}

func resourceAwsApiGatewayRestApiRefreshResources(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	resp, err := conn.GetResources(&apigateway.GetResourcesInput{
		RestApiId: aws.String(d.Id()),
	})
	if err != nil {
		return err
	}

	for _, item := range resp.Items {
		if *item.Path == "/" {
			d.Set("root_resource_id", item.Id)
			break
		}
	}

	return nil
}

func resourceAwsApiGatewayRestApiRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway
	log.Printf("[DEBUG] Reading API Gateway %s", d.Id())

	api, err := conn.GetRestApi(&apigateway.GetRestApiInput{
		RestApiId: aws.String(d.Id()),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NotFoundException" {
			log.Printf("[WARN] API Gateway (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("name", api.Name)
	d.Set("description", api.Description)
	d.Set("binary_media_types", api.BinaryMediaTypes)
	if api.MinimumCompressionSize == nil {
		d.Set("minimum_compression_size", -1)
	} else {
		d.Set("minimum_compression_size", api.MinimumCompressionSize)
	}
	if err := d.Set("created_date", api.CreatedDate.Format(time.RFC3339)); err != nil {
		log.Printf("[DEBUG] Error setting created_date: %s", err)
	}

	return nil
}

func resourceAwsApiGatewayRestApiUpdateOperations(d *schema.ResourceData) []*apigateway.PatchOperation {
	operations := make([]*apigateway.PatchOperation, 0)

	if d.HasChange("name") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/name"),
			Value: aws.String(d.Get("name").(string)),
		})
	}

	if d.HasChange("description") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/description"),
			Value: aws.String(d.Get("description").(string)),
		})
	}

	if d.HasChange("minimum_compression_size") {
		minimumCompressionSize := d.Get("minimum_compression_size").(int)
		var value string
		if minimumCompressionSize > -1 {
			value = strconv.Itoa(minimumCompressionSize)
		}
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/minimumCompressionSize"),
			Value: aws.String(value),
		})
	}

	if d.HasChange("binary_media_types") {
		o, n := d.GetChange("binary_media_types")
		prefix := "binaryMediaTypes"

		old := o.([]interface{})
		new := n.([]interface{})

		// Remove every binary media types. Simpler to remove and add new ones,
		// since there are no replacings.
		for _, v := range old {
			operations = append(operations, &apigateway.PatchOperation{
				Op:   aws.String("remove"),
				Path: aws.String(fmt.Sprintf("/%s/%s", prefix, escapeJsonPointer(v.(string)))),
			})
		}

		// Handle additions
		if len(new) > 0 {
			for _, v := range new {
				operations = append(operations, &apigateway.PatchOperation{
					Op:   aws.String("add"),
					Path: aws.String(fmt.Sprintf("/%s/%s", prefix, escapeJsonPointer(v.(string)))),
				})
			}
		}
	}

	return operations
}

func resourceAwsApiGatewayRestApiUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway
	log.Printf("[DEBUG] Updating API Gateway %s", d.Id())

	if d.HasChange("body") {
		if body, ok := d.GetOk("body"); ok {
			log.Printf("[DEBUG] Updating API Gateway from OpenAPI spec: %s", d.Id())
			_, err := conn.PutRestApi(&apigateway.PutRestApiInput{
				RestApiId: aws.String(d.Id()),
				Mode:      aws.String(apigateway.PutModeOverwrite),
				Body:      []byte(body.(string)),
			})
			if err != nil {
				return errwrap.Wrapf("Error updating API Gateway specification: {{err}}", err)
			}
		}
	}

	_, err := conn.UpdateRestApi(&apigateway.UpdateRestApiInput{
		RestApiId:       aws.String(d.Id()),
		PatchOperations: resourceAwsApiGatewayRestApiUpdateOperations(d),
	})

	if err != nil {
		return err
	}
	log.Printf("[DEBUG] Updated API Gateway %s", d.Id())

	return resourceAwsApiGatewayRestApiRead(d, meta)
}

func resourceAwsApiGatewayRestApiDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway
	log.Printf("[DEBUG] Deleting API Gateway: %s", d.Id())

	return resource.Retry(10*time.Minute, func() *resource.RetryError {
		_, err := conn.DeleteRestApi(&apigateway.DeleteRestApiInput{
			RestApiId: aws.String(d.Id()),
		})
		if err == nil {
			return nil
		}

		if apigatewayErr, ok := err.(awserr.Error); ok && apigatewayErr.Code() == "NotFoundException" {
			return nil
		}

		return resource.NonRetryableError(err)
	})
}
