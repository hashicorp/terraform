package schema

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/config/hcl2shim"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/copystructure"
)

const TimeoutKey = "e2bfb730-ecaa-11e6-8f88-34363bc7c4c0"
const TimeoutsConfigKey = "timeouts"

const (
	TimeoutCreate  = "create"
	TimeoutRead    = "read"
	TimeoutUpdate  = "update"
	TimeoutDelete  = "delete"
	TimeoutDefault = "default"
)

func timeoutKeys() []string {
	return []string{
		TimeoutCreate,
		TimeoutRead,
		TimeoutUpdate,
		TimeoutDelete,
		TimeoutDefault,
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

	if raw, ok := c.Config[TimeoutsConfigKey]; ok {
		var rawTimeouts []map[string]interface{}
		switch raw := raw.(type) {
		case map[string]interface{}:
			rawTimeouts = append(rawTimeouts, raw)
		case []map[string]interface{}:
			rawTimeouts = raw
		case string:
			if raw == hcl2shim.UnknownVariableValue {
				// Timeout is not defined in the config
				// Defaults will be used instead
				return nil
			} else {
				log.Printf("[ERROR] Invalid timeout value: %q", raw)
				return fmt.Errorf("Invalid Timeout value found")
			}
		default:
			log.Printf("[ERROR] Invalid timeout structure: %#v", raw)
			return fmt.Errorf("Invalid Timeout structure found")
		}

		for _, timeoutValues := range rawTimeouts {
			for timeKey, timeValue := range timeoutValues {
				// validate that we're dealing with the normal CRUD actions
				var found bool
				for _, key := range timeoutKeys() {
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
					return fmt.Errorf("Error parsing %q timeout: %s", timeKey, err)
				}

				var timeout *time.Duration
				switch timeKey {
				case TimeoutCreate:
					timeout = t.Create
				case TimeoutUpdate:
					timeout = t.Update
				case TimeoutRead:
					timeout = t.Read
				case TimeoutDelete:
					timeout = t.Delete
				case TimeoutDefault:
					timeout = t.Default
				}

				// If the resource has not delcared this in the definition, then error
				// with an unsupported message
				if timeout == nil {
					return unsupportedTimeoutKeyError(timeKey)
				}

				*timeout = rt
			}
			return nil
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
		m[TimeoutCreate] = t.Create.Nanoseconds()
	}
	if t.Read != nil {
		m[TimeoutRead] = t.Read.Nanoseconds()
	}
	if t.Update != nil {
		m[TimeoutUpdate] = t.Update.Nanoseconds()
	}
	if t.Delete != nil {
		m[TimeoutDelete] = t.Delete.Nanoseconds()
	}
	if t.Default != nil {
		m[TimeoutDefault] = t.Default.Nanoseconds()
		// for any key above that is nil, if default is specified, we need to
		// populate it with the default
		for _, k := range timeoutKeys() {
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

	if v, ok := times[TimeoutCreate]; ok {
		t.Create = DefaultTimeout(v)
	}
	if v, ok := times[TimeoutRead]; ok {
		t.Read = DefaultTimeout(v)
	}
	if v, ok := times[TimeoutUpdate]; ok {
		t.Update = DefaultTimeout(v)
	}
	if v, ok := times[TimeoutDelete]; ok {
		t.Delete = DefaultTimeout(v)
	}
	if v, ok := times[TimeoutDefault]; ok {
		t.Default = DefaultTimeout(v)
	}

	return nil
}
