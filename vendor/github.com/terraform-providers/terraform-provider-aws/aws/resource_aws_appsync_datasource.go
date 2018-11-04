package aws

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/appsync"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsAppsyncDatasource() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsAppsyncDatasourceCreate,
		Read:   resourceAwsAppsyncDatasourceRead,
		Update: resourceAwsAppsyncDatasourceUpdate,
		Delete: resourceAwsAppsyncDatasourceDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"api_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if !regexp.MustCompile(`[_A-Za-z][_0-9A-Za-z]*`).MatchString(value) {
						errors = append(errors, fmt.Errorf("%q must match [_A-Za-z][_0-9A-Za-z]*", k))
					}
					return
				},
			},
			"type": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validation.StringInSlice([]string{
					appsync.DataSourceTypeAwsLambda,
					appsync.DataSourceTypeAmazonDynamodb,
					appsync.DataSourceTypeAmazonElasticsearch,
					appsync.DataSourceTypeHttp,
					appsync.DataSourceTypeNone,
				}, true),
				StateFunc: func(v interface{}) string {
					return strings.ToUpper(v.(string))
				},
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"dynamodb_config": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"region": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"table_name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"use_caller_credentials": {
							Type:     schema.TypeBool,
							Optional: true,
						},
					},
				},
				ConflictsWith: []string{"elasticsearch_config", "http_config", "lambda_config"},
			},
			"elasticsearch_config": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"region": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"endpoint": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				ConflictsWith: []string{"dynamodb_config", "http_config", "lambda_config"},
			},
			"http_config": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"endpoint": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				ConflictsWith: []string{"dynamodb_config", "elasticsearch_config", "lambda_config"},
			},
			"lambda_config": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"function_arn": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				ConflictsWith: []string{"dynamodb_config", "elasticsearch_config", "http_config"},
			},
			"service_role_arn": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsAppsyncDatasourceCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appsyncconn
	region := meta.(*AWSClient).region

	input := &appsync.CreateDataSourceInput{
		ApiId: aws.String(d.Get("api_id").(string)),
		Name:  aws.String(d.Get("name").(string)),
		Type:  aws.String(d.Get("type").(string)),
	}

	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}

	if v, ok := d.GetOk("dynamodb_config"); ok {
		input.DynamodbConfig = expandAppsyncDynamodbDataSourceConfig(v.([]interface{}), region)
	}

	if v, ok := d.GetOk("elasticsearch_config"); ok {
		input.ElasticsearchConfig = expandAppsyncElasticsearchDataSourceConfig(v.([]interface{}), region)
	}

	if v, ok := d.GetOk("http_config"); ok {
		input.HttpConfig = expandAppsyncHTTPDataSourceConfig(v.([]interface{}))
	}

	if v, ok := d.GetOk("lambda_config"); ok {
		input.LambdaConfig = expandAppsyncLambdaDataSourceConfig(v.([]interface{}))
	}

	if v, ok := d.GetOk("service_role_arn"); ok {
		input.ServiceRoleArn = aws.String(v.(string))
	}

	_, err := conn.CreateDataSource(input)
	if err != nil {
		return err
	}

	d.SetId(d.Get("api_id").(string) + "-" + d.Get("name").(string))

	return resourceAwsAppsyncDatasourceRead(d, meta)
}

func resourceAwsAppsyncDatasourceRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appsyncconn

	apiID, name, err := decodeAppsyncDataSourceID(d.Id())

	if err != nil {
		return err
	}

	input := &appsync.GetDataSourceInput{
		ApiId: aws.String(apiID),
		Name:  aws.String(name),
	}

	resp, err := conn.GetDataSource(input)
	if err != nil {
		if isAWSErr(err, appsync.ErrCodeNotFoundException, "") {
			log.Printf("[WARN] AppSync Datasource %q not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("api_id", apiID)
	d.Set("arn", resp.DataSource.DataSourceArn)
	d.Set("description", resp.DataSource.Description)

	if err := d.Set("dynamodb_config", flattenAppsyncDynamodbDataSourceConfig(resp.DataSource.DynamodbConfig)); err != nil {
		return fmt.Errorf("error setting dynamodb_config: %s", err)
	}

	if err := d.Set("elasticsearch_config", flattenAppsyncElasticsearchDataSourceConfig(resp.DataSource.ElasticsearchConfig)); err != nil {
		return fmt.Errorf("error setting elasticsearch_config: %s", err)
	}

	if err := d.Set("http_config", flattenAppsyncHTTPDataSourceConfig(resp.DataSource.HttpConfig)); err != nil {
		return fmt.Errorf("error setting http_config: %s", err)
	}

	if err := d.Set("lambda_config", flattenAppsyncLambdaDataSourceConfig(resp.DataSource.LambdaConfig)); err != nil {
		return fmt.Errorf("error setting lambda_config: %s", err)
	}

	d.Set("name", resp.DataSource.Name)
	d.Set("service_role_arn", resp.DataSource.ServiceRoleArn)
	d.Set("type", resp.DataSource.Type)

	return nil
}

func resourceAwsAppsyncDatasourceUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appsyncconn
	region := meta.(*AWSClient).region

	apiID, name, err := decodeAppsyncDataSourceID(d.Id())

	if err != nil {
		return err
	}

	input := &appsync.UpdateDataSourceInput{
		ApiId: aws.String(apiID),
		Name:  aws.String(name),
		Type:  aws.String(d.Get("type").(string)),
	}

	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}

	if v, ok := d.GetOk("dynamodb_config"); ok {
		input.DynamodbConfig = expandAppsyncDynamodbDataSourceConfig(v.([]interface{}), region)
	}

	if v, ok := d.GetOk("elasticsearch_config"); ok {
		input.ElasticsearchConfig = expandAppsyncElasticsearchDataSourceConfig(v.([]interface{}), region)
	}

	if v, ok := d.GetOk("http_config"); ok {
		input.HttpConfig = expandAppsyncHTTPDataSourceConfig(v.([]interface{}))
	}

	if v, ok := d.GetOk("lambda_config"); ok {
		input.LambdaConfig = expandAppsyncLambdaDataSourceConfig(v.([]interface{}))
	}

	if v, ok := d.GetOk("service_role_arn"); ok {
		input.ServiceRoleArn = aws.String(v.(string))
	}

	_, err = conn.UpdateDataSource(input)
	if err != nil {
		return err
	}
	return resourceAwsAppsyncDatasourceRead(d, meta)
}

func resourceAwsAppsyncDatasourceDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appsyncconn

	apiID, name, err := decodeAppsyncDataSourceID(d.Id())

	if err != nil {
		return err
	}

	input := &appsync.DeleteDataSourceInput{
		ApiId: aws.String(apiID),
		Name:  aws.String(name),
	}

	_, err = conn.DeleteDataSource(input)
	if err != nil {
		if isAWSErr(err, appsync.ErrCodeNotFoundException, "") {
			return nil
		}
		return err
	}

	return nil
}

func decodeAppsyncDataSourceID(id string) (string, string, error) {
	idParts := strings.SplitN(id, "-", 2)
	if len(idParts) != 2 {
		return "", "", fmt.Errorf("expected ID in format ApiID-DataSourceName, received: %s", id)
	}
	return idParts[0], idParts[1], nil
}

func expandAppsyncDynamodbDataSourceConfig(l []interface{}, currentRegion string) *appsync.DynamodbDataSourceConfig {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	configured := l[0].(map[string]interface{})

	result := &appsync.DynamodbDataSourceConfig{
		AwsRegion: aws.String(currentRegion),
		TableName: aws.String(configured["table_name"].(string)),
	}

	if v, ok := configured["region"]; ok && v.(string) != "" {
		result.AwsRegion = aws.String(v.(string))
	}

	if v, ok := configured["use_caller_credentials"]; ok {
		result.UseCallerCredentials = aws.Bool(v.(bool))
	}

	return result
}

func flattenAppsyncDynamodbDataSourceConfig(config *appsync.DynamodbDataSourceConfig) []map[string]interface{} {
	if config == nil {
		return nil
	}

	result := map[string]interface{}{
		"region":     aws.StringValue(config.AwsRegion),
		"table_name": aws.StringValue(config.TableName),
	}

	if config.UseCallerCredentials != nil {
		result["use_caller_credentials"] = aws.BoolValue(config.UseCallerCredentials)
	}

	return []map[string]interface{}{result}
}

func expandAppsyncElasticsearchDataSourceConfig(l []interface{}, currentRegion string) *appsync.ElasticsearchDataSourceConfig {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	configured := l[0].(map[string]interface{})

	result := &appsync.ElasticsearchDataSourceConfig{
		AwsRegion: aws.String(currentRegion),
		Endpoint:  aws.String(configured["endpoint"].(string)),
	}

	if v, ok := configured["region"]; ok && v.(string) != "" {
		result.AwsRegion = aws.String(v.(string))
	}

	return result
}

func flattenAppsyncElasticsearchDataSourceConfig(config *appsync.ElasticsearchDataSourceConfig) []map[string]interface{} {
	if config == nil {
		return nil
	}

	result := map[string]interface{}{
		"endpoint": aws.StringValue(config.Endpoint),
		"region":   aws.StringValue(config.AwsRegion),
	}

	return []map[string]interface{}{result}
}

func expandAppsyncHTTPDataSourceConfig(l []interface{}) *appsync.HttpDataSourceConfig {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	configured := l[0].(map[string]interface{})

	result := &appsync.HttpDataSourceConfig{
		Endpoint: aws.String(configured["endpoint"].(string)),
	}

	return result
}

func flattenAppsyncHTTPDataSourceConfig(config *appsync.HttpDataSourceConfig) []map[string]interface{} {
	if config == nil {
		return nil
	}

	result := map[string]interface{}{
		"endpoint": aws.StringValue(config.Endpoint),
	}

	return []map[string]interface{}{result}
}

func expandAppsyncLambdaDataSourceConfig(l []interface{}) *appsync.LambdaDataSourceConfig {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	configured := l[0].(map[string]interface{})

	result := &appsync.LambdaDataSourceConfig{
		LambdaFunctionArn: aws.String(configured["function_arn"].(string)),
	}

	return result
}

func flattenAppsyncLambdaDataSourceConfig(config *appsync.LambdaDataSourceConfig) []map[string]interface{} {
	if config == nil {
		return nil
	}

	result := map[string]interface{}{
		"function_arn": aws.StringValue(config.LambdaFunctionArn),
	}

	return []map[string]interface{}{result}
}
