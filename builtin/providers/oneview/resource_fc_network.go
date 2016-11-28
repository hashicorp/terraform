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

func resourceFCNetwork() *schema.Resource {
	return &schema.Resource{
		Create: resourceFCNetworkCreate,
		Read:   resourceFCNetworkRead,
		Update: resourceFCNetworkUpdate,
		Delete: resourceFCNetworkDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"fabric_type": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "FabricAttach",
			},
			"link_stability_time": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  30,
			},
			"auto_login_redistribution": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"connection_template_uri": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"type": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "fc-networkV2",
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
			"created": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"modified": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceFCNetworkCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	fcNet := ov.FCNetwork{
		Name:                    d.Get("name").(string),
		FabricType:              d.Get("fabric_type").(string),
		LinkStabilityTime:       d.Get("link_stability_time").(int),
		AutoLoginRedistribution: d.Get("auto_login_redistribution").(bool),
		Type:        d.Get("type").(string),
		Description: d.Get("description").(string),
	}

	fcNetError := config.ovClient.CreateFCNetwork(fcNet)
	d.SetId(d.Get("name").(string))
	if fcNetError != nil {
		d.SetId("")
		return fcNetError
	}
	return resourceFCNetworkRead(d, meta)
}

func resourceFCNetworkRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	fcNet, error := config.ovClient.GetFCNetworkByName(d.Get("name").(string))
	if error != nil || fcNet.URI.IsNil() {
		d.SetId("")
		return nil
	}
	d.Set("name", fcNet.Name)
	d.Set("fabric_type", fcNet.FabricType)
	d.Set("link_stability_time", fcNet.LinkStabilityTime)
	d.Set("auto_login_redistribution", fcNet.AutoLoginRedistribution)
	d.Set("description", fcNet.Description)
	d.Set("type", fcNet.Type)
	d.Set("uri", fcNet.URI.String())
	d.Set("connection_template_uri", fcNet.ConnectionTemplateUri.String())
	d.Set("status", fcNet.Status)
	d.Set("category", fcNet.Category)
	d.Set("state", fcNet.State)
	d.Set("fabric_uri", fcNet.FabricUri.String())
	d.Set("created", fcNet.Created)
	d.Set("modified", fcNet.Modified)
	d.Set("eTag", fcNet.ETAG)
	return nil
}

func resourceFCNetworkUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	fcNet := ov.FCNetwork{
		ETAG:                    d.Get("eTag").(string),
		URI:                     utils.NewNstring(d.Get("uri").(string)),
		Name:                    d.Get("name").(string),
		FabricType:              d.Get("fabric_type").(string),
		LinkStabilityTime:       d.Get("link_stability_time").(int),
		AutoLoginRedistribution: d.Get("auto_login_redistribution").(bool),
		Type: d.Get("type").(string),
		ConnectionTemplateUri: utils.NewNstring(d.Get("connection_template_uri").(string)),
		Description:           d.Get("description").(string),
	}

	err := config.ovClient.UpdateFcNetwork(fcNet)
	if err != nil {
		return err
	}
	d.SetId(d.Get("name").(string))

	return resourceFCNetworkRead(d, meta)
}

func resourceFCNetworkDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	error := config.ovClient.DeleteFCNetwork(d.Get("name").(string))
	if error != nil {
		return error
	}
	return nil
}
