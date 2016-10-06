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
				"viz": &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
				},
				"request": request,
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
	d.Set("graphs", timeboard.Graphs)
	d.Set("template_variables", timeboard.TemplateVariables)
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
