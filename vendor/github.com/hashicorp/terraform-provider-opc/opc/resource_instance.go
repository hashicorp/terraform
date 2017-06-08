package opc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceInstance() *schema.Resource {
	return &schema.Resource{
		Create: resourceInstanceCreate,
		Read:   resourceInstanceRead,
		Update: resourceInstanceUpdate,
		Delete: resourceInstanceDelete,
		Importer: &schema.ResourceImporter{
			State: func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				combined := strings.Split(d.Id(), "/")
				if len(combined) != 2 {
					return nil, fmt.Errorf("Invalid ID specified. Must be in the form of instance_name/instance_id. Got: %s", d.Id())
				}
				d.Set("name", combined[0])
				d.SetId(combined[1])
				return []*schema.ResourceData{d}, nil
			},
		},

		Schema: map[string]*schema.Schema{
			/////////////////////////
			// Required Attributes //
			/////////////////////////
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"shape": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			/////////////////////////
			// Optional Attributes //
			/////////////////////////
			"instance_attributes": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.ValidateJsonString,
			},

			"boot_order": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeInt},
			},

			"hostname": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"image_list": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"label": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"desired_state": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  compute.InstanceDesiredRunning,
			},

			"networking_info": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"dns": {
							// Required for Shared Network Interface, will default if unspecified, however
							// Optional for IP Network Interface
							Type:     schema.TypeList,
							Optional: true,
							Computed: true,
							ForceNew: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},

						"index": {
							Type:     schema.TypeInt,
							ForceNew: true,
							Required: true,
						},

						"ip_address": {
							// Optional, IP Network only
							Type:     schema.TypeString,
							ForceNew: true,
							Optional: true,
						},

						"ip_network": {
							// Required for an IP Network Interface
							Type:     schema.TypeString,
							ForceNew: true,
							Optional: true,
						},

						"mac_address": {
							// Optional, IP Network Only
							Type:     schema.TypeString,
							ForceNew: true,
							Computed: true,
							Optional: true,
						},

						"name_servers": {
							// Optional, IP Network + Shared Network
							Type:     schema.TypeList,
							Optional: true,
							ForceNew: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},

						"nat": {
							// Optional for IP Network
							// Required for Shared Network
							Type:     schema.TypeList,
							Optional: true,
							ForceNew: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},

						"search_domains": {
							// Optional, IP Network + Shared Network
							Type:     schema.TypeList,
							Optional: true,
							ForceNew: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},

						"sec_lists": {
							// Required, Shared Network only. Will default if unspecified however
							Type:     schema.TypeList,
							Optional: true,
							Computed: true,
							ForceNew: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},

						"shared_network": {
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
							Default:  false,
						},

						"vnic": {
							// Optional, IP Network only.
							Type:     schema.TypeString,
							ForceNew: true,
							Optional: true,
						},

						"vnic_sets": {
							// Optional, IP Network only.
							Type:     schema.TypeList,
							Optional: true,
							ForceNew: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
				Set: func(v interface{}) int {
					var buf bytes.Buffer
					m := v.(map[string]interface{})
					buf.WriteString(fmt.Sprintf("%d-", m["index"].(int)))
					buf.WriteString(fmt.Sprintf("%s-", m["vnic"].(string)))
					buf.WriteString(fmt.Sprintf("%s-", m["nat"]))
					return hashcode.String(buf.String())
				},
			},

			"reverse_dns": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
				ForceNew: true,
			},

			"ssh_keys": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"storage": {
				Type:     schema.TypeSet,
				Optional: true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					desired := compute.InstanceDesiredState(d.Get("desired_state").(string))
					state := compute.InstanceState(d.Get("state").(string))
					if desired == compute.InstanceDesiredShutdown || state == compute.InstanceShutdown {
						return true
					}
					return false
				},
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"index": {
							Type:         schema.TypeInt,
							Required:     true,
							ForceNew:     true,
							ValidateFunc: validation.IntBetween(1, 10),
						},
						"volume": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},

			"tags": tagsForceNewSchema(),

			/////////////////////////
			// Computed Attributes //
			/////////////////////////
			"attributes": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"availability_domain": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"domain": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"entry": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"fingerprint": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"fqdn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"image_format": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"ip_address": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"placement_requirements": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"platform": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"priority": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"quota_reservation": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"relationships": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"resolvers": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"site": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"start_time": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"state": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"vcable": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"virtio": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"vnc_address": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceInstanceCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).Instances()

	// Get Required Attributes
	input := &compute.CreateInstanceInput{
		Name:  d.Get("name").(string),
		Shape: d.Get("shape").(string),
	}

	// Get optional instance attributes
	attributes, attrErr := getInstanceAttributes(d)
	if attrErr != nil {
		return attrErr
	}

	if attributes != nil {
		input.Attributes = attributes
	}

	if bootOrder := getIntList(d, "boot_order"); len(bootOrder) > 0 {
		input.BootOrder = bootOrder
	}

	if v, ok := d.GetOk("hostname"); ok {
		input.Hostname = v.(string)
	}

	if v, ok := d.GetOk("image_list"); ok {
		input.ImageList = v.(string)
	}

	if v, ok := d.GetOk("label"); ok {
		input.Label = v.(string)
	}

	interfaces, err := readNetworkInterfacesFromConfig(d)
	if err != nil {
		return err
	}
	if interfaces != nil {
		input.Networking = interfaces
	}

	if v, ok := d.GetOk("reverse_dns"); ok {
		input.ReverseDNS = v.(bool)
	}

	if sshKeys := getStringList(d, "ssh_keys"); len(sshKeys) > 0 {
		input.SSHKeys = sshKeys
	}

	storage := getStorageAttachments(d)
	if len(storage) > 0 {
		input.Storage = storage
	}

	if tags := getStringList(d, "tags"); len(tags) > 0 {
		input.Tags = tags
	}

	result, err := client.CreateInstance(input)
	if err != nil {
		return fmt.Errorf("Error creating instance %s: %s", input.Name, err)
	}

	log.Printf("[DEBUG] Created instance %s: %#v", input.Name, result.ID)

	d.SetId(result.ID)

	return resourceInstanceRead(d, meta)
}

func resourceInstanceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).Instances()

	name := d.Get("name").(string)

	input := &compute.GetInstanceInput{
		ID:   d.Id(),
		Name: name,
	}

	log.Printf("[DEBUG] Reading state of instance %s", name)
	result, err := client.GetInstance(input)
	if err != nil {
		// Instance doesn't exist
		if compute.WasNotFoundError(err) {
			log.Printf("[DEBUG] Instance %s not found", name)
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading instance %s: %s", name, err)
	}

	log.Printf("[DEBUG] Instance '%s' found", name)

	// Update attributes
	return updateInstanceAttributes(d, result)
}

func updateInstanceAttributes(d *schema.ResourceData, instance *compute.InstanceInfo) error {
	d.Set("name", instance.Name)
	d.Set("shape", instance.Shape)

	if err := setInstanceAttributes(d, instance.Attributes); err != nil {
		return err
	}

	if attrs, ok := d.GetOk("instance_attributes"); ok && attrs != nil {
		d.Set("instance_attributes", attrs.(string))
	}

	if err := setIntList(d, "boot_order", instance.BootOrder); err != nil {
		return err
	}

	split_hostname := strings.Split(instance.Hostname, ".")
	if len(split_hostname) == 0 {
		return fmt.Errorf("Unable to parse hostname: %s", instance.Hostname)
	}
	d.Set("hostname", split_hostname[0])
	d.Set("fqdn", instance.Hostname)
	d.Set("image_list", instance.ImageList)
	d.Set("label", instance.Label)

	if err := readNetworkInterfaces(d, instance.Networking); err != nil {
		return err
	}

	d.Set("reverse_dns", instance.ReverseDNS)
	if err := setStringList(d, "ssh_keys", instance.SSHKeys); err != nil {
		return err
	}

	if err := readStorageAttachments(d, instance.Storage); err != nil {
		return err
	}

	if err := setStringList(d, "tags", instance.Tags); err != nil {
		return err
	}
	d.Set("availability_domain", instance.AvailabilityDomain)
	d.Set("domain", instance.Domain)
	d.Set("entry", instance.Entry)
	d.Set("fingerprint", instance.Fingerprint)
	d.Set("image_format", instance.ImageFormat)
	d.Set("ip_address", instance.IPAddress)
	d.Set("desired_state", instance.DesiredState)

	if err := setStringList(d, "placement_requirements", instance.PlacementRequirements); err != nil {
		return err
	}

	d.Set("platform", instance.Platform)
	d.Set("priority", instance.Priority)
	d.Set("quota_reservation", instance.QuotaReservation)

	if err := setStringList(d, "relationships", instance.Relationships); err != nil {
		return err
	}

	if err := setStringList(d, "resolvers", instance.Resolvers); err != nil {
		return err
	}

	d.Set("site", instance.Site)
	d.Set("start_time", instance.StartTime)
	d.Set("state", instance.State)

	if err := setStringList(d, "tags", instance.Tags); err != nil {
		return err
	}

	d.Set("vcable", instance.VCableID)
	d.Set("virtio", instance.Virtio)
	d.Set("vnc_address", instance.VNC)

	return nil
}

func resourceInstanceUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).Instances()

	name := d.Get("name").(string)

	input := &compute.UpdateInstanceInput{
		Name: name,
		ID:   d.Id(),
	}

	if d.HasChange("desired_state") {
		input.DesiredState = compute.InstanceDesiredState(d.Get("desired_state").(string))
	}

	if d.HasChange("tags") {
		tags := getStringList(d, "tags")
		input.Tags = tags

	}

	result, err := client.UpdateInstance(input)
	if err != nil {
		return fmt.Errorf("Error updating instance %s: %s", input.Name, err)
	}

	log.Printf("[DEBUG] Updated instance %s: %#v", result.Name, result.ID)

	return resourceInstanceRead(d, meta)
}

func resourceInstanceDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).Instances()

	name := d.Get("name").(string)

	input := &compute.DeleteInstanceInput{
		ID:   d.Id(),
		Name: name,
	}
	log.Printf("[DEBUG] Deleting instance %s", name)

	if err := client.DeleteInstance(input); err != nil {
		return fmt.Errorf("Error deleting instance %s: %s", name, err)
	}

	return nil
}

func getStorageAttachments(d *schema.ResourceData) []compute.StorageAttachmentInput {
	storageAttachments := []compute.StorageAttachmentInput{}
	storage := d.Get("storage").(*schema.Set)
	for _, i := range storage.List() {
		attrs := i.(map[string]interface{})
		storageAttachments = append(storageAttachments, compute.StorageAttachmentInput{
			Index:  attrs["index"].(int),
			Volume: attrs["volume"].(string),
		})
	}
	return storageAttachments
}

// Parses instance_attributes from a string to a map[string]interface and returns any errors.
func getInstanceAttributes(d *schema.ResourceData) (map[string]interface{}, error) {
	var attrs map[string]interface{}

	// Empty instance attributes
	attributes, ok := d.GetOk("instance_attributes")
	if !ok {
		return attrs, nil
	}

	if err := json.Unmarshal([]byte(attributes.(string)), &attrs); err != nil {
		return attrs, fmt.Errorf("Cannot parse attributes as json: %s", err)
	}

	return attrs, nil
}

// Reads attributes from the returned instance object, and sets the computed attributes string
// as JSON
func setInstanceAttributes(d *schema.ResourceData, attributes map[string]interface{}) error {
	// Shouldn't ever get nil attributes on an instance, but protect against the case either way
	if attributes == nil {
		return nil
	}

	b, err := json.Marshal(attributes)
	if err != nil {
		return fmt.Errorf("Error marshalling returned attributes: %s", err)
	}
	return d.Set("attributes", string(b))
}

// Populates and validates shared network and ip network interfaces to return the of map
// objects needed to create/update an instance's networking_info
func readNetworkInterfacesFromConfig(d *schema.ResourceData) (map[string]compute.NetworkingInfo, error) {
	interfaces := make(map[string]compute.NetworkingInfo)

	if v, ok := d.GetOk("networking_info"); ok {
		vL := v.(*schema.Set).List()
		for _, v := range vL {
			ni := v.(map[string]interface{})
			index, ok := ni["index"].(int)
			if !ok {
				return nil, fmt.Errorf("Index not specified for network interface: %v", ni)
			}

			deviceIndex := fmt.Sprintf("eth%d", index)

			// Verify that the network interface doesn't already exist
			if _, ok := interfaces[deviceIndex]; ok {
				return nil, fmt.Errorf("Duplicate Network interface at eth%d already specified", index)
			}

			// Determine if we're creating a shared network interface or an IP Network interface
			info := compute.NetworkingInfo{}
			var err error
			if ni["shared_network"].(bool) {
				// Populate shared network parameters
				info, err = readSharedNetworkFromConfig(ni)
				// Set 'model' since we're configuring a shared network interface
				info.Model = compute.NICDefaultModel
			} else {
				// Populate IP Network Parameters
				info, err = readIPNetworkFromConfig(ni)
			}
			if err != nil {
				return nil, err
			}
			// And you may find yourself in a beautiful house, with a beautiful wife
			// And you may ask yourself, well, how did I get here?
			interfaces[deviceIndex] = info
		}
	}

	return interfaces, nil
}

// Reads a networking_info config block as a shared network interface
func readSharedNetworkFromConfig(ni map[string]interface{}) (compute.NetworkingInfo, error) {
	info := compute.NetworkingInfo{}
	// Validate the shared network
	if err := validateSharedNetwork(ni); err != nil {
		return info, err
	}
	// Populate shared network fields; checking type casting
	dns := []string{}
	if v, ok := ni["dns"]; ok && v != nil {
		for _, d := range v.([]interface{}) {
			dns = append(dns, d.(string))
		}
		if len(dns) > 0 {
			info.DNS = dns
		}
	}

	if v, ok := ni["model"].(string); ok && v != "" {
		info.Model = compute.NICModel(v)
	}

	nats := []string{}
	if v, ok := ni["nat"]; ok && v != nil {
		for _, nat := range v.([]interface{}) {
			nats = append(nats, nat.(string))
		}
		if len(nats) > 0 {
			info.Nat = nats
		}
	}

	slists := []string{}
	if v, ok := ni["sec_lists"]; ok && v != nil {
		for _, slist := range v.([]interface{}) {
			slists = append(slists, slist.(string))
		}
		if len(slists) > 0 {
			info.SecLists = slists
		}
	}

	nservers := []string{}
	if v, ok := ni["name_servers"]; ok && v != nil {
		for _, nserver := range v.([]interface{}) {
			nservers = append(nservers, nserver.(string))
		}
		if len(nservers) > 0 {
			info.NameServers = nservers
		}
	}

	sdomains := []string{}
	if v, ok := ni["search_domains"]; ok && v != nil {
		for _, sdomain := range v.([]interface{}) {
			sdomains = append(sdomains, sdomain.(string))
		}
		if len(sdomains) > 0 {
			info.SearchDomains = sdomains
		}
	}

	return info, nil
}

// Unfortunately this cannot take place during plan-phase, because we currently cannot have a validation
// function based off of multiple fields in the supplied schema.
func validateSharedNetwork(ni map[string]interface{}) error {
	// A Shared Networking Interface MUST have the following attributes set:
	// - "nat"
	// The following attributes _cannot_ be set for a shared network:
	// - "ip_address"
	// - "ip_network"
	// - "mac_address"
	// - "vnic"
	// - "vnic_sets"

	if _, ok := ni["nat"]; !ok {
		return fmt.Errorf("'nat' field needs to be set for a Shared Networking Interface")
	}

	// Strings only
	nilAttrs := []string{
		"ip_address",
		"ip_network",
		"mac_address",
		"vnic",
	}

	for _, v := range nilAttrs {
		if d, ok := ni[v]; ok && d.(string) != "" {
			return fmt.Errorf("%q field cannot be set in a Shared Networking Interface", v)
		}
	}
	if _, ok := ni["vnic_sets"].([]string); ok {
		return fmt.Errorf("%q field cannot be set in a Shared Networking Interface", "vnic_sets")
	}

	return nil
}

// Populates fields for an IP Network
func readIPNetworkFromConfig(ni map[string]interface{}) (compute.NetworkingInfo, error) {
	info := compute.NetworkingInfo{}
	// Validate the IP Network
	if err := validateIPNetwork(ni); err != nil {
		return info, err
	}
	// Populate fields
	if v, ok := ni["ip_network"].(string); ok && v != "" {
		info.IPNetwork = v
	}

	dns := []string{}
	if v, ok := ni["dns"]; ok && v != nil {
		for _, d := range v.([]interface{}) {
			dns = append(dns, d.(string))
		}
		if len(dns) > 0 {
			info.DNS = dns
		}
	}

	if v, ok := ni["ip_address"].(string); ok && v != "" {
		info.IPAddress = v
	}

	if v, ok := ni["mac_address"].(string); ok && v != "" {
		info.MACAddress = v
	}

	nservers := []string{}
	if v, ok := ni["name_servers"]; ok && v != nil {
		for _, nserver := range v.([]interface{}) {
			nservers = append(nservers, nserver.(string))
		}
		if len(nservers) > 0 {
			info.NameServers = nservers
		}
	}

	nats := []string{}
	if v, ok := ni["nat"]; ok && v != nil {
		for _, nat := range v.([]interface{}) {
			nats = append(nats, nat.(string))
		}
		if len(nats) > 0 {
			info.Nat = nats
		}
	}

	sdomains := []string{}
	if v, ok := ni["search_domains"]; ok && v != nil {
		for _, sdomain := range v.([]interface{}) {
			sdomains = append(sdomains, sdomain.(string))
		}
		if len(sdomains) > 0 {
			info.SearchDomains = sdomains
		}
	}

	if v, ok := ni["vnic"].(string); ok && v != "" {
		info.Vnic = v
	}

	vnicSets := []string{}
	if v, ok := ni["vnic_sets"]; ok && v != nil {
		for _, vnic := range v.([]interface{}) {
			vnicSets = append(vnicSets, vnic.(string))
		}
		if len(vnicSets) > 0 {
			info.VnicSets = vnicSets
		}
	}

	return info, nil
}

// Validates an IP Network config block
func validateIPNetwork(ni map[string]interface{}) error {
	// An IP Networking Interface MUST have the following attributes set:
	// - "ip_network"

	// Required to be set
	if d, ok := ni["ip_network"]; !ok || d.(string) == "" {
		return fmt.Errorf("'ip_network' field is required for an IP Network interface")
	}

	return nil
}

// Reads network interfaces from the config
func readNetworkInterfaces(d *schema.ResourceData, ifaces map[string]compute.NetworkingInfo) error {
	result := make([]map[string]interface{}, 0)

	// Nil check for import case
	if ifaces == nil {
		return d.Set("networking_info", result)
	}

	for index, iface := range ifaces {
		res := make(map[string]interface{})
		// The index returned from the SDK holds the full device_index from the instance.
		// For users convenience, we simply allow them to specify the integer equivalent of the device_index
		// so a user could implement several network interfaces via `count`.
		// Convert the full device_index `ethN` to `N` as an integer.
		index := strings.TrimPrefix(index, "eth")
		indexInt, err := strconv.Atoi(index)
		if err != nil {
			return err
		}
		res["index"] = indexInt

		// Set the proper attributes for this specific network interface
		if iface.DNS != nil {
			res["dns"] = iface.DNS
		}
		if iface.IPAddress != "" {
			res["ip_address"] = iface.IPAddress
		}
		if iface.IPNetwork != "" {
			res["ip_network"] = iface.IPNetwork
		}
		if iface.MACAddress != "" {
			res["mac_address"] = iface.MACAddress
		}
		if iface.Model != "" {
			// Model can only be set on Shared networks
			res["shared_network"] = true
		}
		if iface.NameServers != nil {
			res["name_servers"] = iface.NameServers
		}
		if iface.Nat != nil {
			res["nat"] = iface.Nat
		}
		if iface.SearchDomains != nil {
			res["search_domains"] = iface.SearchDomains
		}
		if iface.SecLists != nil {
			res["sec_lists"] = iface.SecLists
		}
		if iface.Vnic != "" {
			res["vnic"] = iface.Vnic
			// VNIC can only be set on an IP Network
			res["shared_network"] = false
		}
		if iface.VnicSets != nil {
			res["vnic_sets"] = iface.VnicSets
		}

		result = append(result, res)
	}

	return d.Set("networking_info", result)
}

// Flattens the returned slice of storage attachments to a map
func readStorageAttachments(d *schema.ResourceData, attachments []compute.StorageAttachment) error {
	result := make([]map[string]interface{}, 0)

	if attachments == nil || len(attachments) == 0 {
		return d.Set("storage", nil)
	}

	for _, attachment := range attachments {
		res := make(map[string]interface{})
		res["index"] = attachment.Index
		res["volume"] = attachment.StorageVolumeName
		res["name"] = attachment.Name
		result = append(result, res)
	}
	return d.Set("storage", result)
}
