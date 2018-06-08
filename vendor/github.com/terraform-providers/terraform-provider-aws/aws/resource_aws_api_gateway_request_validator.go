package aws

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
	"strings"
)

func resourceAwsApiGatewayRequestValidator() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsApiGatewayRequestValidatorCreate,
		Read:   resourceAwsApiGatewayRequestValidatorRead,
		Update: resourceAwsApiGatewayRequestValidatorUpdate,
		Delete: resourceAwsApiGatewayRequestValidatorDelete,

		Schema: map[string]*schema.Schema{
			"rest_api_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"validate_request_body": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"validate_request_parameters": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
	}
}

func resourceAwsApiGatewayRequestValidatorCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	input := apigateway.CreateRequestValidatorInput{
		Name:                      aws.String(d.Get("name").(string)),
		RestApiId:                 aws.String(d.Get("rest_api_id").(string)),
		ValidateRequestBody:       aws.Bool(d.Get("validate_request_body").(bool)),
		ValidateRequestParameters: aws.Bool(d.Get("validate_request_parameters").(bool)),
	}

	out, err := conn.CreateRequestValidator(&input)
	if err != nil {
		return fmt.Errorf("Error creating Request Validator: %s", err)
	}

	d.SetId(*out.Id)

	return nil
}

func resourceAwsApiGatewayRequestValidatorRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	input := apigateway.GetRequestValidatorInput{
		RequestValidatorId: aws.String(d.Id()),
		RestApiId:          aws.String(d.Get("rest_api_id").(string)),
	}

	out, err := conn.GetRequestValidator(&input)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == apigateway.ErrCodeNotFoundException {
			log.Printf("[WARN] API Gateway Request Validator (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("name", out.Name)
	d.Set("validate_request_body", out.ValidateRequestBody)
	d.Set("validate_request_parameters", out.ValidateRequestParameters)

	return nil
}

func resourceAwsApiGatewayRequestValidatorUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway
	log.Printf("[DEBUG] Updating Request Validator %s", d.Id())

	operations := make([]*apigateway.PatchOperation, 0)

	if d.HasChange("name") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/name"),
			Value: aws.String(d.Get("name").(string)),
		})
	}

	if d.HasChange("validate_request_body") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/validateRequestBody"),
			Value: aws.String(fmt.Sprintf("%t", d.Get("validate_request_body").(bool))),
		})
	}

	if d.HasChange("validate_request_parameters") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/validateRequestParameters"),
			Value: aws.String(fmt.Sprintf("%t", d.Get("validate_request_parameters").(bool))),
		})
	}

	input := apigateway.UpdateRequestValidatorInput{
		RequestValidatorId: aws.String(d.Id()),
		RestApiId:          aws.String(d.Get("rest_api_id").(string)),
		PatchOperations:    operations,
	}

	_, err := conn.UpdateRequestValidator(&input)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Updated Request Validator %s", d.Id())

	return resourceAwsApiGatewayRequestValidatorRead(d, meta)
}

func resourceAwsApiGatewayRequestValidatorDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway
	log.Printf("[DEBUG] Deleting Request Validator %s", d.Id())

	_, err := conn.DeleteRequestValidator(&apigateway.DeleteRequestValidatorInput{
		RequestValidatorId: aws.String(d.Id()),
		RestApiId:          aws.String(d.Get("rest_api_id").(string)),
	})
	if err != nil {
		// XXX: Figure out a way to delete the method that depends on the request validator first
		// otherwise the validator will be dangling until the API is deleted
		if !strings.Contains(err.Error(), apigateway.ErrCodeConflictException) {
			return fmt.Errorf("Deleting Request Validator failed: %s", err)
		}
	}

	return nil
}
