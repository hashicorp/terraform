package diff

import (
	"github.com/hashicorp/terraform/terraform"
)

// ResourceBuilder is a helper that knows about how a single resource
// changes and how those changes affect the diff.
type ResourceBuilder struct {
	CreateComputedAttrs []string
	RequiresNewAttrs    []string
}

// Diff returns the ResourceDiff for a resource given its state and
// configuration.
func (b *ResourceBuilder) Diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig) (*terraform.ResourceDiff, error) {
	attrs := make(map[string]*terraform.ResourceAttrDiff)

	requiresNewSet := make(map[string]struct{})
	for _, k := range b.RequiresNewAttrs {
		requiresNewSet[k] = struct{}{}
	}

	// We require a new resource if the ID is empty. Or, later, we set
	// this to true if any configuration changed that triggers a new resource.
	requiresNew := s.ID == ""

	// Go through the configuration and find the changed attributes
	for k, v := range c.Raw {
		newV := v.(string)

		// If this key is in the cleaned config, then use that value
		// because it'll have its variables properly interpolated
		if cleanV, ok := c.Config[k]; ok {
			newV = cleanV.(string)
		}

		var oldV string
		var ok bool
		if oldV, ok = s.Attributes[k]; ok {
			// Old value exists! We check to see if there is a change
			if oldV == newV {
				continue
			}
		}

		// There has been a change. Record it
		attrs[k] = &terraform.ResourceAttrDiff{
			Old: oldV,
			New: newV,
		}

		// If this requires a new resource, record that and flag our
		// boolean.
		if _, ok := requiresNewSet[k]; ok {
			attrs[k].RequiresNew = true
			attrs[k].Type = terraform.DiffAttrInput
			requiresNew = true
		}
	}

	// If we require a new resource, then process all the attributes
	// that will be changing due to the creation of the resource.
	if requiresNew {
		for _, k := range b.CreateComputedAttrs {
			old := s.Attributes[k]
			attrs[k] = &terraform.ResourceAttrDiff{
				Old:         old,
				NewComputed: true,
				Type:        terraform.DiffAttrOutput,
			}
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
