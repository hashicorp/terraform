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
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceI3SPlan() *schema.Resource {
	return &schema.Resource{
		Create: resourceI3SPlanCreate,
		Read:   resourceI3SPlanRead,
		Update: resourceI3SPlanUpdate,
		Delete: resourceI3SPlanDelete,

		Schema: map[string]*schema.Schema{
			"server_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"os_deployment_plan": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"deployment_attribute": &schema.Schema{
				Optional: true,
				Type:     schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"value": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func resourceI3SPlanCreate(d *schema.ResourceData, meta interface{}) error {

	config := meta.(*Config)

	customizeServer := ov.CustomizeServer{
		ProfileName:           d.Get("server_name").(string),
		OSDeploymentBuildPlan: d.Get("os_deployment_plan").(string),
	}

	if _, ok := d.GetOk("deployment_attribute"); ok {
		deploymentAttributeCount := d.Get("deployment_attribute.#").(int)
		deploymentAttributes := make(map[string]string)
		for i := 0; i < deploymentAttributeCount; i++ {
			deploymentAttributePrefix := fmt.Sprintf("deployment_attribute.%d", i)
			deploymentAttributes[d.Get(deploymentAttributePrefix+".key").(string)] = d.Get(deploymentAttributePrefix + ".value").(string)
		}
		customizeServer.OSDeploymentAttributes = deploymentAttributes
	}

	d.SetId(d.Get("server_name").(string))
	err := config.ovClient.CustomizeServer(customizeServer)
	if err != nil {
		d.SetId("")
		return err
	}

	return resourceI3SPlanRead(d, meta)
}

func resourceI3SPlanRead(d *schema.ResourceData, meta interface{}) error {

	config := meta.(*Config)

	serverProfile, err := config.ovClient.GetProfileByName(d.Id())
	if err != nil || serverProfile.URI.IsNil() {
		d.SetId("")
		return nil
	}
	d.Set("server_name", serverProfile.Name)

	osDeploymentPlan, err := config.ovClient.GetOSDeploymentPlan(serverProfile.OSDeploymentSettings.OSDeploymentPlanUri)
	if err != nil || osDeploymentPlan.URI.IsNil() {
		d.SetId("")
		return nil
	}
	d.Set("os_deployment_plan", osDeploymentPlan.Name)

	/*
	   if _, ok := d.GetOk("deployment_attribute"); ok {
	       deploymentAttributeCount := d.Get("deployment_attribute.#").(int)
	       deploymentAttributes := make([]interface{}, deploymentAttributeCount)
	       for i := 0; i < deploymentAttributeCount; i++ {
	           deploymentAttributePrefix := fmt.Sprintf("deployment_attribute.%d", i)
	           for j := 0; j < len(serverProfile.OSDeploymentSettings.OSCustomAttributes); j++{
	               if d.Get(deploymentAttributePrefix + ".key").(string) == serverProfile.OSDeploymentSettings.Name{
	                   d.Set
	               }
	           }
	       }
	   }*/

	return nil
}

func resourceI3SPlanUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	error := config.ovClient.DeleteOSBuildPlanFromServer(d.Get("server_name").(string))
	if error != nil {
		return error
	}

	customizeServer := ov.CustomizeServer{
		ProfileName:           d.Get("server_name").(string),
		OSDeploymentBuildPlan: d.Get("os_deployment_plan").(string),
	}

	if _, ok := d.GetOk("deployment_attribute"); ok {
		deploymentAttributeCount := d.Get("deployment_attribute.#").(int)
		deploymentAttributes := make(map[string]string)
		for i := 0; i < deploymentAttributeCount; i++ {
			deploymentAttributePrefix := fmt.Sprintf("deployment_attribute.%d", i)
			deploymentAttributes[d.Get(deploymentAttributePrefix+".key").(string)] = d.Get(deploymentAttributePrefix + ".value").(string)
		}
		customizeServer.OSDeploymentAttributes = deploymentAttributes
	}

	d.SetId(d.Get("server_name").(string))
	err := config.ovClient.CustomizeServer(customizeServer)
	if err != nil {
		d.SetId("")
		return err
	}

	return resourceI3SPlanRead(d, meta)
}

func resourceI3SPlanDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	error := config.ovClient.DeleteOSBuildPlanFromServer(d.Get("server_name").(string))
	if error != nil {
		return error
	}
	return nil
}
