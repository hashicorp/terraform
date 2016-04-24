package aws

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsApiGatewayBasePathMapping() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsApiGatewayBasePathMappingCreate,
		Read:   resourceAwsApiGatewayBasePathMappingRead,
		Delete: resourceAwsApiGatewayBasePathMappingDelete,

		Schema: map[string]*schema.Schema{
			"api_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"path": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"stage": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"domain_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsApiGatewayBasePathMappingCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	err := resource.Retry(30*time.Second, func() *resource.RetryError {
		r, err := conn.CreateBasePathMapping(&apigateway.CreateBasePathMappingInput{
			RestApiId:  aws.String(d.Get("api_id").(string)),
			DomainName: aws.String(d.Get("domain_name").(string)),
			BasePath:   aws.String(d.Get("path").(string)),
			Stage:      aws.String(d.Get("stage").(string)),
		})

		if err != nil {
			if err, ok := err.(awserr.Error); ok && err.Code() != "BadRequestException" {
				return &resource.RetryError{
					Err:       err,
					Retryable: false,
				}
			}

			return &resource.RetryError{
				Err:       fmt.Errorf("Error creating Gateway base path mapping: %s", err),
				Retryable: true,
			}
		}

		id := fmt.Sprintf("apigateway-base-path-mapping-%s-%s", r.BasePath, d.Get("domain_name"))

		d.SetId(id)

		return nil
	})

	if err != nil {
		return fmt.Errorf("Error creating Gateway base path mapping: %s", err)
	}

	return resourceAwsApiGatewayBasePathMappingRead(d, meta)
}

func resourceAwsApiGatewayBasePathMappingRead(d *schema.ResourceData, meta interface{}) error {

	return nil
}

func resourceAwsApiGatewayBasePathMappingDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	_, err := conn.DeleteBasePathMapping(&apigateway.DeleteBasePathMappingInput{
		DomainName: aws.String(d.Get("domain_name").(string)),
		BasePath:   aws.String(d.Get("path").(string)),
	})

	if err != nil {
		if err, ok := err.(awserr.Error); ok && err.Code() == "NotFoundException" {
			return nil
		}
		return nil
	}

	return nil
}
