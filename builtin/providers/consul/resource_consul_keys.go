package consul

import (
	"fmt"
	"strconv"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceConsulKeys() *schema.Resource {
	return &schema.Resource{
		Create: resourceConsulKeysCreate,
		Update: resourceConsulKeysUpdate,
		Read:   resourceConsulKeysRead,
		Delete: resourceConsulKeysDelete,

		SchemaVersion: 1,
		MigrateState:  resourceConsulKeysMigrateState,

		Schema: map[string]*schema.Schema{
			"datacenter": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"token": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"key": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"path": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"value": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"default": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"delete": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
					},
				},
			},

			"var": &schema.Schema{
				Type:     schema.TypeMap,
				Computed: true,
			},
		},
	}
}

func resourceConsulKeysCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*consulapi.Client)
	kv := client.KV()
	token := d.Get("token").(string)
	dc, err := getDC(d, client)
	if err != nil {
		return err
	}

	keyClient := newKeyClient(kv, dc, token)

	keys := d.Get("key").(*schema.Set).List()
	for _, raw := range keys {
		_, path, sub, err := parseKey(raw)
		if err != nil {
			return err
		}

		value := sub["value"].(string)
		if value == "" {
			continue
		}

		if err := keyClient.Put(path, value); err != nil {
			return err
		}
	}

	// The ID doesn't matter, since we use provider config, datacenter,
	// and key paths to address consul properly. So we just need to fill it in
	// with some value to indicate the resource has been created.
	d.SetId("consul")

	return resourceConsulKeysRead(d, meta)
}

func resourceConsulKeysUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*consulapi.Client)
	kv := client.KV()
	token := d.Get("token").(string)
	dc, err := getDC(d, client)
	if err != nil {
		return err
	}

	keyClient := newKeyClient(kv, dc, token)

	if d.HasChange("key") {
		o, n := d.GetChange("key")
		if o == nil {
			o = new(schema.Set)
		}
		if n == nil {
			n = new(schema.Set)
		}

		os := o.(*schema.Set)
		ns := n.(*schema.Set)

		remove := os.Difference(ns).List()
		add := ns.Difference(os).List()

		// We'll keep track of what keys we add so that if a key is
		// in both the "remove" and "add" sets -- which will happen if
		// its value is changed in-place -- we will avoid writing the
		// value and then immediately removing it.
		addedPaths := make(map[string]bool)

		// We add before we remove because then it's possible to change
		// a key name (which will result in both an add and a remove)
		// without very temporarily having *neither* value in the store.
		// Instead, both will briefly be present, which should be less
		// disruptive in most cases.
		for _, raw := range add {
			_, path, sub, err := parseKey(raw)
			if err != nil {
				return err
			}

			value := sub["value"].(string)
			if value == "" {
				continue
			}

			if err := keyClient.Put(path, value); err != nil {
				return err
			}
			addedPaths[path] = true
		}

		for _, raw := range remove {
			_, path, sub, err := parseKey(raw)
			if err != nil {
				return err
			}

			// Don't delete something we've just added.
			// (See explanation at the declaration of this variable above.)
			if addedPaths[path] {
				continue
			}

			shouldDelete, ok := sub["delete"].(bool)
			if !ok || !shouldDelete {
				continue
			}

			if err := keyClient.Delete(path); err != nil {
				return err
			}
		}
	}

	// Store the datacenter on this resource, which can be helpful for reference
	// in case it was read from the provider
	d.Set("datacenter", dc)

	return resourceConsulKeysRead(d, meta)
}

func resourceConsulKeysRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*consulapi.Client)
	kv := client.KV()
	token := d.Get("token").(string)
	dc, err := getDC(d, client)
	if err != nil {
		return err
	}

	keyClient := newKeyClient(kv, dc, token)

	vars := make(map[string]string)

	keys := d.Get("key").(*schema.Set).List()
	for _, raw := range keys {
		key, path, sub, err := parseKey(raw)
		if err != nil {
			return err
		}

		value, err := keyClient.Get(path)
		if err != nil {
			return err
		}

		value = attributeValue(sub, value)
		vars[key] = value

		// If there is already a "value" attribute present for this key
		// then it was created as a "write" block. We need to update the
		// given value within the block itself so that Terraform can detect
		// when the Consul-stored value has drifted from what was most
		// recently written by Terraform.
		// We don't do this for "read" blocks; that causes confusing diffs
		// because "value" should not be set for read-only key blocks.
		if oldValue := sub["value"]; oldValue != "" {
			sub["value"] = value
		}
	}

	if err := d.Set("var", vars); err != nil {
		return err
	}
	if err := d.Set("key", keys); err != nil {
		return err
	}

	// Store the datacenter on this resource, which can be helpful for reference
	// in case it was read from the provider
	d.Set("datacenter", dc)

	return nil
}

func resourceConsulKeysDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*consulapi.Client)
	kv := client.KV()
	token := d.Get("token").(string)
	dc, err := getDC(d, client)
	if err != nil {
		return err
	}

	keyClient := newKeyClient(kv, dc, token)

	// Clean up any keys that we're explicitly managing
	keys := d.Get("key").(*schema.Set).List()
	for _, raw := range keys {
		_, path, sub, err := parseKey(raw)
		if err != nil {
			return err
		}

		// Skip if the key is non-managed
		shouldDelete, ok := sub["delete"].(bool)
		if !ok || !shouldDelete {
			continue
		}

		if err := keyClient.Delete(path); err != nil {
			return err
		}
	}

	// Clear the ID
	d.SetId("")
	return nil
}

// parseKey is used to parse a key into a name, path, config or error
func parseKey(raw interface{}) (string, string, map[string]interface{}, error) {
	sub, ok := raw.(map[string]interface{})
	if !ok {
		return "", "", nil, fmt.Errorf("Failed to unroll: %#v", raw)
	}

	key, ok := sub["name"].(string)
	if !ok {
		return "", "", nil, fmt.Errorf("Failed to expand key '%#v'", sub)
	}

	path, ok := sub["path"].(string)
	if !ok {
		return "", "", nil, fmt.Errorf("Failed to get path for key '%s'", key)
	}
	return key, path, sub, nil
}

// attributeValue determines the value for a key, potentially
// using a default value if provided.
func attributeValue(sub map[string]interface{}, readValue string) string {
	// Use the value if given
	if readValue != "" {
		return readValue
	}

	// Use a default if given
	if raw, ok := sub["default"]; ok {
		switch def := raw.(type) {
		case string:
			return def
		case bool:
			return strconv.FormatBool(def)
		}
	}

	// No value
	return ""
}

// getDC is used to get the datacenter of the local agent
func getDC(d *schema.ResourceData, client *consulapi.Client) (string, error) {
	if v, ok := d.GetOk("datacenter"); ok {
		return v.(string), nil
	}
	info, err := client.Agent().Self()
	if err != nil {
		return "", fmt.Errorf("Failed to get datacenter from Consul agent: %v", err)
	}
	return info["Config"]["Datacenter"].(string), nil
}
