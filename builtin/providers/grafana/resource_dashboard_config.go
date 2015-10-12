package grafana

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

func ResourceDashboardConfig() *schema.Resource {
	return &schema.Resource{
		Create: CreateDashboardConfig,
		Delete: DeleteDashboardConfig,
		Read:   ReadDashboardConfig,

		Schema: map[string]*schema.Schema{
			"json": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"title": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"tags": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"timezone": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "browser",
			},

			"editable": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Default:  false,
			},

			"hide_controls": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Default:  true,
			},

			"shared_crosshair": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Default:  false,
			},

			"row": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{

						"collapse": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
							Default:  false,
						},

						"editable": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
							Default:  false,
						},

						"height": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
							Default:  "250px",
						},

						"title": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
							// This is what the Grafana UI sets the title to
							// if the user doesn't specify one.
							Default: "New Row",
						},

						"show_title": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
							Default:  false,
						},

						"panel": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							ForceNew: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{

									"id": &schema.Schema{
										Type:     schema.TypeInt,
										Required: true,
										ForceNew: true,
									},

									"span": &schema.Schema{
										Type:     schema.TypeInt,
										Required: true,
										ForceNew: true,
									},

									"config_json": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
										ForceNew: true,
									},
								},
							},
						},
					},
				},
			},

			"annotation": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{

						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},

						"data_source_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},

						"index_name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"show_line": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
							Default:  true,
						},

						"icon_color": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
							Default:  "#C0C6BE",
						},

						"line_color": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
							Default:  "rgba(255, 96, 96, 0.592157)",
						},

						"icon_size": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: true,
							Default:  13,
						},

						"enable": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
							Default:  true,
						},

						"query": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},

						"title_column": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"title_field": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"time_field": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"tags_column": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"tags_field": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"text_column": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"text_field": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},

			"link": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},

						"icon": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
							Default:  "external link",
						},

						"tags": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							ForceNew: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},

						"open_new_window": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
							Default:  false,
						},

						"title": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},

						"tooltip": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
							Default:  "",
						},

						"url": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
							Default:  "",
						},

						"keep_time_range": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
							Default:  false,
						},

						"keep_variable_values": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
							Default:  false,
						},

						"as_dropdown_menu": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
							Default:  false,
						},
					},
				},
			},

			"template_variable": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},

						"data_source_name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"refresh_on_load": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
							Default:  false,
						},

						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},

						"label": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},

						"hide_label": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
							Default:  false,
						},

						"include_all_option": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
							Default:  false,
						},

						"all_value_format": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
							Default:  "regex wildcard",
						},

						"allow_multiple_values": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
							Default:  false,
						},

						"multiple_values_format": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
							Default:  "regex values",
						},

						"query": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
					},
				},
			},
		},
	}
}

func CreateDashboardConfig(d *schema.ResourceData, meta interface{}) error {
	d.SetId("logical")

	rows := []map[string]interface{}{}
	annotations := []map[string]interface{}{}
	templateVars := []map[string]interface{}{}
	links := []map[string]interface{}{}

	for _, rowI := range d.Get("row").([]interface{}) {
		rowM := rowI.(map[string]interface{})

		panels := []map[string]interface{}{}
		for _, panelI := range rowM["panel"].([]interface{}) {
			panelM := panelI.(map[string]interface{})
			id := panelM["id"].(int)
			span := panelM["span"].(int)
			configJSON := panelM["config_json"].(string)
			panel := make(map[string]interface{})
			err := json.Unmarshal([]byte(configJSON), &panel)
			if err != nil {
				return fmt.Errorf("Invalid config_json for panel %d: %s", id, err)
			}

			// We override the id and span here, since they are significant
			// only in the context of the whole dashboard and so this makes
			// things a little more composable.
			panel["id"] = id
			panel["span"] = span

			panels = append(panels, panel)
		}

		rows = append(rows, map[string]interface{}{
			"collapse":  rowM["collapse"],
			"editable":  rowM["editable"],
			"height":    rowM["height"],
			"title":     rowM["title"],
			"showTitle": rowM["show_title"],
			"panels":    panels,
		})
	}

	for _, annotI := range d.Get("annotation").([]interface{}) {
		annotM := annotI.(map[string]interface{})
		annotations = append(annotations, map[string]interface{}{
			"name":        annotM["name"],
			"datasource":  annotM["data_source_name"],
			"showLine":    annotM["show_line"],
			"iconColor":   annotM["icon_color"],
			"lineColor":   annotM["line_color"],
			"iconSize":    annotM["icon_size"],
			"enable":      annotM["enable"],
			"index":       annotM["index_name"],
			"query":       annotM["query"],
			"timeField":   annotM["time_field"],
			"titleColumn": annotM["title_column"],
			"titleField":  annotM["title_field"],
			"tagsColumn":  annotM["tags_column"],
			"tagsField":   annotM["tags_field"],
			"textColumn":  annotM["text_column"],
			"textField":   annotM["text_field"],
		})
	}

	for _, linkI := range d.Get("link").([]interface{}) {
		linkM := linkI.(map[string]interface{})
		links = append(links, map[string]interface{}{
			"title":       linkM["title"],
			"type":        linkM["type"],
			"icon":        linkM["icon"],
			"tags":        linkM["tags"].(*schema.Set).List(),
			"targetBlank": linkM["open_new_window"],
			"tooltip":     linkM["tooltip"],
			"url":         linkM["url"],
			"keepTime":    linkM["keep_time_range"],
			"includeVars": linkM["keep_variable_values"],
			"asDropdown":  linkM["as_dropdown_menu"],
		})
	}

	for _, varI := range d.Get("template_variable").([]interface{}) {
		varM := varI.(map[string]interface{})
		templateVars = append(templateVars, map[string]interface{}{
			"type":        varM["type"],
			"datasource":  varM["data_source_name"],
			"name":        varM["name"],
			"label":       varM["label"],
			"hideLabel":   varM["hide_label"],
			"includeAll":  varM["include_all_option"],
			"allFormat":   varM["all_value_format"],
			"multi":       varM["allow_multiple_values"],
			"multiFormat": varM["multiple_values_format"],
			"query":       varM["query"],

			// Using underscores for this one is not a mistake; this is an
			// inconsistency in the Grafana dashboard model format.
			"refresh_on_load": varM["refresh_on_load"],
		})
	}

	model := map[string]interface{}{
		"title":           d.Get("title").(string),
		"timezone":        d.Get("timezone").(string),
		"tags":            d.Get("tags").(*schema.Set).List(),
		"editable":        d.Get("editable").(bool),
		"hideControls":    d.Get("hide_controls").(bool),
		"sharedCrosshair": d.Get("shared_crosshair").(bool),
		"rows":            rows,
		"annotations": map[string]interface{}{
			"list": annotations,
		},
		"templating": map[string]interface{}{
			"list": templateVars,
		},
		"links":         links,
		"schemaVersion": 6,
	}

	modelBytes, err := json.Marshal(model)
	if err != nil {
		// Should never happen
		panic(err)
	}

	d.Set("json", string(modelBytes))

	return nil
}

func DeleteDashboardConfig(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")

	return nil
}

func ReadDashboardConfig(d *schema.ResourceData, meta interface{}) error {
	return nil
}
