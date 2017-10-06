package akamai

import (
	"github.com/hashicorp/terraform/helper/schema"
)

var akps_option *schema.Schema = &schema.Schema{
	Type:     schema.TypeSet,
	Optional: true,
	Elem: &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"values": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},
			"value": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"flag": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"type": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "auto",
			},
		},
	},
}

var akps_criteria *schema.Schema = &schema.Schema{
	Type:     schema.TypeSet,
	Optional: true,
	Elem: &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"option": akps_option,
		},
	},
}

var akps_behavior *schema.Schema = &schema.Schema{
	Type:     schema.TypeSet,
	Optional: true,
	Elem: &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"option": akps_option,
		},
	},
}

var akamaiPropertySchema map[string]*schema.Schema = map[string]*schema.Schema{
	// Cloning is unsupported
	// "clone_from": &schema.Schema{},
	"account_id": &schema.Schema{
		Type:     schema.TypeString,
		Required: true,
	},
	"contract_id": &schema.Schema{
		Type:     schema.TypeString,
		Required: true,
	},
	"group_id": &schema.Schema{
		Type:     schema.TypeString,
		Required: true,
	},
	"product_id": &schema.Schema{
		Type:     schema.TypeString,
		Required: true,
	},
	"cp_code": &schema.Schema{
		Type:     schema.TypeString,
		Required: true,
	},
	"property_id": &schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
	},
	"name": &schema.Schema{
		Type:     schema.TypeString,
		Required: true,
	},
	"rule_format": &schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
	},
	"ipv6": &schema.Schema{
		Type:     schema.TypeBool,
		Optional: true,
	},
	"hostname": &schema.Schema{
		Type:     schema.TypeSet,
		Required: true,
		Elem:     &schema.Schema{Type: schema.TypeString},
	},
	"contact": &schema.Schema{
		Type:     schema.TypeSet,
		Required: true,
		Elem:     &schema.Schema{Type: schema.TypeString},
	},
	"edge_hostname": &schema.Schema{
		Type:     schema.TypeMap,
		Computed: true,
		Elem:     &schema.Schema{Type: schema.TypeString},
	},

	"origin": {
		Type:     schema.TypeList,
		Required: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"is_secure": {
					Type:     schema.TypeString,
					Required: true,
				},
				"hostname": {
					Type:     schema.TypeString,
					Required: true,
				},
				"port": {
					Type:     schema.TypeInt,
					Optional: true,
					Default:  80,
				},
				"forward_hostname": {
					Type:     schema.TypeString,
					Optional: true,
					Default:  "ORIGIN_HOSTNAME",
				},
				"cache_key_hostname": {
					Type:     schema.TypeString,
					Optional: true,
					Default:  "ORIGIN_HOSTNAME",
				},
				"gzip_compression": {
					Type:     schema.TypeBool,
					Optional: true,
					Default:  false,
				},
				"true_client_ip_header": {
					Type:     schema.TypeBool,
					Optional: true,
					Default:  false,
				},
			},
		},
	},

	// rules tree can go max 5 levels deep
	"rule": &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"name": {
					Type:     schema.TypeString,
					Required: true,
				},
				"comment": {
					Type:     schema.TypeString,
					Optional: true,
				},
				"criteria_match": {
					Type:     schema.TypeString,
					Optional: true,
					Default:  "all",
				},
				"criteria": akps_criteria,
				"behavior": akps_behavior,
				// "children": [], //TODO
			},
		},
	},

	"compress": {
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"extensions": {
					Type:     schema.TypeSet,
					Elem:     &schema.Schema{Type: schema.TypeString},
					Optional: true,
				},
				"content_types": {
					Type:     schema.TypeSet,
					Elem:     &schema.Schema{Type: schema.TypeString},
					Optional: true,
				},
				"criteria": akps_criteria,
			},
		},
	},
	"cache": {
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"match": {
					Type:     schema.TypeSet,
					Optional: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"extensions": {
								Type:     schema.TypeSet,
								Elem:     &schema.Schema{Type: schema.TypeString},
								Optional: true,
							},
							"paths": {
								Type:     schema.TypeSet,
								Elem:     &schema.Schema{Type: schema.TypeString},
								Optional: true,
							},
						},
					},
				},
				"max_age": {
					Type:     schema.TypeString,
					Optional: true,
				},
				"prefreshing": {
					Type:     schema.TypeBool,
					Optional: true,
				},
				"prefetch": {
					Type:     schema.TypeBool,
					Optional: true,
				},
				"query_params": {
					Type:     schema.TypeBool,
					Optional: true,
				},
				"query_params_sort": {
					Type:     schema.TypeBool,
					Optional: true,
				},
				"cache": {
					Type:     schema.TypeBool,
					Optional: true,
				},
				"criteria": akps_criteria,
			},
		},
	},
}
