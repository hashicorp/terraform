package dvs

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

func _setDVPGPolicy(v interface{}) int {
	asmap := v.(map[string]interface{})
	components := []string{"allow_block_override", "allow_live_port_moving", "allow_network_resources_pool_override", "port_config_reset_disconnect", "allow_shaping_override", "allow_traffic_filter_override", "allow_vendor_config_override"}
	h := ""
	for _, i := range components {
		k, ok := asmap[i]
		if !ok {
			h += "unset"
			continue
		}
		h += fmt.Sprintf("%v-", k)
	}
	return schema.HashString(h)
}

func resourceVSphereDVSSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"name": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
			// ForceNew:		true,
		},
		"folder": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
			// ForceNew:		true,
		},
		"datacenter": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
			// ForceNew:		true,
		},
		"extension_key": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
		},
		"description": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
		},
		"contact": &schema.Schema{
			Type:     schema.TypeMap,
			Optional: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"name": &schema.Schema{
						Type:     schema.TypeString,
						Required: true,
					},
					"infos": &schema.Schema{
						Type:     schema.TypeString,
						Required: true,
					},
				},
			},
		},
		"switch_usage_policy": &schema.Schema{
			Type:     schema.TypeMap,
			Optional: true,
			Default: map[string]bool{
				"auto_preinstall_allowed": false,
				"auto_upgrade_allowed":    false,
				"partial_upgrade_allowed": false,
			},
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"auto_preinstall_allowed": &schema.Schema{
						Type:     schema.TypeBool,
						Optional: true,
						Default:  false,
					},
					"auto_upgrade_allowed": &schema.Schema{
						Type:     schema.TypeBool,
						Optional: true,
						Default:  false,
					},
					"partial_upgrade_allowed": &schema.Schema{
						Type:     schema.TypeBool,
						Optional: true,
						Default:  false,
					},
				},
			},
		},
		"switch_ip_address": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
		},
		"num_standalone_ports": &schema.Schema{
			Type:     schema.TypeInt,
			Optional: true,
		},
		"full_path": &schema.Schema{
			Type:     schema.TypeString,
			Computed: true,
		},
	}
}

/* functions for DistributedVirtualPortgroup */
func resourceVSphereDVPGSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"name": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"switch_id": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"default_vlan": &schema.Schema{
			Type:     schema.TypeInt,
			Optional: true,
		},
		"vlan_range": &schema.Schema{
			Type:     schema.TypeSet,
			Optional: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"start": &schema.Schema{
						Type:     schema.TypeInt,
						Required: true,
					},
					"end": &schema.Schema{
						Type:     schema.TypeInt,
						Required: true,
					},
				},
			},
		},
		"datacenter": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"type": &schema.Schema{
			Type:        schema.TypeString,
			Required:    true,
			Description: "earlyBinding|ephemeral",
		},
		"description": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
		},
		"auto_expand": &schema.Schema{
			Type:     schema.TypeBool,
			Optional: true,
		},
		"num_ports": &schema.Schema{
			Type:     schema.TypeInt,
			Optional: true,
		},
		"port_name_format": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
		},
		"policy": &schema.Schema{
			Type:     schema.TypeSet,
			Computed: true,
			Optional: true,
			MaxItems: 1,
			Set:      _setDVPGPolicy,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"allow_block_override": &schema.Schema{
						Type:     schema.TypeBool,
						Optional: true,
						Default:  false,
					},
					"allow_live_port_moving": &schema.Schema{
						Type:     schema.TypeBool,
						Optional: true,
						Default:  false,
					},
					"allow_network_resources_pool_override": &schema.Schema{
						Type:     schema.TypeBool,
						Optional: true,
						Default:  false,
					},
					"port_config_reset_disconnect": &schema.Schema{
						Type:     schema.TypeBool,
						Optional: true,
						Default:  true,
					},
					"allow_shaping_override": &schema.Schema{
						Type:     schema.TypeBool,
						Optional: true,
						Default:  false,
					},
					"allow_traffic_filter_override": &schema.Schema{
						Type:     schema.TypeBool,
						Optional: true,
						Default:  false,
					},
					"allow_vendor_config_override": &schema.Schema{
						Type:     schema.TypeBool,
						Optional: true,
						Default:  false,
					},
				},
			},
		},
		"full_path": &schema.Schema{
			Type:     schema.TypeString,
			Computed: true,
		},
	}
}

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
