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
	"github.com/HewlettPackard/oneview-golang/ov"
	"github.com/HewlettPackard/oneview-golang/utils"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceNetworkSet() *schema.Resource {
	return &schema.Resource{
		Create: resourceNetworkSetCreate,
		Read:   resourceNetworkSetRead,
		Update: resourceNetworkSetUpdate,
		Delete: resourceNetworkSetDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"network_uris": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"native_network_uri": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"type": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "network-set",
			},
			"connection_template_uri": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"created": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"modified": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"uri": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"category": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"state": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"eTag": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceNetworkSetCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	rawNetUris := d.Get("network_uris").(*schema.Set).List()
	netUris := make([]utils.Nstring, len(rawNetUris))
	for i, raw := range rawNetUris {
		netUris[i] = utils.NewNstring(raw.(string))
	}

	netSet := ov.NetworkSet{
		Name:             d.Get("name").(string),
		NetworkUris:      netUris,
		Type:             d.Get("type").(string),
		NativeNetworkUri: utils.NewNstring(d.Get("native_network_uri").(string)),
	}

	netSetError := config.ovClient.CreateNetworkSet(netSet)
	d.SetId(d.Get("name").(string))
	if netSetError != nil {
		d.SetId("")
		return netSetError
	}
	return resourceNetworkSetRead(d, meta)
}

func resourceNetworkSetRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	netSet, err := config.ovClient.GetNetworkSetByName(d.Id())
	if err != nil || netSet.URI.IsNil() {
		d.SetId("")
		return nil
	}
	d.Set("name", netSet.Name)
	d.Set("type", netSet.Type)
	d.Set("created", netSet.Created)
	d.Set("description", netSet.Description)
	d.Set("eTag", netSet.ETAG)
	d.Set("modified", netSet.Modified)
	d.Set("native_network_uri", netSet.NativeNetworkUri)
	d.Set("uri", netSet.URI.String())
	d.Set("connection_template_uri", netSet.ConnectionTemplateUri.String())
	d.Set("status", netSet.Status)
	d.Set("category", netSet.Category)
	d.Set("state", netSet.State)

	networkUris := make([]interface{}, len(netSet.NetworkUris))
	for i := 0; i < len(netSet.NetworkUris); i++ {
		networkUris[i] = netSet.NetworkUris[i].String()
	}

	rawNetUris := d.Get("network_uris").(*schema.Set).List()
	for i, currNetworkUri := range rawNetUris {
		for j := 0; j < len(networkUris); j++ {
			if currNetworkUri.(string) == networkUris[j] && i <= len(networkUris)-1 {
				networkUris[i], networkUris[j] = networkUris[j], networkUris[i]
			}
		}
	}
	d.Set("network_uris", networkUris)

	return nil

}

func resourceNetworkSetUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	rawNetUris := d.Get("network_uris").(*schema.Set).List()
	netUris := make([]utils.Nstring, len(rawNetUris))
	for i, raw := range rawNetUris {
		netUris[i] = utils.NewNstring(raw.(string))
	}
	newNetSet := ov.NetworkSet{
		ETAG: d.Get("eTag").(string),
		URI:  utils.NewNstring(d.Get("uri").(string)),
		Name: d.Get("name").(string),
		ConnectionTemplateUri: utils.NewNstring(d.Get("connection_template_uri").(string)),
		Type:             d.Get("type").(string),
		NativeNetworkUri: utils.NewNstring(d.Get("native_network_uri").(string)),
		NetworkUris:      netUris,
	}

	err := config.ovClient.UpdateNetworkSet(newNetSet)
	if err != nil {
		return err
	}
	d.SetId(d.Get("name").(string))

	return resourceNetworkSetRead(d, meta)
}

func resourceNetworkSetDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	error := config.ovClient.DeleteNetworkSet(d.Get("name").(string))
	if error != nil {
		return error
	}
	return nil
}
