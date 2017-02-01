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
	rtCreate  = "create"
	rtRead    = "read"
	rtUpdate  = "update"
	rtDelete  = "delete"
	rtDefault = "default"
)

func timeKeys() []string {
	return []string{"create", "read", "update", "delete", "default"}
}

func DefaultTimeout(tx time.Duration) *time.Duration {
	return &tx
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

	if v, ok := c.Config["timeout"]; ok {
		raw := v.([]map[string]interface{})
		for _, tv := range raw {
			for mk, mv := range tv {
				var found bool
				for _, key := range timeKeys() {
					if mk == key {
						found = true
						break
					}
				}

				if !found {
					return fmt.Errorf("Unsupported timeout key found (%s)", mk)
				}

				//TODO-cts: refactor this to remove duplication, using something like a swich
				//on the type
				if mk == "create" {
					if t.Create == nil {
						return fmt.Errorf("Timeout (%s) is not supported", mk)
					} else {
						rt, err := time.ParseDuration(mv.(string))
						if err != nil {
							return fmt.Errorf("Error parsing Timeout for (%s): %s", mk, err)
						}
						t.Create = &rt
						continue
					}
				}

				if mk == "read" {
					if t.Read == nil {
						return fmt.Errorf("Timeout (%s) is not supported", mk)
					} else {
						rt, err := time.ParseDuration(mv.(string))
						if err != nil {
							return fmt.Errorf("Error parsing Timeout for (%s): %s", mk, err)
						}
						t.Read = &rt
						continue
					}
				}

				if mk == "update" {
					if t.Update == nil {
						return fmt.Errorf("Timeout (%s) is not supported", mk)
					} else {
						rt, err := time.ParseDuration(mv.(string))
						if err != nil {
							return fmt.Errorf("Error parsing Timeout for (%s): %s", mk, err)
						}
						t.Update = &rt
						continue
					}
				}

				if mk == "delete" {
					if t.Delete == nil {
						return fmt.Errorf("Timeout (%s) is not supported", mk)
					} else {
						rt, err := time.ParseDuration(mv.(string))
						if err != nil {
							return fmt.Errorf("Error parsing Timeout for (%s): %s", mk, err)
						}
						t.Delete = &rt
						continue
					}
				}

				if mk == "default" {
					if t.Default == nil {
						return fmt.Errorf("Timeout (%s) is not supported", mk)
					} else {
						rt, err := time.ParseDuration(mv.(string))
						if err != nil {
							return fmt.Errorf("Error parsing Timeout for (%s): %s", mk, err)
						}
						t.Default = &rt
						continue
					}
				}
			}
		}
	}

	return nil
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
		m["create"] = t.Create.Nanoseconds()
	}
	if t.Read != nil {
		m["read"] = t.Read.Nanoseconds()
	}
	if t.Update != nil {
		m["update"] = t.Update.Nanoseconds()
	}
	if t.Delete != nil {
		m["delete"] = t.Delete.Nanoseconds()
	}
	if t.Default != nil {
		m["default"] = t.Default.Nanoseconds()
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
			return fmt.Errorf("[ERR] Error matching type for Diff Encode")
		}

	}

	return nil
}

func (t *ResourceTimeout) MetaDecode(id *terraform.InstanceDiff) error {
	//TODO-cts - I don't think this is needed
	if len(id.Meta) == 0 {
		return nil
	}

	tv, ok := id.Meta[TimeoutKey]
	if !ok {
		return nil
	}

	times := tv.(map[string]interface{})

	if v, ok := times[rtCreate]; ok {
		t.Create = DefaultTimeout(time.Duration(v.(int64)))
	}
	if v, ok := times[rtRead]; ok {
		t.Read = DefaultTimeout(time.Duration(v.(int64)))
	}
	if v, ok := times[rtUpdate]; ok {
		t.Update = DefaultTimeout(time.Duration(v.(int64)))
	}
	if v, ok := times[rtDelete]; ok {
		t.Delete = DefaultTimeout(time.Duration(v.(int64)))
	}
	if v, ok := times[rtDefault]; ok {
		t.Default = DefaultTimeout(time.Duration(v.(int64)))
	}

	return nil
}
