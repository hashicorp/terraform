package grafana

import (
	"encoding/json"

	"github.com/hashicorp/terraform/helper/schema"
)

func ResourceDashboardListPanelConfig() *schema.Resource {
	return &schema.Resource{
		Create: CreateDashboardListPanelConfig,
		Delete: DeleteDashboardListPanelConfig,
		Read:   ReadDashboardListPanelConfig,

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

			"mode": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"limit": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
				Default:  10,
			},

			"query": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
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
		},
	}
}

func CreateDashboardListPanelConfig(d *schema.ResourceData, meta interface{}) error {
	d.SetId("logical")

	model := map[string]interface{}{
		"type":  "dashlist",
		"title": d.Get("title").(string),
		"mode":  d.Get("mode").(string),
		"limit": d.Get("limit").(int),
		"query": d.Get("query").(string),
		"tags":  d.Get("tags").(*schema.Set).List(),
	}

	modelBytes, err := json.Marshal(model)
	if err != nil {
		// Should never happen
		panic(err)
	}

	d.Set("json", string(modelBytes))

	return nil
}

func DeleteDashboardListPanelConfig(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")

	return nil
}

func ReadDashboardListPanelConfig(d *schema.ResourceData, meta interface{}) error {
	return nil
}
