package terraform

import (
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/terraform"
)

func resourceSyntheticState() *schema.Resource {
	return &schema.Resource{
		Create: resourceSyntheticStateWrite,
		Update: resourceSyntheticStateWrite,
		Read:   resourceSyntheticStateRead,
		Delete: resourceSyntheticStateDelete,

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

			"outputs": &schema.Schema{
				Type:     schema.TypeMap,
				Required: true,
			},
		},
	}
}

func resourceSyntheticStateWrite(d *schema.ResourceData, meta interface{}) error {
	_, remoteState, err := resourceSyntheticStateClient(d)
	if err != nil {
		return err
	}

	newState := terraform.NewState()
	rootState := newState.RootModule()
	rootState.Outputs = make(map[string]string)

	newOutputs := d.Get("outputs").(map[string]interface{})
	for k, v := range newOutputs {
		rootState.Outputs[k] = v.(string)
	}

	if err = remoteState.WriteState(newState); err != nil {
		return err
	}

	if err = remoteState.PersistState(); err != nil {
		return err
	}

	return resourceRemoteStateRead(d, meta)
}

func resourceSyntheticStateRead(d *schema.ResourceData, meta interface{}) error {
	_, state, err := resourceSyntheticStateClient(d)
	if err != nil {
		return err
	}

	if err := state.RefreshState(); err != nil {
		return err
	}

	var outputs map[string]string
	if !state.State().Empty() {
		outputs = state.State().RootModule().Outputs
	}

	d.SetId("synth-state")
	d.Set("output", outputs)
	return nil
}

func resourceSyntheticStateDelete(d *schema.ResourceData, meta interface{}) error {
	client, _, err := resourceSyntheticStateClient(d)
	if err != nil {
		return err
	}

	err = client.Delete()
	if err == nil {
		d.SetId("")
	}
	return err
}

func resourceSyntheticStateClient(d *schema.ResourceData) (remote.Client, *remote.State, error) {
	backend := d.Get("backend").(string)
	config := make(map[string]string)
	for k, v := range d.Get("config").(map[string]interface{}) {
		config[k] = v.(string)
	}

	// Create the client to access our remote state
	log.Printf("[DEBUG] Initializing remote state client: %s", backend)
	client, err := remote.NewClient(backend, config)
	if err != nil {
		return nil, nil, err
	}

	return client, &remote.State{Client: client}, nil
}
