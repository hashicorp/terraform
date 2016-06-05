package scaleway

import "github.com/hashicorp/terraform/helper/schema"

func resourceScalewayIp() *schema.Resource {
	return &schema.Resource{
		Create: resourceScalewayIpCreate,
		Read:   resourceScalewayIpRead,
		Update: resourceScalewayIpUpdate,
		Delete: resourceScalewayIpDelete,
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

func resourceScalewayIpCreate(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway
	ip, err := scaleway.NewIP()
	if err != nil {
		return err
	}
	d.SetId(ip.IP.ID)
	return resourceScalewayIpUpdate(d, m)
}

func resourceScalewayIpRead(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway
	ip, err := scaleway.GetIP(d.Id())
	if err != nil {
		return err
	}
	d.Set("ip", ip.IP.Address)
	d.Set("server", ip.IP.Server.Identifier)
	return nil
}

func resourceScalewayIpUpdate(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway
	scaleway.AttachIP(d.Id(), d.Get("server").(string))
	return resourceScalewayIpRead(d, m)
}

func resourceScalewayIpDelete(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway
	err := scaleway.DeleteIP(d.Id())
	if err != nil {
		return err
	}
	return nil
}
