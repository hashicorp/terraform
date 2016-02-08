package vcd

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	types "github.com/hmrc/vmware-govcd/types/v56"
)

func resourceVcdVApp() *schema.Resource {
	return &schema.Resource{
		Create: resourceVcdVAppCreate,
		Update: resourceVcdVAppUpdate,
		Read:   resourceVcdVAppRead,
		Delete: resourceVcdVAppDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"template_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"catalog_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"network_href": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"network_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"memory": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"cpus": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"ip": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"initscript": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"metadata": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
			},
			"href": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"power_on": &schema.Schema{
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
	var networkHref string
	net, err := vcdClient.OrgVdc.FindVDCNetwork(d.Get("network_name").(string))
	if err != nil {
		return fmt.Errorf("Error finding OrgVCD Network: %#v", err)
	}
	if attr, ok := d.GetOk("network_href"); ok {
		networkHref = attr.(string)
	} else {
		networkHref = net.OrgVDCNetwork.HREF
	}
	// vapptemplate := govcd.NewVAppTemplate(&vcdClient.Client)
	//
	createvapp := &types.InstantiateVAppTemplateParams{
		Ovf:   "http://schemas.dmtf.org/ovf/envelope/1",
		Xmlns: "http://www.vmware.com/vcloud/v1.5",
		Name:  d.Get("name").(string),
		InstantiationParams: &types.InstantiationParams{
			NetworkConfigSection: &types.NetworkConfigSection{
				Info: "Configuration parameters for logical networks",
				NetworkConfig: &types.VAppNetworkConfiguration{
					NetworkName: d.Get("network_name").(string),
					Configuration: &types.NetworkConfiguration{
						ParentNetwork: &types.Reference{
							HREF: networkHref,
						},
						FenceMode: "bridged",
					},
				},
			},
		},
		Source: &types.Reference{
			HREF: vapptemplate.VAppTemplate.HREF,
		},
	}

	err = retryCall(vcdClient.MaxRetryTimeout, func() error {
		e := vcdClient.OrgVdc.InstantiateVAppTemplate(createvapp)

		if e != nil {
			return fmt.Errorf("Error: %#v", e)
		}

		e = vcdClient.OrgVdc.Refresh()
		if e != nil {
			return fmt.Errorf("Error: %#v", e)
		}
		return nil
	})
	if err != nil {
		return err
	}

	vapp, err := vcdClient.OrgVdc.FindVAppByName(d.Get("name").(string))

	err = retryCall(vcdClient.MaxRetryTimeout, func() error {
		task, err := vapp.ChangeVMName(d.Get("name").(string))
		if err != nil {
			return fmt.Errorf("Error with vm name change: %#v", err)
		}

		return task.WaitTaskCompletion()
	})
	if err != nil {
		return fmt.Errorf("Error changing vmname: %#v", err)
	}

	err = retryCall(vcdClient.MaxRetryTimeout, func() error {
		task, err := vapp.ChangeNetworkConfig(d.Get("network_name").(string), d.Get("ip").(string))
		if err != nil {
			return fmt.Errorf("Error with Networking change: %#v", err)
		}
		return task.WaitTaskCompletion()
	})
	if err != nil {
		return fmt.Errorf("Error changing network: %#v", err)
	}

	if initscript, ok := d.GetOk("initscript"); ok {
		err = retryCall(vcdClient.MaxRetryTimeout, func() error {
			task, err := vapp.RunCustomizationScript(d.Get("name").(string), initscript.(string))
			if err != nil {
				return fmt.Errorf("Error with setting init script: %#v", err)
			}
			return task.WaitTaskCompletion()
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
			err = retryCall(vcdClient.MaxRetryTimeout, func() error {
				task, err := vapp.ChangeMemorySize(d.Get("memory").(int))
				if err != nil {
					return fmt.Errorf("Error changing memory size: %#v", err)
				}

				return task.WaitTaskCompletion()
			})
			if err != nil {
				return err
			}
		}

		if d.HasChange("cpus") {
			err = retryCall(vcdClient.MaxRetryTimeout, func() error {
				task, err := vapp.ChangeCPUcount(d.Get("cpus").(int))
				if err != nil {
					return fmt.Errorf("Error changing cpu count: %#v", err)
				}

				return task.WaitTaskCompletion()
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

	vapp, err := vcdClient.OrgVdc.FindVAppByName(d.Id())
	if err != nil {
		log.Printf("[DEBUG] Unable to find vapp. Removing from tfstate")
		d.SetId("")
		return nil
	}
	d.Set("ip", vapp.VApp.Children.VM[0].NetworkConnectionSection.NetworkConnection.IPAddress)

	return nil
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

	_ = retryCall(vcdClient.MaxRetryTimeout, func() error {
		task, err := vapp.Undeploy()
		if err != nil {
			return fmt.Errorf("Error undeploying: %#v", err)
		}

		return task.WaitTaskCompletion()
	})

	err = retryCall(vcdClient.MaxRetryTimeout, func() error {
		task, err := vapp.Delete()
		if err != nil {
			return fmt.Errorf("Error deleting: %#v", err)
		}

		return task.WaitTaskCompletion()
	})

	return err
}
