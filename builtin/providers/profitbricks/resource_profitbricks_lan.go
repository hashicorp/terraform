package profitbricks

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/profitbricks/profitbricks-sdk-go"
	"log"
	"time"
)

func resourceProfitBricksLan() *schema.Resource {
	return &schema.Resource{
		Create: resourceProfitBricksLanCreate,
		Read:   resourceProfitBricksLanRead,
		Update: resourceProfitBricksLanUpdate,
		Delete: resourceProfitBricksLanDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{

			"public": {
				Type:     schema.TypeBool,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"datacenter_id": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceProfitBricksLanCreate(d *schema.ResourceData, meta interface{}) error {
	username, password, _ := getCredentials(meta)
	profitbricks.SetAuth(username, password)
	profitbricks.SetDepth("5")
	request := profitbricks.Lan{
		Properties: profitbricks.LanProperties{
			Public: d.Get("public").(bool),
		},
	}

	log.Printf("[DEBUG] NAME %s", d.Get("name"))
	if d.Get("name") != nil {
		request.Properties.Name = d.Get("name").(string)
	}

	lan := profitbricks.CreateLan(d.Get("datacenter_id").(string), request)

	log.Printf("[DEBUG] LAN ID: %s", lan.Id)
	log.Printf("[DEBUG] LAN RESPONSE: %s", lan.Response)

	if lan.StatusCode > 299 {
		return fmt.Errorf("An error occured while creating a lan: %s", lan.Response)
	}

	err := waitTillProvisioned(meta, lan.Headers.Get("Location"))
	if err != nil {
		return err
	}
	d.SetId(lan.Id)
	return resourceProfitBricksLanRead(d, meta)
}

func resourceProfitBricksLanRead(d *schema.ResourceData, meta interface{}) error {
	username, password, _ := getCredentials(meta)
	profitbricks.SetAuth(username, password)

	lan := profitbricks.GetLan(d.Get("datacenter_id").(string), d.Id())

	if lan.StatusCode > 299 {
		return fmt.Errorf("An error occured while fetching a lan ID %s %s", d.Id(), lan.Response)
	}

	d.Set("public", lan.Properties.Public)
	d.Set("name", lan.Properties.Name)
	d.Set("datacenter_id", d.Get("datacenter_id").(string))
	return nil
}

func resourceProfitBricksLanUpdate(d *schema.ResourceData, meta interface{}) error {
	username, password, _ := getCredentials(meta)
	profitbricks.SetAuth(username, password)
	properties := &profitbricks.LanProperties{}
	if d.HasChange("public") {
		_, newValue := d.GetChange("public")
		properties.Public = newValue.(bool)
	}
	if d.HasChange("name") {
		_, newValue := d.GetChange("name")
		properties.Name = newValue.(string)
	}
	log.Printf("[DEBUG] LAN UPDATE: %s : %s", properties, d.Get("name"))
	if properties != nil {
		lan := profitbricks.PatchLan(d.Get("datacenter_id").(string), d.Id(), *properties)
		if lan.StatusCode > 299 {
			return fmt.Errorf("An error occured while patching a lan ID %s %s", d.Id(), lan.Response)
		}
		err := waitTillProvisioned(meta, lan.Headers.Get("Location"))
		if err != nil {
			return err
		}
	}
	return resourceProfitBricksLanRead(d, meta)
}

func resourceProfitBricksLanDelete(d *schema.ResourceData, meta interface{}) error {
	username, password, _ := getCredentials(meta)
	profitbricks.SetAuth(username, password)

	resp := profitbricks.DeleteLan(d.Get("datacenter_id").(string), d.Id())
	if resp.StatusCode > 299 {
		//try again in 20 seconds
		time.Sleep(60 * time.Second)
		resp = profitbricks.DeleteLan(d.Get("datacenter_id").(string), d.Id())
		if resp.StatusCode > 299 && resp.StatusCode != 404 {
			return fmt.Errorf("An error occured while deleting a lan dcId %s ID %s %s", d.Get("datacenter_id").(string), d.Id(), string(resp.Body))
		}
	}

	if resp.Headers.Get("Location") != "" {
		err := waitTillProvisioned(meta, resp.Headers.Get("Location"))
		if err != nil {
			return err
		}
	}
	d.SetId("")
	return nil
}
