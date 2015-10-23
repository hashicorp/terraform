package datadog

import (
	"bytes"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/zorkian/go-datadog-api"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

// graphDefintionRequests works around the nested struct in
// https://github.com/zorkian/go-datadog-api/blob/master/dashboards.go#L16
type graphDefintionRequests struct {
	Query   string `json:"q"`
	Stacked bool   `json:"stacked"`
}

// resourceDatadogGraph is a Datadog graph resource. It is a virtual resource as there are no autonomous graphs in
// Datadog.
func resourceDatadogGraph() *schema.Resource {
	return &schema.Resource{
		Create: resourceDatadogGraphCreate,
		Exists: resourceDatadogGraphExists,
		Read:   resourceDatadogGraphRead,
		Delete: resourceDatadogGraphDelete,
		Update: resourceDatadogGraphUpdate,

		Schema: map[string]*schema.Schema{
			"dashboard_id": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
			"title": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"viz": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"request": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"query": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"stacked": &schema.Schema{
							Type:     schema.TypeBool,
							Required: true,
						},
					},
				},
				Set: resourceDatadogRequestHash,
			},
			// TODO: support events.
		},
	}
}

// resourceDatadogGraphCreate creates a graph on associated dashboard(s).
func resourceDatadogGraphCreate(d *schema.ResourceData, meta interface{}) error {
	// This should create graphs associated with dashboards.
	// it's a virtual resource, a la "resource_vpn_connection_route"
	// hence we will need to do a bit of hacking to find out what dashboard.

	// TODO: Delete placeholder graph. See https://github.com/ojongerius/terraform-provider-datadog/issues/8

	if d.Id() == "" {
		ID := int(time.Now().Unix())
		d.SetId(strconv.Itoa(ID)) // Use seconds since Epoch, needs to be a string when saving.

		log.Printf("[INFO] Graph ID: %d", ID)
	}

	resourceDatadogGraphUpdate(d, meta)

	err := resourceDatadogGraphRetrieve(d, meta)

	if err != nil {
		return err
	}

	return nil
}

// resourceDatadogGraphExists verifies is a graph exists.
func resourceDatadogGraphExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*datadog.Client)

	// Verify our Dashboard(s) exist
	_, err := client.GetDashboard(d.Get("dashboard_id").(int))

	if err != nil {
		if strings.EqualFold(err.Error(), "API error: 404 Not Found") {
			return false, nil
		}

		return false, fmt.Errorf("Error retrieving dashboard: %s", err)
	}

	// Verify we exist
	err = resourceDatadogGraphRead(d, meta)

	if err != nil {
		return false, err
	}

	return true, nil
}

// resourceDatadogGraphRead synchronises Datadog and local state. It uses resourceDatadogGraphRetreive for this.
func resourceDatadogGraphRead(d *schema.ResourceData, meta interface{}) error {
	err := resourceDatadogGraphRetrieve(d, meta)

	if err != nil {
		return err
	}

	return nil
}

// resourceDatadogGraphRetreive synchronises Datadog and local state.
func resourceDatadogGraphRetrieve(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)

	// Get the dashboard(s)
	dashBoard, err := client.GetDashboard(d.Get("dashboard_id").(int))

	if err != nil {
		return fmt.Errorf("Error retrieving associated dashboard: %s", err)
	}

	// Walk through the graphs
	for _, g := range dashBoard.Graphs {
		// If it ends with our ID, it's us:
		if strings.HasSuffix(g.Title, fmt.Sprintf("(%s)", d.Id())) {
			log.Printf("[DEBUG] Found matching graph. Start setting/saving state.")
			d.Set("dashboard_id", d.Get("dashboard_id"))
			// Save title to state, but strip ID
			d.Set("title", strings.Replace(g.Title, fmt.Sprintf(" (%s)", d.Id()), "", 1))
			d.Set("viz", g.Definition.Viz)

			// Create an empty schema to hold all the requests.
			request := &schema.Set{F: resourceDatadogRequestHash}

			for _, r := range g.Definition.Requests {
				m := make(map[string]interface{})

				if r.Query != "" {
					m["query"] = r.Query
				}

				m["stacked"] = r.Stacked

				request.Add(m)
			}

			d.Set("request", request)

			return nil

		}
	}

	// If we are still around we've not found ourselves. Set SetId to empty and Terraform will create the resource for us.
	d.SetId("")

	return nil
}

// resourceDatadogGraphUpdate updates a graph.
func resourceDatadogGraphUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)

	// Get the dashboard
	dashboard, err := client.GetDashboard(d.Get("dashboard_id").(int))

	// Make sure the mandatory placeholder is not in there
	dashboard = buildGraph("Mandatory placeholder graph", dashboard)

	if err != nil {
		return err
	}

	// Check if there are changes
	if d.HasChange("request") {
		graphDefinition := datadog.Graph{}.Definition
		graphRequests := datadog.Graph{}.Definition.Requests
		graphDefinition.Viz = d.Get("viz").(string)

		log.Printf("[DEBUG] Request has changed.")
		o, n := d.GetChange("request")
		ors := o.(*schema.Set).Difference(n.(*schema.Set))
		nrs := n.(*schema.Set).Difference(o.(*schema.Set))

		// Loop through all the old requests and delete any obsolete ones
		for _, request := range ors.List() {
			m := request.(map[string]interface{})

			// TODO: implement
			// Delete the query as it no longer exists in the config
			log.Printf("[DEBUG] Deleting graph query %s", m["query"].(string))
			log.Printf("[DEBUG] Deleting graph stacked %t", m["stacked"].(bool))

		}
		// Loop through all the new requests and append them
		for _, request := range nrs.List() {
			m := request.(map[string]interface{})

			// Add the request
			log.Printf("[DEBUG] Adding graph query %s", m["query"].(string))
			log.Printf("[DEBUG] Adding graph stacked %t", m["stacked"].(bool))
			graphRequests = append(graphRequests, graphDefintionRequests{Query: m["query"].(string),
				Stacked: m["stacked"].(bool)})
		}

		// Add requests to the graph definition
		graphDefinition.Requests = graphRequests
		title := d.Get("title").(string) + fmt.Sprintf(" (%s)", d.Id())
		g := datadog.Graph{Title: title, Definition: graphDefinition}

		dashboard.Graphs = append(dashboard.Graphs, g) // Should be done for each
	}

	// Update/commit
	err = client.UpdateDashboard(dashboard)

	if err != nil {
		return err
	}

	return nil
}

// resourceDatadogGraphDelete deletes a graph.
func resourceDatadogGraphDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)

	// Get the dashboard
	dashboard, err := client.GetDashboard(d.Get("dashboard_id").(int))

	if err != nil {
		return fmt.Errorf("Error retrieving associated dashboard: %s", err)
	}

	// Get the dashboard without the Graph to delete
	dashboard = buildGraph(fmt.Sprintf("(%s)", d.Id()), dashboard)

	// Test if we need to reinsert the Mandatory graph
	if len(dashboard.Graphs) < 2 {
		dashboard.Graphs = createPlaceholderGraph()
	}

	// Update/commit
	err = client.UpdateDashboard(dashboard)

	if err != nil {
		return err
	}

	err = resourceDatadogGraphRetrieve(d, meta)

	if err != nil {
		return err
	}

	return nil
}

// resourceDatadogRequestHash creates a hash, used by schema.TypeSet to detect change.
func resourceDatadogRequestHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	if v, ok := m["query"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}

	if v, ok := m["stacked"]; ok {
		buf.WriteString(fmt.Sprintf("%t-", v.(bool)))
	}

	return hashcode.String(buf.String())
}

// buildGraph can be used to create a placeholder graph, used to create new dashboards, as the Datadog API does not
// support creation of dashboards without graphs.
func buildGraph(title string, dashboard *datadog.Dashboard) *datadog.Dashboard {
	// Build a new slice of graphs, excluding graphs matching title.
	g := []datadog.Graph{}
	for _, r := range dashboard.Graphs {
		if r.Title == title {
			continue
		} else {
			g = append(g, r)
		}
	}

	dashboard.Graphs = g

	return dashboard
}
