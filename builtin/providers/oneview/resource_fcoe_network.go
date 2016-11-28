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

func resourceFCoENetwork() *schema.Resource {
	return &schema.Resource{
		Create: resourceFCoENetworkCreate,
		Read:   resourceFCoENetworkRead,
		Update: resourceFCoENetworkUpdate,
		Delete: resourceFCoENetworkDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"vlanId": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"connectionTemplateUri": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "fcoe-network",
			},
			"managedSanUri": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"fabricUri": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"state": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"status": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"eTag": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"modified": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"created": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"category": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"uri": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceFCoENetworkCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	fcoeNet := ov.FCoENetwork{
		Name:   d.Get("name").(string),
		VlanId: d.Get("vlanId").(int),
		Type:   d.Get("type").(string),
	}

	fcoeNetError := config.ovClient.CreateFCoENetwork(fcoeNet)
	d.SetId(d.Get("name").(string))
	if fcoeNetError != nil {
		d.SetId("")
		return fcoeNetError
	}
	return resourceFCoENetworkRead(d, meta)
}

func resourceFCoENetworkRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	fcoeNet, fcoeNetError := config.ovClient.GetFCoENetworkByName(d.Get("name").(string))
	if fcoeNetError != nil || fcoeNet.URI.IsNil() {
		d.SetId("")
		return nil
	}
	d.Set("created", fcoeNet.Created)
	d.Set("modified", fcoeNet.Modified)
	d.Set("uri", fcoeNet.URI.String())
	d.Set("connectionTemplateUri", fcoeNet.ConnectionTemplateUri.String())
	d.Set("status", fcoeNet.Status)
	d.Set("category", fcoeNet.Category)
	d.Set("state", fcoeNet.State)
	d.Set("fabric_uri", fcoeNet.FabricUri.String())
	d.Set("eTag", fcoeNet.ETAG)
	d.Set("managedSanUri", fcoeNet.ManagedSanUri)
	d.Set("description", fcoeNet.Description)

	return nil
}

func resourceFCoENetworkUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	newFCoENet := ov.FCoENetwork{
		ETAG:   d.Get("eTag").(string),
		URI:    utils.NewNstring(d.Get("uri").(string)),
		VlanId: d.Get("vlanId").(int),
		Name:   d.Get("name").(string),
		ConnectionTemplateUri: utils.NewNstring(d.Get("connectionTemplateUri").(string)),
		Type: d.Get("type").(string),
	}

	err := config.ovClient.UpdateFCoENetwork(newFCoENet)
	if err != nil {
		return err
	}
	d.SetId(d.Get("name").(string))

	return resourceFCoENetworkRead(d, meta)
}

func resourceFCoENetworkDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	error := config.ovClient.DeleteFCoENetwork(d.Get("name").(string))
	if error != nil {
		return error
	}
	return nil
}
