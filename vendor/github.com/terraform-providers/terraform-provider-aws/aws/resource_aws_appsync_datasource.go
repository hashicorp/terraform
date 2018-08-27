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
							Required: true,
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
				ConflictsWith: []string{"elasticsearch_config", "lambda_config"},
			},
			"elasticsearch_config": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"region": {
							Type:     schema.TypeString,
							Required: true,
						},
						"endpoint": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				ConflictsWith: []string{"dynamodb_config", "lambda_config"},
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
				ConflictsWith: []string{"dynamodb_config", "elasticsearch_config"},
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

	input := &appsync.CreateDataSourceInput{
		ApiId: aws.String(d.Get("api_id").(string)),
		Name:  aws.String(d.Get("name").(string)),
		Type:  aws.String(d.Get("type").(string)),
	}

	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}

	if v, ok := d.GetOk("service_role_arn"); ok {
		input.ServiceRoleArn = aws.String(v.(string))
	}

	ddbconfig := d.Get("dynamodb_config").([]interface{})
	if len(ddbconfig) > 0 {
		input.DynamodbConfig = expandAppsyncDynamodbDataSourceConfig(ddbconfig[0].(map[string]interface{}))
	}
	esconfig := d.Get("elasticsearch_config").([]interface{})
	if len(esconfig) > 0 {
		input.ElasticsearchConfig = expandAppsyncElasticsearchDataSourceConfig(esconfig[0].(map[string]interface{}))
	}
	lambdaconfig := d.Get("lambda_config").([]interface{})
	if len(lambdaconfig) > 0 {
		input.LambdaConfig = expandAppsyncLambdaDataSourceConfig(lambdaconfig[0].(map[string]interface{}))
	}

	resp, err := conn.CreateDataSource(input)
	if err != nil {
		return err
	}

	d.SetId(d.Get("api_id").(string) + "-" + d.Get("name").(string))
	d.Set("arn", resp.DataSource.DataSourceArn)
	return nil
}

func resourceAwsAppsyncDatasourceRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appsyncconn

	input := &appsync.GetDataSourceInput{
		ApiId: aws.String(d.Get("api_id").(string)),
		Name:  aws.String(d.Get("name").(string)),
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

	d.Set("arn", resp.DataSource.DataSourceArn)
	d.Set("description", resp.DataSource.Description)
	d.Set("dynamodb_config", flattenAppsyncDynamodbDataSourceConfig(resp.DataSource.DynamodbConfig))
	d.Set("elasticsearch_config", flattenAppsyncElasticsearchDataSourceConfig(resp.DataSource.ElasticsearchConfig))
	d.Set("lambda_config", flattenAppsyncLambdaDataSourceConfig(resp.DataSource.LambdaConfig))
	d.Set("name", resp.DataSource.Name)
	d.Set("service_role_arn", resp.DataSource.ServiceRoleArn)
	d.Set("type", resp.DataSource.Type)

	return nil
}

func resourceAwsAppsyncDatasourceUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appsyncconn

	input := &appsync.UpdateDataSourceInput{
		ApiId: aws.String(d.Get("api_id").(string)),
		Name:  aws.String(d.Get("name").(string)),
		Type:  aws.String(d.Get("type").(string)),
	}

	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}

	if v, ok := d.GetOk("service_role_arn"); ok {
		input.ServiceRoleArn = aws.String(v.(string))
	}

	ddbconfig := d.Get("dynamodb_config").([]interface{})
	if len(ddbconfig) > 0 {
		input.DynamodbConfig = expandAppsyncDynamodbDataSourceConfig(ddbconfig[0].(map[string]interface{}))
	}
	esconfig := d.Get("elasticsearch_config").([]interface{})
	if len(esconfig) > 0 {
		input.ElasticsearchConfig = expandAppsyncElasticsearchDataSourceConfig(esconfig[0].(map[string]interface{}))
	}
	lambdaconfig := d.Get("lambda_config").([]interface{})
	if len(lambdaconfig) > 0 {
		input.LambdaConfig = expandAppsyncLambdaDataSourceConfig(lambdaconfig[0].(map[string]interface{}))
	}

	_, err := conn.UpdateDataSource(input)
	if err != nil {
		return err
	}
	return resourceAwsAppsyncDatasourceRead(d, meta)
}

func resourceAwsAppsyncDatasourceDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appsyncconn

	input := &appsync.DeleteDataSourceInput{
		ApiId: aws.String(d.Get("api_id").(string)),
		Name:  aws.String(d.Get("name").(string)),
	}

	_, err := conn.DeleteDataSource(input)
	if err != nil {
		if isAWSErr(err, appsync.ErrCodeNotFoundException, "") {
			return nil
		}
		return err
	}

	return nil
}

func expandAppsyncDynamodbDataSourceConfig(configured map[string]interface{}) *appsync.DynamodbDataSourceConfig {
	result := &appsync.DynamodbDataSourceConfig{
		AwsRegion: aws.String(configured["region"].(string)),
		TableName: aws.String(configured["table_name"].(string)),
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

	result := map[string]interface{}{}

	result["region"] = *config.AwsRegion
	result["table_name"] = *config.TableName
	if config.UseCallerCredentials != nil {
		result["use_caller_credentials"] = *config.UseCallerCredentials
	}

	return []map[string]interface{}{result}
}

func expandAppsyncElasticsearchDataSourceConfig(configured map[string]interface{}) *appsync.ElasticsearchDataSourceConfig {
	result := &appsync.ElasticsearchDataSourceConfig{
		AwsRegion: aws.String(configured["region"].(string)),
		Endpoint:  aws.String(configured["endpoint"].(string)),
	}

	return result
}

func flattenAppsyncElasticsearchDataSourceConfig(config *appsync.ElasticsearchDataSourceConfig) []map[string]interface{} {
	if config == nil {
		return nil
	}

	result := map[string]interface{}{}

	result["region"] = *config.AwsRegion
	result["endpoint"] = *config.Endpoint

	return []map[string]interface{}{result}
}

func expandAppsyncLambdaDataSourceConfig(configured map[string]interface{}) *appsync.LambdaDataSourceConfig {
	result := &appsync.LambdaDataSourceConfig{
		LambdaFunctionArn: aws.String(configured["function_arn"].(string)),
	}

	return result
}

func flattenAppsyncLambdaDataSourceConfig(config *appsync.LambdaDataSourceConfig) []map[string]interface{} {
	if config == nil {
		return nil
	}

	result := map[string]interface{}{}

	result["function_arn"] = *config.LambdaFunctionArn

	return []map[string]interface{}{result}
}
