package akamai

import (
	"github.com/hashicorp/terraform/helper/schema"
)

var akps_option *schema.Schema = &schema.Schema{
	Type:     schema.TypeSet,
	Optional: true,
	Elem: &schema.Resource{
		Schema: map[string]*schema.Schema{
			"key": {
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
	"account_id": &schema.Schema{
		Type:     schema.TypeString,
		Required: true,
	},
	"contract_id": &schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
	},
	"group_id": &schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
	},
	"product_id": &schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
	},

	"network": &schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
		Default:  "staging",
	},

	"activate": &schema.Schema{
		Type:     schema.TypeBool,
		Optional: true,
		Default:  true,
	},

	// Will get added to the default rule
	"cp_code": &schema.Schema{
		Type:     schema.TypeString,
		Required: true,
	},
	"name": &schema.Schema{
		Type:     schema.TypeString,
		Required: true,
	},
	"version": &schema.Schema{
		Type:     schema.TypeInt,
		Computed: true,
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
		Optional: true,
		Elem:     &schema.Schema{Type: schema.TypeString},
	},

	"clone_from": &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"property_id": {
					Type:     schema.TypeString,
					Required: true,
				},
				"version": {
					Type:     schema.TypeInt,
					Optional: true,
				},
				"etag": {
					Type:     schema.TypeString,
					Optional: true,
				},
				"copy_hostnames": {
					Type:     schema.TypeBool,
					Optional: true,
					Default:  false,
				},
			},
		},
	},

	// Will get added to the default rule
	"origin": {
		Type:     schema.TypeSet,
		Optional: true,
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
				"compress": {
					Type:     schema.TypeBool,
					Optional: true,
					Default:  false,
				},
				"enable_true_client_ip": {
					Type:     schema.TypeBool,
					Optional: true,
					Default:  false,
				},
			},
		},
	},

	// rules tree can go max 5 levels deep
	"rules": &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"criteria_match": {
					Type:     schema.TypeString,
					Optional: true,
					Default:  "all",
				},
				"behavior": akps_behavior,
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
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
				"variable": &schema.Schema{
					Type:     schema.TypeSet,
					Optional: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"name": {
								Type:     schema.TypeString,
								Required: true,
							},
							"description": {
								Type:     schema.TypeString,
								Optional: true,
							},
							"hidden": {
								Type:     schema.TypeBool,
								Required: true,
							},
							"sensitive": {
								Type:     schema.TypeBool,
								Required: true,
							},
							"value": {
								Type:     schema.TypeString,
								Optional: true,
							},
						},
					},
				},
			},
		},
	},
}
