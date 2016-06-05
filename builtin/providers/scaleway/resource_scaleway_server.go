package scaleway

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/scaleway/scaleway-cli/pkg/api"
)

func resourceScalewayServer() *schema.Resource {
	return &schema.Resource{
		Create: resourceScalewayServerCreate,
		Read:   resourceScalewayServerRead,
		Update: resourceScalewayServerUpdate,
		Delete: resourceScalewayServerDelete,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"image": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"bootscript": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"tags": &schema.Schema{
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
			},
			"ipv4_address_private": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"ipv4_address_public": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"state": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"dynamic_ip_required": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"state_detail": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"additional_volumes": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func resourceScalewayServerCreate(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway

	image := d.Get("image").(string)
	var server = api.ScalewayServerDefinition{
		Name:         d.Get("name").(string),
		Image:        String(image),
		Organization: scaleway.Organization,
	}

	server.Volumes = make(map[string]string)
	if vl, ok := d.GetOk("additional_volumes"); ok {
		for i, vid := range vl.(*schema.Set).List() {
			volumeIDx := fmt.Sprintf("%d", i+1)
			server.Volumes[volumeIDx] = vid.(string)
		}
	}

	dynamic_ip_required := d.Get("dynamic_ip_required").(bool)
	server.DynamicIPRequired = &dynamic_ip_required
	server.CommercialType = d.Get("type").(string)

	if bootscriptI, ok := d.GetOk("bootscript"); ok {
		bootscript := bootscriptI.(string)
		server.Bootscript = &bootscript
	}

	if tags, ok := d.GetOk("tags"); ok {
		server.Tags = tags.([]string)
	}

	id, err := scaleway.PostServer(server)
	if err != nil {
		serr := err.(api.ScalewayAPIError)

		return fmt.Errorf("Error Posting server with image %s. Reason: %s. %#v\n\n%#v", image, serr.APIMessage, serr, server.Volumes)
	}

	d.SetId(id)
	if d.Get("state").(string) != "stopped" {
		err = scaleway.PostServerAction(id, "poweron")
		if err != nil {
			return err
		}

		_, err = api.WaitForServerState(scaleway, id, "running")
	}

	if err != nil {
		return err
	}

	return resourceScalewayServerRead(d, m)
}

func resourceScalewayServerRead(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway
	server, err := scaleway.GetServer(d.Id())

	if err != nil {
		if serr, ok := err.(api.ScalewayAPIError); ok {
			if serr.StatusCode == 404 {
				d.SetId("")
				return nil
			}
		}

		return err
	}

	d.Set("ipv4_address_private", server.PrivateIP)
	d.Set("ipv4_address_public", server.PublicAddress.IP)
	d.Set("state", server.State)
	d.Set("state_detail", server.StateDetail)

	d.SetConnInfo(map[string]string{
		"type": "ssh",
		"host": server.PublicAddress.IP,
	})

	if len(server.Volumes) > 1 {
		var volumes []string
		for k, v := range server.Volumes {
			if k != "0" {
				volumes = append(volumes, v.Identifier)
			}
		}
		d.Set("additional_volumes", volumes)
	}

	return nil
}

func resourceScalewayServerUpdate(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway

	var server api.ScalewayServerPatchDefinition

	if d.HasChange("name") {
		name := d.Get("name").(string)
		server.Name = &name
	}

	if d.HasChange("additional_volumes") {
		if d.Get("state").(string) != "poweroff" {
			scaleway.PostServerAction(d.Id(), "poweroff")
			api.WaitForServerState(scaleway, d.Id(), "stopped")
		}

		def, _ := scaleway.GetServer(d.Id())
		volumes := make(map[string]api.ScalewayVolume)
		volumes["0"] = def.Volumes["0"]
		if vl, ok := d.GetOk("additional_volumes"); ok {
			for i, vid := range vl.(*schema.Set).List() {
				volumeIDx := fmt.Sprintf("%d", i+1)
				vol, _ := scaleway.GetVolume(vid.(string))
				volumes[volumeIDx] = *vol
			}
		}
		for k, v := range volumes {

			v.Size = 0
			v.CreationDate = ""
			v.Organization = ""
			v.ModificationDate = ""
			v.VolumeType = ""
			v.Server = nil

			volumes[k] = v
		}
		server.Volumes = &volumes
	}

	if d.HasChange("dynamic_ip_required") {
		dynamic_ip_required := d.Get("dynamic_ip_required").(bool)
		server.DynamicIPRequired = &dynamic_ip_required
	}

	if err := scaleway.PatchServer(d.Id(), server); err != nil {
		return fmt.Errorf("%#v", err)
	}

	return nil
}

func resourceScalewayServerDelete(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway

	def, err := scaleway.GetServer(d.Id())
	if err != nil {
		return err
	}

	err = scaleway.DeleteServerSafe(def.Identifier)
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
