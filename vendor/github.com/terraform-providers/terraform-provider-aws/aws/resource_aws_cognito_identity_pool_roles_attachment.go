package aws

import (
	"fmt"
	"log"
	"time"

	"bytes"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cognitoidentity"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsCognitoIdentityPoolRolesAttachment() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCognitoIdentityPoolRolesAttachmentCreate,
		Read:   resourceAwsCognitoIdentityPoolRolesAttachmentRead,
		Update: resourceAwsCognitoIdentityPoolRolesAttachmentUpdate,
		Delete: resourceAwsCognitoIdentityPoolRolesAttachmentDelete,

		Schema: map[string]*schema.Schema{
			"identity_pool_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"role_mapping": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"identity_provider": {
							Type:     schema.TypeString,
							Required: true,
						},
						"ambiguous_role_resolution": {
							Type:         schema.TypeString,
							ValidateFunc: validateCognitoRoleMappingsAmbiguousRoleResolution,
							Optional:     true, // Required if Type equals Token or Rules.
						},
						"mapping_rule": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 25,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"claim": {
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validateCognitoRoleMappingsRulesClaim,
									},
									"match_type": {
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validateCognitoRoleMappingsRulesMatchType,
									},
									"role_arn": {
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validateArn,
									},
									"value": {
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validateCognitoRoleMappingsRulesValue,
									},
								},
							},
						},
						"type": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateCognitoRoleMappingsType,
						},
					},
				},
			},

			"roles": {
				Type:     schema.TypeMap,
				Required: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"authenticated": {
							Type:         schema.TypeString,
							ValidateFunc: validateArn,
							Optional:     true, // Required if unauthenticated isn't defined.
						},
						"unauthenticated": {
							Type:         schema.TypeString,
							ValidateFunc: validateArn,
							Optional:     true, // Required if authenticated isn't defined.
						},
					},
				},
			},
		},
	}
}

func resourceAwsCognitoIdentityPoolRolesAttachmentCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cognitoconn

	// Validates role keys to be either authenticated or unauthenticated,
	// since ValidateFunc validates only the value not the key.
	if errors := validateCognitoRoles(d.Get("roles").(map[string]interface{}), "roles"); len(errors) > 0 {
		return fmt.Errorf("Error validating Roles: %v", errors)
	}

	params := &cognitoidentity.SetIdentityPoolRolesInput{
		IdentityPoolId: aws.String(d.Get("identity_pool_id").(string)),
		Roles:          expandCognitoIdentityPoolRoles(d.Get("roles").(map[string]interface{})),
	}

	if v, ok := d.GetOk("role_mapping"); ok {
		errors := validateRoleMappings(v.(*schema.Set).List())

		if len(errors) > 0 {
			return fmt.Errorf("Error validating ambiguous role resolution: %v", errors)
		}

		params.RoleMappings = expandCognitoIdentityPoolRoleMappingsAttachment(v.(*schema.Set).List())
	}

	log.Printf("[DEBUG] Creating Cognito Identity Pool Roles Association: %#v", params)
	_, err := conn.SetIdentityPoolRoles(params)
	if err != nil {
		return fmt.Errorf("Error creating Cognito Identity Pool Roles Association: %s", err)
	}

	d.SetId(d.Get("identity_pool_id").(string))

	return resourceAwsCognitoIdentityPoolRolesAttachmentRead(d, meta)
}

func resourceAwsCognitoIdentityPoolRolesAttachmentRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cognitoconn
	log.Printf("[DEBUG] Reading Cognito Identity Pool Roles Association: %s", d.Id())

	ip, err := conn.GetIdentityPoolRoles(&cognitoidentity.GetIdentityPoolRolesInput{
		IdentityPoolId: aws.String(d.Get("identity_pool_id").(string)),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "ResourceNotFoundException" {
			log.Printf("[WARN] Cognito Identity Pool Roles Association %s not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	if err := d.Set("roles", flattenCognitoIdentityPoolRoles(ip.Roles)); err != nil {
		return fmt.Errorf("[DEBUG] Error setting roles error: %#v", err)
	}

	if err := d.Set("role_mapping", flattenCognitoIdentityPoolRoleMappingsAttachment(ip.RoleMappings)); err != nil {
		return fmt.Errorf("[DEBUG] Error setting role mappings error: %#v", err)
	}

	return nil
}

func resourceAwsCognitoIdentityPoolRolesAttachmentUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cognitoconn

	// Validates role keys to be either authenticated or unauthenticated,
	// since ValidateFunc validates only the value not the key.
	if errors := validateCognitoRoles(d.Get("roles").(map[string]interface{}), "roles"); len(errors) > 0 {
		return fmt.Errorf("Error validating Roles: %v", errors)
	}

	params := &cognitoidentity.SetIdentityPoolRolesInput{
		IdentityPoolId: aws.String(d.Get("identity_pool_id").(string)),
		Roles:          expandCognitoIdentityPoolRoles(d.Get("roles").(map[string]interface{})),
	}

	if d.HasChange("role_mapping") {
		v, ok := d.GetOk("role_mapping")
		var mappings []interface{}

		if ok {
			errors := validateRoleMappings(v.(*schema.Set).List())

			if len(errors) > 0 {
				return fmt.Errorf("Error validating ambiguous role resolution: %v", errors)
			}
			mappings = v.(*schema.Set).List()
		} else {
			mappings = []interface{}{}
		}

		params.RoleMappings = expandCognitoIdentityPoolRoleMappingsAttachment(mappings)
	}

	log.Printf("[DEBUG] Updating Cognito Identity Pool Roles Association: %#v", params)
	_, err := conn.SetIdentityPoolRoles(params)
	if err != nil {
		return fmt.Errorf("Error updating Cognito Identity Pool Roles Association: %s", err)
	}

	d.SetId(d.Get("identity_pool_id").(string))

	return resourceAwsCognitoIdentityPoolRolesAttachmentRead(d, meta)
}

func resourceAwsCognitoIdentityPoolRolesAttachmentDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cognitoconn
	log.Printf("[DEBUG] Deleting Cognito Identity Pool Roles Association: %s", d.Id())

	return resource.Retry(5*time.Minute, func() *resource.RetryError {
		_, err := conn.SetIdentityPoolRoles(&cognitoidentity.SetIdentityPoolRolesInput{
			IdentityPoolId: aws.String(d.Get("identity_pool_id").(string)),
			Roles:          expandCognitoIdentityPoolRoles(make(map[string]interface{})),
			RoleMappings:   expandCognitoIdentityPoolRoleMappingsAttachment([]interface{}{}),
		})

		if err == nil {
			return nil
		}

		return resource.NonRetryableError(err)
	})
}

// Validating that each role_mapping ambiguous_role_resolution
// is defined when "type" equals Token or Rules.
func validateRoleMappings(roleMappings []interface{}) []error {
	errors := make([]error, 0)

	for _, r := range roleMappings {
		rm := r.(map[string]interface{})

		// If Type equals "Token" or "Rules", ambiguous_role_resolution must be defined.
		// This should be removed as soon as we can have a ValidateFuncAgainst callable on the schema.
		if err := validateCognitoRoleMappingsAmbiguousRoleResolutionAgainstType(rm); len(err) > 0 {
			errors = append(errors, fmt.Errorf("Role Mapping %q: %v", rm["identity_provider"].(string), err))
		}

		// Validating that Rules Configuration is defined when Type equals Rules
		// but not defined when Type equals Token.
		if err := validateCognitoRoleMappingsRulesConfiguration(rm); len(err) > 0 {
			errors = append(errors, fmt.Errorf("Role Mapping %q: %v", rm["identity_provider"].(string), err))
		}
	}

	return errors
}

func cognitoRoleMappingHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["identity_provider"].(string)))

	return hashcode.String(buf.String())
}

func cognitoRoleMappingValueHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["type"].(string)))
	if d, ok := m["ambiguous_role_resolution"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", d.(string)))
	}

	return hashcode.String(buf.String())
}

func cognitoRoleMappingRulesConfigurationHash(v interface{}) int {
	var buf bytes.Buffer
	for _, rule := range v.([]interface{}) {
		r := rule.(map[string]interface{})
		buf.WriteString(fmt.Sprintf("%s-", r["claim"].(string)))
		buf.WriteString(fmt.Sprintf("%s-", r["match_type"].(string)))
		buf.WriteString(fmt.Sprintf("%s-", r["role_arn"].(string)))
		buf.WriteString(fmt.Sprintf("%s-", r["value"].(string)))
	}

	return hashcode.String(buf.String())
}
