package opc

import (
	"fmt"
	"log"

	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceOPCIPNetworkExchange() *schema.Resource {
	return &schema.Resource{
		Create: resourceOPCIPNetworkExchangeCreate,
		Read:   resourceOPCIPNetworkExchangeRead,
		Delete: resourceOPCIPNetworkExchangeDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"tags": tagsForceNewSchema(),
			"uri": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceOPCIPNetworkExchangeCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).IPNetworkExchanges()
	input := compute.CreateIPNetworkExchangeInput{
		Name: d.Get("name").(string),
	}

	log.Printf("[DEBUG] Creating ip network exchange '%s'", input.Name)
	tags := getStringList(d, "tags")
	if len(tags) != 0 {
		input.Tags = tags
	}

	if description, ok := d.GetOk("description"); ok {
		input.Description = description.(string)
	}

	info, err := client.CreateIPNetworkExchange(&input)
	if err != nil {
		return fmt.Errorf("Error creating IP Network Exchange: %s", err)
	}

	d.SetId(info.Name)
	return resourceOPCIPNetworkExchangeRead(d, meta)
}

func resourceOPCIPNetworkExchangeRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).IPNetworkExchanges()

	log.Printf("[DEBUG] Reading state of IP Network Exchange %s", d.Id())
	getInput := compute.GetIPNetworkExchangeInput{
		Name: d.Id(),
	}
	result, err := client.GetIPNetworkExchange(&getInput)
	if err != nil {
		// IP NetworkExchange does not exist
		if compute.WasNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading ip network exchange %s: %s", d.Id(), err)
	}

	d.Set("name", result.Name)
	d.Set("description", result.Description)
	d.Set("uri", result.Uri)

	if err := setStringList(d, "tags", result.Tags); err != nil {
		return err
	}

	return nil
}

func resourceOPCIPNetworkExchangeDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).IPNetworkExchanges()
	name := d.Id()

	log.Printf("[DEBUG] Deleting IP Network Exchange '%s'", name)
	input := compute.DeleteIPNetworkExchangeInput{
		Name: name,
	}
	if err := client.DeleteIPNetworkExchange(&input); err != nil {
		return fmt.Errorf("Error deleting IP Network Exchange '%s': %+v", name, err)
	}
	return nil
}
