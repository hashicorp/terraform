package aws

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsApiGatewayModel() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsApiGatewayModelCreate,
		Read:   resourceAwsApiGatewayModelRead,
		Update: resourceAwsApiGatewayModelUpdate,
		Delete: resourceAwsApiGatewayModelDelete,
		Importer: &schema.ResourceImporter{
			State: func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				idParts := strings.Split(d.Id(), "/")
				if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
					return nil, fmt.Errorf("Unexpected format of ID (%q), expected REST-API-ID/NAME", d.Id())
				}
				restApiID := idParts[0]
				name := idParts[1]
				d.Set("name", name)
				d.Set("rest_api_id", restApiID)

				conn := meta.(*AWSClient).apigateway

				output, err := conn.GetModel(&apigateway.GetModelInput{
					ModelName: aws.String(name),
					RestApiId: aws.String(restApiID),
				})

				if err != nil {
					return nil, err
				}

				d.SetId(aws.StringValue(output.Id))

				return []*schema.ResourceData{d}, nil
			},
		},

		Schema: map[string]*schema.Schema{
			"rest_api_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"schema": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"content_type": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsApiGatewayModelCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway
	log.Printf("[DEBUG] Creating API Gateway Model")

	var description *string
	if v, ok := d.GetOk("description"); ok {
		description = aws.String(v.(string))
	}
	var schema *string
	if v, ok := d.GetOk("schema"); ok {
		schema = aws.String(v.(string))
	}

	var err error
	model, err := conn.CreateModel(&apigateway.CreateModelInput{
		Name:        aws.String(d.Get("name").(string)),
		RestApiId:   aws.String(d.Get("rest_api_id").(string)),
		ContentType: aws.String(d.Get("content_type").(string)),

		Description: description,
		Schema:      schema,
	})

	if err != nil {
		return fmt.Errorf("Error creating API Gateway Model: %s", err)
	}

	d.SetId(*model.Id)

	return nil
}

func resourceAwsApiGatewayModelRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	log.Printf("[DEBUG] Reading API Gateway Model %s", d.Id())
	out, err := conn.GetModel(&apigateway.GetModelInput{
		ModelName: aws.String(d.Get("name").(string)),
		RestApiId: aws.String(d.Get("rest_api_id").(string)),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NotFoundException" {
			log.Printf("[WARN] API Gateway Model (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}
	log.Printf("[DEBUG] Received API Gateway Model: %s", out)

	d.Set("content_type", out.ContentType)
	d.Set("description", out.Description)
	d.Set("schema", out.Schema)

	return nil
}

func resourceAwsApiGatewayModelUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	log.Printf("[DEBUG] Reading API Gateway Model %s", d.Id())
	operations := make([]*apigateway.PatchOperation, 0)
	if d.HasChange("description") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/description"),
			Value: aws.String(d.Get("description").(string)),
		})
	}
	if d.HasChange("schema") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/schema"),
			Value: aws.String(d.Get("schema").(string)),
		})
	}

	out, err := conn.UpdateModel(&apigateway.UpdateModelInput{
		ModelName:       aws.String(d.Get("name").(string)),
		RestApiId:       aws.String(d.Get("rest_api_id").(string)),
		PatchOperations: operations,
	})
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] Received API Gateway Model: %s", out)

	return resourceAwsApiGatewayModelRead(d, meta)
}

func resourceAwsApiGatewayModelDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway
	log.Printf("[DEBUG] Deleting API Gateway Model: %s", d.Id())

	return resource.Retry(5*time.Minute, func() *resource.RetryError {
		log.Printf("[DEBUG] schema is %#v", d)
		_, err := conn.DeleteModel(&apigateway.DeleteModelInput{
			ModelName: aws.String(d.Get("name").(string)),
			RestApiId: aws.String(d.Get("rest_api_id").(string)),
		})
		if err == nil {
			return nil
		}

		apigatewayErr, ok := err.(awserr.Error)
		if apigatewayErr.Code() == "NotFoundException" {
			return nil
		}

		if !ok {
			return resource.NonRetryableError(err)
		}

		return resource.NonRetryableError(err)
	})
}
