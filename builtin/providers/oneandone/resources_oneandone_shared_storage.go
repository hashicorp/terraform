package oneandone

import (
	"fmt"
	"github.com/1and1/oneandone-cloudserver-sdk-go"
	"github.com/hashicorp/terraform/helper/schema"
	"strings"
)

func resourceOneandOneSharedStorage() *schema.Resource {
	return &schema.Resource{
		Create: resourceOneandOneSharedStorageCreate,
		Read:   resourceOneandOneSharedStorageRead,
		Update: resourceOneandOneSharedStorageUpdate,
		Delete: resourceOneandOneSharedStorageDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"size": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"datacenter": {
				Type:     schema.TypeString,
				Required: true,
			},
			"storage_servers": {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Required: true,
						},
						"rights": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Optional: true,
			},
		},
	}
}

func resourceOneandOneSharedStorageCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	req := oneandone.SharedStorageRequest{
		Name: d.Get("name").(string),
		Size: oneandone.Int2Pointer(d.Get("size").(int)),
	}

	if raw, ok := d.GetOk("description"); ok {
		req.Description = raw.(string)

	}

	if raw, ok := d.GetOk("datacenter"); ok {
		dcs, err := config.API.ListDatacenters()

		if err != nil {
			return fmt.Errorf("An error occured while fetching list of datacenters %s", err)

		}

		decenter := raw.(string)
		for _, dc := range dcs {
			if strings.ToLower(dc.CountryCode) == strings.ToLower(decenter) {
				req.DatacenterId = dc.Id
				break
			}
		}
	}

	ss_id, ss, err := config.API.CreateSharedStorage(&req)
	if err != nil {
		return err
	}

	err = config.API.WaitForState(ss, "ACTIVE", 10, config.Retries)
	if err != nil {
		return err
	}
	d.SetId(ss_id)

	if raw, ok := d.GetOk("storage_servers"); ok {

		storage_servers := []oneandone.SharedStorageServer{}

		rawRights := raw.([]interface{})
		for _, raws_ss := range rawRights {
			ss := raws_ss.(map[string]interface{})
			storage_server := oneandone.SharedStorageServer{
				Id:     ss["id"].(string),
				Rights: ss["rights"].(string),
			}
			storage_servers = append(storage_servers, storage_server)
		}

		ss, err := config.API.AddSharedStorageServers(ss_id, storage_servers)

		if err != nil {
			return err
		}

		err = config.API.WaitForState(ss, "ACTIVE", 10, 30)
		if err != nil {
			return err
		}
	}

	return resourceOneandOneSharedStorageRead(d, meta)
}

func resourceOneandOneSharedStorageUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	if d.HasChange("name") || d.HasChange("description") || d.HasChange("size") {
		ssu := oneandone.SharedStorageRequest{}
		if d.HasChange("name") {
			_, n := d.GetChange("name")
			ssu.Name = n.(string)
		}
		if d.HasChange("description") {
			_, n := d.GetChange("description")
			ssu.Description = n.(string)
		}
		if d.HasChange("size") {
			_, n := d.GetChange("size")
			ssu.Size = oneandone.Int2Pointer(n.(int))
		}

		ss, err := config.API.UpdateSharedStorage(d.Id(), &ssu)

		if err != nil {
			return err
		}
		err = config.API.WaitForState(ss, "ACTIVE", 10, 30)
		if err != nil {
			return err
		}

	}

	if d.HasChange("storage_servers") {

		o, n := d.GetChange("storage_servers")

		oldV := o.([]interface{})

		for _, old := range oldV {
			ol := old.(map[string]interface{})

			ss, err := config.API.DeleteSharedStorageServer(d.Id(), ol["id"].(string))
			if err != nil {
				return err
			}

			err = config.API.WaitForState(ss, "ACTIVE", 10, config.Retries)

			if err != nil {
				return err
			}

		}

		newV := n.([]interface{})

		ids := []oneandone.SharedStorageServer{}
		for _, newValue := range newV {
			nn := newValue.(map[string]interface{})
			ids = append(ids, oneandone.SharedStorageServer{
				Id:     nn["id"].(string),
				Rights: nn["rights"].(string),
			})
		}

		if len(ids) > 0 {
			ss, err := config.API.AddSharedStorageServers(d.Id(), ids)
			if err != nil {
				return err
			}

			err = config.API.WaitForState(ss, "ACTIVE", 10, config.Retries)

			if err != nil {
				return err
			}
		}

		//DeleteSharedStorageServer

	}

	return resourceOneandOneSharedStorageRead(d, meta)
}

func resourceOneandOneSharedStorageRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	ss, err := config.API.GetSharedStorage(d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("name", ss.Name)
	d.Set("description", ss.Description)
	d.Set("size", ss.Size)
	d.Set("datacenter", ss.Datacenter.CountryCode)
	d.Set("storage_servers", getStorageServers(ss.Servers))

	return nil
}

func getStorageServers(servers []oneandone.SharedStorageServer) []map[string]interface{} {
	raw := make([]map[string]interface{}, 0, len(servers))

	for _, server := range servers {

		toadd := map[string]interface{}{
			"id":     server.Id,
			"rights": server.Rights,
		}

		raw = append(raw, toadd)
	}

	return raw

}
func resourceOneandOneSharedStorageDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	ss, err := config.API.DeleteSharedStorage(d.Id())
	if err != nil {
		return err
	}
	err = config.API.WaitUntilDeleted(ss)
	if err != nil {
		return err
	}

	return nil
}
