package nsone

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func metaSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeMap,
		Optional: true,
	}
}
