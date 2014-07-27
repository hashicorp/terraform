package consul

import (
	"fmt"
	"log"
	"strconv"

	"github.com/armon/consul-api"
	"github.com/hashicorp/terraform/flatmap"
	"github.com/hashicorp/terraform/helper/config"
	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/terraform"
)

func resource_consul_keys_validation() *config.Validator {
	return &config.Validator{
		Required: []string{
			"key.*.name",
			"key.*.path",
		},
		Optional: []string{
			"datacenter",
			"key.*.value",
			"key.*.default",
			"key.*.delete",
		},
	}
}
func resource_consul_keys_update(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	return resource_consul_keys_create(s, d, meta)
}

func resource_consul_keys_create(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)

	// Merge the diff into the state so that we have all the attributes
	// properly.
	rs := s.MergeDiff(d)
	rs.ID = "consul"

	// Check if the datacenter should be computed
	dc := rs.Attributes["datacenter"]
	if aDiff, ok := d.Attributes["datacenter"]; ok && aDiff.NewComputed {
		var err error
		dc, err = get_dc(p.client)
		if err != nil {
			return rs, fmt.Errorf("Failed to get agent datacenter: %v", err)
		}
		rs.Attributes["datacenter"] = dc
	}

	// Get the keys
	keys, ok := flatmap.Expand(rs.Attributes, "key").([]interface{})
	if !ok {
		return rs, fmt.Errorf("Failed to unroll keys")
	}

	kv := p.client.KV()
	qOpts := consulapi.QueryOptions{Datacenter: dc}
	wOpts := consulapi.WriteOptions{Datacenter: dc}
	for idx, raw := range keys {
		key, path, sub, err := parse_key(raw)
		if err != nil {
			return rs, err
		}

		if valueRaw, ok := sub["value"]; ok {
			value, ok := valueRaw.(string)
			if !ok {
				return rs, fmt.Errorf("Failed to get value for key '%s'", key)
			}

			log.Printf("[DEBUG] Setting key '%s' to '%v' in %s", path, value, dc)
			pair := consulapi.KVPair{Key: path, Value: []byte(value)}
			if _, err := kv.Put(&pair, &wOpts); err != nil {
				return rs, fmt.Errorf("Failed to set Consul key '%s': %v", path, err)
			}
			rs.Attributes[fmt.Sprintf("var.%s", key)] = value
			rs.Attributes[fmt.Sprintf("key.%d.value", idx)] = value

		} else {
			log.Printf("[DEBUG] Getting key '%s' in %s", path, dc)
			pair, _, err := kv.Get(path, &qOpts)
			if err != nil {
				return rs, fmt.Errorf("Failed to get Consul key '%s': %v", path, err)
			}
			rs.Attributes[fmt.Sprintf("var.%s", key)] = attribute_value(sub, key, pair)
		}
	}
	return rs, nil
}

func resource_consul_keys_destroy(
	s *terraform.ResourceState,
	meta interface{}) error {
	p := meta.(*ResourceProvider)
	client := p.client
	kv := client.KV()

	// Get the keys
	keys, ok := flatmap.Expand(s.Attributes, "key").([]interface{})
	if !ok {
		return fmt.Errorf("Failed to unroll keys")
	}

	dc := s.Attributes["datacenter"]
	wOpts := consulapi.WriteOptions{Datacenter: dc}
	for _, raw := range keys {
		_, path, sub, err := parse_key(raw)
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
	return nil
}

func resource_consul_keys_diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.ResourceDiff, error) {

	// Determine the list of computed variables
	var computed []string
	keys, ok := flatmap.Expand(flatmap.Flatten(c.Config), "key").([]interface{})
	if !ok {
		goto AFTER
	}
	for _, sub := range keys {
		key, _, _, err := parse_key(sub)
		if err != nil {
			continue
		}
		computed = append(computed, "var."+key)
	}

AFTER:
	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"datacenter": diff.AttrTypeCreate,
			"key":        diff.AttrTypeUpdate,
		},
		ComputedAttrsUpdate: computed,
	}
	return b.Diff(s, c)
}

func resource_consul_keys_refresh(
	s *terraform.ResourceState,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	client := p.client
	kv := client.KV()

	// Get the list of keys
	keys, ok := flatmap.Expand(s.Attributes, "key").([]interface{})
	if !ok {
		return s, fmt.Errorf("Failed to unroll keys")
	}

	// Update each key
	dc := s.Attributes["datacenter"]
	opts := consulapi.QueryOptions{Datacenter: dc}
	for idx, raw := range keys {
		key, path, sub, err := parse_key(raw)
		if err != nil {
			return s, err
		}

		log.Printf("[DEBUG] Refreshing value of key '%s' in %s", path, dc)
		pair, _, err := kv.Get(path, &opts)
		if err != nil {
			return s, fmt.Errorf("Failed to get value for path '%s' from Consul: %v", path, err)
		}

		setVal := attribute_value(sub, key, pair)
		s.Attributes[fmt.Sprintf("var.%s", key)] = setVal
		if _, ok := sub["value"]; ok {
			s.Attributes[fmt.Sprintf("key.%d.value", idx)] = setVal
		}
	}
	return s, nil
}

// parse_key is used to parse a key into a name, path, config or error
func parse_key(raw interface{}) (string, string, map[string]interface{}, error) {
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

// attribute_value determienes the value for a key
func attribute_value(sub map[string]interface{}, key string, pair *consulapi.KVPair) string {
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

// get_dc is used to get the datacenter of the local agent
func get_dc(client *consulapi.Client) (string, error) {
	info, err := client.Agent().Self()
	if err != nil {
		return "", fmt.Errorf("Failed to get datacenter from Consul agent: %v", err)
	}
	dc := info["Config"]["Datacenter"].(string)
	return dc, nil
}
