package scaleway

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

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
				ForceNew: true,
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
			"enable_ipv6": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"dynamic_ip_required": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"security_group": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"private_ip": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"public_ip": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"state": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"state_detail": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"volumes": &schema.Schema{
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeInt,
				},
				Required: true,
			},
		},
	}
}

func resourceScalewayServerCreate(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway

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

	arch := ""
	if arch == "" {
		server.CommercialType = strings.ToUpper(server.CommercialType)
		switch server.CommercialType[:2] {
		case "C1":
			arch = "arm"
		case "C2", "VC":
			arch = "x86_64"
		default:
			log.Printf("[ERROR] %s wrong commercial type", server.CommercialType)
			return errors.New("Wrong commercial type")
		}
	}

	if bootscript, ok := d.GetOk("bootscript"); ok {
		bootscript_id := bootscript.(string)

		bootscripts, err := scaleway.GetBootscripts()
		if err != nil {
			return err
		}

		for _, b := range *bootscripts {
			if b.Title == bootscript {
				bootscript_id = b.Identifier
			}
		}

		server.Bootscript = &bootscript_id
	}

	if raw, ok := d.GetOk("tags"); ok {
		for _, tag := range raw.([]interface{}) {
			server.Tags = append(server.Tags, tag.(string))
		}
	}

	if raw, ok := d.GetOk("volumes"); ok {
		server.Volumes = make(map[string]string)
		for i, vol := range raw.([]interface{}) {
			var volume = api.ScalewayVolumeDefinition{
				Name:         fmt.Sprintf("%s-%s", server.Name, strconv.Itoa(vol.(int))),
				Size:         uint64(vol.(int)) * gb,
				Type:         "l_ssd",
				Organization: scaleway.Organization,
			}
			vol_id, err := scaleway.PostVolume(volume)
			if err != nil {
				log.Printf("[ERROR] Got error while creating volume: %q\n", err)
				return err
			}
			server.Volumes[strconv.Itoa(i+1)] = vol_id
		}
	}

	log.Printf("creating server: %q\n", server)
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

	d.Set("private_ip", server.PrivateIP)
	d.Set("public_ip", server.PublicAddress.IP)

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

	return resourceScalewayServerRead(d, m)
}

func resourceScalewayServerDelete(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway

	def, err := scaleway.GetServer(d.Id())
	if err != nil {
		if serr, ok := err.(api.ScalewayAPIError); ok {
			if serr.StatusCode == 404 {
				d.SetId("")
				return nil
			}
		}
		return err
	}

	err = deleteServerSafe(scaleway, def.Identifier)
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
