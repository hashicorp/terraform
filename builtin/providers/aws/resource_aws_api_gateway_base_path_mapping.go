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
			"base_path": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"stage_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
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
		_, err := conn.CreateBasePathMapping(&apigateway.CreateBasePathMappingInput{
			RestApiId:  aws.String(d.Get("api_id").(string)),
			DomainName: aws.String(d.Get("domain_name").(string)),
			BasePath:   aws.String(d.Get("base_path").(string)),
			Stage:      aws.String(d.Get("stage_name").(string)),
		})

		if err != nil {
			if err, ok := err.(awserr.Error); ok && err.Code() != "BadRequestException" {
				return resource.NonRetryableError(err)
			}

			return resource.RetryableError(
				fmt.Errorf("Error creating Gateway base path mapping: %s", err),
			)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("Error creating Gateway base path mapping: %s", err)
	}

	id := fmt.Sprintf("%s/%s", d.Get("domain_name").(string), d.Get("base_path").(string))
	d.SetId(id)

	return resourceAwsApiGatewayBasePathMappingRead(d, meta)
}

func resourceAwsApiGatewayBasePathMappingRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	domainName := d.Get("domain_name").(string)
	basePath := d.Get("base_path").(string)

	if domainName == "" {
		return nil
	}

	mapping, err := conn.GetBasePathMapping(&apigateway.GetBasePathMappingInput{
		DomainName: aws.String(domainName),
		BasePath:   aws.String(basePath),
	})
	if err != nil {
		if err, ok := err.(awserr.Error); ok && err.Code() == "NotFoundException" {
			log.Printf("[WARN] API gateway base path mapping %s has vanished\n", d.Id())
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error reading Gateway base path mapping: %s", err)
	}

	d.Set("base_path", mapping.BasePath)
	d.Set("api_id", mapping.RestApiId)
	d.Set("stage_name", mapping.Stage)

	return nil
}

func resourceAwsApiGatewayBasePathMappingDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	_, err := conn.DeleteBasePathMapping(&apigateway.DeleteBasePathMappingInput{
		DomainName: aws.String(d.Get("domain_name").(string)),
		BasePath:   aws.String(d.Get("base_path").(string)),
	})

	if err != nil {
		if err, ok := err.(awserr.Error); ok && err.Code() == "NotFoundException" {
			return nil
		}

		return err
	}

	return nil
}
