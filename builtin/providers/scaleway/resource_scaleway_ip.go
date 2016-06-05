package scaleway

import "github.com/hashicorp/terraform/helper/schema"

func resourceScalewayIP() *schema.Resource {
	return &schema.Resource{
		Create: resourceScalewayIPCreate,
		Read:   resourceScalewayIPRead,
		Update: resourceScalewayIPUpdate,
		Delete: resourceScalewayIPDelete,
		Schema: map[string]*schema.Schema{
			"server": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"ip": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceScalewayIPCreate(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway
	resp, err := scaleway.NewIP()
	if err != nil {
		return err
	}

	d.SetId(resp.IP.ID)
	return resourceScalewayIPUpdate(d, m)
}

func resourceScalewayIPRead(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway
	resp, err := scaleway.GetIP(d.Id())
	if err != nil {
		return err
	}

	d.Set("ip", resp.IP.Address)
	d.Set("server", resp.IP.Server.Identifier)
	return nil
}

func resourceScalewayIPUpdate(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway
	if d.HasChange("server") {
		if err := scaleway.AttachIP(d.Id(), d.Get("server").(string)); err != nil {
			return err
		}
	}

	return resourceScalewayIPRead(d, m)
}

func resourceScalewayIPDelete(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway
	err := scaleway.DeleteIP(d.Id())
	if err != nil {
		return err
	}
	d.SetId("")
	return nil
}
