package vault

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/hashicorp/vault/api"
)

func genericSecretResource() *schema.Resource {
	return &schema.Resource{
		Create: genericSecretResourceWrite,
		Update: genericSecretResourceWrite,
		Delete: genericSecretResourceDelete,
		Read:   genericSecretResourceRead,

		Schema: map[string]*schema.Schema{
			"path": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Full path where the generic secret will be written.",
			},

			// Data is passed as JSON so that an arbitrary structure is
			// possible, rather than forcing e.g. all values to be strings.
			"data_json": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "JSON-encoded secret data to write.",
				// We rebuild the attached JSON string to a simple singleline
				// string. This makes terraform not want to change when an extra
				// space is included in the JSON string. It is also necesarry
				// when allow_read is true for comparing values.
				StateFunc:    NormalizeDataJSON,
				ValidateFunc: ValidateDataJSON,
			},

			"allow_read": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "True if the provided token is allowed to read the secret from vault",
			},
		},
	}
}

func ValidateDataJSON(configI interface{}, k string) ([]string, []error) {
	dataJSON := configI.(string)
	dataMap := map[string]interface{}{}
	err := json.Unmarshal([]byte(dataJSON), &dataMap)
	if err != nil {
		return nil, []error{err}
	}
	return nil, nil
}

func NormalizeDataJSON(configI interface{}) string {
	dataJSON := configI.(string)

	dataMap := map[string]interface{}{}
	err := json.Unmarshal([]byte(dataJSON), &dataMap)
	if err != nil {
		// The validate function should've taken care of this.
		return ""
	}

	ret, err := json.Marshal(dataMap)
	if err != nil {
		// Should never happen.
		return dataJSON
	}

	return string(ret)
}

func genericSecretResourceWrite(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	path := d.Get("path").(string)

	var data map[string]interface{}
	err := json.Unmarshal([]byte(d.Get("data_json").(string)), &data)
	if err != nil {
		return fmt.Errorf("data_json %#v syntax error: %s", d.Get("data_json"), err)
	}

	log.Printf("[DEBUG] Writing generic Vault secret to %s", path)
	_, err = client.Logical().Write(path, data)
	if err != nil {
		return fmt.Errorf("error writing to Vault: %s", err)
	}

	d.SetId(path)

	return nil
}

func genericSecretResourceDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	path := d.Id()

	log.Printf("[DEBUG] Deleting generic Vault from %s", path)
	_, err := client.Logical().Delete(path)
	if err != nil {
		return fmt.Errorf("error deleting from Vault: %s", err)
	}

	return nil
}

func genericSecretResourceRead(d *schema.ResourceData, meta interface{}) error {
	allowed_to_read := d.Get("allow_read").(bool)
	path := d.Get("path").(string)

	if allowed_to_read {
		client := meta.(*api.Client)

		log.Printf("[DEBUG] Reading %s from Vault", path)
		secret, err := client.Logical().Read(path)
		if err != nil {
			return fmt.Errorf("error reading from Vault: %s", err)
		}

		// Ignoring error because this value came from JSON in the
		// first place so no reason why it should fail to re-encode.
		jsonDataBytes, _ := json.Marshal(secret.Data)
		d.Set("data_json", string(jsonDataBytes))
	}

	d.SetId(path)
	log.Printf("[WARN] vault_generic_secret does not automatically refresh if allow_read is set to false")
	return nil
}
