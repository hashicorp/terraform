package aws

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsLbListener() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsLbListenerCreate,
		Read:   resourceAwsLbListenerRead,
		Update: resourceAwsLbListenerUpdate,
		Delete: resourceAwsLbListenerDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Read: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"load_balancer_arn": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"port": {
				Type:         schema.TypeInt,
				Required:     true,
				ValidateFunc: validation.IntBetween(1, 65535),
			},

			"protocol": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "HTTP",
				StateFunc: func(v interface{}) string {
					return strings.ToUpper(v.(string))
				},
				ValidateFunc: validation.StringInSlice([]string{
					elbv2.ProtocolEnumHttp,
					elbv2.ProtocolEnumHttps,
					elbv2.ProtocolEnumTcp,
				}, true),
			},

			"ssl_policy": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"certificate_arn": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"default_action": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								elbv2.ActionTypeEnumAuthenticateCognito,
								elbv2.ActionTypeEnumAuthenticateOidc,
								elbv2.ActionTypeEnumFixedResponse,
								elbv2.ActionTypeEnumForward,
								elbv2.ActionTypeEnumRedirect,
							}, true),
						},
						"order": {
							Type:         schema.TypeInt,
							Optional:     true,
							Computed:     true,
							ValidateFunc: validation.IntBetween(1, 50000),
						},

						"target_group_arn": {
							Type:             schema.TypeString,
							Optional:         true,
							DiffSuppressFunc: suppressIfDefaultActionTypeNot("forward"),
						},

						"redirect": {
							Type:             schema.TypeList,
							Optional:         true,
							DiffSuppressFunc: suppressIfDefaultActionTypeNot("redirect"),
							MaxItems:         1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"host": {
										Type:     schema.TypeString,
										Optional: true,
										Default:  "#{host}",
									},

									"path": {
										Type:     schema.TypeString,
										Optional: true,
										Default:  "/#{path}",
									},

									"port": {
										Type:     schema.TypeString,
										Optional: true,
										Default:  "#{port}",
									},

									"protocol": {
										Type:     schema.TypeString,
										Optional: true,
										Default:  "#{protocol}",
										ValidateFunc: validation.StringInSlice([]string{
											"#{protocol}",
											"HTTP",
											"HTTPS",
										}, false),
									},

									"query": {
										Type:     schema.TypeString,
										Optional: true,
										Default:  "#{query}",
									},

									"status_code": {
										Type:     schema.TypeString,
										Required: true,
										ValidateFunc: validation.StringInSlice([]string{
											"HTTP_301",
											"HTTP_302",
										}, false),
									},
								},
							},
						},

						"fixed_response": {
							Type:             schema.TypeList,
							Optional:         true,
							DiffSuppressFunc: suppressIfDefaultActionTypeNot("fixed-response"),
							MaxItems:         1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"content_type": {
										Type:     schema.TypeString,
										Required: true,
										ValidateFunc: validation.StringInSlice([]string{
											"text/plain",
											"text/css",
											"text/html",
											"application/javascript",
											"application/json",
										}, false),
									},

									"message_body": {
										Type:     schema.TypeString,
										Optional: true,
									},

									"status_code": {
										Type:         schema.TypeString,
										Optional:     true,
										Computed:     true,
										ValidateFunc: validation.StringMatch(regexp.MustCompile(`^[245]\d\d$`), ""),
									},
								},
							},
						},

						"authenticate_cognito": {
							Type:             schema.TypeList,
							Optional:         true,
							DiffSuppressFunc: suppressIfDefaultActionTypeNot("authenticate-cognito"),
							MaxItems:         1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"authentication_request_extra_params": {
										Type:     schema.TypeMap,
										Optional: true,
									},
									"on_unauthenticated_request": {
										Type:     schema.TypeString,
										Optional: true,
										Computed: true,
										ValidateFunc: validation.StringInSlice([]string{
											elbv2.AuthenticateCognitoActionConditionalBehaviorEnumDeny,
											elbv2.AuthenticateCognitoActionConditionalBehaviorEnumAllow,
											elbv2.AuthenticateCognitoActionConditionalBehaviorEnumAuthenticate,
										}, true),
									},
									"scope": {
										Type:     schema.TypeString,
										Optional: true,
										Computed: true,
									},
									"session_cookie_name": {
										Type:     schema.TypeString,
										Optional: true,
										Computed: true,
									},
									"session_timeout": {
										Type:     schema.TypeInt,
										Optional: true,
										Computed: true,
									},
									"user_pool_arn": {
										Type:     schema.TypeString,
										Required: true,
									},
									"user_pool_client_id": {
										Type:     schema.TypeString,
										Required: true,
									},
									"user_pool_domain": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},

						"authenticate_oidc": {
							Type:             schema.TypeList,
							Optional:         true,
							DiffSuppressFunc: suppressIfDefaultActionTypeNot("authenticate-oidc"),
							MaxItems:         1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"authentication_request_extra_params": {
										Type:     schema.TypeMap,
										Optional: true,
									},
									"authorization_endpoint": {
										Type:     schema.TypeString,
										Required: true,
									},
									"client_id": {
										Type:     schema.TypeString,
										Required: true,
									},
									"client_secret": {
										Type:      schema.TypeString,
										Required:  true,
										Sensitive: true,
									},
									"issuer": {
										Type:     schema.TypeString,
										Required: true,
									},
									"on_unauthenticated_request": {
										Type:     schema.TypeString,
										Optional: true,
										Computed: true,
										ValidateFunc: validation.StringInSlice([]string{
											elbv2.AuthenticateOidcActionConditionalBehaviorEnumDeny,
											elbv2.AuthenticateOidcActionConditionalBehaviorEnumAllow,
											elbv2.AuthenticateOidcActionConditionalBehaviorEnumAuthenticate,
										}, true),
									},
									"scope": {
										Type:     schema.TypeString,
										Optional: true,
										Computed: true,
									},
									"session_cookie_name": {
										Type:     schema.TypeString,
										Optional: true,
										Computed: true,
									},
									"session_timeout": {
										Type:     schema.TypeInt,
										Optional: true,
										Computed: true,
									},
									"token_endpoint": {
										Type:     schema.TypeString,
										Required: true,
									},
									"user_info_endpoint": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func suppressIfDefaultActionTypeNot(t string) schema.SchemaDiffSuppressFunc {
	return func(k, old, new string, d *schema.ResourceData) bool {
		take := 2
		i := strings.IndexFunc(k, func(r rune) bool {
			if r == '.' {
				take -= 1
				return take == 0
			}
			return false
		})
		at := k[:i+1] + "type"
		return d.Get(at).(string) != t
	}
}

func resourceAwsLbListenerCreate(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbv2conn

	lbArn := d.Get("load_balancer_arn").(string)

	params := &elbv2.CreateListenerInput{
		LoadBalancerArn: aws.String(lbArn),
		Port:            aws.Int64(int64(d.Get("port").(int))),
		Protocol:        aws.String(d.Get("protocol").(string)),
	}

	if sslPolicy, ok := d.GetOk("ssl_policy"); ok {
		params.SslPolicy = aws.String(sslPolicy.(string))
	}

	if certificateArn, ok := d.GetOk("certificate_arn"); ok {
		params.Certificates = make([]*elbv2.Certificate, 1)
		params.Certificates[0] = &elbv2.Certificate{
			CertificateArn: aws.String(certificateArn.(string)),
		}
	}

	defaultActions := d.Get("default_action").([]interface{})
	params.DefaultActions = make([]*elbv2.Action, len(defaultActions))
	for i, defaultAction := range defaultActions {
		defaultActionMap := defaultAction.(map[string]interface{})

		action := &elbv2.Action{
			Order: aws.Int64(int64(i + 1)),
			Type:  aws.String(defaultActionMap["type"].(string)),
		}

		if order, ok := defaultActionMap["order"]; ok && order.(int) != 0 {
			action.Order = aws.Int64(int64(order.(int)))
		}

		switch defaultActionMap["type"].(string) {
		case "forward":
			action.TargetGroupArn = aws.String(defaultActionMap["target_group_arn"].(string))

		case "redirect":
			redirectList := defaultActionMap["redirect"].([]interface{})

			if len(redirectList) == 1 {
				redirectMap := redirectList[0].(map[string]interface{})

				action.RedirectConfig = &elbv2.RedirectActionConfig{
					Host:       aws.String(redirectMap["host"].(string)),
					Path:       aws.String(redirectMap["path"].(string)),
					Port:       aws.String(redirectMap["port"].(string)),
					Protocol:   aws.String(redirectMap["protocol"].(string)),
					Query:      aws.String(redirectMap["query"].(string)),
					StatusCode: aws.String(redirectMap["status_code"].(string)),
				}
			} else {
				return errors.New("for actions of type 'redirect', you must specify a 'redirect' block")
			}

		case "fixed-response":
			fixedResponseList := defaultActionMap["fixed_response"].([]interface{})

			if len(fixedResponseList) == 1 {
				fixedResponseMap := fixedResponseList[0].(map[string]interface{})

				action.FixedResponseConfig = &elbv2.FixedResponseActionConfig{
					ContentType: aws.String(fixedResponseMap["content_type"].(string)),
					MessageBody: aws.String(fixedResponseMap["message_body"].(string)),
					StatusCode:  aws.String(fixedResponseMap["status_code"].(string)),
				}
			} else {
				return errors.New("for actions of type 'fixed-response', you must specify a 'fixed_response' block")
			}

		case elbv2.ActionTypeEnumAuthenticateCognito:
			authenticateCognitoList := defaultActionMap["authenticate_cognito"].([]interface{})

			if len(authenticateCognitoList) == 1 {
				authenticateCognitoMap := authenticateCognitoList[0].(map[string]interface{})

				authenticationRequestExtraParams := make(map[string]*string)
				for key, value := range authenticateCognitoMap["authentication_request_extra_params"].(map[string]interface{}) {
					authenticationRequestExtraParams[key] = aws.String(value.(string))
				}

				action.AuthenticateCognitoConfig = &elbv2.AuthenticateCognitoActionConfig{
					AuthenticationRequestExtraParams: authenticationRequestExtraParams,
					UserPoolArn:                      aws.String(authenticateCognitoMap["user_pool_arn"].(string)),
					UserPoolClientId:                 aws.String(authenticateCognitoMap["user_pool_client_id"].(string)),
					UserPoolDomain:                   aws.String(authenticateCognitoMap["user_pool_domain"].(string)),
				}

				if onUnauthenticatedRequest, ok := authenticateCognitoMap["on_unauthenticated_request"]; ok && onUnauthenticatedRequest != "" {
					action.AuthenticateCognitoConfig.OnUnauthenticatedRequest = aws.String(onUnauthenticatedRequest.(string))
				}
				if scope, ok := authenticateCognitoMap["scope"]; ok && scope != "" {
					action.AuthenticateCognitoConfig.Scope = aws.String(scope.(string))
				}
				if sessionCookieName, ok := authenticateCognitoMap["session_cookie_name"]; ok && sessionCookieName != "" {
					action.AuthenticateCognitoConfig.SessionCookieName = aws.String(sessionCookieName.(string))
				}
				if sessionTimeout, ok := authenticateCognitoMap["session_timeout"]; ok && sessionTimeout != 0 {
					action.AuthenticateCognitoConfig.SessionTimeout = aws.Int64(int64(sessionTimeout.(int)))
				}
			} else {
				return errors.New("for actions of type 'authenticate-cognito', you must specify a 'authenticate_cognito' block")
			}

		case elbv2.ActionTypeEnumAuthenticateOidc:
			authenticateOidcList := defaultActionMap["authenticate_oidc"].([]interface{})

			if len(authenticateOidcList) == 1 {
				authenticateOidcMap := authenticateOidcList[0].(map[string]interface{})

				authenticationRequestExtraParams := make(map[string]*string)
				for key, value := range authenticateOidcMap["authentication_request_extra_params"].(map[string]interface{}) {
					authenticationRequestExtraParams[key] = aws.String(value.(string))
				}

				action.AuthenticateOidcConfig = &elbv2.AuthenticateOidcActionConfig{
					AuthenticationRequestExtraParams: authenticationRequestExtraParams,
					AuthorizationEndpoint:            aws.String(authenticateOidcMap["authorization_endpoint"].(string)),
					ClientId:                         aws.String(authenticateOidcMap["client_id"].(string)),
					ClientSecret:                     aws.String(authenticateOidcMap["client_secret"].(string)),
					Issuer:                           aws.String(authenticateOidcMap["issuer"].(string)),
					TokenEndpoint:                    aws.String(authenticateOidcMap["token_endpoint"].(string)),
					UserInfoEndpoint:                 aws.String(authenticateOidcMap["user_info_endpoint"].(string)),
				}

				if onUnauthenticatedRequest, ok := authenticateOidcMap["on_unauthenticated_request"]; ok && onUnauthenticatedRequest != "" {
					action.AuthenticateOidcConfig.OnUnauthenticatedRequest = aws.String(onUnauthenticatedRequest.(string))
				}
				if scope, ok := authenticateOidcMap["scope"]; ok && scope != "" {
					action.AuthenticateOidcConfig.Scope = aws.String(scope.(string))
				}
				if sessionCookieName, ok := authenticateOidcMap["session_cookie_name"]; ok && sessionCookieName != "" {
					action.AuthenticateOidcConfig.SessionCookieName = aws.String(sessionCookieName.(string))
				}
				if sessionTimeout, ok := authenticateOidcMap["session_timeout"]; ok && sessionTimeout != 0 {
					action.AuthenticateOidcConfig.SessionTimeout = aws.Int64(int64(sessionTimeout.(int)))
				}
			} else {
				return errors.New("for actions of type 'authenticate-oidc', you must specify a 'authenticate_oidc' block")
			}
		}

		params.DefaultActions[i] = action
	}

	var resp *elbv2.CreateListenerOutput

	err := resource.Retry(5*time.Minute, func() *resource.RetryError {
		var err error
		log.Printf("[DEBUG] Creating LB listener for ARN: %s", d.Get("load_balancer_arn").(string))
		resp, err = elbconn.CreateListener(params)
		if err != nil {
			if isAWSErr(err, elbv2.ErrCodeCertificateNotFoundException, "") {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("Error creating LB Listener: %s", err)
	}

	if len(resp.Listeners) == 0 {
		return errors.New("Error creating LB Listener: no listeners returned in response")
	}

	d.SetId(*resp.Listeners[0].ListenerArn)

	return resourceAwsLbListenerRead(d, meta)
}

func resourceAwsLbListenerRead(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbv2conn

	var resp *elbv2.DescribeListenersOutput
	var request = &elbv2.DescribeListenersInput{
		ListenerArns: []*string{aws.String(d.Id())},
	}

	err := resource.Retry(1*time.Minute, func() *resource.RetryError {
		var err error
		resp, err = elbconn.DescribeListeners(request)
		if d.IsNewResource() && isAWSErr(err, elbv2.ErrCodeListenerNotFoundException, "") {
			return resource.RetryableError(err)
		}
		if err != nil {
			return resource.NonRetryableError(err)
		}
		return nil
	})

	if isAWSErr(err, elbv2.ErrCodeListenerNotFoundException, "") {
		log.Printf("[WARN] ELBv2 Listener (%s) not found - removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("Error retrieving Listener: %s", err)
	}

	if len(resp.Listeners) != 1 {
		return fmt.Errorf("Error retrieving Listener %q", d.Id())
	}

	listener := resp.Listeners[0]

	d.Set("arn", listener.ListenerArn)
	d.Set("load_balancer_arn", listener.LoadBalancerArn)
	d.Set("port", listener.Port)
	d.Set("protocol", listener.Protocol)
	d.Set("ssl_policy", listener.SslPolicy)

	if listener.Certificates != nil && len(listener.Certificates) == 1 && listener.Certificates[0] != nil {
		d.Set("certificate_arn", listener.Certificates[0].CertificateArn)
	}

	sortedActions := sortActionsBasedonTypeinTFFile("default_action", listener.DefaultActions, d)
	defaultActions := make([]interface{}, len(sortedActions))
	for i, defaultAction := range sortedActions {
		defaultActionMap := make(map[string]interface{})
		defaultActionMap["type"] = aws.StringValue(defaultAction.Type)
		defaultActionMap["order"] = aws.Int64Value(defaultAction.Order)

		switch aws.StringValue(defaultAction.Type) {
		case "forward":
			defaultActionMap["target_group_arn"] = aws.StringValue(defaultAction.TargetGroupArn)

		case "redirect":
			defaultActionMap["redirect"] = []map[string]interface{}{
				{
					"host":        aws.StringValue(defaultAction.RedirectConfig.Host),
					"path":        aws.StringValue(defaultAction.RedirectConfig.Path),
					"port":        aws.StringValue(defaultAction.RedirectConfig.Port),
					"protocol":    aws.StringValue(defaultAction.RedirectConfig.Protocol),
					"query":       aws.StringValue(defaultAction.RedirectConfig.Query),
					"status_code": aws.StringValue(defaultAction.RedirectConfig.StatusCode),
				},
			}

		case "fixed-response":
			defaultActionMap["fixed_response"] = []map[string]interface{}{
				{
					"content_type": aws.StringValue(defaultAction.FixedResponseConfig.ContentType),
					"message_body": aws.StringValue(defaultAction.FixedResponseConfig.MessageBody),
					"status_code":  aws.StringValue(defaultAction.FixedResponseConfig.StatusCode),
				},
			}

		case "authenticate-cognito":
			authenticationRequestExtraParams := make(map[string]interface{})
			for key, value := range defaultAction.AuthenticateCognitoConfig.AuthenticationRequestExtraParams {
				authenticationRequestExtraParams[key] = aws.StringValue(value)
			}
			defaultActionMap["authenticate_cognito"] = []map[string]interface{}{
				{
					"authentication_request_extra_params": authenticationRequestExtraParams,
					"on_unauthenticated_request":          aws.StringValue(defaultAction.AuthenticateCognitoConfig.OnUnauthenticatedRequest),
					"scope":                               aws.StringValue(defaultAction.AuthenticateCognitoConfig.Scope),
					"session_cookie_name":                 aws.StringValue(defaultAction.AuthenticateCognitoConfig.SessionCookieName),
					"session_timeout":                     aws.Int64Value(defaultAction.AuthenticateCognitoConfig.SessionTimeout),
					"user_pool_arn":                       aws.StringValue(defaultAction.AuthenticateCognitoConfig.UserPoolArn),
					"user_pool_client_id":                 aws.StringValue(defaultAction.AuthenticateCognitoConfig.UserPoolClientId),
					"user_pool_domain":                    aws.StringValue(defaultAction.AuthenticateCognitoConfig.UserPoolDomain),
				},
			}

		case "authenticate-oidc":
			authenticationRequestExtraParams := make(map[string]interface{})
			for key, value := range defaultAction.AuthenticateOidcConfig.AuthenticationRequestExtraParams {
				authenticationRequestExtraParams[key] = aws.StringValue(value)
			}

			// The LB API currently provides no way to read the ClientSecret
			// Instead we passthrough the configuration value into the state
			clientSecret := d.Get("default_action." + strconv.Itoa(i) + ".authenticate_oidc.0.client_secret").(string)

			defaultActionMap["authenticate_oidc"] = []map[string]interface{}{
				{
					"authentication_request_extra_params": authenticationRequestExtraParams,
					"authorization_endpoint":              aws.StringValue(defaultAction.AuthenticateOidcConfig.AuthorizationEndpoint),
					"client_id":                           aws.StringValue(defaultAction.AuthenticateOidcConfig.ClientId),
					"client_secret":                       clientSecret,
					"issuer":                              aws.StringValue(defaultAction.AuthenticateOidcConfig.Issuer),
					"on_unauthenticated_request":          aws.StringValue(defaultAction.AuthenticateOidcConfig.OnUnauthenticatedRequest),
					"scope":                               aws.StringValue(defaultAction.AuthenticateOidcConfig.Scope),
					"session_cookie_name":                 aws.StringValue(defaultAction.AuthenticateOidcConfig.SessionCookieName),
					"session_timeout":                     aws.Int64Value(defaultAction.AuthenticateOidcConfig.SessionTimeout),
					"token_endpoint":                      aws.StringValue(defaultAction.AuthenticateOidcConfig.TokenEndpoint),
					"user_info_endpoint":                  aws.StringValue(defaultAction.AuthenticateOidcConfig.UserInfoEndpoint),
				},
			}
		}

		defaultActions[i] = defaultActionMap
	}
	d.Set("default_action", defaultActions)

	return nil
}

func resourceAwsLbListenerUpdate(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbv2conn

	params := &elbv2.ModifyListenerInput{
		ListenerArn: aws.String(d.Id()),
		Port:        aws.Int64(int64(d.Get("port").(int))),
		Protocol:    aws.String(d.Get("protocol").(string)),
	}

	if sslPolicy, ok := d.GetOk("ssl_policy"); ok {
		params.SslPolicy = aws.String(sslPolicy.(string))
	}

	if certificateArn, ok := d.GetOk("certificate_arn"); ok {
		params.Certificates = make([]*elbv2.Certificate, 1)
		params.Certificates[0] = &elbv2.Certificate{
			CertificateArn: aws.String(certificateArn.(string)),
		}
	}

	if d.HasChange("default_action") {
		defaultActions := d.Get("default_action").([]interface{})
		params.DefaultActions = make([]*elbv2.Action, len(defaultActions))

		for i, defaultAction := range defaultActions {
			defaultActionMap := defaultAction.(map[string]interface{})

			action := &elbv2.Action{
				Order: aws.Int64(int64(i + 1)),
				Type:  aws.String(defaultActionMap["type"].(string)),
			}

			if order, ok := defaultActionMap["order"]; ok && order.(int) != 0 {
				action.Order = aws.Int64(int64(order.(int)))
			}

			switch defaultActionMap["type"].(string) {
			case "forward":
				action.TargetGroupArn = aws.String(defaultActionMap["target_group_arn"].(string))

			case "redirect":
				redirectList := defaultActionMap["redirect"].([]interface{})

				if len(redirectList) == 1 {
					redirectMap := redirectList[0].(map[string]interface{})

					action.RedirectConfig = &elbv2.RedirectActionConfig{
						Host:       aws.String(redirectMap["host"].(string)),
						Path:       aws.String(redirectMap["path"].(string)),
						Port:       aws.String(redirectMap["port"].(string)),
						Protocol:   aws.String(redirectMap["protocol"].(string)),
						Query:      aws.String(redirectMap["query"].(string)),
						StatusCode: aws.String(redirectMap["status_code"].(string)),
					}
				} else {
					return errors.New("for actions of type 'redirect', you must specify a 'redirect' block")
				}

			case "fixed-response":
				fixedResponseList := defaultActionMap["fixed_response"].([]interface{})

				if len(fixedResponseList) == 1 {
					fixedResponseMap := fixedResponseList[0].(map[string]interface{})

					action.FixedResponseConfig = &elbv2.FixedResponseActionConfig{
						ContentType: aws.String(fixedResponseMap["content_type"].(string)),
						MessageBody: aws.String(fixedResponseMap["message_body"].(string)),
						StatusCode:  aws.String(fixedResponseMap["status_code"].(string)),
					}
				} else {
					return errors.New("for actions of type 'fixed-response', you must specify a 'fixed_response' block")
				}

			case "authenticate-cognito":
				authenticateCognitoList := defaultActionMap["authenticate_cognito"].([]interface{})

				if len(authenticateCognitoList) == 1 {
					authenticateCognitoMap := authenticateCognitoList[0].(map[string]interface{})

					authenticationRequestExtraParams := make(map[string]*string)
					for key, value := range authenticateCognitoMap["authentication_request_extra_params"].(map[string]interface{}) {
						authenticationRequestExtraParams[key] = aws.String(value.(string))
					}

					action.AuthenticateCognitoConfig = &elbv2.AuthenticateCognitoActionConfig{
						AuthenticationRequestExtraParams: authenticationRequestExtraParams,
						UserPoolArn:                      aws.String(authenticateCognitoMap["user_pool_arn"].(string)),
						UserPoolClientId:                 aws.String(authenticateCognitoMap["user_pool_client_id"].(string)),
						UserPoolDomain:                   aws.String(authenticateCognitoMap["user_pool_domain"].(string)),
					}

					if onUnauthenticatedRequest, ok := authenticateCognitoMap["on_unauthenticated_request"]; ok && onUnauthenticatedRequest != "" {
						action.AuthenticateCognitoConfig.OnUnauthenticatedRequest = aws.String(onUnauthenticatedRequest.(string))
					}
					if scope, ok := authenticateCognitoMap["scope"]; ok && scope != "" {
						action.AuthenticateCognitoConfig.Scope = aws.String(scope.(string))
					}
					if sessionCookieName, ok := authenticateCognitoMap["session_cookie_name"]; ok && sessionCookieName != "" {
						action.AuthenticateCognitoConfig.SessionCookieName = aws.String(sessionCookieName.(string))
					}
					if sessionTimeout, ok := authenticateCognitoMap["session_timeout"]; ok && sessionTimeout != 0 {
						action.AuthenticateCognitoConfig.SessionTimeout = aws.Int64(int64(sessionTimeout.(int)))
					}
				} else {
					return errors.New("for actions of type 'authenticate-cognito', you must specify a 'authenticate_cognito' block")
				}

			case "authenticate-oidc":
				authenticateOidcList := defaultActionMap["authenticate_oidc"].([]interface{})

				if len(authenticateOidcList) == 1 {
					authenticateOidcMap := authenticateOidcList[0].(map[string]interface{})

					authenticationRequestExtraParams := make(map[string]*string)
					for key, value := range authenticateOidcMap["authentication_request_extra_params"].(map[string]interface{}) {
						authenticationRequestExtraParams[key] = aws.String(value.(string))
					}

					action.AuthenticateOidcConfig = &elbv2.AuthenticateOidcActionConfig{
						AuthenticationRequestExtraParams: authenticationRequestExtraParams,
						AuthorizationEndpoint:            aws.String(authenticateOidcMap["authorization_endpoint"].(string)),
						ClientId:                         aws.String(authenticateOidcMap["client_id"].(string)),
						ClientSecret:                     aws.String(authenticateOidcMap["client_secret"].(string)),
						Issuer:                           aws.String(authenticateOidcMap["issuer"].(string)),
						TokenEndpoint:                    aws.String(authenticateOidcMap["token_endpoint"].(string)),
						UserInfoEndpoint:                 aws.String(authenticateOidcMap["user_info_endpoint"].(string)),
					}

					if onUnauthenticatedRequest, ok := authenticateOidcMap["on_unauthenticated_request"]; ok && onUnauthenticatedRequest != "" {
						action.AuthenticateOidcConfig.OnUnauthenticatedRequest = aws.String(onUnauthenticatedRequest.(string))
					}
					if scope, ok := authenticateOidcMap["scope"]; ok && scope != "" {
						action.AuthenticateOidcConfig.Scope = aws.String(scope.(string))
					}
					if sessionCookieName, ok := authenticateOidcMap["session_cookie_name"]; ok && sessionCookieName != "" {
						action.AuthenticateOidcConfig.SessionCookieName = aws.String(sessionCookieName.(string))
					}
					if sessionTimeout, ok := authenticateOidcMap["session_timeout"]; ok && sessionTimeout != 0 {
						action.AuthenticateOidcConfig.SessionTimeout = aws.Int64(int64(sessionTimeout.(int)))
					}
				} else {
					return errors.New("for actions of type 'authenticate-oidc', you must specify a 'authenticate_oidc' block")
				}
			}

			params.DefaultActions[i] = action
		}
	}

	err := resource.Retry(5*time.Minute, func() *resource.RetryError {
		_, err := elbconn.ModifyListener(params)
		if err != nil {
			if isAWSErr(err, elbv2.ErrCodeCertificateNotFoundException, "") {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("Error modifying LB Listener: %s", err)
	}

	return resourceAwsLbListenerRead(d, meta)
}

func resourceAwsLbListenerDelete(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbv2conn

	_, err := elbconn.DeleteListener(&elbv2.DeleteListenerInput{
		ListenerArn: aws.String(d.Id()),
	})
	if err != nil {
		return fmt.Errorf("Error deleting Listener: %s", err)
	}

	return nil
}
