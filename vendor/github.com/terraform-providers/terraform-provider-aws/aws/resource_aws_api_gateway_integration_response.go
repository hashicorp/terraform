package aws

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsApiGatewayIntegrationResponse() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsApiGatewayIntegrationResponseCreate,
		Read:   resourceAwsApiGatewayIntegrationResponseRead,
		Update: resourceAwsApiGatewayIntegrationResponseCreate,
		Delete: resourceAwsApiGatewayIntegrationResponseDelete,
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
				d.SetId(fmt.Sprintf("agir-%s-%s-%s-%s", restApiID, resourceID, httpMethod, statusCode))
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

			"selection_pattern": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"response_templates": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"response_parameters": {
				Type:          schema.TypeMap,
				Elem:          &schema.Schema{Type: schema.TypeString},
				Optional:      true,
				ConflictsWith: []string{"response_parameters_in_json"},
			},

			"response_parameters_in_json": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"response_parameters"},
				Deprecated:    "Use field response_parameters instead",
			},

			"content_handling": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateApiGatewayIntegrationContentHandling(),
			},
		},
	}
}

func resourceAwsApiGatewayIntegrationResponseCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	templates := make(map[string]string)
	for k, v := range d.Get("response_templates").(map[string]interface{}) {
		templates[k] = v.(string)
	}

	parameters := make(map[string]string)
	if kv, ok := d.GetOk("response_parameters"); ok {
		for k, v := range kv.(map[string]interface{}) {
			parameters[k] = v.(string)
		}
	}
	if v, ok := d.GetOk("response_parameters_in_json"); ok {
		if err := json.Unmarshal([]byte(v.(string)), &parameters); err != nil {
			return fmt.Errorf("Error unmarshaling response_parameters_in_json: %s", err)
		}
	}
	var contentHandling *string
	if val, ok := d.GetOk("content_handling"); ok {
		contentHandling = aws.String(val.(string))
	}

	input := apigateway.PutIntegrationResponseInput{
		HttpMethod:         aws.String(d.Get("http_method").(string)),
		ResourceId:         aws.String(d.Get("resource_id").(string)),
		RestApiId:          aws.String(d.Get("rest_api_id").(string)),
		StatusCode:         aws.String(d.Get("status_code").(string)),
		ResponseTemplates:  aws.StringMap(templates),
		ResponseParameters: aws.StringMap(parameters),
		ContentHandling:    contentHandling,
	}
	if v, ok := d.GetOk("selection_pattern"); ok {
		input.SelectionPattern = aws.String(v.(string))
	}

	_, err := conn.PutIntegrationResponse(&input)
	if err != nil {
		return fmt.Errorf("Error creating API Gateway Integration Response: %s", err)
	}

	d.SetId(fmt.Sprintf("agir-%s-%s-%s-%s", d.Get("rest_api_id").(string), d.Get("resource_id").(string), d.Get("http_method").(string), d.Get("status_code").(string)))
	log.Printf("[DEBUG] API Gateway Integration Response ID: %s", d.Id())

	return resourceAwsApiGatewayIntegrationResponseRead(d, meta)
}

func resourceAwsApiGatewayIntegrationResponseRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	log.Printf("[DEBUG] Reading API Gateway Integration Response %s", d.Id())
	integrationResponse, err := conn.GetIntegrationResponse(&apigateway.GetIntegrationResponseInput{
		HttpMethod: aws.String(d.Get("http_method").(string)),
		ResourceId: aws.String(d.Get("resource_id").(string)),
		RestApiId:  aws.String(d.Get("rest_api_id").(string)),
		StatusCode: aws.String(d.Get("status_code").(string)),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NotFoundException" {
			log.Printf("[WARN] API Gateway Integration Response (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	log.Printf("[DEBUG] Received API Gateway Integration Response: %s", integrationResponse)

	d.Set("content_handling", integrationResponse.ContentHandling)

	if err := d.Set("response_parameters", aws.StringValueMap(integrationResponse.ResponseParameters)); err != nil {
		return fmt.Errorf("error setting response_parameters: %s", err)
	}

	// KNOWN ISSUE: This next d.Set() is broken as it should be a JSON string of the map,
	//              however leaving as-is since this attribute has been deprecated
	//              for a very long time and will be removed soon in the next major release.
	//              Not worth the effort of fixing, acceptance testing, and potential JSON equivalence bugs.
	if _, ok := d.GetOk("response_parameters_in_json"); ok {
		d.Set("response_parameters_in_json", aws.StringValueMap(integrationResponse.ResponseParameters))
	}

	// We need to explicitly convert key = nil values into key = "", which aws.StringValueMap() removes
	responseTemplateMap := make(map[string]string)
	for key, valuePointer := range integrationResponse.ResponseTemplates {
		responseTemplateMap[key] = aws.StringValue(valuePointer)
	}
	if err := d.Set("response_templates", responseTemplateMap); err != nil {
		return fmt.Errorf("error setting response_templates: %s", err)
	}

	d.Set("selection_pattern", integrationResponse.SelectionPattern)

	return nil
}

func resourceAwsApiGatewayIntegrationResponseDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway
	log.Printf("[DEBUG] Deleting API Gateway Integration Response: %s", d.Id())

	return resource.Retry(5*time.Minute, func() *resource.RetryError {
		_, err := conn.DeleteIntegrationResponse(&apigateway.DeleteIntegrationResponseInput{
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
