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
					appsync.AuthenticationTypeOpenidConnect,
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
			"log_config": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cloudwatch_logs_role_arn": {
							Type:     schema.TypeString,
							Required: true,
						},
						"field_log_level": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								appsync.FieldLogLevelAll,
								appsync.FieldLogLevelError,
								appsync.FieldLogLevelNone,
							}, false),
						},
					},
				},
			},
			"openid_connect_config": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"auth_ttl": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"client_id": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"iat_ttl": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"issuer": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
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
							Optional: true,
							Computed: true,
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
			"uris": {
				Type:     schema.TypeMap,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
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

	if v, ok := d.GetOk("log_config"); ok {
		input.LogConfig = expandAppsyncGraphqlApiLogConfig(v.([]interface{}))
	}

	if v, ok := d.GetOk("openid_connect_config"); ok {
		input.OpenIDConnectConfig = expandAppsyncGraphqlApiOpenIDConnectConfig(v.([]interface{}))
	}

	if v, ok := d.GetOk("user_pool_config"); ok {
		input.UserPoolConfig = expandAppsyncGraphqlApiUserPoolConfig(v.([]interface{}), meta.(*AWSClient).region)
	}

	resp, err := conn.CreateGraphqlApi(input)
	if err != nil {
		return err
	}

	d.SetId(*resp.GraphqlApi.ApiId)

	return resourceAwsAppsyncGraphqlApiRead(d, meta)
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

	d.Set("arn", resp.GraphqlApi.Arn)
	d.Set("authentication_type", resp.GraphqlApi.AuthenticationType)
	d.Set("name", resp.GraphqlApi.Name)

	if err := d.Set("log_config", flattenAppsyncGraphqlApiLogConfig(resp.GraphqlApi.LogConfig)); err != nil {
		return fmt.Errorf("error setting log_config: %s", err)
	}

	if err := d.Set("openid_connect_config", flattenAppsyncGraphqlApiOpenIDConnectConfig(resp.GraphqlApi.OpenIDConnectConfig)); err != nil {
		return fmt.Errorf("error setting openid_connect_config: %s", err)
	}

	if err := d.Set("user_pool_config", flattenAppsyncGraphqlApiUserPoolConfig(resp.GraphqlApi.UserPoolConfig)); err != nil {
		return fmt.Errorf("error setting user_pool_config: %s", err)
	}

	if err := d.Set("uris", aws.StringValueMap(resp.GraphqlApi.Uris)); err != nil {
		return fmt.Errorf("error setting uris")
	}

	return nil
}

func resourceAwsAppsyncGraphqlApiUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appsyncconn

	input := &appsync.UpdateGraphqlApiInput{
		ApiId:              aws.String(d.Id()),
		AuthenticationType: aws.String(d.Get("authentication_type").(string)),
		Name:               aws.String(d.Get("name").(string)),
	}

	if v, ok := d.GetOk("log_config"); ok {
		input.LogConfig = expandAppsyncGraphqlApiLogConfig(v.([]interface{}))
	}

	if v, ok := d.GetOk("openid_connect_config"); ok {
		input.OpenIDConnectConfig = expandAppsyncGraphqlApiOpenIDConnectConfig(v.([]interface{}))
	}

	if v, ok := d.GetOk("user_pool_config"); ok {
		input.UserPoolConfig = expandAppsyncGraphqlApiUserPoolConfig(v.([]interface{}), meta.(*AWSClient).region)
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

func expandAppsyncGraphqlApiLogConfig(l []interface{}) *appsync.LogConfig {
	if len(l) < 1 || l[0] == nil {
		return nil
	}

	m := l[0].(map[string]interface{})

	logConfig := &appsync.LogConfig{
		CloudWatchLogsRoleArn: aws.String(m["cloudwatch_logs_role_arn"].(string)),
		FieldLogLevel:         aws.String(m["field_log_level"].(string)),
	}

	return logConfig
}

func expandAppsyncGraphqlApiOpenIDConnectConfig(l []interface{}) *appsync.OpenIDConnectConfig {
	if len(l) < 1 || l[0] == nil {
		return nil
	}

	m := l[0].(map[string]interface{})

	openIDConnectConfig := &appsync.OpenIDConnectConfig{
		Issuer: aws.String(m["issuer"].(string)),
	}

	if v, ok := m["auth_ttl"].(int); ok && v != 0 {
		openIDConnectConfig.AuthTTL = aws.Int64(int64(v))
	}

	if v, ok := m["client_id"].(string); ok && v != "" {
		openIDConnectConfig.ClientId = aws.String(v)
	}

	if v, ok := m["iat_ttl"].(int); ok && v != 0 {
		openIDConnectConfig.IatTTL = aws.Int64(int64(v))
	}

	return openIDConnectConfig
}

func expandAppsyncGraphqlApiUserPoolConfig(l []interface{}, currentRegion string) *appsync.UserPoolConfig {
	if len(l) < 1 || l[0] == nil {
		return nil
	}

	m := l[0].(map[string]interface{})

	userPoolConfig := &appsync.UserPoolConfig{
		AwsRegion:     aws.String(currentRegion),
		DefaultAction: aws.String(m["default_action"].(string)),
		UserPoolId:    aws.String(m["user_pool_id"].(string)),
	}

	if v, ok := m["app_id_client_regex"].(string); ok && v != "" {
		userPoolConfig.AppIdClientRegex = aws.String(v)
	}

	if v, ok := m["aws_region"].(string); ok && v != "" {
		userPoolConfig.AwsRegion = aws.String(v)
	}

	return userPoolConfig
}

func flattenAppsyncGraphqlApiLogConfig(logConfig *appsync.LogConfig) []interface{} {
	if logConfig == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"cloudwatch_logs_role_arn": aws.StringValue(logConfig.CloudWatchLogsRoleArn),
		"field_log_level":          aws.StringValue(logConfig.FieldLogLevel),
	}

	return []interface{}{m}
}

func flattenAppsyncGraphqlApiOpenIDConnectConfig(openIDConnectConfig *appsync.OpenIDConnectConfig) []interface{} {
	if openIDConnectConfig == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"auth_ttl":  aws.Int64Value(openIDConnectConfig.AuthTTL),
		"client_id": aws.StringValue(openIDConnectConfig.ClientId),
		"iat_ttl":   aws.Int64Value(openIDConnectConfig.IatTTL),
		"issuer":    aws.StringValue(openIDConnectConfig.Issuer),
	}

	return []interface{}{m}
}

func flattenAppsyncGraphqlApiUserPoolConfig(userPoolConfig *appsync.UserPoolConfig) []interface{} {
	if userPoolConfig == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"aws_region":     aws.StringValue(userPoolConfig.AwsRegion),
		"default_action": aws.StringValue(userPoolConfig.DefaultAction),
		"user_pool_id":   aws.StringValue(userPoolConfig.UserPoolId),
	}

	if userPoolConfig.AppIdClientRegex != nil {
		m["app_id_client_regex"] = aws.StringValue(userPoolConfig.AppIdClientRegex)
	}

	return []interface{}{m}
}
