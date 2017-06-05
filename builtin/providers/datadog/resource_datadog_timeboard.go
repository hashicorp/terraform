package datadog

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"gopkg.in/zorkian/go-datadog-api.v2"
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
				"aggregator": {
					Type:         schema.TypeString,
					Optional:     true,
					ValidateFunc: validateAggregatorMethod,
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
								Type:        schema.TypeString,
								Required:    true,
								Description: "Comparator (<, >, etc)",
							},
							"custom_bg_color": &schema.Schema{
								Type:        schema.TypeString,
								Optional:    true,
								Description: "Custom background color (e.g., #205081)",
							},
							"value": &schema.Schema{
								Type:        schema.TypeString,
								Optional:    true,
								Description: "Value that is threshold for conditional format",
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
				"order_direction": &schema.Schema{
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
					Type:        schema.TypeSet,
					Optional:    true,
					Description: "Filter for events to be overlayed on the graph.",
					Elem: &schema.Schema{
						Type: schema.TypeString,
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
					Description: "How many digits to show",
				},
				"custom_unit": &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Use a custom unit (like 'users')",
				},
				"style": &schema.Schema{
					Type:     schema.TypeMap,
					Optional: true,
				},
				"group": &schema.Schema{
					Type:        schema.TypeSet,
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
				"scope": &schema.Schema{
					Type:        schema.TypeSet,
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
			State: resourceDatadogTimeboardImport,
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

func appendConditionalFormats(datadogRequest *datadog.GraphDefinitionRequest, terraformFormats *[]interface{}) {
	for _, t_ := range *terraformFormats {
		t := t_.(map[string]interface{})
		d := datadog.DashboardConditionalFormat{
			Comparator: datadog.String(t["comparator"].(string)),
		}

		if v, ok := t["palette"]; ok {
			d.SetPalette(v.(string))
		}

		if v, ok := t["custom_bg_color"]; ok {
			d.SetCustomBgColor(v.(string))
		}

		if v, ok := t["custom_fg_color"]; ok {
			d.SetCustomFgColor(v.(string))
		}

		if v, ok := t["value"]; ok {
			d.SetValue(json.Number(v.(string)))
		}

		datadogRequest.ConditionalFormats = append(datadogRequest.ConditionalFormats, d)
	}
}

func buildTemplateVariables(terraformTemplateVariables *[]interface{}) *[]datadog.TemplateVariable {
	datadogTemplateVariables := make([]datadog.TemplateVariable, len(*terraformTemplateVariables))
	for i, t_ := range *terraformTemplateVariables {
		t := t_.(map[string]interface{})
		datadogTemplateVariables[i] = datadog.TemplateVariable{
			Name:    datadog.String(t["name"].(string)),
			Prefix:  datadog.String(t["prefix"].(string)),
			Default: datadog.String(t["default"].(string)),
		}
	}
	return &datadogTemplateVariables
}

func appendRequests(datadogGraph *datadog.Graph, terraformRequests *[]interface{}) {
	for _, t_ := range *terraformRequests {
		t := t_.(map[string]interface{})
		d := datadog.GraphDefinitionRequest{
			Query:      datadog.String(t["q"].(string)),
			Type:       datadog.String(t["type"].(string)),
			Aggregator: datadog.String(t["aggregator"].(string)),
		}
		if stacked, ok := t["stacked"]; ok {
			d.SetStacked(stacked.(bool))
		}
		if style, ok := t["style"]; ok {
			s, _ := style.(map[string]interface{})

			style := datadog.GraphDefinitionRequestStyle{}

			if v, ok := s["palette"]; ok {
				style.SetPalette(v.(string))
			}

			if v, ok := s["width"]; ok {
				style.SetWidth(v.(string))
			}

			if v, ok := s["type"]; ok {
				style.SetType(v.(string))
			}

			d.SetStyle(style)
		}

		if v, ok := t["change_type"]; ok {
			d.SetChangeType(v.(string))
		}
		if v, ok := t["compare_to"]; ok {
			d.SetCompareTo(v.(string))
		}
		if v, ok := t["increase_good"]; ok {
			d.SetIncreaseGood(v.(bool))
		}
		if v, ok := t["order_by"]; ok {
			d.SetOrderBy(v.(string))
		}
		if v, ok := t["extra_col"]; ok {
			d.SetExtraCol(v.(string))
		}
		if v, ok := t["order_direction"]; ok {
			d.SetOrderDirection(v.(string))
		}

		if v, ok := t["conditional_format"]; ok {
			v_ := v.([]interface{})
			appendConditionalFormats(&d, &v_)
		}

		datadogGraph.Definition.Requests = append(datadogGraph.Definition.Requests, d)
	}
}

func appendEvents(datadogGraph *datadog.Graph, terraformEvents *[]interface{}) {
	for _, t_ := range *terraformEvents {
		datadogGraph.Definition.Events = append(datadogGraph.Definition.Events, datadog.GraphEvent{
			Query: datadog.String(t_.(string)),
		})
	}
}

func appendMarkers(datadogGraph *datadog.Graph, terraformMarkers *[]interface{}) {
	for _, t_ := range *terraformMarkers {
		t := t_.(map[string]interface{})
		d := datadog.GraphDefinitionMarker{
			Type:  datadog.String(t["type"].(string)),
			Value: datadog.String(t["value"].(string)),
		}
		if v, ok := t["label"]; ok {
			d.SetLabel(v.(string))
		}
		datadogGraph.Definition.Markers = append(datadogGraph.Definition.Markers, d)
	}
}

func buildGraphs(terraformGraphs *[]interface{}) *[]datadog.Graph {
	datadogGraphs := make([]datadog.Graph, len(*terraformGraphs))
	for i, t_ := range *terraformGraphs {
		t := t_.(map[string]interface{})

		datadogGraphs[i] = datadog.Graph{
			Title: datadog.String(t["title"].(string)),
		}

		d := &datadogGraphs[i]
		d.Definition = &datadog.GraphDefinition{}
		d.Definition.SetViz(t["viz"].(string))

		if v, ok := t["yaxis"]; ok {
			yaxis := v.(map[string]interface{})
			if v, ok := yaxis["min"]; ok {
				min, _ := strconv.ParseFloat(v.(string), 64)
				d.Definition.Yaxis.SetMin(min)
			}
			if v, ok := yaxis["max"]; ok {
				max, _ := strconv.ParseFloat(v.(string), 64)
				d.Definition.Yaxis.SetMax(max)
			}
			if v, ok := yaxis["scale"]; ok {
				d.Definition.Yaxis.SetScale(v.(string))
			}
		}

		if v, ok := t["autoscale"]; ok {
			d.Definition.SetAutoscale(v.(bool))
		}

		if v, ok := t["text_align"]; ok {
			d.Definition.SetTextAlign(v.(string))
		}

		if precision, ok := t["precision"]; ok {
			d.Definition.SetPrecision(precision.(string))
		}

		if v, ok := t["custom_unit"]; ok {
			d.Definition.SetCustomUnit(v.(string))
		}

		if style, ok := t["style"]; ok {
			s := style.(map[string]interface{})

			gs := datadog.Style{}

			if v, ok := s["palette"]; ok {
				gs.SetPalette(v.(string))
			}

			if v, ok := s["palette_flip"]; ok {
				pf, _ := strconv.ParseBool(v.(string))
				gs.SetPaletteFlip(pf)
			}
			d.Definition.SetStyle(gs)

		}

		if v, ok := t["group"]; ok {
			for _, g := range v.(*schema.Set).List() {
				d.Definition.Groups = append(d.Definition.Groups, g.(string))
			}
		}

		if includeNoMetricHosts, ok := t["include_no_metric_hosts"]; ok {
			d.Definition.SetIncludeNoMetricHosts(includeNoMetricHosts.(bool))
		}

		if v, ok := t["scope"]; ok {
			for _, s := range v.(*schema.Set).List() {
				d.Definition.Scopes = append(d.Definition.Groups, s.(string))
			}
		}

		if v, ok := t["include_ungrouped_hosts"]; ok {
			d.Definition.SetIncludeUngroupedHosts(v.(bool))
		}
		v := t["marker"].([]interface{})
		appendMarkers(d, &v)

		v = t["events"].(*schema.Set).List()
		appendEvents(d, &v)

		v = t["request"].([]interface{})
		appendRequests(d, &v)
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
		Id:                datadog.Int(id),
		Title:             datadog.String(d.Get("title").(string)),
		Description:       datadog.String(d.Get("description").(string)),
		ReadOnly:          datadog.Bool(d.Get("read_only").(bool)),
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
	d.SetId(strconv.Itoa(timeboard.GetId()))
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

func appendTerraformGraphRequests(datadogRequests []datadog.GraphDefinitionRequest, requests *[]map[string]interface{}) {
	for _, datadogRequest := range datadogRequests {
		request := map[string]interface{}{}
		request["q"] = datadogRequest.GetQuery()
		request["stacked"] = datadogRequest.GetStacked()
		request["type"] = datadogRequest.GetType()
		if v, ok := datadogRequest.GetStyleOk(); ok {
			style := map[string]string{}
			if v, ok := v.GetPaletteOk(); ok {
				style["palette"] = v
			}
			if v, ok := v.GetTypeOk(); ok {
				style["type"] = v
			}
			if v, ok := v.GetWidthOk(); ok {
				style["width"] = v
			}
			request["style"] = style
		}
		conditionalFormats := []map[string]interface{}{}
		for _, cf := range datadogRequest.ConditionalFormats {
			conditionalFormat := map[string]interface{}{
				"palette":         cf.Palette,
				"comparator":      cf.Comparator,
				"custom_bg_color": cf.CustomBgColor,
				"value":           cf.Value,
				"custom_fg_color": cf.CustomFgColor,
			}
			conditionalFormats = append(conditionalFormats, conditionalFormat)
		}
		request["conditional_format"] = conditionalFormats
		request["change_type"] = datadogRequest.GetChangeType()
		request["order_direction"] = datadogRequest.GetOrderDirection()
		request["compare_to"] = datadogRequest.GetCompareTo()
		request["increase_good"] = datadogRequest.GetIncreaseGood()
		request["order_by"] = datadogRequest.GetOrderBy()
		request["extra_col"] = datadogRequest.GetExtraCol()

		*requests = append(*requests, request)
	}
}

func buildTerraformGraph(datadog_graph datadog.Graph) map[string]interface{} {
	graph := map[string]interface{}{}
	graph["title"] = datadog_graph.GetTitle()

	definition := datadog_graph.Definition
	graph["viz"] = definition.GetViz()

	events := []*string{}
	for _, datadog_event := range definition.Events {
		events = append(events, datadog_event.Query)
	}
	graph["events"] = events

	markers := []map[string]interface{}{}
	for _, datadog_marker := range definition.Markers {
		marker := map[string]interface{}{
			"type":  datadog_marker.Type,
			"value": datadog_marker.Value,
			"label": datadog_marker.Label,
		}
		markers = append(markers, marker)
	}
	graph["marker"] = markers

	yaxis := map[string]string{}

	if v, ok := definition.Yaxis.GetMinOk(); ok {
		yaxis["min"] = strconv.FormatFloat(v, 'f', -1, 64)
	}

	if v, ok := definition.Yaxis.GetMaxOk(); ok {
		yaxis["max"] = strconv.FormatFloat(v, 'f', -1, 64)
	}

	if v, ok := definition.Yaxis.GetScaleOk(); ok {
		yaxis["scale"] = v
	}

	graph["yaxis"] = yaxis

	graph["autoscale"] = definition.Autoscale
	graph["text_align"] = definition.TextAlign
	graph["precision"] = definition.Precision
	graph["custom_unit"] = definition.CustomUnit

	if v, ok := definition.GetStyleOk(); ok {
		style := map[string]string{}
		if v, ok := v.GetPaletteOk(); ok {
			style["palette"] = v
		}
		if v, ok := v.GetPaletteFlipOk(); ok {
			style["palette_flip"] = strconv.FormatBool(v)
		}
		graph["style"] = style
	}
	graph["group"] = definition.Groups
	graph["include_no_metric_hosts"] = definition.IncludeNoMetricHosts
	graph["scope"] = definition.Scopes
	graph["include_ungrouped_hosts"] = definition.IncludeUngroupedHosts

	requests := []map[string]interface{}{}
	appendTerraformGraphRequests(definition.Requests, &requests)
	graph["request"] = requests

	return graph
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
	d.Set("title", timeboard.GetTitle())
	d.Set("description", timeboard.GetDescription())

	graphs := []map[string]interface{}{}
	for _, datadog_graph := range timeboard.Graphs {
		graphs = append(graphs, buildTerraformGraph(datadog_graph))
	}
	d.Set("graph", graphs)

	templateVariables := []map[string]*string{}
	for _, templateVariable := range timeboard.TemplateVariables {
		tv := map[string]*string{
			"name":    templateVariable.Name,
			"prefix":  templateVariable.Prefix,
			"default": templateVariable.Default,
		}
		templateVariables = append(templateVariables, tv)
	}
	d.Set("template_variable", templateVariables)

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

func resourceDatadogTimeboardImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	if err := resourceDatadogTimeboardRead(d, meta); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
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

func validateAggregatorMethod(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	validMethods := map[string]struct{}{
		"avg":  {},
		"max":  {},
		"min":  {},
		"sum":  {},
		"last": {},
	}
	if _, ok := validMethods[value]; !ok {
		errors = append(errors, fmt.Errorf(
			`%q contains an invalid method %q. Valid methods are either "avg", "max", "min", "sum", or "last"`, k, value))
	}
	return
}
