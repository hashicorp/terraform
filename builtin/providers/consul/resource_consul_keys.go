package consul

import (
	"bytes"
	"fmt"
	"log"
	"strconv"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceConsulKeys() *schema.Resource {
	return &schema.Resource{
		Create: resourceConsulKeysCreate,
		Update: resourceConsulKeysCreate,
		Read:   resourceConsulKeysRead,
		Delete: resourceConsulKeysDelete,

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
				Set: resourceConsulKeysHash,
			},

			"var": &schema.Schema{
				Type:     schema.TypeMap,
				Computed: true,
			},
		},
	}
}

func resourceConsulKeysHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["path"].(string)))
	return hashcode.String(buf.String())
}

func resourceConsulKeysCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*consulapi.Client)
	kv := client.KV()

	// Resolve the datacenter first, all the other keys are dependent
	// on this.
	var dc string
	if v, ok := d.GetOk("datacenter"); ok {
		dc = v.(string)
		log.Printf("[DEBUG] Consul datacenter: %s", dc)
	} else {
		log.Printf("[DEBUG] Resolving Consul datacenter...")
		var err error
		dc, err = getDC(client)
		if err != nil {
			return err
		}
	}
	var token string
	if v, ok := d.GetOk("token"); ok {
		token = v.(string)
	}

	// Setup the operations using the datacenter
	qOpts := consulapi.QueryOptions{Datacenter: dc, Token: token}
	wOpts := consulapi.WriteOptions{Datacenter: dc, Token: token}

	// Store the computed vars
	vars := make(map[string]string)

	// Extract the keys
	keys := d.Get("key").(*schema.Set).List()
	for _, raw := range keys {
		key, path, sub, err := parseKey(raw)
		if err != nil {
			return err
		}

		value := sub["value"].(string)
		if value != "" {
			log.Printf("[DEBUG] Setting key '%s' to '%v' in %s", path, value, dc)
			pair := consulapi.KVPair{Key: path, Value: []byte(value)}
			if _, err := kv.Put(&pair, &wOpts); err != nil {
				return fmt.Errorf("Failed to set Consul key '%s': %v", path, err)
			}
			vars[key] = value
			sub["value"] = value

		} else {
			log.Printf("[DEBUG] Getting key '%s' in %s", path, dc)
			pair, _, err := kv.Get(path, &qOpts)
			if err != nil {
				return fmt.Errorf("Failed to get Consul key '%s': %v", path, err)
			}
			value := attributeValue(sub, key, pair)
			vars[key] = value
		}
	}

	// Update the resource
	d.SetId("consul")
	d.Set("datacenter", dc)
	d.Set("key", keys)
	d.Set("var", vars)
	return nil
}

func resourceConsulKeysRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*consulapi.Client)
	kv := client.KV()

	// Get the DC, error if not available.
	var dc string
	if v, ok := d.GetOk("datacenter"); ok {
		dc = v.(string)
		log.Printf("[DEBUG] Consul datacenter: %s", dc)
	} else {
		return fmt.Errorf("Missing datacenter configuration")
	}
	var token string
	if v, ok := d.GetOk("token"); ok {
		token = v.(string)
	}

	// Setup the operations using the datacenter
	qOpts := consulapi.QueryOptions{Datacenter: dc, Token: token}

	// Store the computed vars
	vars := make(map[string]string)

	// Extract the keys
	keys := d.Get("key").(*schema.Set).List()
	for _, raw := range keys {
		key, path, sub, err := parseKey(raw)
		if err != nil {
			return err
		}

		log.Printf("[DEBUG] Refreshing value of key '%s' in %s", path, dc)
		pair, _, err := kv.Get(path, &qOpts)
		if err != nil {
			return fmt.Errorf("Failed to get value for path '%s' from Consul: %v", path, err)
		}

		value := attributeValue(sub, key, pair)
		vars[key] = value
		sub["value"] = value
	}

	// Update the resource
	d.Set("key", keys)
	d.Set("var", vars)
	return nil
}

func resourceConsulKeysDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*consulapi.Client)
	kv := client.KV()

	// Get the DC, error if not available.
	var dc string
	if v, ok := d.GetOk("datacenter"); ok {
		dc = v.(string)
		log.Printf("[DEBUG] Consul datacenter: %s", dc)
	} else {
		return fmt.Errorf("Missing datacenter configuration")
	}
	var token string
	if v, ok := d.GetOk("token"); ok {
		token = v.(string)
	}

	// Setup the operations using the datacenter
	wOpts := consulapi.WriteOptions{Datacenter: dc, Token: token}

	// Extract the keys
	keys := d.Get("key").(*schema.Set).List()
	for _, raw := range keys {
		_, path, sub, err := parseKey(raw)
		if err != nil {
			return err
		}

		// Ignore if the key is non-managed
		shouldDelete, ok := sub["delete"].(bool)
		if !ok || !shouldDelete {
			continue
		}

		log.Printf("[DEBUG] Deleting key '%s' in %s", path, dc)
		if _, err := kv.Delete(path, &wOpts); err != nil {
			return fmt.Errorf("Failed to delete Consul key '%s': %v", path, err)
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
func attributeValue(sub map[string]interface{}, key string, pair *consulapi.KVPair) string {
	// Use the value if given
	if pair != nil {
		return string(pair.Value)
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
func getDC(client *consulapi.Client) (string, error) {
	info, err := client.Agent().Self()
	if err != nil {
		return "", fmt.Errorf("Failed to get datacenter from Consul agent: %v", err)
	}
	dc := info["Config"]["Datacenter"].(string)
	return dc, nil
}
