package aws

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsApiGatewayStage() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsApiGatewayStageCreate,
		Read:   resourceAwsApiGatewayStageRead,
		Update: resourceAwsApiGatewayStageUpdate,
		Delete: resourceAwsApiGatewayStageDelete,
		Importer: &schema.ResourceImporter{
			State: func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				idParts := strings.Split(d.Id(), "/")
				if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
					return nil, fmt.Errorf("Unexpected format of ID (%q), expected REST-API-ID/STAGE-NAME", d.Id())
				}
				restApiID := idParts[0]
				stageName := idParts[1]
				d.Set("stage_name", stageName)
				d.Set("rest_api_id", restApiID)
				d.SetId(fmt.Sprintf("ags-%s-%s", restApiID, stageName))
				return []*schema.ResourceData{d}, nil
			},
		},

		Schema: map[string]*schema.Schema{
			"access_log_settings": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"destination_arn": {
							Type:     schema.TypeString,
							Required: true,
							StateFunc: func(arn interface{}) string {
								// arns coming from a TF reference to a log group contain a trailing `:*` which is not valid
								return strings.TrimSuffix(arn.(string), ":*")
							},
						},
						"format": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"cache_cluster_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"cache_cluster_size": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"client_certificate_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"deployment_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"documentation_version": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"execution_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"invoke_url": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"rest_api_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"stage_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"variables": {
				Type:     schema.TypeMap,
				Optional: true,
			},
			"tags": tagsSchema(),
			"xray_tracing_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
			},
		},
	}
}

func resourceAwsApiGatewayStageCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	d.Partial(true)

	input := apigateway.CreateStageInput{
		RestApiId:    aws.String(d.Get("rest_api_id").(string)),
		StageName:    aws.String(d.Get("stage_name").(string)),
		DeploymentId: aws.String(d.Get("deployment_id").(string)),
	}

	waitForCache := false
	if v, ok := d.GetOk("cache_cluster_enabled"); ok {
		input.CacheClusterEnabled = aws.Bool(v.(bool))
		waitForCache = true
	}
	if v, ok := d.GetOk("cache_cluster_size"); ok {
		input.CacheClusterSize = aws.String(v.(string))
		waitForCache = true
	}
	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}
	if v, ok := d.GetOk("xray_tracing_enabled"); ok {
		input.TracingEnabled = aws.Bool(v.(bool))
	}
	if v, ok := d.GetOk("documentation_version"); ok {
		input.DocumentationVersion = aws.String(v.(string))
	}
	if vars, ok := d.GetOk("variables"); ok {
		variables := make(map[string]string)
		for k, v := range vars.(map[string]interface{}) {
			variables[k] = v.(string)
		}
		input.Variables = aws.StringMap(variables)
	}
	if vars, ok := d.GetOk("tags"); ok {
		newMap := make(map[string]string, len(vars.(map[string]interface{})))
		for k, v := range vars.(map[string]interface{}) {
			newMap[k] = v.(string)
		}
		input.Tags = aws.StringMap(newMap)
	}

	out, err := conn.CreateStage(&input)
	if err != nil {
		return fmt.Errorf("Error creating API Gateway Stage: %s", err)
	}

	d.SetId(fmt.Sprintf("ags-%s-%s", d.Get("rest_api_id").(string), d.Get("stage_name").(string)))

	d.SetPartial("rest_api_id")
	d.SetPartial("stage_name")
	d.SetPartial("deployment_id")
	d.SetPartial("description")
	d.SetPartial("variables")
	d.SetPartial("xray_tracing_enabled")

	if waitForCache && *out.CacheClusterStatus != "NOT_AVAILABLE" {
		stateConf := &resource.StateChangeConf{
			Pending: []string{
				"CREATE_IN_PROGRESS",
				"DELETE_IN_PROGRESS",
				"FLUSH_IN_PROGRESS",
			},
			Target: []string{"AVAILABLE"},
			Refresh: apiGatewayStageCacheRefreshFunc(conn,
				d.Get("rest_api_id").(string),
				d.Get("stage_name").(string)),
			Timeout: 90 * time.Minute,
		}

		_, err := stateConf.WaitForState()
		if err != nil {
			return err
		}
	}

	d.SetPartial("cache_cluster_enabled")
	d.SetPartial("cache_cluster_size")
	d.Partial(false)

	if _, ok := d.GetOk("client_certificate_id"); ok {
		return resourceAwsApiGatewayStageUpdate(d, meta)
	}
	if _, ok := d.GetOk("access_log_settings"); ok {
		return resourceAwsApiGatewayStageUpdate(d, meta)
	}
	return resourceAwsApiGatewayStageRead(d, meta)
}

func resourceAwsApiGatewayStageRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	log.Printf("[DEBUG] Reading API Gateway Stage %s", d.Id())
	restApiId := d.Get("rest_api_id").(string)
	stageName := d.Get("stage_name").(string)
	input := apigateway.GetStageInput{
		RestApiId: aws.String(restApiId),
		StageName: aws.String(stageName),
	}
	stage, err := conn.GetStage(&input)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NotFoundException" {
			log.Printf("[WARN] API Gateway Stage (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}
	log.Printf("[DEBUG] Received API Gateway Stage: %s", stage)

	if err := d.Set("access_log_settings", flattenApiGatewayStageAccessLogSettings(stage.AccessLogSettings)); err != nil {
		return fmt.Errorf("error setting access_log_settings: %s", err)
	}

	d.Set("client_certificate_id", stage.ClientCertificateId)

	if stage.CacheClusterStatus != nil && *stage.CacheClusterStatus == "DELETE_IN_PROGRESS" {
		d.Set("cache_cluster_enabled", false)
		d.Set("cache_cluster_size", nil)
	} else {
		d.Set("cache_cluster_enabled", stage.CacheClusterEnabled)
		d.Set("cache_cluster_size", stage.CacheClusterSize)
	}

	d.Set("deployment_id", stage.DeploymentId)
	d.Set("description", stage.Description)
	d.Set("documentation_version", stage.DocumentationVersion)
	d.Set("xray_tracing_enabled", stage.TracingEnabled)

	if err := d.Set("tags", aws.StringValueMap(stage.Tags)); err != nil {
		return fmt.Errorf("error setting tags: %s", err)
	}

	if err := d.Set("variables", aws.StringValueMap(stage.Variables)); err != nil {
		return fmt.Errorf("error setting variables: %s", err)
	}

	region := meta.(*AWSClient).region
	d.Set("invoke_url", buildApiGatewayInvokeURL(restApiId, region, stageName))

	executionArn := arn.ARN{
		Partition: meta.(*AWSClient).partition,
		Service:   "execute-api",
		Region:    meta.(*AWSClient).region,
		AccountID: meta.(*AWSClient).accountid,
		Resource:  fmt.Sprintf("%s/%s", restApiId, stageName),
	}.String()
	d.Set("execution_arn", executionArn)

	return nil
}

func resourceAwsApiGatewayStageUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	d.Partial(true)

	stageArn := arn.ARN{
		Partition: meta.(*AWSClient).partition,
		Region:    meta.(*AWSClient).region,
		Service:   "apigateway",
		Resource:  fmt.Sprintf("/restapis/%s/stages/%s", d.Get("rest_api_id").(string), d.Get("stage_name").(string)),
	}.String()
	if tagErr := setTagsAPIGatewayStage(conn, d, stageArn); tagErr != nil {
		return tagErr
	}
	d.SetPartial("tags")

	operations := make([]*apigateway.PatchOperation, 0)
	waitForCache := false
	if d.HasChange("cache_cluster_enabled") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/cacheClusterEnabled"),
			Value: aws.String(fmt.Sprintf("%t", d.Get("cache_cluster_enabled").(bool))),
		})
		waitForCache = true
	}
	if d.HasChange("cache_cluster_size") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/cacheClusterSize"),
			Value: aws.String(d.Get("cache_cluster_size").(string)),
		})
		waitForCache = true
	}
	if d.HasChange("client_certificate_id") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/clientCertificateId"),
			Value: aws.String(d.Get("client_certificate_id").(string)),
		})
	}
	if d.HasChange("deployment_id") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/deploymentId"),
			Value: aws.String(d.Get("deployment_id").(string)),
		})
	}
	if d.HasChange("description") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/description"),
			Value: aws.String(d.Get("description").(string)),
		})
	}
	if d.HasChange("xray_tracing_enabled") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/tracingEnabled"),
			Value: aws.String(fmt.Sprintf("%t", d.Get("xray_tracing_enabled").(bool))),
		})
	}
	if d.HasChange("documentation_version") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/documentationVersion"),
			Value: aws.String(d.Get("documentation_version").(string)),
		})
	}
	if d.HasChange("variables") {
		o, n := d.GetChange("variables")
		oldV := o.(map[string]interface{})
		newV := n.(map[string]interface{})
		operations = append(operations, diffVariablesOps(oldV, newV)...)
	}
	if d.HasChange("access_log_settings") {
		accessLogSettings := d.Get("access_log_settings").([]interface{})
		if len(accessLogSettings) == 1 {
			operations = append(operations,
				&apigateway.PatchOperation{
					Op:   aws.String("replace"),
					Path: aws.String("/accessLogSettings/destinationArn"),
					// arns coming from a TF reference to a log group contain a trailing `:*` which is not valid
					Value: aws.String(strings.TrimSuffix(d.Get("access_log_settings.0.destination_arn").(string), ":*")),
				}, &apigateway.PatchOperation{
					Op:    aws.String("replace"),
					Path:  aws.String("/accessLogSettings/format"),
					Value: aws.String(d.Get("access_log_settings.0.format").(string)),
				})
		} else if len(accessLogSettings) == 0 {
			operations = append(operations, &apigateway.PatchOperation{
				Op:   aws.String("remove"),
				Path: aws.String("/accessLogSettings"),
			})
		}
	}

	input := apigateway.UpdateStageInput{
		RestApiId:       aws.String(d.Get("rest_api_id").(string)),
		StageName:       aws.String(d.Get("stage_name").(string)),
		PatchOperations: operations,
	}
	log.Printf("[DEBUG] Updating API Gateway Stage: %s", input)
	out, err := conn.UpdateStage(&input)
	if err != nil {
		return fmt.Errorf("Updating API Gateway Stage failed: %s", err)
	}

	d.SetPartial("client_certificate_id")
	d.SetPartial("deployment_id")
	d.SetPartial("description")
	d.SetPartial("xray_tracing_enabled")
	d.SetPartial("variables")

	if waitForCache && *out.CacheClusterStatus != "NOT_AVAILABLE" {
		stateConf := &resource.StateChangeConf{
			Pending: []string{
				"CREATE_IN_PROGRESS",
				"FLUSH_IN_PROGRESS",
			},
			Target: []string{
				"AVAILABLE",
				// There's an AWS API bug (raised & confirmed in Sep 2016 by support)
				// which causes the stage to remain in deletion state forever
				"DELETE_IN_PROGRESS",
			},
			Refresh: apiGatewayStageCacheRefreshFunc(conn,
				d.Get("rest_api_id").(string),
				d.Get("stage_name").(string)),
			Timeout: 30 * time.Minute,
		}

		_, err := stateConf.WaitForState()
		if err != nil {
			return err
		}
	}

	d.SetPartial("cache_cluster_enabled")
	d.SetPartial("cache_cluster_size")
	d.Partial(false)

	return resourceAwsApiGatewayStageRead(d, meta)
}

func diffVariablesOps(oldVars, newVars map[string]interface{}) []*apigateway.PatchOperation {
	ops := make([]*apigateway.PatchOperation, 0)
	prefix := "/variables/"

	for k := range oldVars {
		if _, ok := newVars[k]; !ok {
			ops = append(ops, &apigateway.PatchOperation{
				Op:   aws.String("remove"),
				Path: aws.String(prefix + k),
			})
		}
	}

	for k, v := range newVars {
		newValue := v.(string)

		if oldV, ok := oldVars[k]; ok {
			oldValue := oldV.(string)
			if oldValue == newValue {
				continue
			}
		}
		ops = append(ops, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String(prefix + k),
			Value: aws.String(newValue),
		})
	}

	return ops
}

func apiGatewayStageCacheRefreshFunc(conn *apigateway.APIGateway, apiId, stageName string) func() (interface{}, string, error) {
	return func() (interface{}, string, error) {
		input := apigateway.GetStageInput{
			RestApiId: aws.String(apiId),
			StageName: aws.String(stageName),
		}
		out, err := conn.GetStage(&input)
		if err != nil {
			return 42, "", err
		}

		return out, *out.CacheClusterStatus, nil
	}
}

func resourceAwsApiGatewayStageDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway
	log.Printf("[DEBUG] Deleting API Gateway Stage: %s", d.Id())
	input := apigateway.DeleteStageInput{
		RestApiId: aws.String(d.Get("rest_api_id").(string)),
		StageName: aws.String(d.Get("stage_name").(string)),
	}
	_, err := conn.DeleteStage(&input)
	if err != nil {
		return fmt.Errorf("Deleting API Gateway Stage failed: %s", err)
	}

	return nil
}

func flattenApiGatewayStageAccessLogSettings(accessLogSettings *apigateway.AccessLogSettings) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, 1)
	if accessLogSettings != nil {
		result = append(result, map[string]interface{}{
			"destination_arn": aws.StringValue(accessLogSettings.DestinationArn),
			"format":          aws.StringValue(accessLogSettings.Format),
		})
	}
	return result
}
