package openstack

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/floatingip"
)

func resourceComputeFloatingIPV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeFloatingIPV2Create,
		Read:   resourceComputeFloatingIPV2Read,
		Update: resourceComputeFloatingIPV2Update,
		Delete: resourceComputeFloatingIPV2Delete,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: schema.EnvDefaultFunc("OS_REGION_NAME", ""),
			},

			"pool": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: schema.EnvDefaultFunc("OS_POOL_NAME", nil),
			},

			"address": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"fixed_ip": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"instance_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceComputeFloatingIPV2Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	createOpts := &floatingip.CreateOpts{
		Pool: d.Get("pool").(string),
	}
	log.Printf("[DEBUG] Create Options: %#v", createOpts)
	newFip, err := floatingip.Create(computeClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating Floating IP: %s", err)
	}

	d.SetId(newFip.ID)

	// With the Floating IP created, if an Instance and Fixed IP were specified,
	// attach the Floating IP to the instance.

	// But first, ensure both an instance and fixed IP were specified.
	missingAttachInfo := false
	fixedIP := d.Get("fixed_ip").(string)
	instanceID := d.Get("instance_id").(string)

	// Is there an easier way to do this that I'm overlooking?
	if fixedIP != "" && instanceID == "" {
		missingAttachInfo = true
	}

	if instanceID != "" && fixedIP == "" {
		missingAttachInfo = true
	}

	if missingAttachInfo {
		return fmt.Errorf("Both a Fixed IP and Instance ID are required for Floating IP association")
	}

	if instanceID != "" && fixedIP != "" {
		log.Printf("[DEBUG] Attempting to associate %s to instance %s on fixed IP %s", newFip.IP, instanceID, fixedIP)
		if err := associateFloatingIPToInstance(computeClient, newFip.IP, instanceID, fixedIP); err != nil {
			return fmt.Errorf("Error associating Floating IP %s to Instance %s on Fixed IP %s", newFip.IP, instanceID, fixedIP)
		}
	} else {
		log.Printf("[DEBUG] Neither an instance ID nor a Fixed IP were specified. Not attaching.")
	}

	return resourceComputeFloatingIPV2Read(d, meta)
}

func resourceComputeFloatingIPV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	fip, err := floatingip.Get(computeClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "floating ip")
	}

	log.Printf("[DEBUG] Retrieved Floating IP %s: %+v", d.Id(), fip)

	d.Set("pool", fip.Pool)
	d.Set("address", fip.IP)

	return nil
}

func resourceComputeFloatingIPV2Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	fip, err := floatingip.Get(computeClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "floating ip")
	}

	changeOccurred := false
	newInstanceID := fip.InstanceID
	oldInstanceID := fip.InstanceID
	newFixedIP := fip.FixedIP
	oldFixedIP := fip.FixedIP

	if d.HasChange("instance_id") {
		o, n := d.GetChange("instance_id")
		oldInstanceID = o.(string)
		newInstanceID = n.(string)
		changeOccurred = true
	}

	if d.HasChange("fixed_ip") {
		o, n := d.GetChange("fixed_ip")
		oldFixedIP = o.(string)
		newFixedIP = n.(string)
		changeOccurred = true
	}

	if changeOccurred {
		// Disassociate the IP from the instance
		if oldInstanceID != "" && oldFixedIP != "" {
			log.Printf("[DEBUG] Attempting to disassociate %s from instance %s on fixed IP %s", fip.IP, oldInstanceID, oldFixedIP)
			if err := disassociateFloatingIPFromInstance(computeClient, fip.IP, oldInstanceID, oldFixedIP); err != nil {
				return fmt.Errorf("Error disassociating Floating IP %s from Instance %s on Fixed IP %s", fip.IP, oldInstanceID, oldFixedIP)
			}
		}

		// Associate the IP to the new information
		if newInstanceID != "" && newFixedIP != "" {
			log.Printf("[DEBUG] Attempting to associate %s to instance %s on fixed IP %s", fip.IP, oldInstanceID, oldFixedIP)

			if err := associateFloatingIPToInstance(computeClient, fip.IP, newInstanceID, newFixedIP); err != nil {
				return fmt.Errorf("Error associating Floating IP %s to Instance %s on Fixed IP %s", fip.IP, newInstanceID, newFixedIP)
			}
		}

	}

	return nil
}

func resourceComputeFloatingIPV2Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	log.Printf("[DEBUG] Deleting Floating IP %s", d.Id())
	if err := floatingip.Delete(computeClient, d.Id()).ExtractErr(); err != nil {
		return fmt.Errorf("Error deleting Floating IP: %s", err)
	}

	return nil
}

func associateFloatingIPToInstance(computeClient *gophercloud.ServiceClient, floatingIP string, instanceID string, fixedIP string) error {
	associateOpts := floatingip.AssociateOpts{
		ServerID:   instanceID,
		FloatingIP: floatingIP,
		FixedIP:    fixedIP,
	}

	if err := floatingip.AssociateInstance(computeClient, associateOpts).ExtractErr(); err != nil {
		return fmt.Errorf("Error associating floating IP: %s", err)
	}

	return nil
}

func disassociateFloatingIPFromInstance(computeClient *gophercloud.ServiceClient, floatingIP string, instanceID string, fixedIP string) error {
	associateOpts := floatingip.AssociateOpts{
		ServerID:   instanceID,
		FloatingIP: floatingIP,
		FixedIP:    fixedIP,
	}

	if err := floatingip.DisassociateInstance(computeClient, associateOpts).ExtractErr(); err != nil {
		return fmt.Errorf("Error disassociating floating IP: %s", err)
	}

	return nil
}
