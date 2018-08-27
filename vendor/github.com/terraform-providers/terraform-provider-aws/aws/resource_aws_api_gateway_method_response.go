package aws

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

var resourceAwsApiGatewayMethodResponseMutex = &sync.Mutex{}

func resourceAwsApiGatewayMethodResponse() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsApiGatewayMethodResponseCreate,
		Read:   resourceAwsApiGatewayMethodResponseRead,
		Update: resourceAwsApiGatewayMethodResponseUpdate,
		Delete: resourceAwsApiGatewayMethodResponseDelete,
		Importer: &schema.ResourceImporter{
			State: func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				idParts := strings.Split(d.Id(), "/")
				if len(idParts) != 4 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" || idParts[3] == "" {
					return nil, fmt.Errorf("Unexpected format of ID (%q), expected REST-API-ID/RESOURCE-ID/HTTP-METHOD/STATUS-CODE", d.Id())
				}
				restApiID := idParts[0]
				resourceID := idParts[1]
				httpMethod := idParts[2]
				statusCode := idParts[3]
				d.Set("http_method", httpMethod)
				d.Set("status_code", statusCode)
				d.Set("resource_id", resourceID)
				d.Set("rest_api_id", restApiID)
				d.SetId(fmt.Sprintf("agmr-%s-%s-%s-%s", restApiID, resourceID, httpMethod, statusCode))
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

			"status_code": {
				Type:     schema.TypeString,
				Required: true,
			},

			"response_models": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"response_parameters": {
				Type:          schema.TypeMap,
				Elem:          &schema.Schema{Type: schema.TypeBool},
				Optional:      true,
				ConflictsWith: []string{"response_parameters_in_json"},
			},

			"response_parameters_in_json": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"response_parameters"},
				Deprecated:    "Use field response_parameters instead",
			},
		},
	}
}

func resourceAwsApiGatewayMethodResponseCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	models := make(map[string]string)
	for k, v := range d.Get("response_models").(map[string]interface{}) {
		models[k] = v.(string)
	}

	parameters := make(map[string]bool)
	if kv, ok := d.GetOk("response_parameters"); ok {
		for k, v := range kv.(map[string]interface{}) {
			parameters[k], ok = v.(bool)
			if !ok {
				value, _ := strconv.ParseBool(v.(string))
				parameters[k] = value
			}
		}
	}
	if v, ok := d.GetOk("response_parameters_in_json"); ok {
		if err := json.Unmarshal([]byte(v.(string)), &parameters); err != nil {
			return fmt.Errorf("Error unmarshaling request_parameters_in_json: %s", err)
		}
	}

	resourceAwsApiGatewayMethodResponseMutex.Lock()
	defer resourceAwsApiGatewayMethodResponseMutex.Unlock()

	_, err := retryOnAwsCode(apigateway.ErrCodeConflictException, func() (interface{}, error) {
		return conn.PutMethodResponse(&apigateway.PutMethodResponseInput{
			HttpMethod:         aws.String(d.Get("http_method").(string)),
			ResourceId:         aws.String(d.Get("resource_id").(string)),
			RestApiId:          aws.String(d.Get("rest_api_id").(string)),
			StatusCode:         aws.String(d.Get("status_code").(string)),
			ResponseModels:     aws.StringMap(models),
			ResponseParameters: aws.BoolMap(parameters),
		})
	})

	if err != nil {
		return fmt.Errorf("Error creating API Gateway Method Response: %s", err)
	}

	d.SetId(fmt.Sprintf("agmr-%s-%s-%s-%s", d.Get("rest_api_id").(string), d.Get("resource_id").(string), d.Get("http_method").(string), d.Get("status_code").(string)))
	log.Printf("[DEBUG] API Gateway Method ID: %s", d.Id())

	return nil
}

func resourceAwsApiGatewayMethodResponseRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	log.Printf("[DEBUG] Reading API Gateway Method Response %s", d.Id())
	methodResponse, err := conn.GetMethodResponse(&apigateway.GetMethodResponseInput{
		HttpMethod: aws.String(d.Get("http_method").(string)),
		ResourceId: aws.String(d.Get("resource_id").(string)),
		RestApiId:  aws.String(d.Get("rest_api_id").(string)),
		StatusCode: aws.String(d.Get("status_code").(string)),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NotFoundException" {
			log.Printf("[WARN] API Gateway Response (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	log.Printf("[DEBUG] Received API Gateway Method Response: %s", methodResponse)

	if err := d.Set("response_models", aws.StringValueMap(methodResponse.ResponseModels)); err != nil {
		return fmt.Errorf("error setting response_models: %s", err)
	}

	if err := d.Set("response_parameters", aws.BoolValueMap(methodResponse.ResponseParameters)); err != nil {
		return fmt.Errorf("error setting response_parameters: %s", err)
	}

	// KNOWN ISSUE: This next d.Set() is broken as it should be a JSON string of the map,
	//              however leaving as-is since this attribute has been deprecated
	//              for a very long time and will be removed soon in the next major release.
	//              Not worth the effort of fixing, acceptance testing, and potential JSON equivalence bugs.
	if _, ok := d.GetOk("response_parameters_in_json"); ok {
		d.Set("response_parameters_in_json", aws.BoolValueMap(methodResponse.ResponseParameters))
	}

	return nil
}

func resourceAwsApiGatewayMethodResponseUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	log.Printf("[DEBUG] Updating API Gateway Method Response %s", d.Id())
	operations := make([]*apigateway.PatchOperation, 0)

	if d.HasChange("response_models") {
		operations = append(operations, expandApiGatewayRequestResponseModelOperations(d, "response_models", "responseModels")...)
	}

	if d.HasChange("response_parameters_in_json") {
		ops, err := deprecatedExpandApiGatewayMethodParametersJSONOperations(d, "response_parameters_in_json", "responseParameters")
		if err != nil {
			return err
		}
		operations = append(operations, ops...)
	}

	if d.HasChange("response_parameters") {
		ops, err := expandApiGatewayMethodParametersOperations(d, "response_parameters", "responseParameters")
		if err != nil {
			return err
		}
		operations = append(operations, ops...)
	}

	out, err := conn.UpdateMethodResponse(&apigateway.UpdateMethodResponseInput{
		HttpMethod:      aws.String(d.Get("http_method").(string)),
		ResourceId:      aws.String(d.Get("resource_id").(string)),
		RestApiId:       aws.String(d.Get("rest_api_id").(string)),
		StatusCode:      aws.String(d.Get("status_code").(string)),
		PatchOperations: operations,
	})

	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Received API Gateway Method Response: %s", out)

	return resourceAwsApiGatewayMethodResponseRead(d, meta)
}

func resourceAwsApiGatewayMethodResponseDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway
	log.Printf("[DEBUG] Deleting API Gateway Method Response: %s", d.Id())

	return resource.Retry(5*time.Minute, func() *resource.RetryError {
		_, err := conn.DeleteMethodResponse(&apigateway.DeleteMethodResponseInput{
			HttpMethod: aws.String(d.Get("http_method").(string)),
			ResourceId: aws.String(d.Get("resource_id").(string)),
			RestApiId:  aws.String(d.Get("rest_api_id").(string)),
			StatusCode: aws.String(d.Get("status_code").(string)),
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
