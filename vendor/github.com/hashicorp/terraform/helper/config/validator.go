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
		result = &nestedValidatorKey{
			Parts:    parts,
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
	Parts    []string
	Required bool
}

func (v *nestedValidatorKey) validate(
	m map[string]string,
	prefix string,
	offset int) ([]string, []string, []error) {
	if offset >= len(v.Parts) {
		// We're at the end. Look for a specific key.
		v2 := &basicValidatorKey{Key: prefix, Required: v.Required}
		return v2.Validate(m)
	}

	current := v.Parts[offset]

	// If we're at offset 0, special case to start at the next one.
	if offset == 0 {
		return v.validate(m, current, offset+1)
	}

	// Determine if we're doing a "for all" or a specific key
	if current != "*" {
		// We're looking at a specific key, continue on.
		return v.validate(m, prefix+"."+current, offset+1)
	}

	// We're doing a "for all", so we loop over.
	countStr, ok := m[prefix+".#"]
	if !ok {
		if !v.Required {
			// It wasn't required, so its no problem.
			return nil, nil, nil
		}

		return nil, nil, []error{fmt.Errorf(
			"Key not found: %s", prefix)}
	}

	count, err := strconv.ParseInt(countStr, 0, 0)
	if err != nil {
		// This shouldn't happen if flatmap works properly
		panic("invalid flatmap array")
	}

	var e []error
	var w []string
	u := make([]string, 1, count+1)
	u[0] = prefix + ".#"
	for i := 0; i < int(count); i++ {
		prefix := fmt.Sprintf("%s.%d", prefix, i)

		// Mark that we saw this specific key
		u = append(u, prefix)

		// Mark all prefixes of this
		for k, _ := range m {
			if !strings.HasPrefix(k, prefix+".") {
				continue
			}
			u = append(u, k)
		}

		// If we have more parts, then validate deeper
		if offset+1 < len(v.Parts) {
			u2, w2, e2 := v.validate(m, prefix, offset+1)

			u = append(u, u2...)
			w = append(w, w2...)
			e = append(e, e2...)
		}
	}

	return u, w, e
}

func (v *nestedValidatorKey) Validate(
	m map[string]string) ([]string, []string, []error) {
	return v.validate(m, "", 0)
}
