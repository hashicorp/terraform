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
	"strings"
)

func resourceAwsApiGatewayIntegration() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsApiGatewayIntegrationCreate,
		Read:   resourceAwsApiGatewayIntegrationRead,
		Update: resourceAwsApiGatewayIntegrationUpdate,
		Delete: resourceAwsApiGatewayIntegrationDelete,

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
				ValidateFunc: validateHTTPMethod,
			},

			"type": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateApiGatewayIntegrationType,
			},

			"uri": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"credentials": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"integration_http_method": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateHTTPMethod,
			},

			"request_templates": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem:     schema.TypeString,
			},

			"request_parameters": {
				Type:          schema.TypeMap,
				Elem:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"request_parameters_in_json"},
			},

			"request_parameters_in_json": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"request_parameters"},
				Deprecated:    "Use field request_parameters instead",
			},

			"content_handling": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateApiGatewayIntegrationContentHandling,
			},

			"passthrough_behavior": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validateApiGatewayIntegrationPassthroughBehavior,
			},

			"cache_key_parameters": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
				Optional: true,
			},

			"cache_namespace": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceAwsApiGatewayIntegrationCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	log.Print("[DEBUG] Creating API Gateway Integration")
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
	if kv, ok := d.GetOk("request_parameters"); ok {
		for k, v := range kv.(map[string]interface{}) {
			parameters[k] = v.(string)
		}
	}

	if v, ok := d.GetOk("request_parameters_in_json"); ok {
		if err := json.Unmarshal([]byte(v.(string)), &parameters); err != nil {
			return fmt.Errorf("Error unmarshaling request_parameters_in_json: %s", err)
		}
	}

	var passthroughBehavior *string
	if v, ok := d.GetOk("passthrough_behavior"); ok {
		passthroughBehavior = aws.String(v.(string))
	}

	var credentials *string
	if val, ok := d.GetOk("credentials"); ok {
		credentials = aws.String(val.(string))
	}

	var contentHandling *string
	if val, ok := d.GetOk("content_handling"); ok {
		contentHandling = aws.String(val.(string))
	}

	var cacheKeyParameters []*string
	if v, ok := d.GetOk("cache_key_parameters"); ok {
		cacheKeyParameters = expandStringList(v.(*schema.Set).List())
	}

	var cacheNamespace *string
	if cacheKeyParameters != nil {
		// Use resource_id unless user provides a custom name
		cacheNamespace = aws.String(d.Get("resource_id").(string))
	}
	if v, ok := d.GetOk("cache_namespace"); ok {
		cacheNamespace = aws.String(v.(string))
	}

	_, err := conn.PutIntegration(&apigateway.PutIntegrationInput{
		HttpMethod: aws.String(d.Get("http_method").(string)),
		ResourceId: aws.String(d.Get("resource_id").(string)),
		RestApiId:  aws.String(d.Get("rest_api_id").(string)),
		Type:       aws.String(d.Get("type").(string)),
		IntegrationHttpMethod: integrationHttpMethod,
		Uri:                 uri,
		RequestParameters:   aws.StringMap(parameters),
		RequestTemplates:    aws.StringMap(templates),
		Credentials:         credentials,
		CacheNamespace:      cacheNamespace,
		CacheKeyParameters:  cacheKeyParameters,
		PassthroughBehavior: passthroughBehavior,
		ContentHandling:     contentHandling,
	})
	if err != nil {
		return fmt.Errorf("Error creating API Gateway Integration: %s", err)
	}

	d.SetId(fmt.Sprintf("agi-%s-%s-%s", d.Get("rest_api_id").(string), d.Get("resource_id").(string), d.Get("http_method").(string)))

	return resourceAwsApiGatewayIntegrationRead(d, meta)
}

func resourceAwsApiGatewayIntegrationRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	log.Printf("[DEBUG] Reading API Gateway Integration: %s", d.Id())
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
	d.Set("type", integration.Type)
	d.Set("request_parameters", aws.StringValueMap(integration.RequestParameters))
	d.Set("request_parameters_in_json", aws.StringValueMap(integration.RequestParameters))
	d.Set("passthrough_behavior", integration.PassthroughBehavior)

	if integration.Uri != nil {
		d.Set("uri", integration.Uri)
	}

	if integration.Credentials != nil {
		d.Set("credentials", integration.Credentials)
	}

	if integration.ContentHandling != nil {
		d.Set("content_handling", integration.ContentHandling)
	}

	d.Set("cache_key_parameters", flattenStringList(integration.CacheKeyParameters))

	if integration.CacheNamespace != nil {
		d.Set("cache_namespace", integration.CacheNamespace)
	}

	return nil
}

func resourceAwsApiGatewayIntegrationUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	log.Printf("[DEBUG] Updating API Gateway Integration: %s", d.Id())
	operations := make([]*apigateway.PatchOperation, 0)

	// https://docs.aws.amazon.com/apigateway/api-reference/link-relation/integration-update/#remarks
	// According to the above documentation, only a few parts are addable / removable.
	if d.HasChange("request_templates") {
		o, n := d.GetChange("request_templates")
		prefix := "requestTemplates"

		os := o.(map[string]interface{})
		ns := n.(map[string]interface{})

		// Handle Removal
		for k := range os {
			if _, ok := ns[k]; !ok {
				operations = append(operations, &apigateway.PatchOperation{
					Op:   aws.String("remove"),
					Path: aws.String(fmt.Sprintf("/%s/%s", prefix, strings.Replace(k, "/", "~1", -1))),
				})
			}
		}

		for k, v := range ns {
			// Handle replaces
			if _, ok := os[k]; ok {
				operations = append(operations, &apigateway.PatchOperation{
					Op:    aws.String("replace"),
					Path:  aws.String(fmt.Sprintf("/%s/%s", prefix, strings.Replace(k, "/", "~1", -1))),
					Value: aws.String(v.(string)),
				})
			}

			// Handle additions
			if _, ok := os[k]; !ok {
				operations = append(operations, &apigateway.PatchOperation{
					Op:    aws.String("add"),
					Path:  aws.String(fmt.Sprintf("/%s/%s", prefix, strings.Replace(k, "/", "~1", -1))),
					Value: aws.String(v.(string)),
				})
			}
		}
	}

	if d.HasChange("request_parameters") {
		o, n := d.GetChange("request_parameters")
		prefix := "requestParameters"

		os := o.(map[string]interface{})
		ns := n.(map[string]interface{})

		// Handle Removal
		for k := range os {
			if _, ok := ns[k]; !ok {
				operations = append(operations, &apigateway.PatchOperation{
					Op:   aws.String("remove"),
					Path: aws.String(fmt.Sprintf("/%s/%s", prefix, strings.Replace(k, "/", "~1", -1))),
				})
			}
		}

		for k, v := range ns {
			// Handle replaces
			if _, ok := os[k]; ok {
				operations = append(operations, &apigateway.PatchOperation{
					Op:    aws.String("replace"),
					Path:  aws.String(fmt.Sprintf("/%s/%s", prefix, strings.Replace(k, "/", "~1", -1))),
					Value: aws.String(v.(string)),
				})
			}

			// Handle additions
			if _, ok := os[k]; !ok {
				operations = append(operations, &apigateway.PatchOperation{
					Op:    aws.String("add"),
					Path:  aws.String(fmt.Sprintf("/%s/%s", prefix, strings.Replace(k, "/", "~1", -1))),
					Value: aws.String(v.(string)),
				})
			}
		}
	}

	if d.HasChange("cache_key_parameters") {
		o, n := d.GetChange("cache_key_parameters")

		os := o.(*schema.Set)
		ns := n.(*schema.Set)

		removalList := os.Difference(ns)
		for _, v := range removalList.List() {
			operations = append(operations, &apigateway.PatchOperation{
				Op:    aws.String("remove"),
				Path:  aws.String(fmt.Sprintf("/cacheKeyParameters/%s", v.(string))),
				Value: aws.String(""),
			})
		}

		additionList := ns.Difference(os)
		for _, v := range additionList.List() {
			operations = append(operations, &apigateway.PatchOperation{
				Op:    aws.String("add"),
				Path:  aws.String(fmt.Sprintf("/cacheKeyParameters/%s", v.(string))),
				Value: aws.String(""),
			})
		}
	}

	if d.HasChange("cache_namespace") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/cacheNamespace"),
			Value: aws.String(d.Get("cache_namespace").(string)),
		})
	}

	params := &apigateway.UpdateIntegrationInput{
		HttpMethod:      aws.String(d.Get("http_method").(string)),
		ResourceId:      aws.String(d.Get("resource_id").(string)),
		RestApiId:       aws.String(d.Get("rest_api_id").(string)),
		PatchOperations: operations,
	}

	_, err := conn.UpdateIntegration(params)
	if err != nil {
		return fmt.Errorf("Error updating API Gateway Integration: %s", err)
	}

	d.SetId(fmt.Sprintf("agi-%s-%s-%s", d.Get("rest_api_id").(string), d.Get("resource_id").(string), d.Get("http_method").(string)))

	return resourceAwsApiGatewayIntegrationRead(d, meta)
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
