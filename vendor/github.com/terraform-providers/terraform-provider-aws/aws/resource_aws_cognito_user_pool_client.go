package aws

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsCognitoUserPoolClient() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCognitoUserPoolClientCreate,
		Read:   resourceAwsCognitoUserPoolClientRead,
		Update: resourceAwsCognitoUserPoolClientUpdate,
		Delete: resourceAwsCognitoUserPoolClientDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		// https://docs.aws.amazon.com/cognito-user-identity-pools/latest/APIReference/API_CreateUserPoolClient.html
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"client_secret": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"generate_secret": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"user_pool_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"explicit_auth_flows": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{
						cognitoidentityprovider.ExplicitAuthFlowsTypeAdminNoSrpAuth,
						cognitoidentityprovider.ExplicitAuthFlowsTypeCustomAuthFlowOnly,
						cognitoidentityprovider.ExplicitAuthFlowsTypeUserPasswordAuth,
					}, false),
				},
			},

			"read_attributes": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"write_attributes": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"refresh_token_validity": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      30,
				ValidateFunc: validateIntegerInRange(0, 3650),
			},

			"allowed_oauth_flows": {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 3,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{
						cognitoidentityprovider.OAuthFlowTypeCode,
						cognitoidentityprovider.OAuthFlowTypeImplicit,
						cognitoidentityprovider.OAuthFlowTypeClientCredentials,
					}, false),
				},
			},

			"allowed_oauth_flows_user_pool_client": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			"allowed_oauth_scopes": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			// TODO: analytics_configuration

			"callback_urls": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 100,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validateCognitoUserPoolClientURL,
				},
			},

			"default_redirect_uri": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"logout_urls": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 100,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validateCognitoUserPoolClientURL,
				},
			},

			"supported_identity_providers": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func resourceAwsCognitoUserPoolClientCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cognitoidpconn

	params := &cognitoidentityprovider.CreateUserPoolClientInput{
		ClientName: aws.String(d.Get("name").(string)),
		UserPoolId: aws.String(d.Get("user_pool_id").(string)),
	}

	if v, ok := d.GetOk("generate_secret"); ok {
		params.GenerateSecret = aws.Bool(v.(bool))
	}

	if v, ok := d.GetOk("explicit_auth_flows"); ok {
		params.ExplicitAuthFlows = expandStringList(v.(*schema.Set).List())
	}

	if v, ok := d.GetOk("read_attributes"); ok {
		params.ReadAttributes = expandStringList(v.(*schema.Set).List())
	}

	if v, ok := d.GetOk("write_attributes"); ok {
		params.WriteAttributes = expandStringList(v.(*schema.Set).List())
	}

	if v, ok := d.GetOk("refresh_token_validity"); ok {
		params.RefreshTokenValidity = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("allowed_oauth_flows"); ok {
		params.AllowedOAuthFlows = expandStringList(v.(*schema.Set).List())
	}

	if v, ok := d.GetOk("allowed_oauth_flows_user_pool_client"); ok {
		params.AllowedOAuthFlowsUserPoolClient = aws.Bool(v.(bool))
	}

	if v, ok := d.GetOk("allowed_oauth_scopes"); ok {
		params.AllowedOAuthScopes = expandStringList(v.(*schema.Set).List())
	}

	if v, ok := d.GetOk("callback_urls"); ok {
		params.CallbackURLs = expandStringList(v.([]interface{}))
	}

	if v, ok := d.GetOk("default_redirect_uri"); ok {
		params.DefaultRedirectURI = aws.String(v.(string))
	}

	if v, ok := d.GetOk("logout_urls"); ok {
		params.LogoutURLs = expandStringList(v.([]interface{}))
	}

	if v, ok := d.GetOk("supported_identity_providers"); ok {
		params.SupportedIdentityProviders = expandStringList(v.([]interface{}))
	}

	log.Printf("[DEBUG] Creating Cognito User Pool Client: %s", params)

	resp, err := conn.CreateUserPoolClient(params)

	if err != nil {
		return errwrap.Wrapf("Error creating Cognito User Pool Client: {{err}}", err)
	}

	d.SetId(*resp.UserPoolClient.ClientId)

	return resourceAwsCognitoUserPoolClientRead(d, meta)
}

func resourceAwsCognitoUserPoolClientRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cognitoidpconn

	params := &cognitoidentityprovider.DescribeUserPoolClientInput{
		ClientId:   aws.String(d.Id()),
		UserPoolId: aws.String(d.Get("user_pool_id").(string)),
	}

	log.Printf("[DEBUG] Reading Cognito User Pool Client: %s", params)

	resp, err := conn.DescribeUserPoolClient(params)

	if err != nil {
		if isAWSErr(err, "ResourceNotFoundException", "") {
			log.Printf("[WARN] Cognito User Pool Client %s is already gone", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	d.SetId(*resp.UserPoolClient.ClientId)
	d.Set("user_pool_id", *resp.UserPoolClient.UserPoolId)
	d.Set("name", *resp.UserPoolClient.ClientName)
	d.Set("explicit_auth_flows", flattenStringList(resp.UserPoolClient.ExplicitAuthFlows))
	d.Set("read_attributes", flattenStringList(resp.UserPoolClient.ReadAttributes))
	d.Set("write_attributes", flattenStringList(resp.UserPoolClient.WriteAttributes))
	d.Set("refresh_token_validity", resp.UserPoolClient.RefreshTokenValidity)
	d.Set("client_secret", resp.UserPoolClient.ClientSecret)
	d.Set("allowed_oauth_flows", flattenStringList(resp.UserPoolClient.AllowedOAuthFlows))
	d.Set("allowed_oauth_flows_user_pool_client", resp.UserPoolClient.AllowedOAuthFlowsUserPoolClient)
	d.Set("allowed_oauth_scopes", flattenStringList(resp.UserPoolClient.AllowedOAuthScopes))
	d.Set("callback_urls", flattenStringList(resp.UserPoolClient.CallbackURLs))
	d.Set("default_redirect_uri", resp.UserPoolClient.DefaultRedirectURI)
	d.Set("logout_urls", flattenStringList(resp.UserPoolClient.LogoutURLs))
	d.Set("supported_identity_providers", flattenStringList(resp.UserPoolClient.SupportedIdentityProviders))

	return nil
}

func resourceAwsCognitoUserPoolClientUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cognitoidpconn

	params := &cognitoidentityprovider.UpdateUserPoolClientInput{
		ClientId:   aws.String(d.Id()),
		UserPoolId: aws.String(d.Get("user_pool_id").(string)),
	}

	if d.HasChange("explicit_auth_flows") {
		params.ExplicitAuthFlows = expandStringList(d.Get("explicit_auth_flows").(*schema.Set).List())
	}

	if d.HasChange("read_attributes") {
		params.ReadAttributes = expandStringList(d.Get("read_attributes").(*schema.Set).List())
	}

	if d.HasChange("write_attributes") {
		params.WriteAttributes = expandStringList(d.Get("write_attributes").(*schema.Set).List())
	}

	if d.HasChange("refresh_token_validity") {
		params.RefreshTokenValidity = aws.Int64(d.Get("refresh_token_validity").(int64))
	}

	if d.HasChange("allowed_oauth_flows") {
		params.AllowedOAuthFlows = expandStringList(d.Get("allowed_oauth_flows").(*schema.Set).List())
	}

	if d.HasChange("allowed_oauth_flows_user_pool_client") {
		params.AllowedOAuthFlowsUserPoolClient = aws.Bool(d.Get("allowed_oauth_flows_user_pool_client").(bool))
	}

	if d.HasChange("allowed_oauth_scopes") {
		params.AllowedOAuthScopes = expandStringList(d.Get("allowed_oauth_scopes").(*schema.Set).List())
	}

	if d.HasChange("callback_urls") {
		params.ReadAttributes = expandStringList(d.Get("callback_urls").([]interface{}))
	}

	if d.HasChange("default_redirect_uri") {
		params.DefaultRedirectURI = aws.String(d.Get("default_redirect_uri").(string))
	}

	if d.HasChange("logout_urls") {
		params.LogoutURLs = expandStringList(d.Get("logout_urls").([]interface{}))
	}

	if d.HasChange("supported_identity_providers") {
		params.SupportedIdentityProviders = expandStringList(d.Get("supported_identity_providers").([]interface{}))
	}

	log.Printf("[DEBUG] Updating Cognito User Pool Client: %s", params)

	_, err := conn.UpdateUserPoolClient(params)
	if err != nil {
		return errwrap.Wrapf("Error updating Cognito User Pool Client: {{err}}", err)
	}

	return resourceAwsCognitoUserPoolClientRead(d, meta)
}

func resourceAwsCognitoUserPoolClientDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cognitoidpconn

	params := &cognitoidentityprovider.DeleteUserPoolClientInput{
		ClientId:   aws.String(d.Id()),
		UserPoolId: aws.String(d.Get("user_pool_id").(string)),
	}

	log.Printf("[DEBUG] Deleting Cognito User Pool Client: %s", params)

	_, err := conn.DeleteUserPoolClient(params)

	if err != nil {
		return errwrap.Wrapf("Error deleting Cognito User Pool Client: {{err}}", err)
	}

	return nil
}
