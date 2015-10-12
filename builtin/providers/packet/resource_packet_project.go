package packet

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/packethost/packngo"
)

func resourcePacketProject() *schema.Resource {
	return &schema.Resource{
		Create: resourcePacketProjectCreate,
		Read:   resourcePacketProjectRead,
		Update: resourcePacketProjectUpdate,
		Delete: resourcePacketProjectDelete,

		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"payment_method": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"created": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"updated": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourcePacketProjectCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*packngo.Client)

	createRequest := &packngo.ProjectCreateRequest{
		Name:          d.Get("name").(string),
		PaymentMethod: d.Get("payment_method").(string),
	}

	log.Printf("[DEBUG] Project create configuration: %#v", createRequest)
	project, _, err := client.Projects.Create(createRequest)
	if err != nil {
		return fmt.Errorf("Error creating Project: %s", err)
	}

	d.SetId(project.ID)
	log.Printf("[INFO] Project created: %s", project.ID)

	return resourcePacketProjectRead(d, meta)
}

func resourcePacketProjectRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*packngo.Client)

	key, _, err := client.Projects.Get(d.Id())
	if err != nil {
		// If the project somehow already destroyed, mark as
		// succesfully gone
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving Project: %s", err)
	}

	d.Set("id", key.ID)
	d.Set("name", key.Name)
	d.Set("created", key.Created)
	d.Set("updated", key.Updated)

	return nil
}

func resourcePacketProjectUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*packngo.Client)

	updateRequest := &packngo.ProjectUpdateRequest{
		ID:   d.Get("id").(string),
		Name: d.Get("name").(string),
	}

	if attr, ok := d.GetOk("payment_method"); ok {
		updateRequest.PaymentMethod = attr.(string)
	}

	log.Printf("[DEBUG] Project update: %#v", d.Get("id"))
	_, _, err := client.Projects.Update(updateRequest)
	if err != nil {
		return fmt.Errorf("Failed to update Project: %s", err)
	}

	return resourcePacketProjectRead(d, meta)
}

func resourcePacketProjectDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*packngo.Client)

	log.Printf("[INFO] Deleting Project: %s", d.Id())
	_, err := client.Projects.Delete(d.Id())
	if err != nil {
		return fmt.Errorf("Error deleting SSH key: %s", err)
	}

	d.SetId("")
	return nil
}
