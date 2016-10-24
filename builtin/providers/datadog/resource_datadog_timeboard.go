package datadog

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/zorkian/go-datadog-api"
)

func resourceDatadogTimeboard() *schema.Resource {
	request := &schema.Schema{
		Type:     schema.TypeList,
		Required: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"q": &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
				},
				"stacked": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
					Default:  false,
				},
				"type": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
					Default:  "line",
				},
				"style": &schema.Schema{
					Type:     schema.TypeMap,
					Optional: true,
				},
				"conditional_format": &schema.Schema{
					Type:        schema.TypeList,
					Optional:    true,
					Description: "A list of conditional formatting rules.",
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"palette": &schema.Schema{
								Type:        schema.TypeString,
								Optional:    true,
								Description: "The palette to use if this condition is met.",
							},
							"comparator": &schema.Schema{
								Type:        schema.TypeBool,
								Required:    true,
								Description: "Comparator (<, >, etc)",
							},
							"custom_bg_color": &schema.Schema{
								Type:        schema.TypeString,
								Optional:    true,
								Description: "Custom background color (e.g., #205081)",
							},
							"value": &schema.Schema{
								Type:        schema.TypeFloat,
								Optional:    true,
								Description: "Value that is threshold for conditional format",
							},
							"inverted": &schema.Schema{
								Type:     schema.TypeBool,
								Optional: true,
							},
							"custom_fg_color": &schema.Schema{
								Type:        schema.TypeString,
								Optional:    true,
								Description: "Custom foreground color (e.g., #59afe1)",
							},
						},
					},
				},
				"change_type": &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Type of change for change graphs.",
				},
				"change_direction": &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Sort change graph in ascending or descending order.",
				},
				"compare_to": &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "The time period to compare change against in change graphs.",
				},
				"increase_good": &schema.Schema{
					Type:        schema.TypeBool,
					Optional:    true,
					Description: "Decides whether to represent increases as good or bad in change graphs.",
				},
				"order_by": &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "The field a change graph will be ordered by.",
				},
				"extra_col": &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "If set to 'present', this will include the present values in change graphs.",
				},
			},
		},
	}

	marker := &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"type": &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
				},
				"value": &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
				},
				"label": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
				"val": &schema.Schema{
					Type:     schema.TypeFloat,
					Optional: true,
				},
				"min": &schema.Schema{
					Type:     schema.TypeFloat,
					Optional: true,
				},
				"max": &schema.Schema{
					Type:     schema.TypeFloat,
					Optional: true,
				},
			},
		},
	}

	graph := &schema.Schema{
		Type:        schema.TypeList,
		Required:    true,
		Description: "A list of graph definitions.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"title": &schema.Schema{
					Type:        schema.TypeString,
					Required:    true,
					Description: "The name of the graph.",
				},
				"events": &schema.Schema{
					Type:        schema.TypeList,
					Optional:    true,
					Description: "Filter for events to be overlayed on the graph.",
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"q": &schema.Schema{
								Type:     schema.TypeString,
								Required: true,
							},
						},
					},
				},
				"viz": &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
				},
				"request": request,
				"marker":  marker,
				"yaxis": &schema.Schema{
					Type:     schema.TypeMap,
					Optional: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"min": &schema.Schema{
								Type:     schema.TypeFloat,
								Optional: true,
							},
							"max": &schema.Schema{
								Type:     schema.TypeFloat,
								Optional: true,
							},
							"scale": &schema.Schema{
								Type:     schema.TypeString,
								Optional: true,
							},
						},
					},
				},
				"autoscale": &schema.Schema{
					Type:        schema.TypeBool,
					Optional:    true,
					Description: "Automatically scale graphs",
				},
				"text_align": &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "How to align text",
				},
				"precision": &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "How to align text",
				},
				"custom_unit": &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "How to align text",
				},
				"style": &schema.Schema{
					Type:     schema.TypeMap,
					Optional: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"palette": &schema.Schema{
								Type:     schema.TypeString,
								Optional: true,
							},
							"palette_flip": &schema.Schema{
								Type:     schema.TypeBool,
								Optional: true,
							},
						},
					},
				},
				"groups": &schema.Schema{
					Type:        schema.TypeList,
					Optional:    true,
					Description: "A list of groupings for hostmap type graphs.",
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
				"include_no_metric_hosts": &schema.Schema{
					Type:        schema.TypeBool,
					Optional:    true,
					Description: "Include hosts without metrics in hostmap graphs",
				},
				"scopes": &schema.Schema{
					Type:        schema.TypeList,
					Optional:    true,
					Description: "A list of scope filters for hostmap type graphs.",
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
				"include_ungrouped_hosts": &schema.Schema{
					Type:        schema.TypeBool,
					Optional:    true,
					Description: "Include ungrouped hosts in hostmap graphs",
				},
			},
		},
	}

	template_variable := &schema.Schema{
		Type:        schema.TypeList,
		Optional:    true,
		Description: "A list of template variables for using Dashboard templating.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"name": &schema.Schema{
					Type:        schema.TypeString,
					Required:    true,
					Description: "The name of the variable.",
				},
				"prefix": &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "The tag prefix associated with the variable. Only tags with this prefix will appear in the variable dropdown.",
				},
				"default": &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "The default value for the template variable on dashboard load.",
				},
			},
		},
	}

	return &schema.Resource{
		Create: resourceDatadogTimeboardCreate,
		Update: resourceDatadogTimeboardUpdate,
		Read:   resourceDatadogTimeboardRead,
		Delete: resourceDatadogTimeboardDelete,
		Exists: resourceDatadogTimeboardExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"title": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the dashboard.",
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "A description of the dashboard's content.",
			},
			"read_only": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"graph":             graph,
			"template_variable": template_variable,
		},
	}
}

func buildTemplateVariables(terraformTemplateVariables *[]interface{}) *[]datadog.TemplateVariable {
	datadogTemplateVariables := make([]datadog.TemplateVariable, len(*terraformTemplateVariables))
	for i, t_ := range *terraformTemplateVariables {
		t := t_.(map[string]interface{})
		datadogTemplateVariables[i] = datadog.TemplateVariable{
			Name:    t["name"].(string),
			Prefix:  t["prefix"].(string),
			Default: t["default"].(string)}
	}
	return &datadogTemplateVariables
}

func appendRequests(datadogGraph *datadog.Graph, terraformRequests *[]interface{}) {
	for _, t_ := range *terraformRequests {
		t := t_.(map[string]interface{})
		d := datadog.GraphDefinitionRequest{
			Query: t["q"].(string),
			Type:  t["type"].(string),
		}
		if stacked, ok := t["stacked"]; ok {
			d.Stacked = stacked.(bool)
		}
		if style, ok := t["style"]; ok {
			s, _ := style.(map[string]interface{})
			if palette, ok := s["palette"]; ok {
				d.Style.Palette = palette.(string)
			}
		}
		datadogGraph.Definition.Requests = append(datadogGraph.Definition.Requests, d)
	}
}

func buildGraphs(terraformGraphs *[]interface{}) *[]datadog.Graph {
	datadogGraphs := make([]datadog.Graph, len(*terraformGraphs))
	for i, t_ := range *terraformGraphs {
		t := t_.(map[string]interface{})
		datadogGraphs[i] = datadog.Graph{Title: t["title"].(string)}
		d := &datadogGraphs[i]
		d.Definition.Viz = t["viz"].(string)
		terraformRequests := t["request"].([]interface{})
		appendRequests(d, &terraformRequests)
	}
	return &datadogGraphs
}

func buildTimeboard(d *schema.ResourceData) (*datadog.Dashboard, error) {
	var id int
	if d.Id() != "" {
		var err error
		id, err = strconv.Atoi(d.Id())
		if err != nil {
			return nil, err
		}
	}
	terraformGraphs := d.Get("graph").([]interface{})
	terraformTemplateVariables := d.Get("template_variable").([]interface{})
	return &datadog.Dashboard{
		Id:                id,
		Title:             d.Get("title").(string),
		Description:       d.Get("description").(string),
		ReadOnly:          d.Get("read_only").(bool),
		Graphs:            *buildGraphs(&terraformGraphs),
		TemplateVariables: *buildTemplateVariables(&terraformTemplateVariables),
	}, nil
}

func resourceDatadogTimeboardCreate(d *schema.ResourceData, meta interface{}) error {
	timeboard, err := buildTimeboard(d)
	if err != nil {
		return fmt.Errorf("Failed to parse resource configuration: %s", err.Error())
	}
	timeboard, err = meta.(*datadog.Client).CreateDashboard(timeboard)
	if err != nil {
		return fmt.Errorf("Failed to create timeboard using Datadog API: %s", err.Error())
	}
	d.SetId(strconv.Itoa(timeboard.Id))
	return nil
}

func resourceDatadogTimeboardUpdate(d *schema.ResourceData, meta interface{}) error {
	timeboard, err := buildTimeboard(d)
	if err != nil {
		return fmt.Errorf("Failed to parse resource configuration: %s", err.Error())
	}
	if err = meta.(*datadog.Client).UpdateDashboard(timeboard); err != nil {
		return fmt.Errorf("Failed to update timeboard using Datadog API: %s", err.Error())
	}
	return resourceDatadogTimeboardRead(d, meta)
}

func resourceDatadogTimeboardRead(d *schema.ResourceData, meta interface{}) error {
	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}
	timeboard, err := meta.(*datadog.Client).GetDashboard(id)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] timeboard: %v", timeboard)
	d.Set("title", timeboard.Title)
	d.Set("description", timeboard.Description)

	graphs := []map[string]interface{}{}
	for _, datadog_graph := range timeboard.Graphs {
		graph := map[string]interface{}{}
		graph["title"] = datadog_graph.Title
		graph["viz"] = datadog_graph.Definition.Viz

		requests := []map[string]interface{}{}
		for _, datadog_request := range datadog_graph.Definition.Requests {
			request := map[string]interface{}{}
			request["q"] = datadog_request.Query
			request["stacked"] = datadog_request.Stacked
			request["type"] = datadog_request.Type
			request["style"] = map[string]string{
				"palette": datadog_request.Style.Palette,
			}
			conditional_formats := []map[string]interface{}{}
			for _, cf := range datadog_request.ConditionalFormats {
				conditional_format := map[string]interface{}{
					"palette":         cf.Palette,
					"comparator":      cf.Comparator,
					"custom_bg_color": cf.CustomBgColor,
					"value":           cf.Value,
					"inverted":        cf.Inverted,
					"custom_fg_color": cf.CustomFgColor,
				}
				conditional_formats = append(conditional_formats, conditional_format)
			}
			request["conditional_format"] = conditional_formats

			requests = append(requests, request)
		}
		graph["request"] = requests

		graphs = append(graphs, graph)
	}
	d.Set("graph", graphs)
	// TODO template variables (list)
	d.Set("template_variable", timeboard.TemplateVariables)
	return nil
}

func resourceDatadogTimeboardDelete(d *schema.ResourceData, meta interface{}) error {
	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}
	if err = meta.(*datadog.Client).DeleteDashboard(id); err != nil {
		return err
	}
	return nil
}

func resourceDatadogTimeboardExists(d *schema.ResourceData, meta interface{}) (b bool, e error) {
	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return false, err
	}
	if _, err = meta.(*datadog.Client).GetDashboard(id); err != nil {
		if strings.Contains(err.Error(), "404 Not Found") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
