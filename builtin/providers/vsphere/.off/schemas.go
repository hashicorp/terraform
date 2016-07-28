package dvs

import "github.com/hashicorp/terraform/helper/schema"

/* functions for MapHostDVS */

/* MapHostDVS functions */
func resourceVSphereMapHostDVSSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"host": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"switch": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"nic_names": &schema.Schema{
			Type:     schema.TypeSet,
			Optional: true,
			Computed: true,
			ForceNew: true,
			Elem:     &schema.Schema{Type: schema.TypeString},
			Set:      schema.HashString,
		},
	}
}

/* Functions for MapVMDVPG */

func resourceVSphereMapVMDVPGSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"vm": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"nic_label": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"portgroup": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
	}
}
