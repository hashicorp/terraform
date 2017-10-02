package akamai

import (
	"github.com/akamai/AkamaiOPEN-edgegrid-golang/papi-v1"
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
	"network": &schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
		Default:  string(papi.NetworkStaging),
	},
	"group": &schema.Schema{
		Type:     schema.TypeString,
		Required: true,
	},
	"contract": &schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
	},
	"product_id": &schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
	},
	"cpcode": &schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
	},
	"name": &schema.Schema{
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
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"is_secure": {
					Type:     schema.TypeString,
					Optional: true,
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
	// rules tree can go max 5 levels deep
	"rule": &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"name": {
					Type:    schema.TypeString,
					Default: "default",
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
								Type:    schema.TypeString,
								Default: "default",
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
											Type:    schema.TypeString,
											Default: "default",
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
														Type:    schema.TypeString,
														Default: "default",
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
																	Type:    schema.TypeString,
																	Default: "default",
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
			},
		},
	},
}
