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

			"format": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
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

			"url": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"zone": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"is_dynamically_scalable": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"is_extractable": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"is_featured": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"is_public": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"password_enabled": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"is_ready": &schema.Schema{
				Type:     schema.TypeBool,
				Computed: true,
			},

			"is_ready_timeout": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  300,
			},
		},
	}
}

func resourceCloudStackTemplateCreate(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	if err := verifyTemplateParams(d); err != nil {
		return err
	}

	name := d.Get("name").(string)

	// Compute/set the display text
	displaytext := d.Get("display_text").(string)
	if displaytext == "" {
		displaytext = name
	}

	// Retrieve the os_type ID
	ostypeid, e := retrieveID(cs, "os_type", d.Get("os_type").(string))
	if e != nil {
		return e.Error()
	}

	// Retrieve the zone ID
	zoneid, e := retrieveID(cs, "zone", d.Get("zone").(string))
	if e != nil {
		return e.Error()
	}

	// Create a new parameter struct
	p := cs.Template.NewRegisterTemplateParams(
		displaytext,
		d.Get("format").(string),
		d.Get("hypervisor").(string),
		name,
		ostypeid,
		d.Get("url").(string),
		zoneid)

	// Set optional parameters
	if v, ok := d.GetOk("is_dynamically_scalable"); ok {
		p.SetIsdynamicallyscalable(v.(bool))
	}

	if v, ok := d.GetOk("is_extractable"); ok {
		p.SetIsextractable(v.(bool))
	}

	if v, ok := d.GetOk("is_featured"); ok {
		p.SetIsfeatured(v.(bool))
	}

	if v, ok := d.GetOk("is_public"); ok {
		p.SetIspublic(v.(bool))
	}

	if v, ok := d.GetOk("password_enabled"); ok {
		p.SetPasswordenabled(v.(bool))
	}

	// If there is a project supplied, we retrieve and set the project id
	if err := setProjectid(p, cs, d); err != nil {
		return err
	}

	// Create the new template
	r, err := cs.Template.RegisterTemplate(p)
	if err != nil {
		return fmt.Errorf("Error creating template %s: %s", name, err)
	}

	d.SetId(r.RegisterTemplate[0].Id)

	// Wait until the template is ready to use, or timeout with an error...
	currentTime := time.Now().Unix()
	timeout := int64(d.Get("is_ready_timeout").(int))
	for {
		// Start with the sleep so the register action has a few seconds
		// to process the registration correctly. Without this wait
		time.Sleep(10 * time.Second)

		err := resourceCloudStackTemplateRead(d, meta)
		if err != nil {
			return err
		}

		if d.Get("is_ready").(bool) {
			return nil
		}

		if time.Now().Unix()-currentTime > timeout {
			return fmt.Errorf("Timeout while waiting for template to become ready")
		}
	}
}

func resourceCloudStackTemplateRead(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Get the template details
	t, count, err := cs.Template.GetTemplateByID(
		d.Id(),
		"executable",
		cloudstack.WithProject(d.Get("project").(string)),
	)
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
	d.Set("format", t.Format)
	d.Set("hypervisor", t.Hypervisor)
	d.Set("is_dynamically_scalable", t.Isdynamicallyscalable)
	d.Set("is_extractable", t.Isextractable)
	d.Set("is_featured", t.Isfeatured)
	d.Set("is_public", t.Ispublic)
	d.Set("password_enabled", t.Passwordenabled)
	d.Set("is_ready", t.Isready)

	setValueOrID(d, "os_type", t.Ostypename, t.Ostypeid)
	setValueOrID(d, "project", t.Project, t.Projectid)
	setValueOrID(d, "zone", t.Zonename, t.Zoneid)

	return nil
}

func resourceCloudStackTemplateUpdate(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)
	name := d.Get("name").(string)

	// Create a new parameter struct
	p := cs.Template.NewUpdateTemplateParams(d.Id())

	if d.HasChange("name") {
		p.SetName(name)
	}

	if d.HasChange("display_text") {
		p.SetDisplaytext(d.Get("display_text").(string))
	}

	if d.HasChange("format") {
		p.SetFormat(d.Get("format").(string))
	}

	if d.HasChange("is_dynamically_scalable") {
		p.SetIsdynamicallyscalable(d.Get("is_dynamically_scalable").(bool))
	}

	if d.HasChange("os_type") {
		ostypeid, e := retrieveID(cs, "os_type", d.Get("os_type").(string))
		if e != nil {
			return e.Error()
		}
		p.SetOstypeid(ostypeid)
	}

	if d.HasChange("password_enabled") {
		p.SetPasswordenabled(d.Get("password_enabled").(bool))
	}

	_, err := cs.Template.UpdateTemplate(p)
	if err != nil {
		return fmt.Errorf("Error updating template %s: %s", name, err)
	}

	return resourceCloudStackTemplateRead(d, meta)
}

func resourceCloudStackTemplateDelete(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Create a new parameter struct
	p := cs.Template.NewDeleteTemplateParams(d.Id())

	// Delete the template
	log.Printf("[INFO] Deleting template: %s", d.Get("name").(string))
	_, err := cs.Template.DeleteTemplate(p)
	if err != nil {
		// This is a very poor way to be told the ID does no longer exist :(
		if strings.Contains(err.Error(), fmt.Sprintf(
			"Invalid parameter id value=%s due to incorrect long value format, "+
				"or entity does not exist", d.Id())) {
			return nil
		}

		return fmt.Errorf("Error deleting template %s: %s", d.Get("name").(string), err)
	}
	return nil
}

func verifyTemplateParams(d *schema.ResourceData) error {
	format := d.Get("format").(string)
	if format != "OVA" && format != "QCOW2" && format != "RAW" && format != "VHD" && format != "VMDK" {
		return fmt.Errorf(
			"%s is not a valid format. Valid options are 'OVA','QCOW2', 'RAW', 'VHD' and 'VMDK'", format)
	}

	return nil
}
