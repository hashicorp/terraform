package grafana

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"

	gapi "github.com/apparentlymart/go-grafana-api"
)

func ResourceDashboard() *schema.Resource {
	return &schema.Resource{
		Create: CreateDashboard,
		Delete: DeleteDashboard,
		Read:   ReadDashboard,

		Schema: map[string]*schema.Schema{
			"slug": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"config_json": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				StateFunc:    NormalizeDashboardConfigJSON,
				ValidateFunc: ValidateDashboardConfigJSON,
			},
		},
	}
}

func CreateDashboard(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

	model := prepareDashboardModel(d.Get("config_json").(string))

	resp, err := client.SaveDashboard(model, false)
	if err != nil {
		return err
	}

	d.SetId(resp.Slug)

	return ReadDashboard(d, meta)
}

func ReadDashboard(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

	slug := d.Id()

	dashboard, err := client.Dashboard(slug)
	if err != nil {
		return err
	}

	configJSONBytes, err := json.Marshal(dashboard.Model)
	if err != nil {
		return err
	}

	configJSON := NormalizeDashboardConfigJSON(string(configJSONBytes))

	d.SetId(dashboard.Meta.Slug)
	d.Set("slug", dashboard.Meta.Slug)
	d.Set("config_json", configJSON)

	return nil
}

func DeleteDashboard(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

	slug := d.Id()
	return client.DeleteDashboard(slug)
}

func prepareDashboardModel(configJSON string) map[string]interface{} {
	configMap := map[string]interface{}{}
	err := json.Unmarshal([]byte(configJSON), &configMap)
	if err != nil {
		// The validate function should've taken care of this.
		panic(fmt.Errorf("Invalid JSON got into prepare func"))
	}

	delete(configMap, "id")
	configMap["version"] = 0

	return configMap
}

func ValidateDashboardConfigJSON(configI interface{}, k string) ([]string, []error) {
	configJSON := configI.(string)
	configMap := map[string]interface{}{}
	err := json.Unmarshal([]byte(configJSON), &configMap)
	if err != nil {
		return nil, []error{err}
	}
	return nil, nil
}

func NormalizeDashboardConfigJSON(configI interface{}) string {
	configJSON := configI.(string)

	configMap := map[string]interface{}{}
	err := json.Unmarshal([]byte(configJSON), &configMap)
	if err != nil {
		// The validate function should've taken care of this.
		return ""
	}

	// Some properties are managed by this provider and are thus not
	// significant when included in the JSON.
	delete(configMap, "id")
	delete(configMap, "version")

	ret, err := json.Marshal(configMap)
	if err != nil {
		// Should never happen.
		return configJSON
	}

	return string(ret)
}
