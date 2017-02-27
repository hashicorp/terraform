package schema

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/copystructure"
)

const TimeoutKey = "e2bfb730-ecaa-11e6-8f88-34363bc7c4c0"

const (
	resourceTimeoutCreateKey  = "create"
	resourceTimeoutReadKey    = "read"
	resourceTimeoutUpdateKey  = "update"
	resourceTimeoutDeleteKey  = "delete"
	resourceTimeoutDefaultKey = "default"
)

func timeKeys() []string {
	return []string{
		resourceTimeoutCreateKey,
		resourceTimeoutReadKey,
		resourceTimeoutUpdateKey,
		resourceTimeoutDeleteKey,
		resourceTimeoutDefaultKey,
	}
}

// could be time.Duration, int64 or float64
func DefaultTimeout(tx interface{}) *time.Duration {
	var td time.Duration
	switch raw := tx.(type) {
	case time.Duration:
		return &raw
	case int64:
		td = time.Duration(raw)
	case float64:
		td = time.Duration(int64(raw))
	default:
		log.Printf("[WARN] Unknown type in DefaultTimeout: %#v", tx)
	}
	return &td
}

type ResourceTimeout struct {
	Create, Read, Update, Delete, Default *time.Duration
}

// ConfigDecode takes a schema and the configuration (available in Diff) and
// validates, parses the timeouts into `t`
func (t *ResourceTimeout) ConfigDecode(s *Resource, c *terraform.ResourceConfig) error {
	if s.Timeouts != nil {
		raw, err := copystructure.Copy(s.Timeouts)
		if err != nil {
			log.Printf("[DEBUG] Error with deep copy: %s", err)
		}
		*t = *raw.(*ResourceTimeout)
	}

	if raw, ok := c.Config["timeout"]; ok {
		configTimeouts := raw.([]map[string]interface{})
		for _, timeoutValues := range configTimeouts {
			// loop through each Timeout given in the configuration and validate they
			// the Timeout defined in the resource
			for timeKey, timeValue := range timeoutValues {
				// validate that we're dealing with the normal CRUD actions
				var found bool
				for _, key := range timeKeys() {
					if timeKey == key {
						found = true
						break
					}
				}

				if !found {
					return fmt.Errorf("Unsupported Timeout configuration key found (%s)", timeKey)
				}

				// Get timeout
				rt, err := time.ParseDuration(timeValue.(string))
				if err != nil {
					return fmt.Errorf("Error parsing Timeout for (%s): %s", timeKey, err)
				}

				var timeout *time.Duration
				switch timeKey {
				case resourceTimeoutCreateKey:
					timeout = t.Create
				case resourceTimeoutUpdateKey:
					timeout = t.Update
				case resourceTimeoutReadKey:
					timeout = t.Read
				case resourceTimeoutDeleteKey:
					timeout = t.Delete
				case resourceTimeoutDefaultKey:
					timeout = t.Default
				}

				// If the resource has not delcared this in the definition, then error
				// with an unsupported message
				if timeout == nil {
					return unsupportedTimeoutKeyError(timeKey)
				}

				*timeout = rt
			}
		}
	}

	return nil
}

func unsupportedTimeoutKeyError(key string) error {
	return fmt.Errorf("Timeout Key (%s) is not supported", key)
}

// DiffEncode, StateEncode, and MetaDecode are analogous to the Go stdlib JSONEncoder
// interface: they encode/decode a timeouts struct from an instance diff, which is
// where the timeout data is stored after a diff to pass into Apply.
//
// StateEncode encodes the timeout into the ResourceData's InstanceState for
// saving to state
//
// TODO: when should this error?
// func (t *ResourceTimeout) DiffEncode(id *terraform.InstanceDiff) error {
func (t *ResourceTimeout) DiffEncode(id *terraform.InstanceDiff) error {
	return t.metaEncode(id)
}

func (t *ResourceTimeout) StateEncode(is *terraform.InstanceState) error {
	return t.metaEncode(is)
}

// metaEncode encodes the ResourceTimeout into a map[string]interface{} format
// and stores it in the Meta field of the interface it's given.
// Assumes the interface is either *terraform.InstanceState or
// *terraform.InstanceDiff, returns an error otherwise
func (t *ResourceTimeout) metaEncode(ids interface{}) error {
	m := make(map[string]interface{})

	if t.Create != nil {
		m[resourceTimeoutCreateKey] = t.Create.Nanoseconds()
	}
	if t.Read != nil {
		m[resourceTimeoutReadKey] = t.Read.Nanoseconds()
	}
	if t.Update != nil {
		m[resourceTimeoutUpdateKey] = t.Update.Nanoseconds()
	}
	if t.Delete != nil {
		m[resourceTimeoutDeleteKey] = t.Delete.Nanoseconds()
	}
	if t.Default != nil {
		m[resourceTimeoutDefaultKey] = t.Default.Nanoseconds()
		// for any key above that is nil, if default is specified, we need to
		// populate it with the default
		for _, k := range timeKeys() {
			if _, ok := m[k]; !ok {
				m[k] = t.Default.Nanoseconds()
			}
		}
	}

	// only add the Timeout to the Meta if we have values
	if len(m) > 0 {
		switch instance := ids.(type) {
		case *terraform.InstanceDiff:
			if instance.Meta == nil {
				instance.Meta = make(map[string]interface{})
			}
			instance.Meta[TimeoutKey] = m
		case *terraform.InstanceState:
			if instance.Meta == nil {
				instance.Meta = make(map[string]interface{})
			}
			instance.Meta[TimeoutKey] = m
		default:
			return fmt.Errorf("Error matching type for Diff Encode")
		}
	}

	return nil
}

func (t *ResourceTimeout) StateDecode(id *terraform.InstanceState) error {
	return t.metaDecode(id)
}
func (t *ResourceTimeout) DiffDecode(is *terraform.InstanceDiff) error {
	return t.metaDecode(is)
}

func (t *ResourceTimeout) metaDecode(ids interface{}) error {
	var rawMeta interface{}
	var ok bool
	switch rawInstance := ids.(type) {
	case *terraform.InstanceDiff:
		rawMeta, ok = rawInstance.Meta[TimeoutKey]
		if !ok {
			return nil
		}
	case *terraform.InstanceState:
		rawMeta, ok = rawInstance.Meta[TimeoutKey]
		if !ok {
			return nil
		}
	default:
		return fmt.Errorf("Unknown or unsupported type in metaDecode: %#v", ids)
	}

	times := rawMeta.(map[string]interface{})
	if len(times) == 0 {
		return nil
	}

	if v, ok := times[resourceTimeoutCreateKey]; ok {
		t.Create = DefaultTimeout(v)
	}
	if v, ok := times[resourceTimeoutReadKey]; ok {
		t.Read = DefaultTimeout(v)
	}
	if v, ok := times[resourceTimeoutUpdateKey]; ok {
		t.Update = DefaultTimeout(v)
	}
	if v, ok := times[resourceTimeoutDeleteKey]; ok {
		t.Delete = DefaultTimeout(v)
	}
	if v, ok := times[resourceTimeoutDefaultKey]; ok {
		t.Default = DefaultTimeout(v)
	}

	return nil
}
