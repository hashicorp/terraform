package datadog

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/zorkian/go-datadog-api"
)

// resourceDatadogDashboard is a Datadog dashboard resource.
func resourceDatadogDashboard() *schema.Resource {
	return &schema.Resource{
		Create: resourceDatadogDashboardCreate,
		Read:   resourceDatadogDashboardRead,
		Exists: resourceDatadogDashboardExists,
		Update: resourceDatadogDashboardUpdate,
		Delete: resourceDatadogDashboardDelete,

		Schema: map[string]*schema.Schema{
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"title": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"template_variable": templateVariablesSchema(),
		},
	}
}

// resourceDatadogDashboardCreate creates a new dashboard.
func resourceDatadogDashboardCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)

	opts := datadog.Dashboard{}
	opts.Description = d.Get("description").(string)
	opts.Title = d.Get("title").(string)
	opts.Graphs = createPlaceholderGraph()
	opts.TemplateVariables = []datadog.TemplateVariable{}

	dashboard, err := client.CreateDashboard(&opts)

	if err != nil {
		return fmt.Errorf("Error creating Dashboard: %s", err)
	}

	d.SetId(strconv.Itoa(dashboard.Id))

	err = resourceDatadogDashboardUpdate(d, meta)

	if err != nil {
		return fmt.Errorf("Error updating Dashboard: %s", err)
	}

	return nil
}

// resourceDatadogDashboardCreate deletes an existing dashboard.
func resourceDatadogDashboardDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)

	log.Printf("[DEBUG] Deleting Dashboard: %s", d.Id())

	id, _ := strconv.Atoi(d.Id())

	err := client.DeleteDashboard(id)

	if err != nil {
		return fmt.Errorf("Error deleting Dashboard: %s", err)
	}

	return nil
}

// resourceDatadogDashboardExists verifies a dashboard exists.
func resourceDatadogDashboardExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*datadog.Client)

	id, _ := strconv.Atoi(d.Id())

	_, err := client.GetDashboard(id)

	if err != nil {
		if strings.EqualFold(err.Error(), "API error: 404 Not Found") {
			return false, nil
		}

		return false, fmt.Errorf("Error retrieving dashboard: %s", err)
	}

	return true, nil
}

// resourceDatadogDashboardRead synchronises Datadog and local state.
func resourceDatadogDashboardRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)

	id, _ := strconv.Atoi(d.Id())

	resp, err := client.GetDashboard(id)

	if err != nil {
		return fmt.Errorf("Error retrieving dashboard: %s", err)
	}

	d.Set("id", resp.Id)
	d.Set("descripton", resp.Description)
	d.Set("title", resp.Title)
	d.Set("graphs", resp.Graphs)

	t := &schema.Set{F: templateVariablesHash}

	for _, v := range resp.TemplateVariables {

		m := make(map[string]interface{})

		if v.Name != "" {
			m["name"] = v.Name
		}
		if v.Prefix != "" {
			m["prefix"] = v.Prefix
		}

		if v.Default != "" {
			m["default"] = v.Default
		}

		t.Add(m)
	}

	d.Set("template_variable", resp.TemplateVariables)

	return nil
}

// resourceDatadogDashboardUpdate updates an existing dashboard.
func resourceDatadogDashboardUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)

	opts := datadog.Dashboard{}

	id, _ := strconv.Atoi(d.Id())

	opts.Id = id
	opts.Description = d.Get("description").(string)
	opts.Title = d.Get("title").(string)
	opts.Graphs = createPlaceholderGraph()

	v := []datadog.TemplateVariable{}

	if d.HasChange("template_variable") {
		o, n := d.GetChange("template_variable")

		variables := o.(*schema.Set).Intersection(n.(*schema.Set))

		nvs := n.(*schema.Set).Difference(o.(*schema.Set))
		for _, variable := range nvs.List() {
			m := variable.(map[string]interface{})

			v = append(v, datadog.TemplateVariable{
				Name:    m["name"].(string),
				Prefix:  m["prefix"].(string),
				Default: m["default"].(string),
			})
			variables.Add(variable)
		}
		d.Set("template_variable", variables)
	}

	opts.TemplateVariables = v

	err := client.UpdateDashboard(&opts)

	if err != nil {
		return fmt.Errorf("Error updating Dashboard: %s", err)
	}

	return nil

}
