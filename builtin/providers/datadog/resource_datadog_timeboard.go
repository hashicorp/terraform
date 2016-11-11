package datadog

import (
	"encoding/json"
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
			Comparator: t["comparator"].(string),
		}

		if palette, ok := t["palette"]; ok {
			d.Palette = palette.(string)
		}

		if customBgColor, ok := t["custom_bg_color"]; ok {
			d.CustomBgColor = customBgColor.(string)
		}

		if customFgColor, ok := t["custom_fg_color"]; ok {
			d.CustomFgColor = customFgColor.(string)
		}

		if value, ok := t["value"]; ok {
			d.Value = json.Number(value.(string))
		}

		datadogRequest.ConditionalFormats = append(datadogRequest.ConditionalFormats, d)
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

			style := struct {
				Palette *string `json:"palette,omitempty"`
				Width   *string `json:"width,omitempty"`
				Type    *string `json:"type,omitempty"`
			}{}

			if palette_, ok := s["palette"]; ok {
				palette := palette_.(string)
				style.Palette = &palette
			}

			if width, ok := s["width"]; ok {
				width := width.(string)
				style.Width = &width
			}

			if type_, ok := s["type"]; ok {
				style_type := type_.(string)
				style.Type = &style_type
			}

			d.Style = &style
		}

		if changeType, ok := t["change_type"]; ok {
			d.ChangeType = changeType.(string)
		}
		if compareTo, ok := t["compare_to"]; ok {
			d.CompareTo = compareTo.(string)
		}
		if increaseGood, ok := t["increase_good"]; ok {
			d.IncreaseGood = increaseGood.(bool)
		}
		if orderBy, ok := t["order_by"]; ok {
			d.OrderBy = orderBy.(string)
		}
		if extraCol, ok := t["extra_col"]; ok {
			d.ExtraCol = extraCol.(string)
		}
		if orderDirection, ok := t["order_direction"]; ok {
			d.OrderDirection = orderDirection.(string)
		}

		if terraformConditionalFormats, ok := t["conditional_format"]; ok {
			formats := terraformConditionalFormats.([]interface{})
			appendConditionalFormats(&d, &formats)
		}

		datadogGraph.Definition.Requests = append(datadogGraph.Definition.Requests, d)
	}
}

func appendEvents(datadogGraph *datadog.Graph, terraformEvents *[]interface{}) {
	for _, t_ := range *terraformEvents {
		d := struct {
			Query string `json:"q"`
		}{
			t_.(string),
		}
		datadogGraph.Definition.Events = append(datadogGraph.Definition.Events, d)
	}
}

func appendMarkers(datadogGraph *datadog.Graph, terraformMarkers *[]interface{}) {
	for _, t_ := range *terraformMarkers {
		t := t_.(map[string]interface{})
		d := datadog.GraphDefinitionMarker{
			Type:  t["type"].(string),
			Value: t["value"].(string),
		}
		if label, ok := t["label"]; ok {
			d.Label = label.(string)
		}
		datadogGraph.Definition.Markers = append(datadogGraph.Definition.Markers, d)
	}
}

func buildGraphs(terraformGraphs *[]interface{}) *[]datadog.Graph {
	datadogGraphs := make([]datadog.Graph, len(*terraformGraphs))
	for i, t_ := range *terraformGraphs {
		t := t_.(map[string]interface{})
		datadogGraphs[i] = datadog.Graph{Title: t["title"].(string)}
		d := &datadogGraphs[i]
		d.Definition.Viz = t["viz"].(string)

		if yaxis_, ok := t["yaxis"]; ok {
			yaxis := yaxis_.(map[string]interface{})
			if min_, ok := yaxis["min"]; ok {
				min, _ := strconv.ParseFloat(min_.(string), 64)
				d.Definition.Yaxis.Min = &min
			}
			if max_, ok := yaxis["max"]; ok {
				max, _ := strconv.ParseFloat(max_.(string), 64)
				d.Definition.Yaxis.Max = &max
			}
			if scale_, ok := yaxis["scale"]; ok {
				scale := scale_.(string)
				d.Definition.Yaxis.Scale = &scale
			}
		}

		if autoscale, ok := t["autoscale"]; ok {
			d.Definition.Autoscale = autoscale.(bool)
		}

		if textAlign, ok := t["text_align"]; ok {
			d.Definition.TextAlign = textAlign.(string)
		}

		if precision, ok := t["precision"]; ok {
			d.Definition.Precision = precision.(string)
		}

		if customUnit, ok := t["custom_unit"]; ok {
			d.Definition.CustomUnit = customUnit.(string)
		}

		if style, ok := t["style"]; ok {
			s := style.(map[string]interface{})

			style := struct {
				Palette     *string `json:"palette,omitempty"`
				PaletteFlip *bool   `json:"paletteFlip,omitempty"`
			}{}

			if palette_, ok := s["palette"]; ok {
				palette := palette_.(string)
				style.Palette = &palette
			}

			if paletteFlip_, ok := s["palette_flip"]; ok {
				paletteFlip, _ := strconv.ParseBool(paletteFlip_.(string))
				style.PaletteFlip = &paletteFlip
			}
			d.Definition.Style = &style

		}

		if groups, ok := t["group"]; ok {
			for _, g := range groups.(*schema.Set).List() {
				d.Definition.Groups = append(d.Definition.Groups, g.(string))
			}
		}

		if includeNoMetricHosts, ok := t["include_no_metric_hosts"]; ok {
			d.Definition.IncludeNoMetricHosts = includeNoMetricHosts.(bool)
		}

		if scopes, ok := t["scope"]; ok {
			for _, s := range scopes.(*schema.Set).List() {
				d.Definition.Scopes = append(d.Definition.Groups, s.(string))
			}
		}

		if includeUngroupedHosts, ok := t["include_ungrouped_hosts"]; ok {
			d.Definition.IncludeUngroupedHosts = includeUngroupedHosts.(bool)
		}
		terraformMarkers := t["marker"].([]interface{})
		appendMarkers(d, &terraformMarkers)

		terraformEvents := t["events"].(*schema.Set).List()
		appendEvents(d, &terraformEvents)

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

func appendTerraformGraphRequests(datadogRequests []datadog.GraphDefinitionRequest, requests *[]map[string]interface{}) {
	for _, datadogRequest := range datadogRequests {
		request := map[string]interface{}{}
		request["q"] = datadogRequest.Query
		request["stacked"] = datadogRequest.Stacked
		request["type"] = datadogRequest.Type
		if datadogRequest.Style != nil {
			style := map[string]string{}
			if datadogRequest.Style.Palette != nil {
				style["palette"] = *datadogRequest.Style.Palette
			}
			if datadogRequest.Style.Type != nil {
				style["type"] = *datadogRequest.Style.Type
			}
			if datadogRequest.Style.Width != nil {
				style["width"] = *datadogRequest.Style.Width
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
		request["change_type"] = datadogRequest.ChangeType
		request["order_direction"] = datadogRequest.OrderDirection
		request["compare_to"] = datadogRequest.CompareTo
		request["increase_good"] = datadogRequest.IncreaseGood
		request["order_by"] = datadogRequest.OrderBy
		request["extra_col"] = datadogRequest.ExtraCol

		*requests = append(*requests, request)
	}
}

func buildTerraformGraph(datadog_graph datadog.Graph) map[string]interface{} {
	graph := map[string]interface{}{}
	graph["title"] = datadog_graph.Title

	definition := datadog_graph.Definition
	graph["viz"] = definition.Viz

	events := []string{}
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

	if definition.Yaxis.Min != nil {
		yaxis["min"] = strconv.FormatFloat(*definition.Yaxis.Min, 'f', -1, 64)
	}

	if definition.Yaxis.Max != nil {
		yaxis["max"] = strconv.FormatFloat(*definition.Yaxis.Max, 'f', -1, 64)
	}

	if definition.Yaxis.Scale != nil {
		yaxis["scale"] = *definition.Yaxis.Scale
	}

	graph["yaxis"] = yaxis

	graph["autoscale"] = definition.Autoscale
	graph["text_align"] = definition.TextAlign
	graph["precision"] = definition.Precision
	graph["custom_unit"] = definition.CustomUnit

	if definition.Style != nil {
		style := map[string]string{}
		if definition.Style.Palette != nil {
			style["palette"] = *definition.Style.Palette
		}
		if definition.Style.PaletteFlip != nil {
			style["palette_flip"] = strconv.FormatBool(*definition.Style.PaletteFlip)
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
	d.Set("title", timeboard.Title)
	d.Set("description", timeboard.Description)

	graphs := []map[string]interface{}{}
	for _, datadog_graph := range timeboard.Graphs {
		graphs = append(graphs, buildTerraformGraph(datadog_graph))
	}
	d.Set("graph", graphs)

	templateVariables := []map[string]string{}
	for _, templateVariable := range timeboard.TemplateVariables {
		tv := map[string]string{
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
