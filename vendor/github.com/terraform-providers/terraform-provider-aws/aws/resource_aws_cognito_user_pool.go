package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsCognitoUserPool() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCognitoUserPoolCreate,
		Read:   resourceAwsCognitoUserPoolRead,
		Update: resourceAwsCognitoUserPoolUpdate,
		Delete: resourceAwsCognitoUserPoolDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		// https://docs.aws.amazon.com/cognito-user-identity-pools/latest/APIReference/API_CreateUserPool.html
		Schema: map[string]*schema.Schema{
			"admin_create_user_config": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"allow_admin_create_user_only": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"invite_message_template": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"email_message": {
										Type:         schema.TypeString,
										Optional:     true,
										ValidateFunc: validateCognitoUserPoolInviteTemplateEmailMessage,
									},
									"email_subject": {
										Type:         schema.TypeString,
										Optional:     true,
										ValidateFunc: validateCognitoUserPoolTemplateEmailSubject,
									},
									"sms_message": {
										Type:         schema.TypeString,
										Optional:     true,
										ValidateFunc: validateCognitoUserPoolInviteTemplateSmsMessage,
									},
								},
							},
						},
						"unused_account_validity_days": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      7,
							ValidateFunc: validation.IntBetween(0, 90),
						},
					},
				},
			},

			"alias_attributes": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{
						cognitoidentityprovider.AliasAttributeTypeEmail,
						cognitoidentityprovider.AliasAttributeTypePhoneNumber,
						cognitoidentityprovider.AliasAttributeTypePreferredUsername,
					}, false),
				},
				ConflictsWith: []string{"username_attributes"},
			},

			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"auto_verified_attributes": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{
						cognitoidentityprovider.VerifiedAttributeTypePhoneNumber,
						cognitoidentityprovider.VerifiedAttributeTypeEmail,
					}, false),
				},
			},

			"creation_date": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"device_configuration": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"challenge_required_on_new_device": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"device_only_remembered_on_user_prompt": {
							Type:     schema.TypeBool,
							Optional: true,
						},
					},
				},
			},

			"email_configuration": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"reply_to_email_address": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateCognitoUserPoolReplyEmailAddress,
						},
						"source_arn": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateArn,
						},
					},
				},
			},

			"email_verification_subject": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateCognitoUserPoolEmailVerificationSubject,
			},

			"email_verification_message": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateCognitoUserPoolEmailVerificationMessage,
			},

			"endpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"lambda_config": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"create_auth_challenge": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateArn,
						},
						"custom_message": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateArn,
						},
						"define_auth_challenge": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateArn,
						},
						"post_authentication": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateArn,
						},
						"post_confirmation": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateArn,
						},
						"pre_authentication": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateArn,
						},
						"pre_sign_up": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateArn,
						},
						"pre_token_generation": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateArn,
						},
						"user_migration": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateArn,
						},
						"verify_auth_challenge_response": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateArn,
						},
					},
				},
			},

			"last_modified_date": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"mfa_configuration": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  cognitoidentityprovider.UserPoolMfaTypeOff,
				ValidateFunc: validation.StringInSlice([]string{
					cognitoidentityprovider.UserPoolMfaTypeOff,
					cognitoidentityprovider.UserPoolMfaTypeOn,
					cognitoidentityprovider.UserPoolMfaTypeOptional,
				}, false),
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"password_policy": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"minimum_length": {
							Type:         schema.TypeInt,
							Optional:     true,
							ValidateFunc: validation.IntBetween(6, 99),
						},
						"require_lowercase": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"require_numbers": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"require_symbols": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"require_uppercase": {
							Type:     schema.TypeBool,
							Optional: true,
						},
					},
				},
			},

			"schema": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				MinItems: 1,
				MaxItems: 50,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"attribute_data_type": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
							ValidateFunc: validation.StringInSlice([]string{
								cognitoidentityprovider.AttributeDataTypeString,
								cognitoidentityprovider.AttributeDataTypeNumber,
								cognitoidentityprovider.AttributeDataTypeDateTime,
								cognitoidentityprovider.AttributeDataTypeBoolean,
							}, false),
						},
						"developer_only_attribute": {
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
						},
						"mutable": {
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
						},
						"name": {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							ValidateFunc: validateCognitoUserPoolSchemaName,
						},
						"number_attribute_constraints": {
							Type:     schema.TypeList,
							Optional: true,
							ForceNew: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"min_value": {
										Type:     schema.TypeString,
										Optional: true,
										ForceNew: true,
									},
									"max_value": {
										Type:     schema.TypeString,
										Optional: true,
										ForceNew: true,
									},
								},
							},
						},
						"required": {
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
						},
						"string_attribute_constraints": {
							Type:     schema.TypeList,
							Optional: true,
							ForceNew: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"min_length": {
										Type:     schema.TypeString,
										Optional: true,
										ForceNew: true,
									},
									"max_length": {
										Type:     schema.TypeString,
										Optional: true,
										ForceNew: true,
									},
								},
							},
						},
					},
				},
			},

			"sms_authentication_message": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateCognitoUserPoolSmsAuthenticationMessage,
			},

			"sms_configuration": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"external_id": {
							Type:     schema.TypeString,
							Required: true,
						},
						"sns_caller_arn": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateArn,
						},
					},
				},
			},

			"sms_verification_message": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateCognitoUserPoolSmsVerificationMessage,
			},

			"tags": tagsSchema(),

			"username_attributes": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{
						cognitoidentityprovider.UsernameAttributeTypeEmail,
						cognitoidentityprovider.UsernameAttributeTypePhoneNumber,
					}, false),
				},
				ConflictsWith: []string{"alias_attributes"},
			},

			"verification_message_template": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"default_email_option": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  cognitoidentityprovider.DefaultEmailOptionTypeConfirmWithCode,
							ValidateFunc: validation.StringInSlice([]string{
								cognitoidentityprovider.DefaultEmailOptionTypeConfirmWithLink,
								cognitoidentityprovider.DefaultEmailOptionTypeConfirmWithCode,
							}, false),
						},
						"email_message": {
							Type:         schema.TypeString,
							Optional:     true,
							Computed:     true,
							ValidateFunc: validateCognitoUserPoolTemplateEmailMessage,
						},
						"email_message_by_link": {
							Type:         schema.TypeString,
							Optional:     true,
							Computed:     true,
							ValidateFunc: validateCognitoUserPoolTemplateEmailMessageByLink,
						},
						"email_subject": {
							Type:         schema.TypeString,
							Optional:     true,
							Computed:     true,
							ValidateFunc: validateCognitoUserPoolTemplateEmailSubject,
						},
						"email_subject_by_link": {
							Type:         schema.TypeString,
							Optional:     true,
							Computed:     true,
							ValidateFunc: validateCognitoUserPoolTemplateEmailSubjectByLink,
						},
						"sms_message": {
							Type:         schema.TypeString,
							Optional:     true,
							Computed:     true,
							ValidateFunc: validateCognitoUserPoolTemplateSmsMessage,
						},
					},
				},
			},
		},
	}
}

func resourceAwsCognitoUserPoolCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cognitoidpconn

	params := &cognitoidentityprovider.CreateUserPoolInput{
		PoolName: aws.String(d.Get("name").(string)),
	}

	if v, ok := d.GetOk("admin_create_user_config"); ok {
		configs := v.([]interface{})
		config, ok := configs[0].(map[string]interface{})

		if ok && config != nil {
			params.AdminCreateUserConfig = expandCognitoUserPoolAdminCreateUserConfig(config)
		}
	}

	if v, ok := d.GetOk("alias_attributes"); ok {
		params.AliasAttributes = expandStringList(v.(*schema.Set).List())
	}

	if v, ok := d.GetOk("auto_verified_attributes"); ok {
		params.AutoVerifiedAttributes = expandStringList(v.(*schema.Set).List())
	}

	if v, ok := d.GetOk("email_configuration"); ok {
		configs := v.([]interface{})
		config, ok := configs[0].(map[string]interface{})

		if ok && config != nil {
			emailConfigurationType := &cognitoidentityprovider.EmailConfigurationType{}

			if v, ok := config["reply_to_email_address"]; ok && v.(string) != "" {
				emailConfigurationType.ReplyToEmailAddress = aws.String(v.(string))
			}

			if v, ok := config["source_arn"]; ok && v.(string) != "" {
				emailConfigurationType.SourceArn = aws.String(v.(string))
			}

			params.EmailConfiguration = emailConfigurationType
		}
	}

	if v, ok := d.GetOk("admin_create_user_config"); ok {
		configs := v.([]interface{})
		config, ok := configs[0].(map[string]interface{})

		if ok && config != nil {
			params.AdminCreateUserConfig = expandCognitoUserPoolAdminCreateUserConfig(config)
		}
	}

	if v, ok := d.GetOk("device_configuration"); ok {
		configs := v.([]interface{})
		config, ok := configs[0].(map[string]interface{})

		if ok && config != nil {
			params.DeviceConfiguration = expandCognitoUserPoolDeviceConfiguration(config)
		}
	}

	if v, ok := d.GetOk("email_verification_subject"); ok {
		params.EmailVerificationSubject = aws.String(v.(string))
	}

	if v, ok := d.GetOk("email_verification_message"); ok {
		params.EmailVerificationMessage = aws.String(v.(string))
	}

	if v, ok := d.GetOk("lambda_config"); ok {
		configs := v.([]interface{})
		config, ok := configs[0].(map[string]interface{})

		if ok && config != nil {
			params.LambdaConfig = expandCognitoUserPoolLambdaConfig(config)
		}
	}

	if v, ok := d.GetOk("mfa_configuration"); ok {
		params.MfaConfiguration = aws.String(v.(string))
	}

	if v, ok := d.GetOk("password_policy"); ok {
		configs := v.([]interface{})
		config, ok := configs[0].(map[string]interface{})

		if ok && config != nil {
			policies := &cognitoidentityprovider.UserPoolPolicyType{}
			policies.PasswordPolicy = expandCognitoUserPoolPasswordPolicy(config)
			params.Policies = policies
		}
	}

	if v, ok := d.GetOk("schema"); ok {
		configs := v.(*schema.Set).List()
		params.Schema = expandCognitoUserPoolSchema(configs)
	}

	if v, ok := d.GetOk("sms_authentication_message"); ok {
		params.SmsAuthenticationMessage = aws.String(v.(string))
	}

	if v, ok := d.GetOk("sms_configuration"); ok {
		configs := v.([]interface{})
		config, ok := configs[0].(map[string]interface{})

		if ok && config != nil {
			params.SmsConfiguration = expandCognitoUserPoolSmsConfiguration(config)
		}
	}

	if v, ok := d.GetOk("username_attributes"); ok {
		params.UsernameAttributes = expandStringList(v.([]interface{}))
	}

	if v, ok := d.GetOk("verification_message_template"); ok {
		configs := v.([]interface{})
		config, ok := configs[0].(map[string]interface{})

		if ok && config != nil {
			params.VerificationMessageTemplate = expandCognitoUserPoolVerificationMessageTemplate(config)
		}
	}

	if v, ok := d.GetOk("sms_verification_message"); ok {
		params.SmsVerificationMessage = aws.String(v.(string))
	}

	if v, ok := d.GetOk("tags"); ok {
		params.UserPoolTags = tagsFromMapGeneric(v.(map[string]interface{}))
	}
	log.Printf("[DEBUG] Creating Cognito User Pool: %s", params)

	// IAM roles & policies can take some time to propagate and be attached
	// to the User Pool
	var resp *cognitoidentityprovider.CreateUserPoolOutput
	err := resource.Retry(2*time.Minute, func() *resource.RetryError {
		var err error
		resp, err = conn.CreateUserPool(params)
		if isAWSErr(err, "InvalidSmsRoleTrustRelationshipException", "Role does not have a trust relationship allowing Cognito to assume the role") {
			log.Printf("[DEBUG] Received %s, retrying CreateUserPool", err)
			return resource.RetryableError(err)
		}
		if isAWSErr(err, "InvalidSmsRoleAccessPolicyException", "Role does not have permission to publish with SNS") {
			log.Printf("[DEBUG] Received %s, retrying CreateUserPool", err)
			return resource.RetryableError(err)
		}

		return resource.NonRetryableError(err)
	})
	if err != nil {
		return fmt.Errorf("Error creating Cognito User Pool: %s", err)
	}

	d.SetId(*resp.UserPool.Id)

	return resourceAwsCognitoUserPoolRead(d, meta)
}

func resourceAwsCognitoUserPoolRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cognitoidpconn

	params := &cognitoidentityprovider.DescribeUserPoolInput{
		UserPoolId: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Reading Cognito User Pool: %s", params)

	resp, err := conn.DescribeUserPool(params)

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "ResourceNotFoundException" {
			log.Printf("[WARN] Cognito User Pool %s is already gone", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	if err := d.Set("admin_create_user_config", flattenCognitoUserPoolAdminCreateUserConfig(resp.UserPool.AdminCreateUserConfig)); err != nil {
		return fmt.Errorf("Failed setting admin_create_user_config: %s", err)
	}
	if resp.UserPool.AliasAttributes != nil {
		d.Set("alias_attributes", flattenStringList(resp.UserPool.AliasAttributes))
	}
	arn := arn.ARN{
		Partition: meta.(*AWSClient).partition,
		Region:    meta.(*AWSClient).region,
		Service:   "cognito-idp",
		AccountID: meta.(*AWSClient).accountid,
		Resource:  fmt.Sprintf("userpool/%s", d.Id()),
	}
	d.Set("arn", arn.String())
	d.Set("endpoint", fmt.Sprintf("cognito-idp.%s.amazonaws.com/%s", meta.(*AWSClient).region, d.Id()))
	d.Set("auto_verified_attributes", flattenStringList(resp.UserPool.AutoVerifiedAttributes))

	if resp.UserPool.EmailVerificationSubject != nil {
		d.Set("email_verification_subject", *resp.UserPool.EmailVerificationSubject)
	}
	if resp.UserPool.EmailVerificationMessage != nil {
		d.Set("email_verification_message", *resp.UserPool.EmailVerificationMessage)
	}
	if err := d.Set("lambda_config", flattenCognitoUserPoolLambdaConfig(resp.UserPool.LambdaConfig)); err != nil {
		return fmt.Errorf("Failed setting lambda_config: %s", err)
	}
	if resp.UserPool.MfaConfiguration != nil {
		d.Set("mfa_configuration", *resp.UserPool.MfaConfiguration)
	}
	if resp.UserPool.SmsVerificationMessage != nil {
		d.Set("sms_verification_message", *resp.UserPool.SmsVerificationMessage)
	}
	if resp.UserPool.SmsAuthenticationMessage != nil {
		d.Set("sms_authentication_message", *resp.UserPool.SmsAuthenticationMessage)
	}

	if err := d.Set("device_configuration", flattenCognitoUserPoolDeviceConfiguration(resp.UserPool.DeviceConfiguration)); err != nil {
		return fmt.Errorf("Failed setting device_configuration: %s", err)
	}

	if err := d.Set("email_configuration", flattenCognitoUserPoolEmailConfiguration(resp.UserPool.EmailConfiguration)); err != nil {
		return fmt.Errorf("Failed setting email_configuration: %s", err)
	}

	if resp.UserPool.Policies != nil && resp.UserPool.Policies.PasswordPolicy != nil {
		if err := d.Set("password_policy", flattenCognitoUserPoolPasswordPolicy(resp.UserPool.Policies.PasswordPolicy)); err != nil {
			return fmt.Errorf("Failed setting password_policy: %s", err)
		}
	}

	var configuredSchema []interface{}
	if v, ok := d.GetOk("schema"); ok {
		configuredSchema = v.(*schema.Set).List()
	}
	if err := d.Set("schema", flattenCognitoUserPoolSchema(expandCognitoUserPoolSchema(configuredSchema), resp.UserPool.SchemaAttributes)); err != nil {
		return fmt.Errorf("Failed setting schema: %s", err)
	}

	if err := d.Set("sms_configuration", flattenCognitoUserPoolSmsConfiguration(resp.UserPool.SmsConfiguration)); err != nil {
		return fmt.Errorf("Failed setting sms_configuration: %s", err)
	}

	if resp.UserPool.UsernameAttributes != nil {
		d.Set("username_attributes", flattenStringList(resp.UserPool.UsernameAttributes))
	}

	if err := d.Set("verification_message_template", flattenCognitoUserPoolVerificationMessageTemplate(resp.UserPool.VerificationMessageTemplate)); err != nil {
		return fmt.Errorf("Failed setting verification_message_template: %s", err)
	}

	d.Set("creation_date", resp.UserPool.CreationDate.Format(time.RFC3339))
	d.Set("last_modified_date", resp.UserPool.LastModifiedDate.Format(time.RFC3339))
	d.Set("name", resp.UserPool.Name)
	d.Set("tags", tagsToMapGeneric(resp.UserPool.UserPoolTags))

	return nil
}

func resourceAwsCognitoUserPoolUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cognitoidpconn

	params := &cognitoidentityprovider.UpdateUserPoolInput{
		UserPoolId: aws.String(d.Id()),
	}

	if v, ok := d.GetOk("admin_create_user_config"); ok {
		configs := v.([]interface{})
		config, ok := configs[0].(map[string]interface{})

		if ok && config != nil {
			params.AdminCreateUserConfig = expandCognitoUserPoolAdminCreateUserConfig(config)
		}
	}

	if v, ok := d.GetOk("auto_verified_attributes"); ok {
		params.AutoVerifiedAttributes = expandStringList(v.(*schema.Set).List())
	}

	if v, ok := d.GetOk("device_configuration"); ok {
		configs := v.([]interface{})
		config, ok := configs[0].(map[string]interface{})

		if ok && config != nil {
			params.DeviceConfiguration = expandCognitoUserPoolDeviceConfiguration(config)
		}
	}

	if v, ok := d.GetOk("email_configuration"); ok {
		configs := v.([]interface{})
		config, ok := configs[0].(map[string]interface{})

		if ok && config != nil {
			emailConfigurationType := &cognitoidentityprovider.EmailConfigurationType{}

			if v, ok := config["reply_to_email_address"]; ok && v.(string) != "" {
				emailConfigurationType.ReplyToEmailAddress = aws.String(v.(string))
			}

			if v, ok := config["source_arn"]; ok && v.(string) != "" {
				emailConfigurationType.SourceArn = aws.String(v.(string))
			}

			params.EmailConfiguration = emailConfigurationType
		}
	}

	if v, ok := d.GetOk("email_verification_subject"); ok {
		params.EmailVerificationSubject = aws.String(v.(string))
	}

	if v, ok := d.GetOk("email_verification_message"); ok {
		params.EmailVerificationMessage = aws.String(v.(string))
	}

	if v, ok := d.GetOk("lambda_config"); ok {
		configs := v.([]interface{})
		config, ok := configs[0].(map[string]interface{})

		if ok && config != nil {
			params.LambdaConfig = expandCognitoUserPoolLambdaConfig(config)
		}
	}

	if v, ok := d.GetOk("mfa_configuration"); ok {
		params.MfaConfiguration = aws.String(v.(string))
	}

	if v, ok := d.GetOk("password_policy"); ok {
		configs := v.([]interface{})
		config, ok := configs[0].(map[string]interface{})

		if ok && config != nil {
			policies := &cognitoidentityprovider.UserPoolPolicyType{}
			policies.PasswordPolicy = expandCognitoUserPoolPasswordPolicy(config)
			params.Policies = policies
		}
	}

	if v, ok := d.GetOk("sms_authentication_message"); ok {
		params.SmsAuthenticationMessage = aws.String(v.(string))
	}

	if v, ok := d.GetOk("sms_configuration"); ok {
		configs := v.([]interface{})
		config, ok := configs[0].(map[string]interface{})

		if ok && config != nil {
			params.SmsConfiguration = expandCognitoUserPoolSmsConfiguration(config)
		}
	}

	if v, ok := d.GetOk("verification_message_template"); ok {
		configs := v.([]interface{})
		config, ok := configs[0].(map[string]interface{})

		if d.HasChange("email_verification_message") {
			config["email_message"] = d.Get("email_verification_message")
		}
		if d.HasChange("email_verification_subject") {
			config["email_subject"] = d.Get("email_verification_subject")
		}
		if d.HasChange("sms_verification_message") {
			config["sms_message"] = d.Get("sms_verification_message")
		}

		if ok && config != nil {
			params.VerificationMessageTemplate = expandCognitoUserPoolVerificationMessageTemplate(config)
		}
	}

	if v, ok := d.GetOk("sms_verification_message"); ok {
		params.SmsVerificationMessage = aws.String(v.(string))
	}

	if v, ok := d.GetOk("tags"); ok {
		params.UserPoolTags = tagsFromMapGeneric(v.(map[string]interface{}))
	}

	log.Printf("[DEBUG] Updating Cognito User Pool: %s", params)

	// IAM roles & policies can take some time to propagate and be attached
	// to the User Pool.
	err := resource.Retry(2*time.Minute, func() *resource.RetryError {
		var err error
		_, err = conn.UpdateUserPool(params)
		if isAWSErr(err, "InvalidSmsRoleTrustRelationshipException", "Role does not have a trust relationship allowing Cognito to assume the role") {
			log.Printf("[DEBUG] Received %s, retrying UpdateUserPool", err)
			return resource.RetryableError(err)
		}
		if isAWSErr(err, "InvalidSmsRoleAccessPolicyException", "Role does not have permission to publish with SNS") {
			log.Printf("[DEBUG] Received %s, retrying UpdateUserPool", err)
			return resource.RetryableError(err)
		}

		return resource.NonRetryableError(err)
	})
	if err != nil {
		return fmt.Errorf("Error updating Cognito User pool: %s", err)
	}

	return resourceAwsCognitoUserPoolRead(d, meta)
}

func resourceAwsCognitoUserPoolDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cognitoidpconn

	params := &cognitoidentityprovider.DeleteUserPoolInput{
		UserPoolId: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Deleting Cognito User Pool: %s", params)

	_, err := conn.DeleteUserPool(params)

	if err != nil {
		return fmt.Errorf("Error deleting user pool: %s", err)
	}

	return nil
}
