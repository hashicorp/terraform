package diff

import (
	"strings"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/flatmap"
	"github.com/hashicorp/terraform/terraform"
)

// AttrType is an enum that tells the ResourceBuilder what type of attribute
// an attribute is, affecting the overall diff output.
//
// The valid values are:
//
//   * AttrTypeCreate - This attribute can only be set or updated on create.
//       This means that if this attribute is changed, it will require a new
//       resource to be created if it is already created.
//
//   * AttrTypeUpdate - This attribute can be set at create time or updated
//       in-place. Changing this attribute does not require a new resource.
//
type AttrType byte

const (
	AttrTypeUnknown AttrType = iota
	AttrTypeCreate
	AttrTypeUpdate
)

// ResourceBuilder is a helper that knows about how a single resource
// changes and how those changes affect the diff.
type ResourceBuilder struct {
	// Attrs are the mapping of attributes that can be set from the
	// configuration, and the affect they have. See the documentation for
	// AttrType for more info.
	//
	// Sometimes attributes in here are also computed. For example, an
	// "availability_zone" might be optional, but will be chosen for you
	// by AWS. In that case, specify it both here and in ComputedAttrs.
	// This will make sure that the absence of the configuration won't
	// cause a diff by setting it to the empty string.
	Attrs map[string]AttrType

	// ComputedAttrs are the attributes that are computed at
	// resource creation time.
	ComputedAttrs []string

	// ComputedAttrsUpdate are the attributes that are computed
	// at resource update time (this includes creation).
	ComputedAttrsUpdate []string

	// PreProcess is a mapping of exact keys that are sent through
	// a pre-processor before comparing values. The original value will
	// be put in the "NewExtra" field of the diff.
	PreProcess map[string]PreProcessFunc
}

// PreProcessFunc is used with the PreProcess field in a ResourceBuilder
type PreProcessFunc func(string) string

// Diff returns the ResourceDiff for a resource given its state and
// configuration.
func (b *ResourceBuilder) Diff(
	s *terraform.InstanceState,
	c *terraform.ResourceConfig) (*terraform.InstanceDiff, error) {
	attrs := make(map[string]*terraform.ResourceAttrDiff)

	// We require a new resource if the ID is empty. Or, later, we set
	// this to true if any configuration changed that triggers a new resource.
	requiresNew := s.ID == ""

	// Flatten the raw and processed configuration
	flatRaw := flatmap.Flatten(c.Raw)
	flatConfig := flatmap.Flatten(c.Config)

	for ak, at := range b.Attrs {
		// Keep track of all the keys we saw in the raw structure
		// so that we can prune our attributes later.
		seenKeys := make([]string, 0)

		// Go through and find the added/changed keys in flatRaw
		for k, v := range flatRaw {
			// Find only the attributes that match our prefix
			if !strings.HasPrefix(k, ak) {
				continue
			}

			// Track that we saw this key
			seenKeys = append(seenKeys, k)

			// We keep track of this in case we have a pre-processor
			// so that we can store the original value still.
			originalV := v

			// If this key is in the cleaned config, then use that value
			// because it'll have its variables properly interpolated
			if cleanV, ok := flatConfig[k]; ok && cleanV != config.UnknownVariableValue {
				v = cleanV
				originalV = v

				// If we have a pre-processor for this, run it.
				if pp, ok := b.PreProcess[k]; ok {
					v = pp(v)
				}
			}

			oldV, ok := s.Attributes[k]

			// If there is an old value and they're the same, no change
			if ok && oldV == v {
				continue
			}

			// Record the change
			attrs[k] = &terraform.ResourceAttrDiff{
				Old:      oldV,
				New:      v,
				NewExtra: originalV,
				Type:     terraform.DiffAttrInput,
			}

			// If this requires a new resource, record that and flag our
			// boolean.
			if at == AttrTypeCreate {
				attrs[k].RequiresNew = true
				requiresNew = true
			}
		}

		// Find all the keys that are in our attributes right now that
		// we also care about.
		matchingKeys := make(map[string]struct{})
		for k, _ := range s.Attributes {
			// Find only the attributes that match our prefix
			if !strings.HasPrefix(k, ak) {
				continue
			}

			// If this key is computed, then we don't ever delete it
			comp := false
			for _, ck := range b.ComputedAttrs {
				if ck == k {
					comp = true
					break
				}

				// If the key is prefixed with the computed key, don't
				// mark it for delete, ever.
				if strings.HasPrefix(k, ck+".") {
					comp = true
					break
				}
			}
			if comp {
				continue
			}

			matchingKeys[k] = struct{}{}
		}

		// Delete the keys we saw in the configuration from the keys
		// that are currently set.
		for _, k := range seenKeys {
			delete(matchingKeys, k)
		}
		for k, _ := range matchingKeys {
			attrs[k] = &terraform.ResourceAttrDiff{
				Old:        s.Attributes[k],
				NewRemoved: true,
				Type:       terraform.DiffAttrInput,
			}
		}
	}

	// If we require a new resource, then process all the attributes
	// that will be changing due to the creation of the resource.
	if requiresNew {
		for _, k := range b.ComputedAttrs {
			if _, ok := attrs[k]; ok {
				continue
			}

			old := s.Attributes[k]
			attrs[k] = &terraform.ResourceAttrDiff{
				Old:         old,
				NewComputed: true,
				Type:        terraform.DiffAttrOutput,
			}
		}
	}

	// If we're changing anything, then mark the updated
	// attributes.
	if len(attrs) > 0 {
		for _, k := range b.ComputedAttrsUpdate {
			if _, ok := attrs[k]; ok {
				continue
			}

			old := s.Attributes[k]
			attrs[k] = &terraform.ResourceAttrDiff{
				Old:         old,
				NewComputed: true,
				Type:        terraform.DiffAttrOutput,
			}
		}
	}

	// Build our resulting diff if we had attributes change
	var result *terraform.InstanceDiff
	if len(attrs) > 0 {
		result = &terraform.InstanceDiff{
			Attributes: attrs,
		}
	}

	return result, nil
}
