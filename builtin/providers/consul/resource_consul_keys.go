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

func resource_consul_keys_create(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)

	// Merge the diff into the state so that we have all the attributes
	// properly.
	rs := s.MergeDiff(d)

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
		return s, fmt.Errorf("Failed to unroll keys")
	}

	kv := p.client.KV()
	qOpts := consulapi.QueryOptions{Datacenter: dc}
	wOpts := consulapi.WriteOptions{Datacenter: dc}
	for _, raw := range keys {
		sub := raw.(map[string]interface{})
		if !ok {
			return s, fmt.Errorf("Failed to unroll: %#v", raw)
		}

		key, ok := sub["name"].(string)
		if !ok {
			return s, fmt.Errorf("Failed to expand key '%#v'", sub)
		}

		path, ok := sub["path"].(string)
		if !ok {
			return s, fmt.Errorf("Failed to get path for key '%s'", key)
		}

		valueRaw, shouldSet := sub["value"]
		if shouldSet {
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
		} else {
			log.Printf("[DEBUG] Getting key '%s' in %s", path, dc)
			pair, _, err := kv.Get(path, &qOpts)
			if err != nil {
				return rs, fmt.Errorf("Failed to get Consul key '%s': %v", path, err)
			}

			// Check for a default value
			var defaultVal string
			setDefault := false
			if raw, ok := sub["default"]; ok {
				switch def := raw.(type) {
				case string:
					setDefault = true
					defaultVal = def
				case bool:
					setDefault = true
					defaultVal = strconv.FormatBool(def)
				}
			}

			if pair == nil && setDefault {
				rs.Attributes[fmt.Sprintf("var.%s", key)] = defaultVal
			} else if pair == nil {
				rs.Attributes[fmt.Sprintf("var.%s", key)] = ""
			} else {
				rs.Attributes[fmt.Sprintf("var.%s", key)] = string(pair.Value)
			}
		}
	}

	// Set an ID
	rs.ID = "consul"
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
		sub := raw.(map[string]interface{})
		if !ok {
			return fmt.Errorf("Failed to unroll: %#v", raw)
		}

		// Ignore if the key is non-managed
		shouldDelete, ok := sub["delete"].(bool)
		if !ok || !shouldDelete {
			continue
		}

		key, ok := sub["name"].(string)
		if !ok {
			return fmt.Errorf("Failed to expand key '%#v'", sub)
		}

		path, ok := sub["path"].(string)
		if !ok {
			return fmt.Errorf("Failed to get path for key '%s'", key)
		}

		log.Printf("[DEBUG] Deleting key '%s' in %s", path, dc)
		_, err := kv.Delete(path, &wOpts)
		if err != nil {
			return fmt.Errorf("Failed to delete Consul key '%s': %v", path, err)
		}
	}
	return nil
}

func resource_consul_keys_update(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	panic("cannot update")
	return s, nil
}

func resource_consul_keys_diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.ResourceDiff, error) {

	// Get the list of keys
	var computed []string
	keys, ok := flatmap.Expand(flatmap.Flatten(c.Config), "key").([]interface{})
	if !ok {
		goto AFTER
	}
	for _, sub := range keys {
		subMap, ok := sub.(map[string]interface{})
		if !ok {
			continue
		}
		nameRaw, ok := subMap["name"]
		if !ok {
			continue
		}
		name, ok := nameRaw.(string)
		if !ok {
			continue
		}
		computed = append(computed, "var."+name)
	}

AFTER:
	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"datacenter": diff.AttrTypeCreate,
			"key":        diff.AttrTypeUpdate,
		},
		ComputedAttrs: computed,
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
	for _, raw := range keys {
		sub := raw.(map[string]interface{})
		if !ok {
			return s, fmt.Errorf("Failed to unroll: %#v", raw)
		}

		key, ok := sub["name"].(string)
		if !ok {
			return s, fmt.Errorf("Failed to expand key '%#v'", sub)
		}

		path, ok := sub["path"].(string)
		if !ok {
			return s, fmt.Errorf("Failed to get path for key '%s'", key)
		}

		log.Printf("[DEBUG] Refreshing value of key '%s' in %s", path, dc)
		pair, _, err := kv.Get(path, &opts)
		if err != nil {
			return s, fmt.Errorf("Failed to get value for path '%s' from Consul: %v", path, err)
		}

		// Check for a default value
		var defaultVal string
		setDefault := false
		if raw, ok := sub["default"]; ok {
			switch def := raw.(type) {
			case string:
				setDefault = true
				defaultVal = def
			case bool:
				setDefault = true
				defaultVal = strconv.FormatBool(def)
			}
		}

		if pair == nil && setDefault {
			s.Attributes[fmt.Sprintf("var.%s", key)] = defaultVal
		} else if pair == nil {
			s.Attributes[fmt.Sprintf("var.%s", key)] = ""
		} else {
			s.Attributes[fmt.Sprintf("var.%s", key)] = string(pair.Value)
		}
	}
	return s, nil
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
