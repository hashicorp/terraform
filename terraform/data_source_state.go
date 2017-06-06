package terraform

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/backend"
	backendinit "github.com/hashicorp/terraform/backend/init"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
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

			"environment": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  backend.DefaultStateName,
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

	// Get the configuration in a type we want.
	rawConfig, err := config.NewRawConfig(d.Get("config").(map[string]interface{}))
	if err != nil {
		return fmt.Errorf("error initializing backend: %s", err)
	}

	// Don't break people using the old _local syntax - but note warning above
	if backend == "_local" {
		log.Println(`[INFO] Switching old (unsupported) backend "_local" to "local"`)
		backend = "local"
	}

	// Create the client to access our remote state
	log.Printf("[DEBUG] Initializing remote state backend: %s", backend)
	f := backendinit.Backend(backend)
	if f == nil {
		return fmt.Errorf("Unknown backend type: %s", backend)
	}
	b := f()

	// Configure the backend
	if err := b.Configure(terraform.NewResourceConfig(rawConfig)); err != nil {
		return fmt.Errorf("error initializing backend: %s", err)
	}

	// Get the state
	env := d.Get("environment").(string)
	state, err := b.State(env)
	if err != nil {
		return fmt.Errorf("error loading the remote state: %s", err)
	}
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
