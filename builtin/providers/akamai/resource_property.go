package akamai

import (
	"errors"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourcePAPI() *schema.Resource {
	return &schema.Resource{
		Create: resourcePAPICreate,
		Read:   resourcePAPIRead,
		Update: resourcePAPICreate,
		Delete: resourcePAPIDelete,
		Exists: resourcePAPIExists,
		Importer: &schema.ResourceImporter{
			State: importRecord,
		},
		Schema: map[string]*schema.Schema{
			// Terraform-only Params
			"group": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"cpcode": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			// Aallow multiple hostnames:
			"hostname": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"origin": {
				Type:     schema.TypeSet,
				Required: true,
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
					},
				},
			},

			"compress": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"extensions": {
							Type:     schema.TypeList,
							Elem:     schema.TypeString,
							Optional: true,
						},
						"content_types": {
							Type:     schema.TypeList,
							Elem:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},

			"cache": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"match": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"extensions": {
										Type:     schema.TypeList,
										Elem:     schema.TypeString,
										Optional: true,
									},
									"path": {
										Type:     schema.TypeList,
										Elem:     schema.TypeString,
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
					},
				},
			},

			"rule": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"criteria": {
							Type:     schema.TypeSet,
							Required: true,
							Elem: map[string]*schema.Schema{
								"options": {
									Type:     schema.TypeMap,
									Required: true,
								},
							},
						},
						"behaviors": {
							Type:     schema.TypeString,
							Required: true,
							Elem: map[string]*schema.Schema{
								"options": {
									Type:     schema.TypeMap,
									Required: true,
								},
							},
						},
						"comment": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func resourcePAPICreate(d *schema.ResourceData, meta interface{}) error {
	return errors.New("resourcePAPICreate")
}

func resourcePAPIRead(d *schema.ResourceData, meta interface{}) error {
	return errors.New("resourcePAPIRead")
}

func resourcePAPIDelete(d *schema.ResourceData, meta interface{}) error {
	return errors.New("resourcePAPIDelete")
}

func resourcePAPIExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	return false, errors.New("resourcePAPIExists")
}
