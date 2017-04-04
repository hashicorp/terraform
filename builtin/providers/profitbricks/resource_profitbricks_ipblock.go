package profitbricks

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/profitbricks/profitbricks-sdk-go"
	"log"
	"strings"
)

func resourceProfitBricksIPBlock() *schema.Resource {
	return &schema.Resource{
		Create: resourceProfitBricksIPBlockCreate,
		Read:   resourceProfitBricksIPBlockRead,
		//Update: resourceProfitBricksIPBlockUpdate,
		Delete: resourceProfitBricksIPBlockDelete,
		Schema: map[string]*schema.Schema{
			"location": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"size": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"ips": {
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
			},
		},
	}
}

func resourceProfitBricksIPBlockCreate(d *schema.ResourceData, meta interface{}) error {
	ipblock := profitbricks.IpBlock{
		Properties: profitbricks.IpBlockProperties{
			Size:     d.Get("size").(int),
			Location: d.Get("location").(string),
		},
	}

	ipblock = profitbricks.ReserveIpBlock(ipblock)

	if ipblock.StatusCode > 299 {
		return fmt.Errorf("An error occured while reserving an ip block: %s", ipblock.Response)
	}
	err := waitTillProvisioned(meta, ipblock.Headers.Get("Location"))
	if err != nil {
		return err
	}
	d.SetId(ipblock.Id)

	return resourceProfitBricksIPBlockRead(d, meta)
}

func resourceProfitBricksIPBlockRead(d *schema.ResourceData, meta interface{}) error {
	ipblock := profitbricks.GetIpBlock(d.Id())

	if ipblock.StatusCode > 299 {
		if ipblock.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("An error occured while fetching an ip block ID %s %s", d.Id(), ipblock.Response)
	}

	log.Printf("[INFO] IPS: %s", strings.Join(ipblock.Properties.Ips, ","))

	d.Set("ips", ipblock.Properties.Ips)
	d.Set("location", ipblock.Properties.Location)
	d.Set("size", ipblock.Properties.Size)

	return nil
}

func resourceProfitBricksIPBlockDelete(d *schema.ResourceData, meta interface{}) error {
	resp := profitbricks.ReleaseIpBlock(d.Id())
	if resp.StatusCode > 299 {
		return fmt.Errorf("An error occured while releasing an ipblock ID: %s %s", d.Id(), string(resp.Body))
	}

	err := waitTillProvisioned(meta, resp.Headers.Get("Location"))
	if err != nil {
		return err
	}
	d.SetId("")
	return nil
}
