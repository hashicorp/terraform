package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsLambdaAlias() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsLambdaAliasCreate,
		Read:   resourceAwsLambdaAliasRead,
		Update: resourceAwsLambdaAliasUpdate,
		Delete: resourceAwsLambdaAliasDelete,

		Schema: map[string]*schema.Schema{
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"function_name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"function_version": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"invoke_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"routing_config": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"additional_version_weights": {
							Type:     schema.TypeMap,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeFloat},
						},
					},
				},
			},
		},
	}
}

// resourceAwsLambdaAliasCreate maps to:
// CreateAlias in the API / SDK
func resourceAwsLambdaAliasCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lambdaconn

	functionName := d.Get("function_name").(string)
	aliasName := d.Get("name").(string)

	log.Printf("[DEBUG] Creating Lambda alias: alias %s for function %s", aliasName, functionName)

	params := &lambda.CreateAliasInput{
		Description:     aws.String(d.Get("description").(string)),
		FunctionName:    aws.String(functionName),
		FunctionVersion: aws.String(d.Get("function_version").(string)),
		Name:            aws.String(aliasName),
		RoutingConfig:   expandLambdaAliasRoutingConfiguration(d.Get("routing_config").([]interface{})),
	}

	aliasConfiguration, err := conn.CreateAlias(params)
	if err != nil {
		return fmt.Errorf("Error creating Lambda alias: %s", err)
	}

	d.SetId(*aliasConfiguration.AliasArn)

	return resourceAwsLambdaAliasRead(d, meta)
}

// resourceAwsLambdaAliasRead maps to:
// GetAlias in the API / SDK
func resourceAwsLambdaAliasRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lambdaconn

	log.Printf("[DEBUG] Fetching Lambda alias: %s:%s", d.Get("function_name"), d.Get("name"))

	params := &lambda.GetAliasInput{
		FunctionName: aws.String(d.Get("function_name").(string)),
		Name:         aws.String(d.Get("name").(string)),
	}

	aliasConfiguration, err := conn.GetAlias(params)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "ResourceNotFoundException" && strings.Contains(awsErr.Message(), "Cannot find alias arn") {
				d.SetId("")
				return nil
			}
		}
		return err
	}

	d.Set("description", aliasConfiguration.Description)
	d.Set("function_version", aliasConfiguration.FunctionVersion)
	d.Set("name", aliasConfiguration.Name)
	d.Set("arn", aliasConfiguration.AliasArn)

	invokeArn := lambdaFunctionInvokeArn(*aliasConfiguration.AliasArn, meta)
	d.Set("invoke_arn", invokeArn)

	if err := d.Set("routing_config", flattenLambdaAliasRoutingConfiguration(aliasConfiguration.RoutingConfig)); err != nil {
		return fmt.Errorf("error setting routing_config: %s", err)
	}

	return nil
}

// resourceAwsLambdaAliasDelete maps to:
// DeleteAlias in the API / SDK
func resourceAwsLambdaAliasDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lambdaconn

	log.Printf("[INFO] Deleting Lambda alias: %s:%s", d.Get("function_name"), d.Get("name"))

	params := &lambda.DeleteAliasInput{
		FunctionName: aws.String(d.Get("function_name").(string)),
		Name:         aws.String(d.Get("name").(string)),
	}

	_, err := conn.DeleteAlias(params)
	if err != nil {
		return fmt.Errorf("Error deleting Lambda alias: %s", err)
	}

	return nil
}

// resourceAwsLambdaAliasUpdate maps to:
// UpdateAlias in the API / SDK
func resourceAwsLambdaAliasUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lambdaconn

	log.Printf("[DEBUG] Updating Lambda alias: %s:%s", d.Get("function_name"), d.Get("name"))

	params := &lambda.UpdateAliasInput{
		Description:     aws.String(d.Get("description").(string)),
		FunctionName:    aws.String(d.Get("function_name").(string)),
		FunctionVersion: aws.String(d.Get("function_version").(string)),
		Name:            aws.String(d.Get("name").(string)),
		RoutingConfig:   expandLambdaAliasRoutingConfiguration(d.Get("routing_config").([]interface{})),
	}

	_, err := conn.UpdateAlias(params)
	if err != nil {
		return fmt.Errorf("Error updating Lambda alias: %s", err)
	}

	return nil
}

func expandLambdaAliasRoutingConfiguration(l []interface{}) *lambda.AliasRoutingConfiguration {
	aliasRoutingConfiguration := &lambda.AliasRoutingConfiguration{}

	if len(l) == 0 || l[0] == nil {
		return aliasRoutingConfiguration
	}

	m := l[0].(map[string]interface{})

	if v, ok := m["additional_version_weights"]; ok {
		aliasRoutingConfiguration.AdditionalVersionWeights = expandFloat64Map(v.(map[string]interface{}))
	}

	return aliasRoutingConfiguration
}
