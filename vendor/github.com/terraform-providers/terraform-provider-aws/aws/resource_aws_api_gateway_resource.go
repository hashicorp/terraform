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

func resourceAwsApiGatewayResource() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsApiGatewayResourceCreate,
		Read:   resourceAwsApiGatewayResourceRead,
		Update: resourceAwsApiGatewayResourceUpdate,
		Delete: resourceAwsApiGatewayResourceDelete,

		Schema: map[string]*schema.Schema{
			"rest_api_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"parent_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"path_part": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"path": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsApiGatewayResourceCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway
	log.Printf("[DEBUG] Creating API Gateway Resource for API %s", d.Get("rest_api_id").(string))

	var err error
	resource, err := conn.CreateResource(&apigateway.CreateResourceInput{
		ParentId:  aws.String(d.Get("parent_id").(string)),
		PathPart:  aws.String(d.Get("path_part").(string)),
		RestApiId: aws.String(d.Get("rest_api_id").(string)),
	})

	if err != nil {
		return fmt.Errorf("Error creating API Gateway Resource: %s", err)
	}

	d.SetId(*resource.Id)

	return resourceAwsApiGatewayResourceRead(d, meta)
}

func resourceAwsApiGatewayResourceRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	log.Printf("[DEBUG] Reading API Gateway Resource %s", d.Id())
	resource, err := conn.GetResource(&apigateway.GetResourceInput{
		ResourceId: aws.String(d.Id()),
		RestApiId:  aws.String(d.Get("rest_api_id").(string)),
	})

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NotFoundException" {
			log.Printf("[WARN] API Gateway Resource (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("parent_id", resource.ParentId)
	d.Set("path_part", resource.PathPart)
	d.Set("path", resource.Path)

	return nil
}

func resourceAwsApiGatewayResourceUpdateOperations(d *schema.ResourceData) []*apigateway.PatchOperation {
	operations := make([]*apigateway.PatchOperation, 0)
	if d.HasChange("path_part") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/pathPart"),
			Value: aws.String(d.Get("path_part").(string)),
		})
	}

	if d.HasChange("parent_id") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/parentId"),
			Value: aws.String(d.Get("parent_id").(string)),
		})
	}
	return operations
}

func resourceAwsApiGatewayResourceUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	log.Printf("[DEBUG] Updating API Gateway Resource %s", d.Id())
	_, err := conn.UpdateResource(&apigateway.UpdateResourceInput{
		ResourceId:      aws.String(d.Id()),
		RestApiId:       aws.String(d.Get("rest_api_id").(string)),
		PatchOperations: resourceAwsApiGatewayResourceUpdateOperations(d),
	})

	if err != nil {
		return err
	}

	return resourceAwsApiGatewayResourceRead(d, meta)
}

func resourceAwsApiGatewayResourceDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway
	log.Printf("[DEBUG] Deleting API Gateway Resource: %s", d.Id())

	return resource.Retry(5*time.Minute, func() *resource.RetryError {
		log.Printf("[DEBUG] schema is %#v", d)
		_, err := conn.DeleteResource(&apigateway.DeleteResourceInput{
			ResourceId: aws.String(d.Id()),
			RestApiId:  aws.String(d.Get("rest_api_id").(string)),
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
