package scaleway

import (
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/scaleway/scaleway-cli/pkg/api"
)

func resourceScalewayIP() *schema.Resource {
	return &schema.Resource{
		Create: resourceScalewayIPCreate,
		Read:   resourceScalewayIPRead,
		Update: resourceScalewayIPUpdate,
		Delete: resourceScalewayIPDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"server": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"ip": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceScalewayIPCreate(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway

	mu.Lock()
	resp, err := scaleway.NewIP()
	mu.Unlock()
	if err != nil {
		return err
	}

	d.SetId(resp.IP.ID)
	return resourceScalewayIPUpdate(d, m)
}

func resourceScalewayIPRead(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway
	log.Printf("[DEBUG] Reading IP\n")

	resp, err := scaleway.GetIP(d.Id())
	if err != nil {
		log.Printf("[DEBUG] Error reading ip: %q\n", err)
		if serr, ok := err.(api.ScalewayAPIError); ok {
			if serr.StatusCode == 404 {
				d.SetId("")
				return nil
			}
		}
		return err
	}

	d.Set("ip", resp.IP.Address)
	if resp.IP.Server != nil {
		d.Set("server", resp.IP.Server.Identifier)
	}
	return nil
}

func resourceScalewayIPUpdate(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway

	mu.Lock()
	defer mu.Unlock()

	if d.HasChange("server") {
		if d.Get("server").(string) != "" {
			log.Printf("[DEBUG] Attaching IP %q to server %q\n", d.Id(), d.Get("server").(string))
			if err := scaleway.AttachIP(d.Id(), d.Get("server").(string)); err != nil {
				return err
			}
		} else {
			log.Printf("[DEBUG] Detaching IP %q\n", d.Id())
			return scaleway.DetachIP(d.Id())
		}
	}

	return resourceScalewayIPRead(d, m)
}

func resourceScalewayIPDelete(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway

	mu.Lock()
	defer mu.Unlock()

	err := scaleway.DeleteIP(d.Id())
	if err != nil {
		return err
	}
	d.SetId("")
	return nil
}
