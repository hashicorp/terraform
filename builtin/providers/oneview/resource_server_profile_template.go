// (C) Copyright 2016 Hewlett Packard Enterprise Development LP
//
// Licensed under the Apache License, Version 2.0 (the "License");
// You may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software distributed
// under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
// CONDITIONS OF ANY KIND, either express or implied. See the License for the
// specific language governing permissions and limitations under the License.

package oneview

import (
	"fmt"
	"github.com/HewlettPackard/oneview-golang/ov"
	"github.com/HewlettPackard/oneview-golang/utils"
	"github.com/hashicorp/terraform/helper/schema"
	"strconv"
)

func resourceServerProfileTemplate() *schema.Resource {
	return &schema.Resource{
		Create: resourceServerProfileTemplateCreate,
		Read:   resourceServerProfileTemplateRead,
		Update: resourceServerProfileTemplateUpdate,
		Delete: resourceServerProfileTemplateDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"boot_order": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"network": {
				Optional: true,
				Type:     schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"function_type": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"network_uri": {
							Type:     schema.TypeString,
							Required: true,
						},
						"port_id": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "Lom 1:1-a",
						},
						"requested_mbps": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "2500",
						},
						"id": {
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
				},
			},
			"type": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "ServerProfileTemplateV1",
			},
			"server_hardware_type": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"enclosure_group": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"affinity": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Bay",
			},
			"hide_unused_flex_nics": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"uri": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"etag": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"serial_number_type": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Virtual",
				ForceNew: true,
			},
			"wwn_type": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Virtual",
				ForceNew: true,
			},
			"mac_type": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Virtual",
				ForceNew: true,
			},
		},
	}
}

func resourceServerProfileTemplateCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	serverProfileTemplate := ov.ServerProfile{
		Name:               d.Get("name").(string),
		Type:               d.Get("type").(string),
		Affinity:           d.Get("affinity").(string),
		SerialNumberType:   d.Get("serial_number_type").(string),
		WWNType:            d.Get("wwn_type").(string),
		MACType:            d.Get("mac_type").(string),
		HideUnusedFlexNics: d.Get("hide_unused_flex_nics").(bool),
	}

	enclosureGroup, err := config.ovClient.GetEnclosureGroupByName(d.Get("enclosure_group").(string))
	if err != nil {
		return err
	}
	serverProfileTemplate.EnclosureGroupURI = enclosureGroup.URI

	serverHardwareType, err := config.ovClient.GetServerHardwareTypeByName(d.Get("server_hardware_type").(string))
	if err != nil {
		return err
	}
	serverProfileTemplate.ServerHardwareTypeURI = serverHardwareType.URI

	networkCount := d.Get("network.#").(int)
	networks := make([]ov.Connection, 0)
	for i := 0; i < networkCount; i++ {
		networkPrefix := fmt.Sprintf("network.%d", i)
		networks = append(networks, ov.Connection{
			Name:          d.Get(networkPrefix + ".name").(string),
			FunctionType:  d.Get(networkPrefix + ".function_type").(string),
			NetworkURI:    utils.NewNstring(d.Get(networkPrefix + ".network_uri").(string)),
			PortID:        d.Get(networkPrefix + ".port_id").(string),
			RequestedMbps: d.Get(networkPrefix + ".requested_mbps").(string),
			ID:            i + 1,
		})
	}
	serverProfileTemplate.Connections = networks

	if val, ok := d.GetOk("boot_order"); ok {
		rawBootOrder := val.(*schema.Set).List()
		bootOrder := make([]string, len(rawBootOrder))
		for i, raw := range rawBootOrder {
			bootOrder[i] = raw.(string)
		}
		serverProfileTemplate.Boot.ManageBoot = true
		serverProfileTemplate.Boot.Order = bootOrder
	}

	sptError := config.ovClient.CreateProfileTemplate(serverProfileTemplate)
	d.SetId(d.Get("name").(string))
	if sptError != nil {
		d.SetId("")
		return sptError
	}
	return resourceServerProfileTemplateRead(d, meta)
}

func resourceServerProfileTemplateRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	spt, err := config.ovClient.GetProfileTemplateByName(d.Id())
	if err != nil || spt.URI.IsNil() {
		d.SetId("")
		return nil
	}

	d.Set("name", spt.Name)
	d.Set("type", spt.Type)

	enclosureGroup, err := config.ovClient.GetEnclosureGroupByUri(spt.EnclosureGroupURI)
	if err != nil {
		return err
	}
	d.Set("enclosure_group_uri", enclosureGroup.Name)

	serverHardwareType, err := config.ovClient.GetServerHardwareTypeByUri(spt.ServerHardwareTypeURI)
	if err != nil {
		return err
	}
	d.Set("server_hardware_type_uri", serverHardwareType.Name)
	d.Set("affinity", spt.Affinity)
	d.Set("uri", spt.URI.String())
	d.Set("etag", spt.ETAG)
	d.Set("serial_number_type", spt.SerialNumberType)
	d.Set("wwn_type", spt.WWNType)
	d.Set("mac_type", spt.MACType)
	d.Set("hide_unused_flex_nics", spt.HideUnusedFlexNics)

	networks := make([]map[string]interface{}, 0, len(spt.Connections))
	for _, network := range spt.Connections {

		networks = append(networks, map[string]interface{}{
			"name":           network.Name,
			"function_type":  network.FunctionType,
			"network_uri":    network.NetworkURI,
			"port_id":        network.PortID,
			"requested_mbps": network.RequestedMbps,
			"id":             network.ID,
		})
	}

	networkCount := d.Get("network.#").(int)
	if networkCount > 0 {
		for i := 0; i < networkCount; i++ {
			currNetworkId := d.Get("network." + strconv.Itoa(i) + ".id")
			for j := 0; j < len(spt.Connections); j++ {
				if spt.Connections[j].ID == currNetworkId && i <= len(spt.Connections)-1 {
					networks[i], networks[j] = networks[j], networks[i]
				}
			}
		}
		d.Set("network", networks)
	}

	if spt.Boot.ManageBoot {
		bootOrder := make([]interface{}, 0)
		for _, currBoot := range spt.Boot.Order {
			rawBootOrder := d.Get("boot_order").(*schema.Set).List()
			for _, raw := range rawBootOrder {
				if raw == currBoot {
					bootOrder = append(bootOrder, currBoot)
				}
			}
		}
		d.Set("boot_order", bootOrder)
	}
	return nil
}

func resourceServerProfileTemplateUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	serverProfileTemplate := ov.ServerProfile{
		Name:               d.Get("name").(string),
		Type:               d.Get("type").(string),
		Affinity:           d.Get("affinity").(string),
		URI:                utils.NewNstring(d.Get("uri").(string)),
		ETAG:               d.Get("etag").(string),
		SerialNumberType:   d.Get("serial_number_type").(string),
		WWNType:            d.Get("wwn_type").(string),
		MACType:            d.Get("mac_type").(string),
		HideUnusedFlexNics: d.Get("hide_unused_flex_nics").(bool),
	}

	enclosureGroup, err := config.ovClient.GetEnclosureGroupByName(d.Get("enclosure_group").(string))
	if err != nil {
		return err
	}
	serverProfileTemplate.EnclosureGroupURI = enclosureGroup.URI

	serverHardwareType, err := config.ovClient.GetServerHardwareTypeByName(d.Get("server_hardware_type").(string))
	if err != nil {
		return err
	}
	serverProfileTemplate.ServerHardwareTypeURI = serverHardwareType.URI

	networkCount := d.Get("network.#").(int)
	networks := make([]ov.Connection, 0)
	for i := 0; i < networkCount; i++ {
		networkPrefix := fmt.Sprintf("network.%d", i)
		networks = append(networks, ov.Connection{
			Name:          d.Get(networkPrefix + ".name").(string),
			FunctionType:  d.Get(networkPrefix + ".function_type").(string),
			NetworkURI:    utils.NewNstring(d.Get(networkPrefix + ".network_uri").(string)),
			PortID:        d.Get(networkPrefix + ".port_id").(string),
			RequestedMbps: d.Get(networkPrefix + ".requested_mbps").(string),
			ID:            d.Get(networkPrefix + ".id").(int),
		})
	}
	serverProfileTemplate.Connections = networks

	if val, ok := d.GetOk("boot_order"); ok {
		rawBootOrder := val.(*schema.Set).List()
		bootOrder := make([]string, len(rawBootOrder))
		for i, raw := range rawBootOrder {
			bootOrder[i] = raw.(string)
		}
		serverProfileTemplate.Boot.ManageBoot = true
		serverProfileTemplate.Boot.Order = bootOrder
	}

	err = config.ovClient.UpdateProfileTemplate(serverProfileTemplate)
	if err != nil {
		return err
	}
	d.SetId(d.Get("name").(string))

	return resourceServerProfileTemplateRead(d, meta)
}

func resourceServerProfileTemplateDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	error := config.ovClient.DeleteProfileTemplate(d.Get("name").(string))
	if error != nil {
		return error
	}
	return nil
}
