package grafana

import (
	"encoding/json"

	"github.com/hashicorp/terraform/helper/schema"
)

func ResourceTextPanelConfig() *schema.Resource {
	return &schema.Resource{
		Create: CreateTextPanelConfig,
		Delete: DeleteTextPanelConfig,
		Read:   ReadTextPanelConfig,

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
				Optional: true,
				ForceNew: true,
				Default:  "markdown",
			},

			"content": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"style": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func CreateTextPanelConfig(d *schema.ResourceData, meta interface{}) error {
	d.SetId("logical")

	model := map[string]interface{}{
		"type":    "text",
		"title":   d.Get("title").(string),
		"mode":    d.Get("mode").(string),
		"content": d.Get("content").(string),
		"style":   d.Get("style"),
	}

	modelBytes, err := json.Marshal(model)
	if err != nil {
		// Should never happen
		panic(err)
	}

	d.Set("json", string(modelBytes))

	return nil
}

func DeleteTextPanelConfig(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")

	return nil
}

func ReadTextPanelConfig(d *schema.ResourceData, meta interface{}) error {
	return nil
}
