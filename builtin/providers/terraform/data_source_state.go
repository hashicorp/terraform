package terraform

import (
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/state/remote"
)

func dataSourceRemoteState() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceRemoteStateRead,

		Schema: map[string]*schema.Schema{
			"backend": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"config": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
			},

			"output": &schema.Schema{
				Type:     schema.TypeMap,
				Computed: true,
			},
		},
	}
}

func dataSourceRemoteStateRead(d *schema.ResourceData, meta interface{}) error {
	backend := d.Get("backend").(string)
	config := make(map[string]string)
	for k, v := range d.Get("config").(map[string]interface{}) {
		config[k] = v.(string)
	}

	// Create the client to access our remote state
	log.Printf("[DEBUG] Initializing remote state client: %s", backend)
	client, err := remote.NewClient(backend, config)
	if err != nil {
		return err
	}

	// Create the remote state itself and refresh it in order to load the state
	log.Printf("[DEBUG] Loading remote state...")
	state := &remote.State{Client: client}
	if err := state.RefreshState(); err != nil {
		return err
	}

	var outputs map[string]interface{}
	if !state.State().Empty() {
		outputValueMap := make(map[string]string)
		for key, output := range state.State().RootModule().Outputs {
			//This is ok for 0.6.17 as outputs will have been strings
			outputValueMap[key] = output.Value.(string)
		}
	}

	d.SetId(time.Now().UTC().String())
	d.Set("output", outputs)
	return nil
}
