package azurerm

import (
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
)

func locationSchema() *schema.Schema {
	return &schema.Schema{
		Type:             schema.TypeString,
		Required:         true,
		ForceNew:         true,
		StateFunc:        azureRMNormalizeLocation,
		DiffSuppressFunc: azureRMSuppressLocationDiff,
	}
}

func deprecatedLocationSchema() *schema.Schema {
	return &schema.Schema{
		Type:             schema.TypeString,
		ForceNew:         true,
		Optional:         true,
		StateFunc:        azureRMNormalizeLocation,
		DiffSuppressFunc: azureRMSuppressLocationDiff,
		Deprecated:       "location is no longer used",
	}
}

// azureRMNormalizeLocation is a function which normalises human-readable region/location
// names (e.g. "West US") to the values used and returned by the Azure API (e.g. "westus").
// In state we track the API internal version as it is easier to go from the human form
// to the canonical form than the other way around.
func azureRMNormalizeLocation(location interface{}) string {
	input := location.(string)
	return strings.Replace(strings.ToLower(input), " ", "", -1)
}

func azureRMSuppressLocationDiff(k, old, new string, d *schema.ResourceData) bool {
	return azureRMNormalizeLocation(old) == azureRMNormalizeLocation(new)
}
