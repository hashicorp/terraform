package diff

import (
	"strings"

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
	Attrs map[string]AttrType

	// ComputedAttrs are the attributes that are computed at
	// resource creation time.
	ComputedAttrs []string
}

// Diff returns the ResourceDiff for a resource given its state and
// configuration.
func (b *ResourceBuilder) Diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig) (*terraform.ResourceDiff, error) {
	attrs := make(map[string]*terraform.ResourceAttrDiff)

	// We require a new resource if the ID is empty. Or, later, we set
	// this to true if any configuration changed that triggers a new resource.
	requiresNew := s.ID == ""

	// Flatten the raw and processed configuration
	flatRaw := flatmap.Flatten(c.Raw)
	flatConfig := flatmap.Flatten(c.Config)

	for k, v := range flatRaw {
		// Make sure this is an attribute that actually affects
		// the diff in some way.
		var attr AttrType
		for ak, at := range b.Attrs {
			if strings.HasPrefix(k, ak) {
				attr = at
				break
			}
		}
		if attr == AttrTypeUnknown {
			continue
		}

		// If this key is in the cleaned config, then use that value
		// because it'll have its variables properly interpolated
		if cleanV, ok := flatConfig[k]; ok {
			v = cleanV
		}

		oldV, ok := s.Attributes[k]

		// If there is an old value and they're the same, no change
		if ok && oldV == v {
			continue
		}

		// Record the change
		attrs[k] = &terraform.ResourceAttrDiff{
			Old:  oldV,
			New:  v,
			Type: terraform.DiffAttrInput,
		}

		// If this requires a new resource, record that and flag our
		// boolean.
		if attr == AttrTypeCreate {
			attrs[k].RequiresNew = true
			requiresNew = true
		}
	}

	// If we require a new resource, then process all the attributes
	// that will be changing due to the creation of the resource.
	if requiresNew {
		for _, k := range b.ComputedAttrs {
			old := s.Attributes[k]
			attrs[k] = &terraform.ResourceAttrDiff{
				Old:         old,
				NewComputed: true,
				Type:        terraform.DiffAttrOutput,
			}
		}

		// The ID will change
		attrs["id"] = &terraform.ResourceAttrDiff{
			Old:         s.ID,
			NewComputed: true,
			Type:        terraform.DiffAttrOutput,
		}
	}

	// Build our resulting diff if we had attributes change
	var result *terraform.ResourceDiff
	if len(attrs) > 0 {
		result = &terraform.ResourceDiff{
			Attributes: attrs,
		}
	}

	return result, nil
}
