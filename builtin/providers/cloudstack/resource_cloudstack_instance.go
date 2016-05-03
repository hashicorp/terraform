package cloudstack

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

func resourceCloudStackInstance() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudStackInstanceCreate,
		Read:   resourceCloudStackInstanceRead,
		Update: resourceCloudStackInstanceUpdate,
		Delete: resourceCloudStackInstanceDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"display_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"service_offering": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"network_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"network": &schema.Schema{
				Type:       schema.TypeString,
				Optional:   true,
				ForceNew:   true,
				Deprecated: "Please use the `network_id` field instead",
			},

			"ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"ipaddress": &schema.Schema{
				Type:       schema.TypeString,
				Optional:   true,
				Computed:   true,
				ForceNew:   true,
				Deprecated: "Please use the `ip_address` field instead",
			},

			"template": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"zone": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"keypair": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"user_data": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				StateFunc: func(v interface{}) string {
					switch v.(type) {
					case string:
						hash := sha1.Sum([]byte(v.(string)))
						return hex.EncodeToString(hash[:])
					default:
						return ""
					}
				},
			},

			"expunge": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"group": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceCloudStackInstanceCreate(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Retrieve the service_offering ID
	serviceofferingid, e := retrieveID(cs, "service_offering", d.Get("service_offering").(string))
	if e != nil {
		return e.Error()
	}

	// Retrieve the zone ID
	zoneid, e := retrieveID(cs, "zone", d.Get("zone").(string))
	if e != nil {
		return e.Error()
	}

	// Retrieve the zone object
	zone, _, err := cs.Zone.GetZoneByID(zoneid)
	if err != nil {
		return err
	}

	// Retrieve the template ID
	templateid, e := retrieveTemplateID(cs, zone.Id, d.Get("template").(string))
	if e != nil {
		return e.Error()
	}

	// Create a new parameter struct
	p := cs.VirtualMachine.NewDeployVirtualMachineParams(serviceofferingid, templateid, zone.Id)

	// Set the name
	name, hasName := d.GetOk("name")
	if hasName {
		p.SetName(name.(string))
	}

	// Set the display name
	if displayname, ok := d.GetOk("display_name"); ok {
		p.SetDisplayname(displayname.(string))
	} else if hasName {
		p.SetDisplayname(name.(string))
	}

	if zone.Networktype == "Advanced" {
		network, ok := d.GetOk("network_id")
		if !ok {
			network, ok = d.GetOk("network")
		}
		if !ok {
			return errors.New(
				"Either `network_id` or [deprecated] `network` must be provided when using a zone with network type `advanced`.")
		}

		// Retrieve the network ID
		networkid, e := retrieveID(
			cs,
			"network",
			network.(string),
			cloudstack.WithProject(d.Get("project").(string)),
		)
		if e != nil {
			return e.Error()
		}

		// Set the default network ID
		p.SetNetworkids([]string{networkid})
	}

	// If there is a ipaddres supplied, add it to the parameter struct
	ipaddress, ok := d.GetOk("ip_address")
	if !ok {
		ipaddress, ok = d.GetOk("ipaddress")
	}
	if ok {
		p.SetIpaddress(ipaddress.(string))
	}

	// If there is a project supplied, we retrieve and set the project id
	if err := setProjectid(p, cs, d); err != nil {
		return err
	}

	// If a keypair is supplied, add it to the parameter struct
	if keypair, ok := d.GetOk("keypair"); ok {
		p.SetKeypair(keypair.(string))
	}

	// If the user data contains any info, it needs to be base64 encoded and
	// added to the parameter struct
	if userData, ok := d.GetOk("user_data"); ok {
		ud := base64.StdEncoding.EncodeToString([]byte(userData.(string)))

		// deployVirtualMachine uses POST by default, so max userdata is 32K
		maxUD := 32768

		if cs.HTTPGETOnly {
			// deployVirtualMachine using GET instead, so max userdata is 2K
			maxUD = 2048
		}

		if len(ud) > maxUD {
			return fmt.Errorf(
				"The supplied user_data contains %d bytes after encoding, "+
					"this exeeds the limit of %d bytes", len(ud), maxUD)
		}

		p.SetUserdata(ud)
	}

	// If there is a group supplied, add it to the parameter struct
	if group, ok := d.GetOk("group"); ok {
		p.SetGroup(group.(string))
	}

	// Create the new instance
	r, err := cs.VirtualMachine.DeployVirtualMachine(p)
	if err != nil {
		return fmt.Errorf("Error creating the new instance %s: %s", name, err)
	}

	d.SetId(r.Id)

	// Set the connection info for any configured provisioners
	d.SetConnInfo(map[string]string{
		"host":     r.Nic[0].Ipaddress,
		"password": r.Password,
	})

	return resourceCloudStackInstanceRead(d, meta)
}

func resourceCloudStackInstanceRead(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Get the virtual machine details
	vm, count, err := cs.VirtualMachine.GetVirtualMachineByID(
		d.Id(),
		cloudstack.WithProject(d.Get("project").(string)),
	)
	if err != nil {
		if count == 0 {
			log.Printf("[DEBUG] Instance %s does no longer exist", d.Get("name").(string))
			d.SetId("")
			return nil
		}

		return err
	}

	// Update the config
	d.Set("name", vm.Name)
	d.Set("display_name", vm.Displayname)
	d.Set("network_id", vm.Nic[0].Networkid)
	d.Set("ip_address", vm.Nic[0].Ipaddress)
	d.Set("group", vm.Group)

	setValueOrID(d, "service_offering", vm.Serviceofferingname, vm.Serviceofferingid)
	setValueOrID(d, "template", vm.Templatename, vm.Templateid)
	setValueOrID(d, "project", vm.Project, vm.Projectid)
	setValueOrID(d, "zone", vm.Zonename, vm.Zoneid)

	return nil
}

func resourceCloudStackInstanceUpdate(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)
	d.Partial(true)

	name := d.Get("name").(string)

	// Check if the display name is changed and if so, update the virtual machine
	if d.HasChange("display_name") {
		log.Printf("[DEBUG] Display name changed for %s, starting update", name)

		// Create a new parameter struct
		p := cs.VirtualMachine.NewUpdateVirtualMachineParams(d.Id())

		// Set the new display name
		p.SetDisplayname(d.Get("display_name").(string))

		// Update the display name
		_, err := cs.VirtualMachine.UpdateVirtualMachine(p)
		if err != nil {
			return fmt.Errorf(
				"Error updating the display name for instance %s: %s", name, err)
		}

		d.SetPartial("display_name")
	}

	// Check if the group is changed and if so, update the virtual machine
	if d.HasChange("group") {
		log.Printf("[DEBUG] Group changed for %s, starting update", name)

		// Create a new parameter struct
		p := cs.VirtualMachine.NewUpdateVirtualMachineParams(d.Id())

		// Set the new group
		p.SetGroup(d.Get("group").(string))

		// Update the display name
		_, err := cs.VirtualMachine.UpdateVirtualMachine(p)
		if err != nil {
			return fmt.Errorf(
				"Error updating the group for instance %s: %s", name, err)
		}

		d.SetPartial("group")
	}

	// Attributes that require reboot to update
	if d.HasChange("name") || d.HasChange("service_offering") || d.HasChange("keypair") {
		// Before we can actually make these changes, the virtual machine must be stopped
		_, err := cs.VirtualMachine.StopVirtualMachine(
			cs.VirtualMachine.NewStopVirtualMachineParams(d.Id()))
		if err != nil {
			return fmt.Errorf(
				"Error stopping instance %s before making changes: %s", name, err)
		}

		// Check if the name has changed and if so, update the name
		if d.HasChange("name") {
			log.Printf("[DEBUG] Name for %s changed to %s, starting update", d.Id(), name)

			// Create a new parameter struct
			p := cs.VirtualMachine.NewUpdateVirtualMachineParams(d.Id())

			// Set the new name
			p.SetName(name)

			// Update the display name
			_, err := cs.VirtualMachine.UpdateVirtualMachine(p)
			if err != nil {
				return fmt.Errorf(
					"Error updating the name for instance %s: %s", name, err)
			}

			d.SetPartial("name")
		}

		// Check if the service offering is changed and if so, update the offering
		if d.HasChange("service_offering") {
			log.Printf("[DEBUG] Service offering changed for %s, starting update", name)

			// Retrieve the service_offering ID
			serviceofferingid, e := retrieveID(cs, "service_offering", d.Get("service_offering").(string))
			if e != nil {
				return e.Error()
			}

			// Create a new parameter struct
			p := cs.VirtualMachine.NewChangeServiceForVirtualMachineParams(d.Id(), serviceofferingid)

			// Change the service offering
			_, err = cs.VirtualMachine.ChangeServiceForVirtualMachine(p)
			if err != nil {
				return fmt.Errorf(
					"Error changing the service offering for instance %s: %s", name, err)
			}
			d.SetPartial("service_offering")
		}

		if d.HasChange("keypair") {
			log.Printf("[DEBUG] SSH keypair changed for %s, starting update", name)

			p := cs.SSH.NewResetSSHKeyForVirtualMachineParams(d.Id(), d.Get("keypair").(string))

			// Change the ssh keypair
			_, err = cs.SSH.ResetSSHKeyForVirtualMachine(p)
			if err != nil {
				return fmt.Errorf(
					"Error changing the SSH keypair for instance %s: %s", name, err)
			}
			d.SetPartial("keypair")
		}

		// Start the virtual machine again
		_, err = cs.VirtualMachine.StartVirtualMachine(
			cs.VirtualMachine.NewStartVirtualMachineParams(d.Id()))
		if err != nil {
			return fmt.Errorf(
				"Error starting instance %s after making changes", name)
		}
	}

	d.Partial(false)
	return resourceCloudStackInstanceRead(d, meta)
}

func resourceCloudStackInstanceDelete(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Create a new parameter struct
	p := cs.VirtualMachine.NewDestroyVirtualMachineParams(d.Id())

	if d.Get("expunge").(bool) {
		p.SetExpunge(true)
	}

	log.Printf("[INFO] Destroying instance: %s", d.Get("name").(string))
	if _, err := cs.VirtualMachine.DestroyVirtualMachine(p); err != nil {
		// This is a very poor way to be told the ID does no longer exist :(
		if strings.Contains(err.Error(), fmt.Sprintf(
			"Invalid parameter id value=%s due to incorrect long value format, "+
				"or entity does not exist", d.Id())) {
			return nil
		}

		return fmt.Errorf("Error destroying instance: %s", err)
	}

	return nil
}
