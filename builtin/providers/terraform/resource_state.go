package terraform

import (
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/state/remote"
)

func resourceRemoteState() *schema.Resource {
	return &schema.Resource{
		Create: resourceRemoteStateCreate,
		Read:   resourceRemoteStateRead,
		Delete: resourceRemoteStateDelete,

		Schema: map[string]*schema.Schema{
			"backend": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"config": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},

			"output": &schema.Schema{
				Type:     schema.TypeMap,
				Computed: true,
			},
		},
	}
}

func resourceRemoteStateCreate(d *schema.ResourceData, meta interface{}) error {
	return resourceRemoteStateRead(d, meta)
}

func resourceRemoteStateRead(d *schema.ResourceData, meta interface{}) error {
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

	var outputs map[string]string
	if !state.State().Empty() {
		outputs = state.State().RootModule().Outputs
	}

	d.SetId(time.Now().UTC().String())
	d.Set("output", outputs)
	return nil
}

func resourceRemoteStateDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
