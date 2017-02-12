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
			"backend": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					if vStr, ok := v.(string); ok && vStr == "_local" {
						ws = append(ws, "Use of the %q backend is now officially "+
							"supported as %q. Please update your configuration to ensure "+
							"compatibility with future versions of Terraform.",
							"_local", "local")
					}

					return
				},
			},

			"config": {
				Type:     schema.TypeMap,
				Optional: true,
			},

			"__has_dynamic_attributes": {
				Type:     schema.TypeString,
				Optional: true,
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

	// Don't break people using the old _local syntax - but note warning above
	if backend == "_local" {
		log.Println(`[INFO] Switching old (unsupported) backend "_local" to "local"`)
		backend = "local"
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

	d.SetId(time.Now().UTC().String())

	outputMap := make(map[string]interface{})

	remoteState := state.State()
	if remoteState.Empty() {
		log.Println("[DEBUG] empty remote state")
		return nil
	}

	for key, val := range remoteState.RootModule().Outputs {
		outputMap[key] = val.Value
	}

	mappedOutputs := remoteStateFlatten(outputMap)

	for key, val := range mappedOutputs {
		d.UnsafeSetFieldRaw(key, val)
	}
	return nil
}
