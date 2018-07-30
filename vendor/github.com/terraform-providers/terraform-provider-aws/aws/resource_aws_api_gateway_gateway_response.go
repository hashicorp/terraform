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

func resourceAwsApiGatewayGatewayResponse() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsApiGatewayGatewayResponsePut,
		Read:   resourceAwsApiGatewayGatewayResponseRead,
		Update: resourceAwsApiGatewayGatewayResponsePut,
		Delete: resourceAwsApiGatewayGatewayResponseDelete,

		Schema: map[string]*schema.Schema{
			"rest_api_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"response_type": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"status_code": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"response_templates": {
				Type:     schema.TypeMap,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},

			"response_parameters": {
				Type:     schema.TypeMap,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},
		},
	}
}

func resourceAwsApiGatewayGatewayResponsePut(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	templates := make(map[string]string)
	if kv, ok := d.GetOk("response_templates"); ok {
		for k, v := range kv.(map[string]interface{}) {
			templates[k] = v.(string)
		}
	}

	parameters := make(map[string]string)
	if kv, ok := d.GetOk("response_parameters"); ok {
		for k, v := range kv.(map[string]interface{}) {
			parameters[k] = v.(string)
		}
	}

	input := apigateway.PutGatewayResponseInput{
		RestApiId:          aws.String(d.Get("rest_api_id").(string)),
		ResponseType:       aws.String(d.Get("response_type").(string)),
		ResponseTemplates:  aws.StringMap(templates),
		ResponseParameters: aws.StringMap(parameters),
	}

	if v, ok := d.GetOk("status_code"); ok {
		input.StatusCode = aws.String(v.(string))
	}

	log.Printf("[DEBUG] Putting API Gateway Gateway Response: %s", input)

	_, err := conn.PutGatewayResponse(&input)
	if err != nil {
		return fmt.Errorf("Error putting API Gateway Gateway Response: %s", err)
	}

	d.SetId(fmt.Sprintf("aggr-%s-%s", d.Get("rest_api_id").(string), d.Get("response_type").(string)))
	log.Printf("[DEBUG] API Gateway Gateway Response put (%q)", d.Id())

	return resourceAwsApiGatewayGatewayResponseRead(d, meta)
}

func resourceAwsApiGatewayGatewayResponseRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	log.Printf("[DEBUG] Reading API Gateway Gateway Response %s", d.Id())
	gatewayResponse, err := conn.GetGatewayResponse(&apigateway.GetGatewayResponseInput{
		RestApiId:    aws.String(d.Get("rest_api_id").(string)),
		ResponseType: aws.String(d.Get("response_type").(string)),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NotFoundException" {
			log.Printf("[WARN] API Gateway Gateway Response (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	log.Printf("[DEBUG] Received API Gateway Gateway Response: %s", gatewayResponse)

	d.Set("response_type", gatewayResponse.ResponseType)
	d.Set("status_code", gatewayResponse.StatusCode)
	d.Set("response_templates", aws.StringValueMap(gatewayResponse.ResponseTemplates))
	d.Set("response_parameters", aws.StringValueMap(gatewayResponse.ResponseParameters))

	return nil
}

func resourceAwsApiGatewayGatewayResponseDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway
	log.Printf("[DEBUG] Deleting API Gateway Gateway Response: %s", d.Id())

	return resource.Retry(1*time.Minute, func() *resource.RetryError {
		_, err := conn.DeleteGatewayResponse(&apigateway.DeleteGatewayResponseInput{
			RestApiId:    aws.String(d.Get("rest_api_id").(string)),
			ResponseType: aws.String(d.Get("response_type").(string)),
		})

		if err == nil {
			return nil
		}

		apigatewayErr, ok := err.(awserr.Error)

		if ok && apigatewayErr.Code() == "NotFoundException" {
			return nil
		}

		return resource.NonRetryableError(err)
	})
}
