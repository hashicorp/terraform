package google

import (
	"google.golang.org/api/bigquery/v2"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceBigQueryTable() *schema.Resource {
	return &schema.Resource{
		Create: resourceBigQueryTableCreate,
		Read:   resourceBigQueryTableRead,
		Delete: resourceBigQueryTableDelete,
		Update: resourceBigQueryTableUpdate,

		Schema: map[string]*schema.Schema{
			"tableId": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"datasetId": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			
			"expirationTime": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			
			"friendlyName": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			
			"schema": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     schema.Resource{
						Schema: map[string]*schema.Schema{
								"fields": &schema.Schema{
										Type:	 schema.TypeList,
										Optiona: true,
										Elem:	 schema.Resource{
												Schema: map[string]*schema.Schema{
														"description": &schema.Schema{
																Type:     schema.TypeString,
																Optional: true,
														},
														"mode": &schema.Schema{
																Type:     schema.TypeString,
																Optional: true,
														},
														"name": &schema.Schema{
																Type:     schema.TypeString,
																Required: true,
														},
														"type": &schema.Schema{
																Type:     schema.TypeString,
																Required: true,
														},
														"fields": &schema.Schema{
																Type:	 schema.TypeList,
																Optiona: true,
																Elem:	 schema.Resource{
																		Schema: map[string]*schema.Schema{
																				"description": &schema.Schema{
																						Type:     schema.TypeString,
																						Optional: true,
																				},
																				"mode": &schema.Schema{
																						Type:     schema.TypeString,
																						Optional: true,
																				},
																				"name": &schema.Schema{
																						Type:     schema.TypeString,
																						Required: true,
																				},
																				"type": &schema.Schema{
																						Type:     schema.TypeString,
																						Required: true,
																				},
																				"fields": &schema.Schema{
																						Type:	 schema.TypeList,
																						Optiona: true,
																						Elem:	 schema.Resource{
																								Schema: map[string]*schema.Schema{
																										"description": &schema.Schema{
																												Type:     schema.TypeString,
																												Optional: true,
																										},
																										"mode": &schema.Schema{
																												Type:     schema.TypeString,
																												Optional: true,
																										},
																										"name": &schema.Schema{
																												Type:     schema.TypeString,
																												Required: true,
																										},
																										"type": &schema.Schema{
																												Type:     schema.TypeString,
																												Required: true,
																										},
																										"fields": &schema.Schema{
																												Type:	 schema.TypeList,
																												Optiona: true,
																												Elem:	 schema.Resource{
																														Schema: map[string]*schema.Schema{
																																"description": &schema.Schema{
																																		Type:     schema.TypeString,
																																		Optional: true,
																																},
																																"mode": &schema.Schema{
																																		Type:     schema.TypeString,
																																		Optional: true,
																																},
																																"name": &schema.Schema{
																																		Type:     schema.TypeString,
																																		Required: true,
																																},
																																"type": &schema.Schema{
																																		Type:     schema.TypeString,
																																		Required: true,
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
			},
			
			"self_link": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

		},
	},
}

func resourceBigQueryTableCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	datasetId := d.Get("datasetId").(string)
	tableId := d.Get("tableId").(string)
	tRef := &bigquery.TableReference{DatasetId: datasetId, ProjectId: config.Project, TableId: tableName}
	
	table := &bigquery.Table{TableReference: tRef}

	call := config.clientBigQuery.Tables.Insert(config.Project, datasetId, table)
	_, err := call.Do()
	if err != nil {
		return err
	}
	
	err = resourceBigQueryTableRead(d, meta)
	if err != nil {
		return err
	}

	return nil
}

func resourceBigQueryTableRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	
	call := config.clientBigQuery.Tables.Get(config.Project, d.Get("datasetId").(string), d.Get("name").(string))
	res, err := call.Do()
	if err != nil {
		return err
	}

	d.SetId(res.Id)
	return nil
}

func resourceBigQueryTableUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceBigQueryTableDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	d.SetId("")	
	return nil
}
