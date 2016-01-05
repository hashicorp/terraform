package google

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/bigquery/v2"
	"google.golang.org/api/googleapi"
)

func resourceBigQueryDataset() *schema.Resource {
	return &schema.Resource{
		Create: resourceBigQueryDatasetCreate,
		Read:   resourceBigQueryDatasetRead,
		Update: resourceBigQueryDatasetUpdate,
		Delete: resourceBigQueryDatasetDelete,

		Schema: map[string]*schema.Schema{
			"datasetId": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"friendlyName": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"location": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"defaultTableExpirationMs": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},

			"access": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"role": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"userByEmail": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"groupByEmail": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"domain": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"specialGroup": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"view": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"projectId": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},

									"datasetId": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},

									"tableId": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
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

			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"lastModifiedTime": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
	}
}

func resourceBigQueryDatasetCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	datasetRef := &bigquery.DatasetReference{DatasetId: d.Get("datasetId").(string), ProjectId: config.Project}

	dataset := &bigquery.Dataset{DatasetReference: datasetRef}

	if v, ok := d.GetOk("friendlyName"); ok {
		dataset.FriendlyName = v.(string)
	}

	if v, ok := d.GetOk("description"); ok {
		dataset.Description = v.(string)
	}

	if v, ok := d.GetOk("location"); ok {
		dataset.Location = v.(string)
	}

	if v, ok := d.GetOk("defaultTableExpirationMs"); ok {
		dataset.DefaultTableExpirationMs = v.(int64)
	}

	if v, ok := d.GetOk("access"); ok {
		accessList := make([]*bigquery.DatasetAccess, 0)
		for _, access_interface := range v.([]interface{}) {
			access_parsed := &bigquery.DatasetAccess{}
			access_raw := access_interface.(map[string]interface{})
			if role, ok := access_raw["role"]; ok {
				access_parsed.Role = role.(string)
			}
			if userByEmail, ok := access_raw["userByEmail"]; ok {
				access_parsed.UserByEmail = userByEmail.(string)
			}
			if groupByEmail, ok := access_raw["groupByEmail"]; ok {
				access_parsed.GroupByEmail = groupByEmail.(string)
			}
			if domain, ok := access_raw["domain"]; ok {
				access_parsed.Domain = domain.(string)
			}
			if specialGroup, ok := access_raw["specialGroup"]; ok {
				access_parsed.SpecialGroup = specialGroup.(string)
			}
			if view, ok := access_raw["view"]; ok {
				view_raw := view.([]interface{})
				if len(view_raw) > 1 {
					fmt.Errorf("There are more then one view records in a single access record, this is not valid.")
				}
				view_parsed := &bigquery.TableReference{}
				view_zero := view_raw[0].(map[string]interface{})
				if projectId, ok := view_zero["projectId"]; ok {
					view_parsed.ProjectId = projectId.(string)
				}
				if datasetId, ok := view_zero["datasetId"]; ok {
					view_parsed.DatasetId = datasetId.(string)
				}
				if tableId, ok := view_zero["tableId"]; ok {
					view_parsed.TableId = tableId.(string)
				}
				access_parsed.View = view_parsed
			}

			accessList = append(accessList, access_parsed)
		}

		dataset.Access = accessList
	}

	call := config.clientBigQuery.Datasets.Insert(config.Project, dataset)
	_, err := call.Do()
	if err != nil {
		return err
	}

	return resourceBigQueryDatasetRead(d, meta)
}

func resourceBigQueryDatasetRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	call := config.clientBigQuery.Datasets.Get(config.Project, d.Get("datasetId").(string))
	res, err := call.Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			// The resource doesn't exist anymore
			d.SetId("")

			return nil
		}
		return fmt.Errorf("Failed to read bigquery dataset %s with err: %q", d.Get("datasetId").(string), err)
	}

	d.SetId(res.Id)
	d.Set("self_link", res.SelfLink)
	d.Set("lastModifiedTime", res.LastModifiedTime)
	d.Set("id", res.Id)
	return nil
}

func resourceBigQueryDatasetUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceBigQueryDatasetDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	call := config.clientBigQuery.Datasets.Delete(config.Project, d.Get("datasetId").(string))
	err := call.Do()
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
