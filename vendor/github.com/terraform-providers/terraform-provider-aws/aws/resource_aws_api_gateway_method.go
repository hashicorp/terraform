package aws

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
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
		Importer: &schema.ResourceImporter{
			State: func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				idParts := strings.Split(d.Id(), "/")
				if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
					return nil, fmt.Errorf("Unexpected format of ID (%q), expected REST-API-ID/RESOURCE-ID/HTTP-METHOD", d.Id())
				}
				restApiID := idParts[0]
				resourceID := idParts[1]
				httpMethod := idParts[2]
				d.Set("http_method", httpMethod)
				d.Set("resource_id", resourceID)
				d.Set("rest_api_id", restApiID)
				d.SetId(fmt.Sprintf("agm-%s-%s-%s", restApiID, resourceID, httpMethod))
				return []*schema.ResourceData{d}, nil
			},
		},

		Schema: map[string]*schema.Schema{
			"rest_api_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"resource_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"http_method": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateHTTPMethod(),
			},

			"authorization": {
				Type:     schema.TypeString,
				Required: true,
			},

			"authorizer_id": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"authorization_scopes": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
				Optional: true,
			},

			"api_key_required": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"request_models": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"request_parameters": {
				Type:          schema.TypeMap,
				Elem:          &schema.Schema{Type: schema.TypeBool},
				Optional:      true,
				ConflictsWith: []string{"request_parameters_in_json"},
			},

			"request_parameters_in_json": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"request_parameters"},
				Deprecated:    "Use field request_parameters instead",
			},

			"request_validator_id": {
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

	if v, ok := d.GetOk("authorization_scopes"); ok {
		input.AuthorizationScopes = expandStringList(v.(*schema.Set).List())
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

	d.Set("api_key_required", out.ApiKeyRequired)

	if err := d.Set("authorization_scopes", flattenStringList(out.AuthorizationScopes)); err != nil {
		return fmt.Errorf("error setting authorization_scopes: %s", err)
	}

	d.Set("authorization", out.AuthorizationType)
	d.Set("authorizer_id", out.AuthorizerId)

	if err := d.Set("request_models", aws.StringValueMap(out.RequestModels)); err != nil {
		return fmt.Errorf("error setting request_models: %s", err)
	}

	// KNOWN ISSUE: This next d.Set() is broken as it should be a JSON string of the map,
	//              however leaving as-is since this attribute has been deprecated
	//              for a very long time and will be removed soon in the next major release.
	//              Not worth the effort of fixing, acceptance testing, and potential JSON equivalence bugs.
	if _, ok := d.GetOk("request_parameters_in_json"); ok {
		d.Set("request_parameters_in_json", aws.BoolValueMap(out.RequestParameters))
	}

	if err := d.Set("request_parameters", aws.BoolValueMap(out.RequestParameters)); err != nil {
		return fmt.Errorf("error setting request_models: %s", err)
	}

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

	if d.HasChange("authorization_scopes") {
		old, new := d.GetChange("authorization_scopes")
		path := "/authorizationScopes"

		os := old.(*schema.Set)
		ns := new.(*schema.Set)

		additionList := ns.Difference(os)
		for _, v := range additionList.List() {
			operations = append(operations, &apigateway.PatchOperation{
				Op:    aws.String("add"),
				Path:  aws.String(path),
				Value: aws.String(v.(string)),
			})
		}

		removalList := os.Difference(ns)
		for _, v := range removalList.List() {
			operations = append(operations, &apigateway.PatchOperation{
				Op:    aws.String("remove"),
				Path:  aws.String(path),
				Value: aws.String(v.(string)),
			})
		}
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
