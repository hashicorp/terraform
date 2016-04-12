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

func resourceAwsApiGatewaySwaggerApi() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsApiSwaggerApiCreate,
		Read:   resourceAwsApiSwaggerApiRead,
		Delete: resourceAwsApiSwaggerApiDelete,

		Schema: map[string]*schema.Schema{
			"swagger": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"failonwarnings": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
		},
	}
}

func resourceAwsApiGatewaySwaggerApiCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	req := &apigateway.ImportRestApiInput{
		Body: d.Get("swagger").([]byte),
	}

	if d.Get("failonwarnings") != nil {
		req.FailOnWarnings = aws.Bool(d.Get("failonwarnings").(bool))
	}

	res, err := conn.ImportRestApi(req)

	if err != nil {
		return err
	}

	for w := range res.Warnings {
		log.Printf("[WARN] Swagger import warning: %s", w)
	}
	d.SetId(*res.Id)

	return resourceAwsApiSwaggerApiRead(d, meta)
}

func resourceAwsApiSwaggerApiRead(d *schema.ResourceDta, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	api, err := conn.GetRestApi(&apigateway.GetRestApiInput{
		RestApiId: aws.String(d.Id()),
	})

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NotFoundException" {
			d.SetId("")
			return nil
		}
		return err
	}

	return nil
}

func resourceAwsApiGatewaySwaggerApiDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway
	log.Printf("[DEBUG] Deleting API Gateway: %s", d.Id())

	return resource.Retry(5*time.Minute, func() *resource.RetryError {
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
