package scaleway

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/scaleway/scaleway-cli/pkg/api"
)

func resourceScalewayServer() *schema.Resource {
	return &schema.Resource{
		Create: resourceScalewayServerCreate,
		Read:   resourceScalewayServerRead,
		Update: resourceScalewayServerUpdate,
		Delete: resourceScalewayServerDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"image": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"type": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"bootscript": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"tags": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
			},
			"security_group": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"volume": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"size_in_gb": {
							Type:         schema.TypeInt,
							Required:     true,
							ValidateFunc: validateVolumeSize,
						},
						"type": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateVolumeType,
						},
						"volume_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"enable_ipv6": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"dynamic_ip_required": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"private_ip": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"public_ip": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"public_ipv6": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"state": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"state_detail": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceScalewayServerCreate(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway

	mu.Lock()
	defer mu.Unlock()

	image := d.Get("image").(string)
	var server = api.ScalewayServerDefinition{
		Name:          d.Get("name").(string),
		Image:         String(image),
		Organization:  scaleway.Organization,
		EnableIPV6:    d.Get("enable_ipv6").(bool),
		SecurityGroup: d.Get("security_group").(string),
	}

	server.DynamicIPRequired = Bool(d.Get("dynamic_ip_required").(bool))
	server.CommercialType = d.Get("type").(string)

	if bootscript, ok := d.GetOk("bootscript"); ok {
		server.Bootscript = String(bootscript.(string))
	}

	if vs, ok := d.GetOk("volume"); ok {
		server.Volumes = make(map[string]string)

		volumes := vs.([]interface{})
		for i, v := range volumes {
			volume := v.(map[string]interface{})

			volumeID, err := scaleway.PostVolume(api.ScalewayVolumeDefinition{
				Size: uint64(volume["size_in_gb"].(int)) * gb,
				Type: volume["type"].(string),
				Name: fmt.Sprintf("%s-%d", server.Name, volume["size_in_gb"].(int)),
			})
			if err != nil {
				return err
			}
			volume["volume_id"] = volumeID
			volumes[i] = volume
			server.Volumes[fmt.Sprintf("%d", i+1)] = volumeID
		}
		d.Set("volume", volumes)
	}

	if raw, ok := d.GetOk("tags"); ok {
		for _, tag := range raw.([]interface{}) {
			server.Tags = append(server.Tags, tag.(string))
		}
	}

	id, err := scaleway.PostServer(server)
	if err != nil {
		return err
	}

	d.SetId(id)
	if d.Get("state").(string) != "stopped" {
		err = scaleway.PostServerAction(id, "poweron")
		if err != nil {
			return err
		}

		err = waitForServerState(scaleway, id, "running")

		if v, ok := d.GetOk("public_ip"); ok {
			if ips, err := scaleway.GetIPS(); err != nil {
				return err
			} else {
				for _, ip := range ips.IPS {
					if ip.Address == v.(string) {
						log.Printf("[DEBUG] Attaching IP %q to server %q\n", ip.ID, d.Id())
						if err := scaleway.AttachIP(ip.ID, d.Id()); err != nil {
							return err
						}
						break
					}
				}
			}
		}
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
			log.Printf("[DEBUG] Error reading server: %q\n", serr.APIMessage)

			if serr.StatusCode == 404 {
				d.SetId("")
				return nil
			}
		}

		return err
	}

	d.Set("name", server.Name)
	d.Set("image", server.Image.Identifier)
	d.Set("type", server.CommercialType)
	d.Set("enable_ipv6", server.EnableIPV6)
	d.Set("private_ip", server.PrivateIP)
	d.Set("public_ip", server.PublicAddress.IP)

	if server.EnableIPV6 {
		d.Set("public_ipv6", server.IPV6.Address)
	}

	d.Set("state", server.State)
	d.Set("state_detail", server.StateDetail)
	d.Set("tags", server.Tags)

	d.SetConnInfo(map[string]string{
		"type": "ssh",
		"host": server.PublicAddress.IP,
	})

	return nil
}

func resourceScalewayServerUpdate(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway

	mu.Lock()
	defer mu.Unlock()

	var req api.ScalewayServerPatchDefinition
	if d.HasChange("name") {
		name := d.Get("name").(string)
		req.Name = &name
	}

	if d.HasChange("tags") {
		if raw, ok := d.GetOk("tags"); ok {
			var tags []string
			for _, tag := range raw.([]interface{}) {
				tags = append(tags, tag.(string))
			}
			req.Tags = &tags
		}
	}

	if d.HasChange("enable_ipv6") {
		req.EnableIPV6 = Bool(d.Get("enable_ipv6").(bool))
	}

	if d.HasChange("dynamic_ip_required") {
		req.DynamicIPRequired = Bool(d.Get("dynamic_ip_required").(bool))
	}

	if d.HasChange("security_group") {
		req.SecurityGroup = &api.ScalewaySecurityGroup{
			Identifier: d.Get("security_group").(string),
		}
	}

	if err := scaleway.PatchServer(d.Id(), req); err != nil {
		return fmt.Errorf("Failed patching scaleway server: %q", err)
	}

	if d.HasChange("public_ip") {
		ips, err := scaleway.GetIPS()
		if err != nil {
			return err
		}
		if v, ok := d.GetOk("public_ip"); ok {
			for _, ip := range ips.IPS {
				if ip.Address == v.(string) {
					log.Printf("[DEBUG] Attaching IP %q to server %q\n", ip.ID, d.Id())
					if err := scaleway.AttachIP(ip.ID, d.Id()); err != nil {
						return err
					}
					break
				}
			}
		} else {
			for _, ip := range ips.IPS {
				if ip.Server != nil && ip.Server.Identifier == d.Id() {
					log.Printf("[DEBUG] Detaching IP %q to server %q\n", ip.ID, d.Id())
					if err := scaleway.DetachIP(ip.ID); err != nil {
						return err
					}
					break
				}
			}
		}
	}

	return resourceScalewayServerRead(d, m)
}

func resourceScalewayServerDelete(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway

	mu.Lock()
	defer mu.Unlock()

	s, err := scaleway.GetServer(d.Id())
	if err != nil {
		return err
	}

	if s.State == "stopped" {
		return deleteStoppedServer(scaleway, s)
	}

	err = deleteRunningServer(scaleway, s)

	if err == nil {
		d.SetId("")
	}

	return err
}
