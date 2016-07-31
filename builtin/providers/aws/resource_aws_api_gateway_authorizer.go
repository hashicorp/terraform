package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsApiGatewayAuthorizer() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsApiGatewayAuthorizerCreate,
		Read:   resourceAwsApiGatewayAuthorizerRead,
		Update: resourceAwsApiGatewayAuthorizerUpdate,
		Delete: resourceAwsApiGatewayAuthorizerDelete,

		Schema: map[string]*schema.Schema{
			"authorizer_uri": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"identity_source": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "method.request.header.Authorization",
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"rest_api_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "TOKEN",
			},
			"authorizer_credentials": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"authorizer_result_ttl_in_seconds": &schema.Schema{
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validateIntegerInRange(0, 3600),
			},
			"identity_validation_expression": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceAwsApiGatewayAuthorizerCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	input := apigateway.CreateAuthorizerInput{
		AuthorizerUri:  aws.String(d.Get("authorizer_uri").(string)),
		IdentitySource: aws.String(d.Get("identity_source").(string)),
		Name:           aws.String(d.Get("name").(string)),
		RestApiId:      aws.String(d.Get("rest_api_id").(string)),
		Type:           aws.String(d.Get("type").(string)),
	}

	if v, ok := d.GetOk("authorizer_credentials"); ok {
		input.AuthorizerCredentials = aws.String(v.(string))
	}
	if v, ok := d.GetOk("authorizer_result_ttl_in_seconds"); ok {
		input.AuthorizerResultTtlInSeconds = aws.Int64(int64(v.(int)))
	}
	if v, ok := d.GetOk("identity_validation_expression"); ok {
		input.IdentityValidationExpression = aws.String(v.(string))
	}

	log.Printf("[INFO] Creating API Gateway Authorizer: %s", input)
	out, err := conn.CreateAuthorizer(&input)
	if err != nil {
		return fmt.Errorf("Error creating API Gateway Authorizer: %s", err)
	}

	d.SetId(*out.Id)

	return resourceAwsApiGatewayAuthorizerRead(d, meta)
}

func resourceAwsApiGatewayAuthorizerRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	log.Printf("[INFO] Reading API Gateway Authorizer %s", d.Id())
	input := apigateway.GetAuthorizerInput{
		AuthorizerId: aws.String(d.Id()),
		RestApiId:    aws.String(d.Get("rest_api_id").(string)),
	}

	authorizer, err := conn.GetAuthorizer(&input)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NotFoundException" {
			log.Printf("[WARN] No API Gateway Authorizer found: %s", input)
			d.SetId("")
			return nil
		}
		return err
	}
	log.Printf("[DEBUG] Received API Gateway Authorizer: %s", authorizer)

	d.Set("authorizer_credentials", authorizer.AuthorizerCredentials)
	d.Set("authorizer_result_ttl_in_seconds", authorizer.AuthorizerResultTtlInSeconds)
	d.Set("authorizer_uri", authorizer.AuthorizerUri)
	d.Set("identity_source", authorizer.IdentitySource)
	d.Set("identity_validation_expression", authorizer.IdentityValidationExpression)
	d.Set("name", authorizer.Name)
	d.Set("type", authorizer.Type)

	return nil
}

func resourceAwsApiGatewayAuthorizerUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	input := apigateway.UpdateAuthorizerInput{
		AuthorizerId: aws.String(d.Id()),
		RestApiId:    aws.String(d.Get("rest_api_id").(string)),
	}

	operations := make([]*apigateway.PatchOperation, 0)

	if d.HasChange("authorizer_uri") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/authorizerUri"),
			Value: aws.String(d.Get("authorizer_uri").(string)),
		})
	}
	if d.HasChange("identity_source") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/identitySource"),
			Value: aws.String(d.Get("identity_source").(string)),
		})
	}
	if d.HasChange("name") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/name"),
			Value: aws.String(d.Get("name").(string)),
		})
	}
	if d.HasChange("type") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/type"),
			Value: aws.String(d.Get("type").(string)),
		})
	}
	if d.HasChange("authorizer_credentials") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/authorizerCredentials"),
			Value: aws.String(d.Get("authorizer_credentials").(string)),
		})
	}
	if d.HasChange("authorizer_result_ttl_in_seconds") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/authorizerResultTtlInSeconds"),
			Value: aws.String(fmt.Sprintf("%d", d.Get("authorizer_result_ttl_in_seconds").(int))),
		})
	}
	if d.HasChange("identity_validation_expression") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/identityValidationExpression"),
			Value: aws.String(d.Get("identity_validation_expression").(string)),
		})
	}
	input.PatchOperations = operations

	log.Printf("[INFO] Updating API Gateway Authorizer: %s", input)
	_, err := conn.UpdateAuthorizer(&input)
	if err != nil {
		return fmt.Errorf("Updating API Gateway Authorizer failed: %s", err)
	}

	return resourceAwsApiGatewayAuthorizerRead(d, meta)
}

func resourceAwsApiGatewayAuthorizerDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway
	input := apigateway.DeleteAuthorizerInput{
		AuthorizerId: aws.String(d.Id()),
		RestApiId:    aws.String(d.Get("rest_api_id").(string)),
	}
	log.Printf("[INFO] Deleting API Gateway Authorizer: %s", input)
	_, err := conn.DeleteAuthorizer(&input)
	if err != nil {
		return fmt.Errorf("Deleting API Gateway Authorizer failed: %s", err)
	}

	return nil
}
