package packet

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/packethost/packngo"
)

func resourcePacketDevice() *schema.Resource {
	return &schema.Resource{
		Create: resourcePacketDeviceCreate,
		Read:   resourcePacketDeviceRead,
		Update: resourcePacketDeviceUpdate,
		Delete: resourcePacketDeviceDelete,

		Schema: map[string]*schema.Schema{
			"project_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"hostname": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"operating_system": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"facility": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"plan": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"billing_cycle": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"state": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"locked": &schema.Schema{
				Type:     schema.TypeBool,
				Computed: true,
			},

			"network": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"address": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},

						"gateway": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},

						"family": &schema.Schema{
							Type:     schema.TypeInt,
							Computed: true,
						},

						"cidr": &schema.Schema{
							Type:     schema.TypeInt,
							Computed: true,
						},

						"public": &schema.Schema{
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},

			"created": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"updated": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"user_data": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"tags": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourcePacketDeviceCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*packngo.Client)

	createRequest := &packngo.DeviceCreateRequest{
		HostName:     d.Get("hostname").(string),
		Plan:         d.Get("plan").(string),
		Facility:     d.Get("facility").(string),
		OS:           d.Get("operating_system").(string),
		BillingCycle: d.Get("billing_cycle").(string),
		ProjectID:    d.Get("project_id").(string),
	}

	if attr, ok := d.GetOk("user_data"); ok {
		createRequest.UserData = attr.(string)
	}

	tags := d.Get("tags.#").(int)
	if tags > 0 {
		createRequest.Tags = make([]string, 0, tags)
		for i := 0; i < tags; i++ {
			key := fmt.Sprintf("tags.%d", i)
			createRequest.Tags = append(createRequest.Tags, d.Get(key).(string))
		}
	}

	log.Printf("[DEBUG] Device create configuration: %#v", createRequest)

	newDevice, _, err := client.Devices.Create(createRequest)
	if err != nil {
		return fmt.Errorf("Error creating device: %s", err)
	}

	// Assign the device id
	d.SetId(newDevice.ID)

	log.Printf("[INFO] Device ID: %s", d.Id())

	_, err = WaitForDeviceAttribute(d, "active", []string{"provisioning"}, "state", meta)
	if err != nil {
		return fmt.Errorf(
			"Error waiting for device (%s) to become ready: %s", d.Id(), err)
	}

	return resourcePacketDeviceRead(d, meta)
}

func resourcePacketDeviceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*packngo.Client)

	// Retrieve the device properties for updating the state
	device, _, err := client.Devices.Get(d.Id())
	if err != nil {
		return fmt.Errorf("Error retrieving device: %s", err)
	}

	d.Set("name", device.Hostname)
	d.Set("plan", device.Plan.Slug)
	d.Set("facility", device.Facility.Code)
	d.Set("operating_system", device.OS.Slug)
	d.Set("state", device.State)
	d.Set("billing_cycle", device.BillingCycle)
	d.Set("locked", device.Locked)
	d.Set("created", device.Created)
	d.Set("udpated", device.Updated)

	tags := make([]string, 0)
	for _, tag := range device.Tags {
		tags = append(tags, tag)
	}
	d.Set("tags", tags)

	networks := make([]map[string]interface{}, 0, 1)
	for _, ip := range device.Network {
		network := make(map[string]interface{})
		network["address"] = ip.Address
		network["gateway"] = ip.Gateway
		network["family"] = ip.Family
		network["cidr"] = ip.Cidr
		network["public"] = ip.Public
		networks = append(networks, network)
	}
	d.Set("network", networks)

	return nil
}

func resourcePacketDeviceUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*packngo.Client)

	if d.HasChange("locked") && d.Get("locked").(bool) {
		_, err := client.Devices.Lock(d.Id())

		if err != nil {
			return fmt.Errorf(
				"Error locking device (%s): %s", d.Id(), err)
		}
	} else if d.HasChange("locked") {
		_, err := client.Devices.Unlock(d.Id())

		if err != nil {
			return fmt.Errorf(
				"Error unlocking device (%s): %s", d.Id(), err)
		}
	}

	return resourcePacketDeviceRead(d, meta)
}

func resourcePacketDeviceDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*packngo.Client)

	log.Printf("[INFO] Deleting device: %s", d.Id())
	if _, err := client.Devices.Delete(d.Id()); err != nil {
		return fmt.Errorf("Error deleting device: %s", err)
	}

	return nil
}

func WaitForDeviceAttribute(
	d *schema.ResourceData, target string, pending []string, attribute string, meta interface{}) (interface{}, error) {
	// Wait for the device so we can get the networking attributes
	// that show up after a while
	log.Printf(
		"[INFO] Waiting for device (%s) to have %s of %s",
		d.Id(), attribute, target)

	stateConf := &resource.StateChangeConf{
		Pending:    pending,
		Target:     target,
		Refresh:    newDeviceStateRefreshFunc(d, attribute, meta),
		Timeout:    60 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	return stateConf.WaitForState()
}

func newDeviceStateRefreshFunc(
	d *schema.ResourceData, attribute string, meta interface{}) resource.StateRefreshFunc {
	client := meta.(*packngo.Client)
	return func() (interface{}, string, error) {
		err := resourcePacketDeviceRead(d, meta)
		if err != nil {
			return nil, "", err
		}

		// See if we can access our attribute
		if attr, ok := d.GetOk(attribute); ok {
			// Retrieve the device properties
			device, _, err := client.Devices.Get(d.Id())
			if err != nil {
				return nil, "", fmt.Errorf("Error retrieving device: %s", err)
			}

			return &device, attr.(string), nil
		}

		return nil, "", nil
	}
}

// Powers on the device and waits for it to be active
func powerOnAndWait(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*packngo.Client)
	_, err := client.Devices.PowerOn(d.Id())
	if err != nil {
		return err
	}

	// Wait for power on
	_, err = WaitForDeviceAttribute(d, "active", []string{"off"}, "state", client)
	if err != nil {
		return err
	}

	return nil
}
