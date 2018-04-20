package aws

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsApiGatewayMethod() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsApiGatewayMethodCreate,
		Read:   resourceAwsApiGatewayMethodRead,
		Update: resourceAwsApiGatewayMethodUpdate,
		Delete: resourceAwsApiGatewayMethodDelete,

		Schema: map[string]*schema.Schema{
			"rest_api_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"resource_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"http_method": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateHTTPMethod(),
			},

			"authorization": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"authorizer_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"api_key_required": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"request_models": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				Elem:     schema.TypeString,
			},

			"request_parameters": &schema.Schema{
				Type:          schema.TypeMap,
				Elem:          schema.TypeBool,
				Optional:      true,
				ConflictsWith: []string{"request_parameters_in_json"},
			},

			"request_parameters_in_json": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"request_parameters"},
				Deprecated:    "Use field request_parameters instead",
			},

			"request_validator_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceAwsApiGatewayMethodCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	input := apigateway.PutMethodInput{
		AuthorizationType: aws.String(d.Get("authorization").(string)),
		HttpMethod:        aws.String(d.Get("http_method").(string)),
		ResourceId:        aws.String(d.Get("resource_id").(string)),
		RestApiId:         aws.String(d.Get("rest_api_id").(string)),
		ApiKeyRequired:    aws.Bool(d.Get("api_key_required").(bool)),
	}

	models := make(map[string]string)
	for k, v := range d.Get("request_models").(map[string]interface{}) {
		models[k] = v.(string)
	}
	if len(models) > 0 {
		input.RequestModels = aws.StringMap(models)
	}

	parameters := make(map[string]bool)
	if kv, ok := d.GetOk("request_parameters"); ok {
		for k, v := range kv.(map[string]interface{}) {
			parameters[k], ok = v.(bool)
			if !ok {
				value, _ := strconv.ParseBool(v.(string))
				parameters[k] = value
			}
		}
		input.RequestParameters = aws.BoolMap(parameters)
	}
	if v, ok := d.GetOk("request_parameters_in_json"); ok {
		if err := json.Unmarshal([]byte(v.(string)), &parameters); err != nil {
			return fmt.Errorf("Error unmarshaling request_parameters_in_json: %s", err)
		}
		input.RequestParameters = aws.BoolMap(parameters)
	}

	if v, ok := d.GetOk("authorizer_id"); ok {
		input.AuthorizerId = aws.String(v.(string))
	}

	if v, ok := d.GetOk("request_validator_id"); ok {
		input.RequestValidatorId = aws.String(v.(string))
	}

	_, err := conn.PutMethod(&input)
	if err != nil {
		return fmt.Errorf("Error creating API Gateway Method: %s", err)
	}

	d.SetId(fmt.Sprintf("agm-%s-%s-%s", d.Get("rest_api_id").(string), d.Get("resource_id").(string), d.Get("http_method").(string)))
	log.Printf("[DEBUG] API Gateway Method ID: %s", d.Id())

	return nil
}

func resourceAwsApiGatewayMethodRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	log.Printf("[DEBUG] Reading API Gateway Method %s", d.Id())
	out, err := conn.GetMethod(&apigateway.GetMethodInput{
		HttpMethod: aws.String(d.Get("http_method").(string)),
		ResourceId: aws.String(d.Get("resource_id").(string)),
		RestApiId:  aws.String(d.Get("rest_api_id").(string)),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NotFoundException" {
			log.Printf("[WARN] API Gateway Method (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}
	log.Printf("[DEBUG] Received API Gateway Method: %s", out)
	d.SetId(fmt.Sprintf("agm-%s-%s-%s", d.Get("rest_api_id").(string), d.Get("resource_id").(string), d.Get("http_method").(string)))
	d.Set("request_parameters", aws.BoolValueMap(out.RequestParameters))
	d.Set("request_parameters_in_json", aws.BoolValueMap(out.RequestParameters))
	d.Set("api_key_required", out.ApiKeyRequired)
	d.Set("authorization", out.AuthorizationType)
	d.Set("authorizer_id", out.AuthorizerId)
	d.Set("request_models", aws.StringValueMap(out.RequestModels))
	d.Set("request_validator_id", out.RequestValidatorId)

	return nil
}

func resourceAwsApiGatewayMethodUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	log.Printf("[DEBUG] Reading API Gateway Method %s", d.Id())
	operations := make([]*apigateway.PatchOperation, 0)
	if d.HasChange("resource_id") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/resourceId"),
			Value: aws.String(d.Get("resource_id").(string)),
		})
	}

	if d.HasChange("request_models") {
		operations = append(operations, expandApiGatewayRequestResponseModelOperations(d, "request_models", "requestModels")...)
	}

	if d.HasChange("request_parameters_in_json") {
		ops, err := deprecatedExpandApiGatewayMethodParametersJSONOperations(d, "request_parameters_in_json", "requestParameters")
		if err != nil {
			return err
		}
		operations = append(operations, ops...)
	}

	if d.HasChange("request_parameters") {
		parameters := make(map[string]bool)
		var ok bool
		for k, v := range d.Get("request_parameters").(map[string]interface{}) {
			parameters[k], ok = v.(bool)
			if !ok {
				value, _ := strconv.ParseBool(v.(string))
				parameters[k] = value
			}
		}
		ops, err := expandApiGatewayMethodParametersOperations(d, "request_parameters", "requestParameters")
		if err != nil {
			return err
		}
		operations = append(operations, ops...)
	}

	if d.HasChange("authorization") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/authorizationType"),
			Value: aws.String(d.Get("authorization").(string)),
		})
	}

	if d.HasChange("authorizer_id") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/authorizerId"),
			Value: aws.String(d.Get("authorizer_id").(string)),
		})
	}

	if d.HasChange("api_key_required") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/apiKeyRequired"),
			Value: aws.String(fmt.Sprintf("%t", d.Get("api_key_required").(bool))),
		})
	}

	if d.HasChange("request_validator_id") {
		var request_validator_id *string
		if v, ok := d.GetOk("request_validator_id"); ok {
			// requestValidatorId cannot be an empty string; it must either be nil
			// or it must have some value. Otherwise, updating fails.
			if s := v.(string); len(s) > 0 {
				request_validator_id = &s
			}
		}
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/requestValidatorId"),
			Value: request_validator_id,
		})
	}

	method, err := conn.UpdateMethod(&apigateway.UpdateMethodInput{
		HttpMethod:      aws.String(d.Get("http_method").(string)),
		ResourceId:      aws.String(d.Get("resource_id").(string)),
		RestApiId:       aws.String(d.Get("rest_api_id").(string)),
		PatchOperations: operations,
	})

	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Received API Gateway Method: %s", method)

	return resourceAwsApiGatewayMethodRead(d, meta)
}

func resourceAwsApiGatewayMethodDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway
	log.Printf("[DEBUG] Deleting API Gateway Method: %s", d.Id())

	return resource.Retry(5*time.Minute, func() *resource.RetryError {
		_, err := conn.DeleteMethod(&apigateway.DeleteMethodInput{
			HttpMethod: aws.String(d.Get("http_method").(string)),
			ResourceId: aws.String(d.Get("resource_id").(string)),
			RestApiId:  aws.String(d.Get("rest_api_id").(string)),
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
