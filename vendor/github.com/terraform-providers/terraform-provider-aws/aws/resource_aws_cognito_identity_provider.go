package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsCognitoIdentityProvider() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCognitoIdentityProviderCreate,
		Read:   resourceAwsCognitoIdentityProviderRead,
		Update: resourceAwsCognitoIdentityProviderUpdate,
		Delete: resourceAwsCognitoIdentityProviderDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"attribute_mapping": {
				Type:     schema.TypeMap,
				Optional: true,
			},

			"idp_identifiers": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"provider_details": {
				Type:     schema.TypeMap,
				Required: true,
			},

			"provider_name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"provider_type": {
				Type:     schema.TypeString,
				Required: true,
			},

			"user_pool_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsCognitoIdentityProviderCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cognitoidpconn
	log.Print("[DEBUG] Creating Cognito Identity Provider")

	providerName := d.Get("provider_name").(string)
	userPoolID := d.Get("user_pool_id").(string)
	params := &cognitoidentityprovider.CreateIdentityProviderInput{
		ProviderName: aws.String(providerName),
		ProviderType: aws.String(d.Get("provider_type").(string)),
		UserPoolId:   aws.String(userPoolID),
	}

	if v, ok := d.GetOk("attribute_mapping"); ok {
		params.AttributeMapping = stringMapToPointers(v.(map[string]interface{}))
	}

	if v, ok := d.GetOk("provider_details"); ok {
		params.ProviderDetails = stringMapToPointers(v.(map[string]interface{}))
	}

	if v, ok := d.GetOk("idp_identifiers"); ok {
		params.IdpIdentifiers = expandStringList(v.([]interface{}))
	}

	_, err := conn.CreateIdentityProvider(params)
	if err != nil {
		return fmt.Errorf("Error creating Cognito Identity Provider: %s", err)
	}

	d.SetId(fmt.Sprintf("%s:%s", userPoolID, providerName))

	return resourceAwsCognitoIdentityProviderRead(d, meta)
}

func resourceAwsCognitoIdentityProviderRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cognitoidpconn
	log.Printf("[DEBUG] Reading Cognito Identity Provider: %s", d.Id())

	userPoolID, providerName, err := decodeCognitoIdentityProviderID(d.Id())
	if err != nil {
		return err
	}

	ret, err := conn.DescribeIdentityProvider(&cognitoidentityprovider.DescribeIdentityProviderInput{
		ProviderName: aws.String(providerName),
		UserPoolId:   aws.String(userPoolID),
	})

	if err != nil {
		if isAWSErr(err, cognitoidentityprovider.ErrCodeResourceNotFoundException, "") {
			log.Printf("[WARN] Cognito Identity Provider %q not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	if ret == nil || ret.IdentityProvider == nil {
		log.Printf("[WARN] Cognito Identity Provider %q not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	ip := ret.IdentityProvider
	d.Set("provider_name", ip.ProviderName)
	d.Set("provider_type", ip.ProviderType)
	d.Set("user_pool_id", ip.UserPoolId)

	if err := d.Set("attribute_mapping", aws.StringValueMap(ip.AttributeMapping)); err != nil {
		return fmt.Errorf("error setting attribute_mapping error: %s", err)
	}

	if err := d.Set("provider_details", aws.StringValueMap(ip.ProviderDetails)); err != nil {
		return fmt.Errorf("error setting provider_details error: %s", err)
	}

	if err := d.Set("idp_identifiers", flattenStringList(ip.IdpIdentifiers)); err != nil {
		return fmt.Errorf("error setting idp_identifiers error: %s", err)
	}

	return nil
}

func resourceAwsCognitoIdentityProviderUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cognitoidpconn
	log.Print("[DEBUG] Updating Cognito Identity Provider")

	userPoolID, providerName, err := decodeCognitoIdentityProviderID(d.Id())
	if err != nil {
		return err
	}

	params := &cognitoidentityprovider.UpdateIdentityProviderInput{
		ProviderName: aws.String(providerName),
		UserPoolId:   aws.String(userPoolID),
	}

	if d.HasChange("attribute_mapping") {
		params.AttributeMapping = stringMapToPointers(d.Get("attribute_mapping").(map[string]interface{}))
	}

	if d.HasChange("provider_details") {
		params.ProviderDetails = stringMapToPointers(d.Get("provider_details").(map[string]interface{}))
	}

	if d.HasChange("idp_identifiers") {
		params.IdpIdentifiers = expandStringList(d.Get("supported_login_providers").([]interface{}))
	}

	_, err = conn.UpdateIdentityProvider(params)
	if err != nil {
		return fmt.Errorf("Error updating Cognito Identity Provider: %s", err)
	}

	return resourceAwsCognitoIdentityProviderRead(d, meta)
}

func resourceAwsCognitoIdentityProviderDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cognitoidpconn
	log.Printf("[DEBUG] Deleting Cognito Identity Provider: %s", d.Id())

	userPoolID, providerName, err := decodeCognitoIdentityProviderID(d.Id())
	if err != nil {
		return err
	}

	_, err = conn.DeleteIdentityProvider(&cognitoidentityprovider.DeleteIdentityProviderInput{
		ProviderName: aws.String(providerName),
		UserPoolId:   aws.String(userPoolID),
	})

	if err != nil {
		if isAWSErr(err, cognitoidentityprovider.ErrCodeResourceNotFoundException, "") {
			return nil
		}
		return err
	}

	return nil
}

func decodeCognitoIdentityProviderID(id string) (string, string, error) {
	idParts := strings.Split(id, ":")
	if len(idParts) != 2 {
		return "", "", fmt.Errorf("expected ID in format UserPoolID:ProviderName, received: %s", id)
	}
	return idParts[0], idParts[1], nil
}
