package aws

import (
	"fmt"
	"log"
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

		Schema: map[string]*schema.Schema{
			"rest_api_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"schema": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"content_type": &schema.Schema{
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
			d.SetId("")
			return nil
		}
		return err
	}
	log.Printf("[DEBUG] Received API Gateway Model: %s", out)
	d.SetId(*out.Id)
	d.Set("description", out.Description)
	d.Set("schema", out.Schema)
	d.Set("content_type", out.ContentType)

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
