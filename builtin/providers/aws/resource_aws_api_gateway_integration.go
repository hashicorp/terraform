package aws

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsApiGatewayIntegration() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsApiGatewayIntegrationCreate,
		Read:   resourceAwsApiGatewayIntegrationRead,
		Update: resourceAwsApiGatewayIntegrationUpdate,
		Delete: resourceAwsApiGatewayIntegrationDelete,

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
				ValidateFunc: validateHTTPMethod,
			},

			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if value != "MOCK" && value != "AWS" && value != "HTTP" {
						errors = append(errors, fmt.Errorf(
							"%q must be one of 'AWS', 'MOCK', 'HTTP'", k))
					}
					return
				},
			},

			"uri": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"credentials": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"integration_http_method": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateHTTPMethod,
			},

			"request_templates": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				Elem:     schema.TypeString,
			},

			"request_parameters_in_json": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceAwsApiGatewayIntegrationCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	var integrationHttpMethod *string
	if v, ok := d.GetOk("integration_http_method"); ok {
		integrationHttpMethod = aws.String(v.(string))
	}
	var uri *string
	if v, ok := d.GetOk("uri"); ok {
		uri = aws.String(v.(string))
	}
	templates := make(map[string]string)
	for k, v := range d.Get("request_templates").(map[string]interface{}) {
		templates[k] = v.(string)
	}

	parameters := make(map[string]string)
	if v, ok := d.GetOk("request_parameters_in_json"); ok {
		if err := json.Unmarshal([]byte(v.(string)), &parameters); err != nil {
			return fmt.Errorf("Error unmarshaling request_parameters_in_json: %s", err)
		}
	}

	var credentials *string
	if val, ok := d.GetOk("credentials"); ok {
		credentials = aws.String(val.(string))
	}

	_, err := conn.PutIntegration(&apigateway.PutIntegrationInput{
		HttpMethod: aws.String(d.Get("http_method").(string)),
		ResourceId: aws.String(d.Get("resource_id").(string)),
		RestApiId:  aws.String(d.Get("rest_api_id").(string)),
		Type:       aws.String(d.Get("type").(string)),
		IntegrationHttpMethod: integrationHttpMethod,
		Uri: uri,
		// TODO reimplement once [GH-2143](https://github.com/hashicorp/terraform/issues/2143) has been implemented
		RequestParameters:  aws.StringMap(parameters),
		RequestTemplates:   aws.StringMap(templates),
		Credentials:        credentials,
		CacheNamespace:     nil,
		CacheKeyParameters: nil,
	})
	if err != nil {
		return fmt.Errorf("Error creating API Gateway Integration: %s", err)
	}

	d.SetId(fmt.Sprintf("agi-%s-%s-%s", d.Get("rest_api_id").(string), d.Get("resource_id").(string), d.Get("http_method").(string)))

	return nil
}

func resourceAwsApiGatewayIntegrationRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	log.Printf("[DEBUG] Reading API Gateway Integration %s", d.Id())
	integration, err := conn.GetIntegration(&apigateway.GetIntegrationInput{
		HttpMethod: aws.String(d.Get("http_method").(string)),
		ResourceId: aws.String(d.Get("resource_id").(string)),
		RestApiId:  aws.String(d.Get("rest_api_id").(string)),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NotFoundException" {
			d.SetId("")
			return nil
		}
		return err
	}
	log.Printf("[DEBUG] Received API Gateway Integration: %s", integration)
	d.SetId(fmt.Sprintf("agi-%s-%s-%s", d.Get("rest_api_id").(string), d.Get("resource_id").(string), d.Get("http_method").(string)))

	// AWS converts "" to null on their side, convert it back
	if v, ok := integration.RequestTemplates["application/json"]; ok && v == nil {
		integration.RequestTemplates["application/json"] = aws.String("")
	}

	d.Set("request_templates", aws.StringValueMap(integration.RequestTemplates))
	d.Set("credentials", integration.Credentials)
	d.Set("type", integration.Type)
	d.Set("uri", integration.Uri)
	d.Set("request_parameters_in_json", aws.StringValueMap(integration.RequestParameters))

	return nil
}

func resourceAwsApiGatewayIntegrationUpdate(d *schema.ResourceData, meta interface{}) error {
	return resourceAwsApiGatewayIntegrationCreate(d, meta)
}

func resourceAwsApiGatewayIntegrationDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway
	log.Printf("[DEBUG] Deleting API Gateway Integration: %s", d.Id())

	return resource.Retry(5*time.Minute, func() *resource.RetryError {
		_, err := conn.DeleteIntegration(&apigateway.DeleteIntegrationInput{
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
