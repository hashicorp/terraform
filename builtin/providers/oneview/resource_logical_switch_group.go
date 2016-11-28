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
)

func resourceLogicalSwitchGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceLogicalSwitchGroupCreate,
		Read:   resourceLogicalSwitchGroupRead,
		Update: resourceLogicalSwitchGroupUpdate,
		Delete: resourceLogicalSwitchGroupDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "logical-switch-group",
			},
			"switch_type_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"switch_count": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
			"location_entry_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "StackingMemberId",
			},
			"created": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"modified": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"uri": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"status": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"category": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"state": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"fabric_uri": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"eTag": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"switch_type_uri": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceLogicalSwitchGroupCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	switchType, err := config.ovClient.GetSwitchTypeByName(d.Get("switch_type_name").(string))
	if err != nil {
		return err
	}
	d.Set("switch_type_uri", switchType.URI)

	if switchType.Name == "" {
		return fmt.Errorf("Can't find %s switchType", d.Get("switch_type_name").(string))
	}

	switchMapEntryTemplates := make([]ov.SwitchMapEntry, d.Get("switch_count").(int))
	for i := 0; i < d.Get("switch_count").(int); i++ {
		locationEntries := make([]ov.LocationEntry, 1)
		locationEntry := ov.LocationEntry{
			RelativeValue: i + 1,
			Type:          d.Get("location_entry_type").(string),
		}
		locationEntries[0] = locationEntry
		logicalLocation := ov.LogicalLocation{
			LocationEntries: locationEntries,
		}
		switchMapEntry := ov.SwitchMapEntry{
			LogicalLocation:        logicalLocation,
			PermittedSwitchTypeUri: switchType.URI,
		}
		switchMapEntryTemplates[i] = switchMapEntry
	}

	switchMapTemplate := ov.SwitchMapTemplate{
		SwitchMapEntryTemplates: switchMapEntryTemplates,
	}

	logicalSwitchGroup := ov.LogicalSwitchGroup{
		Name:              d.Get("name").(string),
		Type:              d.Get("type").(string),
		SwitchMapTemplate: switchMapTemplate,
	}

	logicalSwitchGroupError := config.ovClient.CreateLogicalSwitchGroup(logicalSwitchGroup)
	d.SetId(d.Get("name").(string))
	if logicalSwitchGroupError != nil {
		d.SetId("")
		return logicalSwitchGroupError
	}
	return resourceLogicalSwitchGroupRead(d, meta)
}

func resourceLogicalSwitchGroupRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	logicalSwitchGroup, err := config.ovClient.GetLogicalSwitchGroupByName(d.Id())
	if err != nil || logicalSwitchGroup.URI.IsNil() {
		d.SetId("")
		return nil
	}

	d.Set("name", logicalSwitchGroup.Name)
	d.Set("type", logicalSwitchGroup.Type)
	d.Set("switch_count", len(logicalSwitchGroup.SwitchMapTemplate.SwitchMapEntryTemplates))
	d.Set("created", logicalSwitchGroup.Created)
	d.Set("modified", logicalSwitchGroup.Modified)
	d.Set("uri", logicalSwitchGroup.URI.String())
	d.Set("status", logicalSwitchGroup.Status)
	d.Set("category", logicalSwitchGroup.Category)
	d.Set("state", logicalSwitchGroup.State)
	d.Set("fabric_uri", logicalSwitchGroup.FabricUri.String())
	d.Set("eTag", logicalSwitchGroup.ETAG)
	d.Set("description", logicalSwitchGroup.Description)
	return nil
}

func resourceLogicalSwitchGroupDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	error := config.ovClient.DeleteLogicalSwitchGroup(d.Get("name").(string))
	if error != nil {
		return error
	}
	return nil
}

func resourceLogicalSwitchGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	switchType, err := config.ovClient.GetSwitchTypeByName(d.Get("switch_type_name").(string))
	if err != nil {
		return err
	}
	if switchType.Name == "" {
		return fmt.Errorf("Can't find %s switchType", d.Get("switch_type_name").(string))
	}

	switchMapEntryTemplates := make([]ov.SwitchMapEntry, d.Get("switch_count").(int))
	for i := 0; i < d.Get("switch_count").(int); i++ {
		locationEntries := make([]ov.LocationEntry, 1)
		locationEntry := ov.LocationEntry{
			RelativeValue: i + 1,
			Type:          d.Get("location_entry_type").(string),
		}
		locationEntries[0] = locationEntry
		logicalLocation := ov.LogicalLocation{
			LocationEntries: locationEntries,
		}
		switchMapEntry := ov.SwitchMapEntry{
			LogicalLocation:        logicalLocation,
			PermittedSwitchTypeUri: switchType.URI,
		}
		switchMapEntryTemplates[i] = switchMapEntry
	}

	switchMapTemplate := ov.SwitchMapTemplate{
		SwitchMapEntryTemplates: switchMapEntryTemplates,
	}

	newLogicalSwitchGroup := ov.LogicalSwitchGroup{
		ETAG:              d.Get("eTag").(string),
		URI:               utils.NewNstring(d.Get("uri").(string)),
		Name:              d.Get("name").(string),
		Type:              d.Get("type").(string),
		SwitchMapTemplate: switchMapTemplate,
	}
	err = config.ovClient.UpdateLogicalSwitchGroup(newLogicalSwitchGroup)
	if err != nil {
		return err
	}
	d.SetId(d.Get("name").(string))
	return resourceLogicalSwitchGroupRead(d, meta)
}
