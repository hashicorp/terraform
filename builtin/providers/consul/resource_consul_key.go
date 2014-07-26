package consul

import (
	"fmt"
	"log"

	"github.com/armon/consul-api"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/mapstructure"
)

type consulKeys map[string]*consulKey

type consulKey struct {
	Key     string
	Value   string
	Default string
	Delete  bool

	SetValue   bool `mapstructure:"-"`
	SetDefault bool `mapstructure:"-"`
}

func resource_consul_keys_validate(c *terraform.ResourceConfig) (ws []string, es []error) {
	conf := c.Raw
	for k, v := range conf {
		// datacenter is special and can be ignored
		if k == "datacenter" {
			continue
		}

		keyList, ok := v.([]map[string]interface{})
		if !ok {
			es = append(es, fmt.Errorf("Field '%s' must be map containing a key", k))
			continue
		}
		if len(keyList) > 1 {
			es = append(es, fmt.Errorf("Field '%s' is defined more than once", k))
			continue
		}
		key := keyList[0]

		for sub, val := range key {
			// Verify the sub-key is supported
			switch sub {
			case "key":
			case "value":
			case "default":
			case "delete":
			default:
				es = append(es, fmt.Errorf("Field '%s' has unsupported config '%s'", k, sub))
				continue
			}

			// Verify value is of the correct type
			_, isStr := val.(string)
			_, isBool := val.(bool)
			if !isStr && sub != "delete" {
				es = append(es, fmt.Errorf("Field '%s' must set '%s' as a string", key, sub))
			}
			if !isBool && sub == "delete" {
				es = append(es, fmt.Errorf("Field '%s' must set '%s' as a bool", key, sub))
			}
		}
	}
	return
}

func resource_consul_keys_create(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	if s.Attributes == nil {
		s.Attributes = make(map[string]string)
	}

	// Load the configuration
	var config map[string]interface{}
	for _, attr := range d.Attributes {
		if attr.NewExtra != nil {
			config = attr.NewExtra.(map[string]interface{})
			break
		}
	}
	if config == nil {
		return s, fmt.Errorf("Missing configuration state")
	}
	dc, keys, err := partsFromConfig(config)
	if err != nil {
		return s, err
	}

	// Check if we are missing a datacenter
	if dc == "" {
		dc, err = get_dc(p.client)
	}
	s.Attributes["datacenter"] = dc

	// Handle each of the keys
	kv := p.client.KV()
	qOpts := consulapi.QueryOptions{Datacenter: dc}
	wOpts := consulapi.WriteOptions{Datacenter: dc}
	for name, conf := range keys {
		if conf.SetValue {
			log.Printf("[DEBUG] Setting key '%s' to '%v' in %s", conf.Key, conf.Value, dc)
			pair := consulapi.KVPair{Key: conf.Key, Value: []byte(conf.Value)}
			if _, err := kv.Put(&pair, &wOpts); err != nil {
				return s, fmt.Errorf("Failed to set Consul key '%s': %v", conf.Key, err)
			}
			s.Attributes[name] = conf.Value
		} else {
			log.Printf("[DEBUG] Getting key '%s' in %s", conf.Key, dc)
			pair, _, err := kv.Get(conf.Key, &qOpts)
			if err != nil {
				return s, fmt.Errorf("Failed to get Consul key '%s': %v", conf.Key, err)
			}
			if pair == nil && conf.SetDefault {
				s.Attributes[name] = conf.Default
			} else if pair == nil {
				s.Attributes[name] = ""
			} else {
				s.Attributes[name] = string(pair.Value)
			}
		}
	}

	// Set an ID, store the config
	s.ID = "consul"
	s.Extra = config
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

func resource_consul_keys_destroy(
	s *terraform.ResourceState,
	meta interface{}) error {
	p := meta.(*ResourceProvider)
	client := p.client
	kv := client.KV()

	// Restore our configuration
	dc, keys, err := partsFromConfig(s.Extra)
	if err != nil {
		return err
	}

	// Load the DC if not given
	if dc == "" {
		dc = s.Attributes["datacenter"]
	}
	opts := consulapi.WriteOptions{Datacenter: dc}
	for _, key := range keys {
		// Skip any non-managed keys
		if !key.Delete {
			continue
		}
		log.Printf("[DEBUG] Deleting key '%s' in %s", key.Key, dc)
		if _, err := kv.Delete(key.Key, &opts); err != nil {
			return fmt.Errorf("Failed to delete Consul key '%s': %v", key.Key, err)
		}
	}
	return nil
}

func resource_consul_keys_update(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	// TODO
	panic("update not supported")
	return s, nil
}

func resource_consul_keys_diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.ResourceDiff, error) {
	// Parse the configuration
	dc, keys, err := partsFromConfig(c.Config)
	if err != nil {
		return nil, err
	}

	// Get the old values
	oldValues := s.Attributes

	// Initialize the diff set
	attrs := make(map[string]*terraform.ResourceAttrDiff)
	diff := &terraform.ResourceDiff{Attributes: attrs}

	// Handle removed attributes
	for key, oldVal := range oldValues {
		if key == "datacenter" {
			continue
		}
		if _, keep := keys[key]; !keep {
			attrs[key] = &terraform.ResourceAttrDiff{
				Old:        oldVal,
				NewRemoved: true,
			}
		}
	}

	// Handle added or changed attributes
	for key, conf := range keys {
		aDiff := &terraform.ResourceAttrDiff{
			Type: terraform.DiffAttrInput,
		}
		oldVal, ok := oldValues[key]
		if conf.SetValue {
			aDiff.New = conf.Value
		} else {
			aDiff.NewComputed = true
		}
		if ok {
			aDiff.Old = oldVal
		}

		// If this is new or changed we need to refresh
		if !ok || (conf.SetValue && oldVal != conf.Value) {
			attrs[key] = aDiff
		}
	}

	// If the DC has changed, require a destroy!
	if old := oldValues["datacenter"]; dc != old {
		aDiff := &terraform.ResourceAttrDiff{
			Old:         old,
			New:         dc,
			RequiresNew: true,
			Type:        terraform.DiffAttrInput,
		}
		if aDiff.New == "" {
			aDiff.NewComputed = true
		}
		attrs["datacenter"] = aDiff
	}

	// Make sure one of the attributes contains the configuration
	if len(attrs) > 0 {
		for _, aDiff := range attrs {
			aDiff.NewExtra = c.Config
			break
		}
	}
	return diff, nil
}

func resource_consul_keys_refresh(
	s *terraform.ResourceState,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	client := p.client
	agent := client.Agent()
	kv := client.KV()

	// Restore our configuration
	dc, keys, err := partsFromConfig(s.Extra)
	if err != nil {
		return s, err
	}

	// Check if we are missing a datacenter
	if dc == "" {
		info, err := agent.Self()
		if err != nil {
			return s, fmt.Errorf("Failed to get datacenter from Consul agent: %v", err)
		}
		dc = info["Config"]["Datacenter"].(string)
	}

	// Update the attributes
	s.Attributes["datacenter"] = dc
	opts := consulapi.QueryOptions{Datacenter: dc}
	for name, key := range keys {
		pair, _, err := kv.Get(key.Key, &opts)
		if err != nil {
			return s, fmt.Errorf("Failed to get key '%s' from Consul: %v", key.Key, err)
		}
		if pair == nil && key.SetDefault {
			s.Attributes[name] = key.Default
		} else if pair == nil {
			s.Attributes[name] = ""
		} else {
			s.Attributes[name] = string(pair.Value)
		}
	}
	return s, nil
}

// partsFromConfig extracts the relevant configuration from the raw format
func partsFromConfig(raw map[string]interface{}) (string, consulKeys, error) {
	var dc string
	keys := make(map[string]*consulKey)
	for k, v := range raw {
		// datacenter is special and can be ignored
		if k == "datacenter" {
			vStr, ok := v.(string)
			if !ok {
				return "", nil, fmt.Errorf("datacenter must be a string")
			}
			dc = vStr
			continue
		}

		confs, ok := v.([]map[string]interface{})
		if !ok {
			return "", nil, fmt.Errorf("Field '%s' must be map containing a key", k)
		}
		if len(confs) > 1 {
			return "", nil, fmt.Errorf("Field '%s' has duplicate definitions", k)
		}
		conf := confs[0]

		key := &consulKey{}
		if err := mapstructure.WeakDecode(conf, key); err != nil {
			return "", nil, fmt.Errorf("Field '%s' failed to decode: %v", k, err)
		}
		for sub := range conf {
			switch sub {
			case "value":
				key.SetValue = true
			case "default":
				key.SetDefault = true
			}
		}
		keys[k] = key
	}
	return dc, keys, nil
}
