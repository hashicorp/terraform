package vault

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/hashicorp/vault/api"
)

func genericSecretDataSource() *schema.Resource {
	return &schema.Resource{
		Read: genericSecretDataSourceRead,

		Schema: map[string]*schema.Schema{
			"path": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Full path from which a secret will be read.",
			},

			"data_json": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "JSON-encoded secret data read from Vault.",
			},

			"data": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "Map of strings read from Vault.",
			},

			"lease_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Lease identifier assigned by vault.",
			},

			"lease_duration": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Lease duration in seconds relative to the time in lease_start_time.",
			},

			"lease_start_time": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Time at which the lease was read, using the clock of the system where Terraform was running",
			},

			"lease_renewable": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "True if the duration of this lease can be extended through renewal.",
			},
		},
	}
}

func genericSecretDataSourceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	path := d.Get("path").(string)

	log.Printf("[DEBUG] Reading %s from Vault", path)
	secret, err := client.Logical().Read(path)
	if err != nil {
		return fmt.Errorf("error reading from Vault: %s", err)
	}
	if secret == nil {
		return fmt.Errorf("No secret found at %q", path)
	}

	d.SetId(secret.RequestID)

	// Ignoring error because this value came from JSON in the
	// first place so no reason why it should fail to re-encode.
	jsonDataBytes, _ := json.Marshal(secret.Data)
	d.Set("data_json", string(jsonDataBytes))

	// Since our "data" map can only contain string values, we
	// will take strings from Data and write them in as-is,
	// and write everything else in as a JSON serialization of
	// whatever value we get so that complex types can be
	// passed around and processed elsewhere if desired.
	dataMap := map[string]string{}
	for k, v := range secret.Data {
		if vs, ok := v.(string); ok {
			dataMap[k] = vs
		} else {
			// Again ignoring error because we know this value
			// came from JSON in the first place and so must be valid.
			vBytes, _ := json.Marshal(v)
			dataMap[k] = string(vBytes)
		}
	}
	d.Set("data", dataMap)

	d.Set("lease_id", secret.LeaseID)
	d.Set("lease_duration", secret.LeaseDuration)
	d.Set("lease_start_time", time.Now().Format("RFC3339"))
	d.Set("lease_renewable", secret.Renewable)

	return nil
}
