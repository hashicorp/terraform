package vcd

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	types "github.com/ukcloud/govcloudair/types/v56"
)

func resourceVcdVApp() *schema.Resource {
	return &schema.Resource{
		Create: resourceVcdVAppCreate,
		Update: resourceVcdVAppUpdate,
		Read:   resourceVcdVAppRead,
		Delete: resourceVcdVAppDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"template_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"catalog_name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"network_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"memory": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"cpus": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"ip": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"storage_profile": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "terrafrom vapp",
			},
			"initscript": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"metadata": {
				Type:     schema.TypeMap,
				Optional: true,
			},
			"href": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"power_on": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
		},
	}
}

func resourceVcdVAppCreate(d *schema.ResourceData, meta interface{}) error {
	vcdClient := meta.(*VCDClient)

	catalog, err := vcdClient.Org.FindCatalog(d.Get("catalog_name").(string))
	if err != nil {
		return fmt.Errorf("Error finding catalog: %#v", err)
	}

	catalogitem, err := catalog.FindCatalogItem(d.Get("template_name").(string))
	if err != nil {
		return fmt.Errorf("Error finding catelog item: %#v", err)
	}

	vapptemplate, err := catalogitem.GetVAppTemplate()
	if err != nil {
		return fmt.Errorf("Error finding VAppTemplate: %#v", err)
	}

	log.Printf("[DEBUG] VAppTemplate: %#v", vapptemplate)
	net, err := vcdClient.OrgVdc.FindVDCNetwork(d.Get("network_name").(string))
	if err != nil {
		return fmt.Errorf("Error finding OrgVCD Network: %#v", err)
	}

	storage_profile_reference := types.Reference{}

	// Use default storage profile by default
	storage_profile_query_params := make(map[string]string)
	storage_profile_query_params["type"] = "orgVdcStorageProfile"
	storage_profile_query_params["format"] = "records"
	storage_profiles, err := vcdClient.VCDClient.Query(storage_profile_query_params)
	default_storage_profile, err := vcdClient.OrgVdc.GetDefaultStorageProfileReference(storage_profiles.Results)
	if err != nil {
		return fmt.Errorf("Couldn't find storage_profile and no default storage_profile avaiable %s", err)
	}

	// Override default storage profile if we found the given storage profile
	if d.Get("storage_profile").(string) != "" {
		storage_profile_reference, err = vcdClient.OrgVdc.FindStorageProfileReference(d.Get("storage_profile").(string))
		if err != nil {
			storage_profile_reference = default_storage_profile
			log.Printf("Using default storage_profile: %s", storage_profile_reference)
		}
	} else {
		storage_profile_reference = default_storage_profile
	}

	log.Printf("storage_profile %s", storage_profile_reference)

	vapp, err := vcdClient.OrgVdc.FindVAppByName(d.Get("name").(string))

	if err != nil {
		vapp = vcdClient.NewVApp(&vcdClient.Client)

		err = retryCall(vcdClient.MaxRetryTimeout, func() *resource.RetryError {
			task, err := vapp.ComposeVApp(net, vapptemplate, storage_profile_reference, d.Get("name").(string), d.Get("description").(string))
			if err != nil {
				return resource.RetryableError(fmt.Errorf("Error creating vapp: %#v", err))
			}

			return resource.RetryableError(task.WaitTaskCompletion())
		})

		if err != nil {
			return fmt.Errorf("Error creating vapp: %#v", err)
		}
	}

	err = retryCall(vcdClient.MaxRetryTimeout, func() *resource.RetryError {
		task, err := vapp.ChangeVMName(d.Get("name").(string))
		if err != nil {
			return resource.RetryableError(fmt.Errorf("Error with vm name change: %#v", err))
		}

		return resource.RetryableError(task.WaitTaskCompletion())
	})
	if err != nil {
		return fmt.Errorf("Error changing vmname: %#v", err)
	}

	err = retryCall(vcdClient.MaxRetryTimeout, func() *resource.RetryError {
		task, err := vapp.ChangeNetworkConfig(d.Get("network_name").(string), d.Get("ip").(string))
		if err != nil {
			return resource.RetryableError(fmt.Errorf("Error with Networking change: %#v", err))
		}
		return resource.RetryableError(task.WaitTaskCompletion())
	})
	if err != nil {
		return fmt.Errorf("Error changing network: %#v", err)
	}

	if initscript, ok := d.GetOk("initscript"); ok {
		err = retryCall(vcdClient.MaxRetryTimeout, func() *resource.RetryError {
			task, err := vapp.RunCustomizationScript(d.Get("name").(string), initscript.(string))
			if err != nil {
				return resource.RetryableError(fmt.Errorf("Error with setting init script: %#v", err))
			}
			return resource.RetryableError(task.WaitTaskCompletion())
		})
		if err != nil {
			return fmt.Errorf("Error completing tasks: %#v", err)
		}
	}

	d.SetId(d.Get("name").(string))

	return resourceVcdVAppUpdate(d, meta)
}

func resourceVcdVAppUpdate(d *schema.ResourceData, meta interface{}) error {
	vcdClient := meta.(*VCDClient)
	vapp, err := vcdClient.OrgVdc.FindVAppByName(d.Id())

	if err != nil {
		return fmt.Errorf("Error finding VApp: %#v", err)
	}

	status, err := vapp.GetStatus()
	if err != nil {
		return fmt.Errorf("Error getting VApp status: %#v", err)
	}

	if d.HasChange("metadata") {
		oraw, nraw := d.GetChange("metadata")
		metadata := oraw.(map[string]interface{})
		for k := range metadata {
			task, err := vapp.DeleteMetadata(k)
			if err != nil {
				return fmt.Errorf("Error deleting metadata: %#v", err)
			}
			err = task.WaitTaskCompletion()
			if err != nil {
				return fmt.Errorf("Error completing tasks: %#v", err)
			}
		}
		metadata = nraw.(map[string]interface{})
		for k, v := range metadata {
			task, err := vapp.AddMetadata(k, v.(string))
			if err != nil {
				return fmt.Errorf("Error adding metadata: %#v", err)
			}
			err = task.WaitTaskCompletion()
			if err != nil {
				return fmt.Errorf("Error completing tasks: %#v", err)
			}
		}

	}

	if d.HasChange("storage_profile") {
		err = retryCall(vcdClient.MaxRetryTimeout, func() *resource.RetryError {
			task, err := vapp.ChangeStorageProfile(d.Get("storage_profile").(string))
			if err != nil {
				return resource.RetryableError(fmt.Errorf("Error changing storage_profile: %#v", err))
			}

			return resource.RetryableError(task.WaitTaskCompletion())
		})
		if err != nil {
			return err
		}
	}

	if d.HasChange("memory") || d.HasChange("cpus") || d.HasChange("power_on") {
		if status != "POWERED_OFF" {
			task, err := vapp.PowerOff()
			if err != nil {
				return fmt.Errorf("Error Powering Off: %#v", err)
			}
			err = task.WaitTaskCompletion()
			if err != nil {
				return fmt.Errorf("Error completing tasks: %#v", err)
			}
		}

		if d.HasChange("memory") {
			err = retryCall(vcdClient.MaxRetryTimeout, func() *resource.RetryError {
				task, err := vapp.ChangeMemorySize(d.Get("memory").(int))
				if err != nil {
					return resource.RetryableError(fmt.Errorf("Error changing memory size: %#v", err))
				}

				return resource.RetryableError(task.WaitTaskCompletion())
			})
			if err != nil {
				return err
			}
		}

		if d.HasChange("cpus") {
			err = retryCall(vcdClient.MaxRetryTimeout, func() *resource.RetryError {
				task, err := vapp.ChangeCPUcount(d.Get("cpus").(int))
				if err != nil {
					return resource.RetryableError(fmt.Errorf("Error changing cpu count: %#v", err))
				}

				return resource.RetryableError(task.WaitTaskCompletion())
			})
			if err != nil {
				return fmt.Errorf("Error completing task: %#v", err)
			}
		}

		if d.Get("power_on").(bool) {
			task, err := vapp.PowerOn()
			if err != nil {
				return fmt.Errorf("Error Powering Up: %#v", err)
			}
			err = task.WaitTaskCompletion()
			if err != nil {
				return fmt.Errorf("Error completing tasks: %#v", err)
			}
		}

	}

	return resourceVcdVAppRead(d, meta)
}

func resourceVcdVAppRead(d *schema.ResourceData, meta interface{}) error {
	vcdClient := meta.(*VCDClient)

	err := vcdClient.OrgVdc.Refresh()
	if err != nil {
		return fmt.Errorf("Error refreshing vdc: %#v", err)
	}

	_, err = vcdClient.OrgVdc.FindVAppByName(d.Id())
	if err != nil {
		log.Printf("[DEBUG] Unable to find vapp. Removing from tfstate")
		d.SetId("")
		return nil
	}

	ip, err := getVAppIPAddress(d, meta)
	if err != nil {
		return err
	}
	d.Set("ip", ip)

	return nil
}

func getVAppIPAddress(d *schema.ResourceData, meta interface{}) (string, error) {
	vcdClient := meta.(*VCDClient)
	var ip string

	err := retryCall(vcdClient.MaxRetryTimeout, func() *resource.RetryError {
		err := vcdClient.OrgVdc.Refresh()
		if err != nil {
			return resource.RetryableError(fmt.Errorf("Error refreshing vdc: %#v", err))
		}
		vapp, err := vcdClient.OrgVdc.FindVAppByName(d.Id())
		if err != nil {
			return resource.RetryableError(fmt.Errorf("Unable to find vapp."))
		}
		ip = vapp.VApp.Children.VM[0].NetworkConnectionSection.NetworkConnection.IPAddress
		if ip == "" {
			return resource.RetryableError(fmt.Errorf("Timeout: VM did not aquire IP address"))
		}
		return nil
	})

	return ip, err
}

func resourceVcdVAppDelete(d *schema.ResourceData, meta interface{}) error {
	vcdClient := meta.(*VCDClient)
	vapp, err := vcdClient.OrgVdc.FindVAppByName(d.Id())

	if err != nil {
		return fmt.Errorf("error finding vapp: %s", err)
	}

	if err != nil {
		return fmt.Errorf("Error getting VApp status: %#v", err)
	}

	_ = retryCall(vcdClient.MaxRetryTimeout, func() *resource.RetryError {
		task, err := vapp.Undeploy()
		if err != nil {
			return resource.RetryableError(fmt.Errorf("Error undeploying: %#v", err))
		}

		return resource.RetryableError(task.WaitTaskCompletion())
	})

	err = retryCall(vcdClient.MaxRetryTimeout, func() *resource.RetryError {
		task, err := vapp.Delete()
		if err != nil {
			return resource.RetryableError(fmt.Errorf("Error deleting: %#v", err))
		}

		return resource.RetryableError(task.WaitTaskCompletion())
	})

	return err
}
