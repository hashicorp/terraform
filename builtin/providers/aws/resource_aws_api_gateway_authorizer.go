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

func resourceAwsApiGatewayAuthorizer() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsApiGatewayAuthorizerCreate,
		Read:   resourceAwsApiGatewayAuthorizerRead,
		Update: resourceAwsApiGatewayAuthorizerUpdate,
		Delete: resourceAwsApiGatewayAuthorizerDelete,

		Schema: map[string]*schema.Schema{

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"authorizer_uri": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"identity_source": &schema.Schema{
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
				Required: true,
				ForceNew: true,
			},

			"credentials": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"result_in_ttl": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
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
	// Create the gateway
	log.Printf("[DEBUG] Creating API Gateway Authorizer")

	var credentials *string
	if v, ok := d.GetOk("credentials"); ok {
		credentials = aws.String(v.(string))
	}

	var result_in_ttl *int64
	if v, ok := d.GetOk("result_in_ttl"); ok {
		result_in_ttl = aws.Int64(int64(v.(int)))
	}

	var identity_validation_expression *string
	if v, ok := d.GetOk("identity_validation_expression"); ok {
		identity_validation_expression = aws.String(v.(string))
	}

	var err error
	authorizer, err := conn.CreateAuthorizer(&apigateway.CreateAuthorizerInput{
		AuthorizerUri:  aws.String(d.Get("authorizer_uri").(string)),
		IdentitySource: aws.String(d.Get("identity_source").(string)),
		Name:           aws.String(d.Get("name").(string)),
		RestApiId:      aws.String(d.Get("rest_api_id").(string)),
		Type:           aws.String(d.Get("type").(string)),

		AuthorizerCredentials:        credentials,
		AuthorizerResultTtlInSeconds: result_in_ttl,
		IdentityValidationExpression: identity_validation_expression,
	})
	if err != nil {
		return fmt.Errorf("Error creating API Gateway Authorizer: %s", err)
	}

	d.SetId(*authorizer.Id)
	log.Printf("[DEBUG] API Gateway Authorizer ID: %s", d.Id())

	return nil
}

func resourceAwsApiGatewayAuthorizerRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	log.Printf("[DEBUG] Reading API Gateway Deployment %s", d.Id())
	out, err := conn.GetAuthorizer(&apigateway.GetAuthorizerInput{
		RestApiId:    aws.String(d.Get("rest_api_id").(string)),
		AuthorizerId: aws.String(d.Id()),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NotFoundException" {
			d.SetId("")
			return nil
		}
		return err
	}
	log.Printf("[DEBUG] Received API Gateway Authorizer: %s", out)
	d.SetId(*out.Id)

	return nil
}

func resourceAwsApiGatewayAuthorizerUpdateOperations(d *schema.ResourceData) []*apigateway.PatchOperation {
	operations := make([]*apigateway.PatchOperation, 0)

	if d.HasChange("description") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/description"),
			Value: aws.String(d.Get("description").(string)),
		})
	}

	return operations
}

func resourceAwsApiGatewayAuthorizerUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	log.Printf("[DEBUG] Updating API Gateway API Key: %s", d.Id())

	_, err := conn.UpdateAuthorizer(&apigateway.UpdateAuthorizerInput{
		AuthorizerId:    aws.String(d.Id()),
		RestApiId:       aws.String(d.Get("rest_api_id").(string)),
		PatchOperations: resourceAwsApiGatewayAuthorizerUpdateOperations(d),
	})
	if err != nil {
		return err
	}

	return resourceAwsApiGatewayAuthorizerRead(d, meta)
}

func resourceAwsApiGatewayAuthorizerDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway
	log.Printf("[DEBUG] Deleting API Gateway Authorizer: %s", d.Id())

	return resource.Retry(5*time.Minute, func() *resource.RetryError {
		log.Printf("[DEBUG] schema is %#v", d)
		_, err := conn.DeleteAuthorizer(&apigateway.DeleteAuthorizerInput{
			AuthorizerId: aws.String(d.Id()),
			RestApiId:    aws.String(d.Get("rest_api_id").(string)),
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
