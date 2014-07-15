package config

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/flatmap"
	"github.com/hashicorp/terraform/terraform"
)

// Validator is a helper that helps you validate the configuration
// of your resource, resource provider, etc.
//
// At the most basic level, set the Required and Optional lists to be
// specifiers of keys that are required or optional. If a key shows up
// that isn't in one of these two lists, then an error is generated.
//
// The "specifiers" allowed in this is a fairly rich syntax to help
// describe the format of your configuration:
//
//   * Basic keys are just strings. For example: "foo" will match the
//       "foo" key.
//
//   * Nested structure keys can be matched by doing
//       "listener.*.foo". This will verify that there is at least one
//       listener element that has the "foo" key set.
//
//   * The existence of a nested structure can be checked by simply
//       doing "listener.*" which will verify that there is at least
//       one element in the "listener" structure. This is NOT
//       validating that "listener" is an array. It is validating
//       that it is a nested structure in the configuration.
//
type Validator struct {
	Required []string
	Optional []string
}

func (v *Validator) Validate(
	c *terraform.ResourceConfig) (ws []string, es []error) {
	// Flatten the configuration so it is easier to reason about
	flat := flatmap.Flatten(c.Raw)

	keySet := make(map[string]validatorKey)
	for i, vs := range [][]string{v.Required, v.Optional} {
		req := i == 0
		for _, k := range vs {
			vk, err := newValidatorKey(k, req)
			if err != nil {
				es = append(es, err)
				continue
			}

			keySet[k] = vk
		}
	}

	purged := make([]string, 0)
	for _, kv := range keySet {
		p, w, e := kv.Validate(flat)
		if len(w) > 0 {
			ws = append(ws, w...)
		}
		if len(e) > 0 {
			es = append(es, e...)
		}

		purged = append(purged, p...)
	}

	// Delete all the keys we processed in order to find
	// the unknown keys.
	for _, p := range purged {
		delete(flat, p)
	}

	// The rest are unknown
	for k, _ := range flat {
		es = append(es, fmt.Errorf("Unknown configuration: %s", k))
	}

	return
}

type validatorKey interface {
	// Validate validates the given configuration and returns viewed keys,
	// warnings, and errors.
	Validate(map[string]string) ([]string, []string, []error)
}

func newValidatorKey(k string, req bool) (validatorKey, error) {
	var result validatorKey

	parts := strings.Split(k, ".")
	if len(parts) > 1 && parts[1] == "*" {
		key := ""
		if len(parts) >= 3 {
			key = parts[2]
		}

		result = &nestedValidatorKey{
			Prefix:   parts[0],
			Key:      key,
			Required: req,
		}
	} else {
		result = &basicValidatorKey{
			Key:      k,
			Required: req,
		}
	}

	return result, nil
}

// basicValidatorKey validates keys that are basic such as "foo"
type basicValidatorKey struct {
	Key      string
	Required bool
}

func (v *basicValidatorKey) Validate(
	m map[string]string) ([]string, []string, []error) {
	for k, _ := range m {
		// If we have the exact key its a match
		if k == v.Key {
			return []string{k}, nil, nil
		}
	}

	if !v.Required {
		return nil, nil, nil
	}

	return nil, nil, []error{fmt.Errorf(
		"Key not found: %s", v.Key)}
}

type nestedValidatorKey struct {
	Prefix   string
	Key      string
	Required bool
}

func (v *nestedValidatorKey) Validate(
	m map[string]string) ([]string, []string, []error) {
	countStr, ok := m[v.Prefix+".#"]
	if !ok {
		if !v.Required || v.Key != "" {
			// Not present, that is okay
			return nil, nil, nil
		} else {
			// Required and isn't present
			return nil, nil, []error{fmt.Errorf(
				"Key not found: %s", v.Prefix)}
		}
	}

	count, err := strconv.ParseInt(countStr, 0, 0)
	if err != nil {
		// This shouldn't happen if flatmap works properly
		panic("invalid flatmap array")
	}

	var errs []error
	used := make([]string, 1, count+1)
	used[0] = v.Prefix + ".#"
	for i := 0; i < int(count); i++ {
		prefix := fmt.Sprintf("%s.%d.", v.Prefix, i)

		if v.Key != "" {
			key := prefix + v.Key
			if _, ok := m[key]; !ok {
				errs = append(errs, fmt.Errorf(
					"%s[%d]: does not contain required key %s",
					v.Prefix,
					i,
					v.Key))
			}
		}

		for k, _ := range m {
			if k != prefix[:len(prefix)-1] {
				if !strings.HasPrefix(k, prefix) {
					continue
				}
			}

			used = append(used, k)
		}
	}

	return used, nil, errs
}
