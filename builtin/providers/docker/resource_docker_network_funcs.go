package docker

import (
	"fmt"

	dc "github.com/fsouza/go-dockerclient"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDockerNetworkCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*dc.Client)

	createOpts := dc.CreateNetworkOptions{
		Name: d.Get("name").(string),
	}
	if v, ok := d.GetOk("check_duplicate"); ok {
		createOpts.CheckDuplicate = v.(bool)
	}
	if v, ok := d.GetOk("driver"); ok {
		createOpts.Driver = v.(string)
	}
	if v, ok := d.GetOk("options"); ok {
		createOpts.Options = v.(map[string]interface{})
	}

	ipamOpts := dc.IPAMOptions{}
	ipamOptsSet := false
	if v, ok := d.GetOk("ipam_driver"); ok {
		ipamOpts.Driver = v.(string)
		ipamOptsSet = true
	}
	if v, ok := d.GetOk("ipam_config"); ok {
		ipamOpts.Config = ipamConfigSetToIpamConfigs(v.(*schema.Set))
		ipamOptsSet = true
	}

	if ipamOptsSet {
		createOpts.IPAM = ipamOpts
	}

	var err error
	var retNetwork *dc.Network
	if retNetwork, err = client.CreateNetwork(createOpts); err != nil {
		return fmt.Errorf("Unable to create network: %s", err)
	}
	if retNetwork == nil {
		return fmt.Errorf("Returned network is nil")
	}

	d.SetId(retNetwork.ID)
	d.Set("name", retNetwork.Name)
	d.Set("scope", retNetwork.Scope)
	d.Set("driver", retNetwork.Driver)
	d.Set("options", retNetwork.Options)

	return nil
}

func resourceDockerNetworkRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*dc.Client)

	var err error
	var retNetwork *dc.Network
	if retNetwork, err = client.NetworkInfo(d.Id()); err != nil {
		if _, ok := err.(*dc.NoSuchNetwork); !ok {
			return fmt.Errorf("Unable to inspect network: %s", err)
		}
	}
	if retNetwork == nil {
		d.SetId("")
		return nil
	}

	d.Set("scope", retNetwork.Scope)
	d.Set("driver", retNetwork.Driver)
	d.Set("options", retNetwork.Options)

	return nil
}

func resourceDockerNetworkDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*dc.Client)

	if err := client.RemoveNetwork(d.Id()); err != nil {
		if _, ok := err.(*dc.NoSuchNetwork); !ok {
			return fmt.Errorf("Error deleting network %s: %s", d.Id(), err)
		}
	}

	d.SetId("")
	return nil
}

func ipamConfigSetToIpamConfigs(ipamConfigSet *schema.Set) []dc.IPAMConfig {
	ipamConfigs := make([]dc.IPAMConfig, ipamConfigSet.Len())

	for i, ipamConfigInt := range ipamConfigSet.List() {
		ipamConfigRaw := ipamConfigInt.(map[string]interface{})

		ipamConfig := dc.IPAMConfig{}
		ipamConfig.Subnet = ipamConfigRaw["subnet"].(string)
		ipamConfig.IPRange = ipamConfigRaw["ip_range"].(string)
		ipamConfig.Gateway = ipamConfigRaw["gateway"].(string)

		auxAddressRaw := ipamConfigRaw["aux_address"].(map[string]interface{})
		ipamConfig.AuxAddress = make(map[string]string, len(auxAddressRaw))
		for k, v := range auxAddressRaw {
			ipamConfig.AuxAddress[k] = v.(string)
		}

		ipamConfigs[i] = ipamConfig
	}

	return ipamConfigs
}
