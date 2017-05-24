package oneandone

import (
	"fmt"
	"github.com/1and1/oneandone-cloudserver-sdk-go"
	"github.com/hashicorp/terraform/helper/schema"
	"strings"
)

func resourceOneandOnePrivateNetwork() *schema.Resource {
	return &schema.Resource{

		Create: resourceOneandOnePrivateNetworkCreate,
		Read:   resourceOneandOnePrivateNetworkRead,
		Update: resourceOneandOnePrivateNetworkUpdate,
		Delete: resourceOneandOnePrivateNetworkDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"datacenter": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"network_address": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"subnet_mask": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"server_ids": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},
		},
	}
}

func resourceOneandOnePrivateNetworkCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	req := oneandone.PrivateNetworkRequest{
		Name: d.Get("name").(string),
	}

	if raw, ok := d.GetOk("description"); ok {
		req.Description = raw.(string)
	}

	if raw, ok := d.GetOk("network_address"); ok {
		req.NetworkAddress = raw.(string)
	}

	if raw, ok := d.GetOk("subnet_mask"); ok {
		req.SubnetMask = raw.(string)
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

	prn_id, prn, err := config.API.CreatePrivateNetwork(&req)
	if err != nil {
		return err
	}
	err = config.API.WaitForState(prn, "ACTIVE", 30, config.Retries)

	if err != nil {
		return err
	}

	d.SetId(prn_id)

	var ids []string
	if raw, ok := d.GetOk("server_ids"); ok {

		rawIps := raw.(*schema.Set).List()

		for _, raw := range rawIps {
			ids = append(ids, raw.(string))
			server, err := config.API.ShutdownServer(raw.(string), false)
			if err != nil {
				return err
			}
			err = config.API.WaitForState(server, "POWERED_OFF", 10, config.Retries)
			if err != nil {
				return err
			}

		}
	}

	prn, err = config.API.AttachPrivateNetworkServers(d.Id(), ids)
	if err != nil {
		return err
	}

	err = config.API.WaitForState(prn, "ACTIVE", 30, config.Retries)
	if err != nil {
		return err
	}

	for _, id := range ids {
		server, err := config.API.StartServer(id)
		if err != nil {
			return err
		}

		err = config.API.WaitForState(server, "POWERED_ON", 10, config.Retries)
		if err != nil {
			return err
		}
	}

	return resourceOneandOnePrivateNetworkRead(d, meta)
}

func resourceOneandOnePrivateNetworkUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	if d.HasChange("name") || d.HasChange("description") || d.HasChange("network_address") || d.HasChange("subnet_mask") {
		pnset := oneandone.PrivateNetworkRequest{}

		pnset.Name = d.Get("name").(string)

		pnset.Description = d.Get("description").(string)
		pnset.NetworkAddress = d.Get("network_address").(string)
		pnset.SubnetMask = d.Get("subnet_mask").(string)

		prn, err := config.API.UpdatePrivateNetwork(d.Id(), &pnset)

		if err != nil {
			return err
		}

		err = config.API.WaitForState(prn, "ACTIVE", 30, config.Retries)
		if err != nil {
			return err
		}
	}

	if d.HasChange("server_ids") {
		o, n := d.GetChange("server_ids")

		newValues := n.(*schema.Set).List()
		oldValues := o.(*schema.Set).List()

		var ids []string
		for _, newV := range oldValues {
			ids = append(ids, newV.(string))
		}
		for _, id := range ids {
			server, err := config.API.ShutdownServer(id, false)
			if err != nil {
				return err
			}
			err = config.API.WaitForState(server, "POWERED_OFF", 10, config.Retries)
			if err != nil {
				return err
			}

			_, err = config.API.RemoveServerPrivateNetwork(id, d.Id())
			if err != nil {
				return err
			}

			prn, _ := config.API.GetPrivateNetwork(d.Id())

			err = config.API.WaitForState(prn, "ACTIVE", 10, config.Retries)
			if err != nil {
				return err
			}

		}

		var newids []string

		for _, newV := range newValues {
			newids = append(newids, newV.(string))
		}
		pn, err := config.API.AttachPrivateNetworkServers(d.Id(), newids)

		if err != nil {
			return err
		}
		err = config.API.WaitForState(pn, "ACTIVE", 30, config.Retries)
		if err != nil {
			return err
		}

		for _, id := range newids {
			server, err := config.API.StartServer(id)
			if err != nil {
				return err
			}

			err = config.API.WaitForState(server, "POWERED_ON", 10, config.Retries)
			if err != nil {
				return err
			}
		}
	}

	return resourceOneandOnePrivateNetworkRead(d, meta)
}

func resourceOneandOnePrivateNetworkRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	pn, err := config.API.GetPrivateNetwork(d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("name", pn.Name)
	d.Set("description", pn.Description)
	d.Set("network_address", pn.NetworkAddress)
	d.Set("subnet_mask", pn.SubnetMask)
	d.Set("datacenter", pn.Datacenter.CountryCode)

	var toAdd []string
	for _, s := range pn.Servers {
		toAdd = append(toAdd, s.Id)
	}
	d.Set("server_ids", toAdd)
	return nil
}

func resourceOneandOnePrivateNetworkDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	pn, err := config.API.GetPrivateNetwork(d.Id())

	for _, server := range pn.Servers {
		srv, err := config.API.ShutdownServer(server.Id, false)
		if err != nil {
			return err
		}
		err = config.API.WaitForState(srv, "POWERED_OFF", 10, config.Retries)
		if err != nil {
			return err
		}
	}

	pn, err = config.API.DeletePrivateNetwork(d.Id())
	if err != nil {
		return err
	}

	err = config.API.WaitUntilDeleted(pn)
	if err != nil {
		return err
	}

	for _, server := range pn.Servers {
		srv, err := config.API.StartServer(server.Id)
		if err != nil {
			return err
		}
		err = config.API.WaitForState(srv, "POWERED_ON", 10, config.Retries)
		if err != nil {
			return err
		}
	}

	return nil
}
