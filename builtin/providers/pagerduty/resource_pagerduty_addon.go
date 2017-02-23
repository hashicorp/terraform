package pagerduty

import (
	"log"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourcePagerDutyAddon() *schema.Resource {
	return &schema.Resource{
		Create: resourcePagerDutyAddonCreate,
		Read:   resourcePagerDutyAddonRead,
		Update: resourcePagerDutyAddonUpdate,
		Delete: resourcePagerDutyAddonDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"src": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func buildAddonStruct(d *schema.ResourceData) *pagerduty.Addon {
	addon := pagerduty.Addon{
		Name: d.Get("name").(string),
		Src:  d.Get("src").(string),
		APIObject: pagerduty.APIObject{
			Type: "full_page_addon",
		},
	}

	return &addon
}

func resourcePagerDutyAddonCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	addon := buildAddonStruct(d)

	log.Printf("[INFO] Creating PagerDuty add-on %s", addon.Name)

	addon, err := client.InstallAddon(*addon)
	if err != nil {
		return err
	}

	d.SetId(addon.ID)

	return resourcePagerDutyAddonRead(d, meta)
}

func resourcePagerDutyAddonRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	log.Printf("[INFO] Reading PagerDuty add-on %s", d.Id())

	addon, err := client.GetAddon(d.Id())
	if err != nil {
		if isNotFound(err) {
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("name", addon.Name)
	d.Set("src", addon.Src)

	return nil
}

func resourcePagerDutyAddonUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	addon := buildAddonStruct(d)

	log.Printf("[INFO] Updating PagerDuty add-on %s", d.Id())

	if _, err := client.UpdateAddon(d.Id(), *addon); err != nil {
		return err
	}

	return nil
}

func resourcePagerDutyAddonDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	log.Printf("[INFO] Deleting PagerDuty add-on %s", d.Id())

	if err := client.DeleteAddon(d.Id()); err != nil {
		return err
	}

	d.SetId("")

	return nil
}
