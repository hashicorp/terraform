package aws

import (
	"fmt"
	"log"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/appsync"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsAppsyncGraphqlApi() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsAppsyncGraphqlApiCreate,
		Read:   resourceAwsAppsyncGraphqlApiRead,
		Update: resourceAwsAppsyncGraphqlApiUpdate,
		Delete: resourceAwsAppsyncGraphqlApiDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"authentication_type": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validation.StringInSlice([]string{
					appsync.AuthenticationTypeApiKey,
					appsync.AuthenticationTypeAwsIam,
					appsync.AuthenticationTypeAmazonCognitoUserPools,
				}, false),
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
			"user_pool_config": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"app_id_client_regex": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"aws_region": {
							Type:     schema.TypeString,
							Required: true,
						},
						"default_action": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								appsync.DefaultActionAllow,
								appsync.DefaultActionDeny,
							}, false),
						},
						"user_pool_id": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsAppsyncGraphqlApiCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appsyncconn

	input := &appsync.CreateGraphqlApiInput{
		AuthenticationType: aws.String(d.Get("authentication_type").(string)),
		Name:               aws.String(d.Get("name").(string)),
	}

	if v, ok := d.GetOk("user_pool_config"); ok {
		input.UserPoolConfig = expandAppsyncGraphqlApiUserPoolConfig(v.([]interface{}))
	}

	resp, err := conn.CreateGraphqlApi(input)
	if err != nil {
		return err
	}

	d.SetId(*resp.GraphqlApi.ApiId)
	d.Set("arn", resp.GraphqlApi.Arn)
	return nil
}

func resourceAwsAppsyncGraphqlApiRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appsyncconn

	input := &appsync.GetGraphqlApiInput{
		ApiId: aws.String(d.Id()),
	}

	resp, err := conn.GetGraphqlApi(input)
	if err != nil {
		if isAWSErr(err, appsync.ErrCodeNotFoundException, "") {
			log.Printf("[WARN] No such entity found for Appsync Graphql API (%s)", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("authentication_type", resp.GraphqlApi.AuthenticationType)
	d.Set("name", resp.GraphqlApi.Name)
	d.Set("user_pool_config", flattenAppsyncGraphqlApiUserPoolConfig(resp.GraphqlApi.UserPoolConfig))
	d.Set("arn", resp.GraphqlApi.Arn)
	return nil
}

func resourceAwsAppsyncGraphqlApiUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appsyncconn

	input := &appsync.UpdateGraphqlApiInput{
		ApiId: aws.String(d.Id()),
		Name:  aws.String(d.Get("name").(string)),
	}

	if d.HasChange("authentication_type") {
		input.AuthenticationType = aws.String(d.Get("authentication_type").(string))
	}
	if d.HasChange("user_pool_config") {
		input.UserPoolConfig = expandAppsyncGraphqlApiUserPoolConfig(d.Get("user_pool_config").([]interface{}))
	}

	_, err := conn.UpdateGraphqlApi(input)
	if err != nil {
		return err
	}

	return resourceAwsAppsyncGraphqlApiRead(d, meta)
}

func resourceAwsAppsyncGraphqlApiDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appsyncconn

	input := &appsync.DeleteGraphqlApiInput{
		ApiId: aws.String(d.Id()),
	}
	_, err := conn.DeleteGraphqlApi(input)
	if err != nil {
		if isAWSErr(err, appsync.ErrCodeNotFoundException, "") {
			return nil
		}
		return err
	}

	return nil
}

func expandAppsyncGraphqlApiUserPoolConfig(config []interface{}) *appsync.UserPoolConfig {
	if len(config) < 1 {
		return nil
	}
	cg := config[0].(map[string]interface{})
	upc := &appsync.UserPoolConfig{
		AwsRegion:     aws.String(cg["aws_region"].(string)),
		DefaultAction: aws.String(cg["default_action"].(string)),
		UserPoolId:    aws.String(cg["user_pool_id"].(string)),
	}
	if v, ok := cg["app_id_client_regex"].(string); ok && v != "" {
		upc.AppIdClientRegex = aws.String(v)
	}
	return upc
}

func flattenAppsyncGraphqlApiUserPoolConfig(upc *appsync.UserPoolConfig) []interface{} {
	if upc == nil {
		return []interface{}{}
	}
	m := make(map[string]interface{}, 1)

	m["aws_region"] = *upc.AwsRegion
	m["default_action"] = *upc.DefaultAction
	m["user_pool_id"] = *upc.UserPoolId
	if upc.AppIdClientRegex != nil {
		m["app_id_client_regex"] = *upc.AppIdClientRegex
	}

	return []interface{}{m}
}
