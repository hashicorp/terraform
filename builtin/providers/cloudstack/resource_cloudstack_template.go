package cloudstack

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

func resourceCloudStackTemplate() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudStackTemplateCreate,
		Read:   resourceCloudStackTemplateRead,
		Update: resourceCloudStackTemplateUpdate,
		Delete: resourceCloudStackTemplateDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"display_text": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"url": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"hypervisor": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"os_type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"format": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"zone": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"requires_hvm": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},

			"is_featured": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},

			"password_enabled": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},

			"template_tag": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"ssh_key_enabled": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},

			"is_routing": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},

			"is_public": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},

			"is_extractable": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},

			"bits": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},

			"is_dynamically_scalable": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},

			"checksum": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"is_ready": &schema.Schema{
				Type:     schema.TypeBool,
				Computed: true,
			},
		},
	}
}

func resourceCloudStackTemplateCreate(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	//Retrieving required parameters
	format := d.Get("format").(string)

	hypervisor := d.Get("hypervisor").(string)

	name := d.Get("name").(string)

	// Retrieve the os_type UUID
	ostypeid, e := retrieveUUID(cs, "os_type", d.Get("os_type").(string))
	if e != nil {
		return e.Error()
	}

	url := d.Get("url").(string)

	//Retrieve the zone UUID
	zoneid, e := retrieveUUID(cs, "zone", d.Get("zone").(string))
	if e != nil {
		return e.Error()
	}

	// Compute/set the display text
	displaytext, ok := d.GetOk("display_text")
	if !ok {
		displaytext = name
	}

	// Create a new parameter struct
	p := cs.Template.NewRegisterTemplateParams(displaytext.(string), format, hypervisor, name, ostypeid, url, zoneid)
	//Set optional parameters
	p.SetPasswordenabled(d.Get("password_enabled").(bool))
	p.SetSshkeyenabled(d.Get("ssh_key_enabled").(bool))
	p.SetIsdynamicallyscalable(d.Get("is_dynamically_scalable").(bool))
	p.SetRequireshvm(d.Get("requires_hvm").(bool))
	p.SetIsfeatured(d.Get("is_featured").(bool))
	ttag := d.Get("template_tag").(string)
	if ttag != "" {
		//error if we give this a value as non-root
		p.SetTemplatetag(ttag)
	}
	ir := d.Get("is_routing").(bool)
	if ir == true {
		p.SetIsrouting(ir)
	}
	p.SetIspublic(d.Get("is_public").(bool))
	p.SetIsextractable(d.Get("is_extractable").(bool))
	p.SetBits(d.Get("bits").(int))
	p.SetChecksum(d.Get("checksum").(string))
	//TODO: set project ref / details

	// Create the new template
	r, err := cs.Template.RegisterTemplate(p)
	if err != nil {
		return fmt.Errorf("Error creating template %s: %s", name, err)
	}
	log.Printf("[DEBUG] Register template response: %+v\n", r)
	d.SetId(r.RegisterTemplate[0].Id)

	//dont return until the template is ready to use
	result := resourceCloudStackTemplateRead(d, meta)

	for !d.Get("is_ready").(bool) {
		time.Sleep(5 * time.Second)
		result = resourceCloudStackTemplateRead(d, meta)
	}
	return result
}

func resourceCloudStackTemplateRead(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)
	log.Printf("[DEBUG] looking for template %s", d.Id())
	// Get the template details
	t, count, err := cs.Template.GetTemplateByID(d.Id(), "all")
	if err != nil {
		if count == 0 {
			log.Printf(
				"[DEBUG] Template %s no longer exists", d.Get("name").(string))
			d.SetId("")
			return nil
		}

		return err
	}
	d.Set("name", t.Name)
	d.Set("display_text", t.Displaytext)
	d.Set("zone", t.Zonename)
	d.Set("hypervisor", t.Hypervisor)
	d.Set("os_type", t.Ostypename)
	d.Set("format", t.Format)
	d.Set("zone", t.Zonename)
	d.Set("is_featured", t.Isfeatured)
	d.Set("password_enabled", t.Passwordenabled)
	d.Set("template_tag", t.Templatetag)
	d.Set("ssh_key_enabled", t.Sshkeyenabled)
	d.Set("is_public", t.Ispublic)
	d.Set("is_extractable", t.Isextractable)
	d.Set("is_dynamically_scalable", t.Isdynamicallyscalable)
	d.Set("checksum", t.Checksum)
	d.Set("is_ready", t.Isready)
	log.Printf("[DEBUG] Read template values: %+v\n", d)
	return nil
}

func resourceCloudStackTemplateUpdate(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)
	d.Partial(true)
	name := d.Get("name").(string)

	sim_attrs := []string{"name", "display_text", "bootable", "is_dynamically_scalable",
		"is_routing"}
	for _, attr := range sim_attrs {
		if d.HasChange(attr) {
			log.Printf("[DEBUG] %s changed for %s, starting update", attr, name)
			p := cs.Template.NewUpdateTemplateParams(d.Id())
			switch attr {
			case "name":
				p.SetName(name)
			case "display_text":
				p.SetDisplaytext(d.Get("display_name").(string))
			case "bootable":
				p.SetBootable(d.Get("bootable").(bool))
			case "is_dynamically_scalable":
				p.SetIsdynamicallyscalable(d.Get("is_dynamically_scalable").(bool))
			case "is_routing":
				p.SetIsrouting(d.Get("is_routing").(bool))
			default:
				return fmt.Errorf("Unhandleable updateable attribute was declared, fix the code here.")
			}
			_, err := cs.Template.UpdateTemplate(p)
			if err != nil {
				return fmt.Errorf("Error updating the %s for instance %s: %s", attr, name, err)
			}
			d.SetPartial(attr)
		}
	}
	if d.HasChange("os_type") {
		log.Printf("[DEBUG] OS type changed for %s, starting update", name)
		p := cs.Template.NewUpdateTemplateParams(d.Id())
		ostypeid, e := retrieveUUID(cs, "os_type", d.Get("os_type").(string))
		if e != nil {
			return e.Error()
		}
		p.SetOstypeid(ostypeid)
		_, err := cs.Template.UpdateTemplate(p)
		if err != nil {
			return fmt.Errorf("Error updating the OS type for instance %s: %s", name, err)
		}
		d.SetPartial("os_type")

	}
	d.Partial(false)
	return resourceCloudStackTemplateRead(d, meta)
}

func resourceCloudStackTemplateDelete(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Create a new parameter struct
	p := cs.Template.NewDeleteTemplateParams(d.Id())

	// Delete the template
	log.Printf("[INFO] Destroying instance: %s", d.Get("name").(string))
	_, err := cs.Template.DeleteTemplate(p)
	if err != nil {
		// This is a very poor way to be told the UUID does no longer exist :(
		if strings.Contains(err.Error(), fmt.Sprintf(
			"Invalid parameter id value=%s due to incorrect long value format, "+
				"or entity does not exist", d.Id())) {
			return nil
		}

		return fmt.Errorf("Error deleting template %s: %s", d.Get("name").(string), err)
	}
	return nil
}
