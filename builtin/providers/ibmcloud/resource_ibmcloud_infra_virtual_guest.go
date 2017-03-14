package ibmcloud

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/softlayer/softlayer-go/datatypes"
	"github.com/softlayer/softlayer-go/filter"
	"github.com/softlayer/softlayer-go/helpers/product"
	"github.com/softlayer/softlayer-go/helpers/virtual"
	"github.com/softlayer/softlayer-go/services"
	"github.com/softlayer/softlayer-go/session"
	"github.com/softlayer/softlayer-go/sl"
)

const (
	staticIPRouted = "STATIC_IP_ROUTED"

	upgradeTransaction = "UPGRADE"
	pendingUpgrade     = "pending_upgrade"
	inProgressUpgrade  = "upgrade_started"

	activeTransaction = "active"
	idleTransaction   = "idle"

	virtualGuestAvailable    = "available"
	virtualGuestProvisioning = "provisioning"
)

func resourceIBMCloudInfraVirtualGuest() *schema.Resource {
	return &schema.Resource{
		Create:   resourceIBMCloudInfraVirtualGuestCreate,
		Read:     resourceIBMCloudInfraVirtualGuestRead,
		Update:   resourceIBMCloudInfraVirtualGuestUpdate,
		Delete:   resourceIBMCloudInfraVirtualGuestDelete,
		Exists:   resourceIBMCloudInfraVirtualGuestExists,
		Importer: &schema.ResourceImporter{},

		Schema: map[string]*schema.Schema{
			"hostname": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: genID,
				DiffSuppressFunc: func(k, o, n string, d *schema.ResourceData) bool {
					// FIXME: Work around another bug in terraform.
					// When a default function is used with an optional property,
					// terraform will always execute it on apply, even when the property
					// already has a value in the state for it. This causes a false diff.
					// Making the property Computed:true does not make a difference.
					if strings.HasPrefix(o, "terraformed-") && strings.HasPrefix(n, "terraformed-") {
						return true
					}
					return o == n
				},
			},

			"domain": {
				Type:     schema.TypeString,
				Required: true,
			},

			"os_reference_code": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"image_id"},
			},

			"hourly_billing": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
				ForceNew: true,
			},

			"private_network_only": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},

			"datacenter": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"cores": {
				Type:     schema.TypeInt,
				Required: true,
			},

			"memory": {
				Type:     schema.TypeInt,
				Required: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					memoryInMB := float64(v.(int))

					// Validate memory to match gigs format
					remaining := math.Mod(memoryInMB, 1024)
					if remaining > 0 {
						suggested := math.Ceil(memoryInMB/1024) * 1024
						errors = append(errors, fmt.Errorf(
							"Invalid 'memory' value %d megabytes, must be a multiple of 1024 (e.g. use %d)", int(memoryInMB), int(suggested)))
					}

					return
				},
			},

			"dedicated_acct_host_only": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"public_vlan_id": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"public_subnet": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"private_vlan_id": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"private_subnet": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"disks": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeInt},
			},

			"network_speed": {
				Type:     schema.TypeInt,
				Optional: true,
				DiffSuppressFunc: func(k, o, n string, d *schema.ResourceData) bool {
					if privateNetworkOnly, ok := d.GetOk("private_network_only"); ok {
						if privateNetworkOnly.(bool) {
							return true
						}
					}
					return o == n
				},
				Default:      100,
				ValidateFunc: validateNetworkSpeed,
			},

			"ipv4_address": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"ipv4_address_private": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"ip_address_id": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"ip_address_id_private": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"ipv6_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Default:  false,
			},

			"ipv6_address": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"ipv6_address_id": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			// SoftLayer does not support public_ipv6_subnet configuration in vm creation. So, public_ipv6_subnet
			// is defined as a computed parameter.
			"public_ipv6_subnet": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"secondary_ip_count": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
				DiffSuppressFunc: func(k, o, n string, d *schema.ResourceData) bool {
					// secondary_ip_count is only used when a virtual_guest resource is created.
					if d.State() == nil {
						return false
					}
					return true
				},
				ValidateFunc: validateSecondaryIPCount,
			},

			"secondary_ip_addresses": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"ssh_key_ids": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeInt},
			},

			"user_metadata": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"local_disk": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
				ForceNew: true,
			},

			"post_install_script_uri": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"image_id": {
				Type:          schema.TypeInt,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"os_reference_code"},
			},

			"tags": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"wait_time_minutes": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  90,
			},
		},
	}
}

func getSubnetID(subnet string, meta interface{}) (int, error) {
	service := services.GetAccountService(meta.(ClientSession).SoftLayerSession())

	subnetInfo := strings.Split(subnet, "/")
	if len(subnetInfo) != 2 {
		return 0, fmt.Errorf(
			"Unable to parse the provided subnet: %s", subnet)
	}

	networkIdentifier := subnetInfo[0]
	cidr := subnetInfo[1]

	subnets, err := service.
		Mask("id").
		Filter(
			filter.Build(
				filter.Path("subnets.cidr").Eq(cidr),
				filter.Path("subnets.networkIdentifier").Eq(networkIdentifier),
			),
		).
		GetSubnets()

	if err != nil {
		return 0, fmt.Errorf("Error looking up Subnet: %s", err)
	}

	if len(subnets) < 1 {
		return 0, fmt.Errorf(
			"Unable to locate a subnet matching the provided subnet: %s", subnet)
	}

	return *subnets[0].Id, nil
}

func getNameForBlockDevice(i int) string {
	// skip 1, which is reserved for the swap disk.
	// so we get 0, 2, 3, 4, 5 ...
	if i == 0 {
		return "0"
	}

	return strconv.Itoa(i + 1)
}

func getBlockDevices(d *schema.ResourceData) []datatypes.Virtual_Guest_Block_Device {
	numBlocks := d.Get("disks.#").(int)
	if numBlocks == 0 {
		return nil
	}

	blocks := make([]datatypes.Virtual_Guest_Block_Device, 0, numBlocks)
	for i := 0; i < numBlocks; i++ {
		blockRef := fmt.Sprintf("disks.%d", i)
		name := getNameForBlockDevice(i)
		capacity := d.Get(blockRef).(int)
		block := datatypes.Virtual_Guest_Block_Device{
			Device: &name,
			DiskImage: &datatypes.Virtual_Disk_Image{
				Capacity: &capacity,
			},
		}
		blocks = append(blocks, block)
	}
	return blocks
}
func getVirtualGuestTemplateFromResourceData(d *schema.ResourceData, meta interface{}) (datatypes.Virtual_Guest, error) {

	dc := datatypes.Location{
		Name: sl.String(d.Get("datacenter").(string)),
	}
	// FIXME: Work around bug in terraform (?)
	// For properties that have a default value set and a diff suppress function,
	// it is not using the default value.
	networkSpeed := d.Get("network_speed").(int)
	if networkSpeed == 0 {
		networkSpeed = resourceIBMCloudInfraVirtualGuest().Schema["network_speed"].Default.(int)
	}

	networkComponent := datatypes.Virtual_Guest_Network_Component{
		MaxSpeed: &networkSpeed,
	}

	opts := datatypes.Virtual_Guest{
		Hostname:               sl.String(d.Get("hostname").(string)),
		Domain:                 sl.String(d.Get("domain").(string)),
		HourlyBillingFlag:      sl.Bool(d.Get("hourly_billing").(bool)),
		PrivateNetworkOnlyFlag: sl.Bool(d.Get("private_network_only").(bool)),
		Datacenter:             &dc,
		StartCpus:              sl.Int(d.Get("cores").(int)),
		MaxMemory:              sl.Int(d.Get("memory").(int)),
		NetworkComponents:      []datatypes.Virtual_Guest_Network_Component{networkComponent},
		BlockDevices:           getBlockDevices(d),
		LocalDiskFlag:          sl.Bool(d.Get("local_disk").(bool)),
		PostInstallScriptUri:   sl.String(d.Get("post_install_script_uri").(string)),
	}

	if dedicatedAcctHostOnly, ok := d.GetOk("dedicated_acct_host_only"); ok {
		opts.DedicatedAccountHostOnlyFlag = sl.Bool(dedicatedAcctHostOnly.(bool))
	}

	if imgID, ok := d.GetOk("image_id"); ok {
		imageID := imgID.(int)
		service := services.
			GetVirtualGuestBlockDeviceTemplateGroupService(meta.(ClientSession).SoftLayerSession())

		image, err := service.
			Mask("id,globalIdentifier").Id(imageID).
			GetObject()
		if err != nil {
			return opts, fmt.Errorf("Error looking up image %d: %s", imageID, err)
		} else if image.GlobalIdentifier == nil {
			return opts, fmt.Errorf(
				"Image template %d does not have a global identifier", imageID)
		}

		opts.BlockDeviceTemplateGroup = &datatypes.Virtual_Guest_Block_Device_Template_Group{
			GlobalIdentifier: image.GlobalIdentifier,
		}
	}

	if operatingSystemReferenceCode, ok := d.GetOk("os_reference_code"); ok {
		opts.OperatingSystemReferenceCode = sl.String(operatingSystemReferenceCode.(string))
	}

	publicVlanID := d.Get("public_vlan_id").(int)
	publicSubnet := d.Get("public_subnet").(string)
	privateVlanID := d.Get("private_vlan_id").(int)
	privateSubnet := d.Get("private_subnet").(string)

	primaryNetworkComponent := datatypes.Virtual_Guest_Network_Component{
		NetworkVlan: &datatypes.Network_Vlan{},
	}

	if publicVlanID > 0 {
		primaryNetworkComponent.NetworkVlan.Id = &publicVlanID
	}

	// Apply public subnet if provided
	if publicSubnet != "" {
		primarySubnetID, err := getSubnetID(publicSubnet, meta)
		if err != nil {
			return opts, fmt.Errorf("Error creating virtual guest: %s", err)
		}
		primaryNetworkComponent.NetworkVlan.PrimarySubnetId = &primarySubnetID
	}

	if publicVlanID > 0 || publicSubnet != "" {
		opts.PrimaryNetworkComponent = &primaryNetworkComponent
	}

	primaryBackendNetworkComponent := datatypes.Virtual_Guest_Network_Component{
		NetworkVlan: &datatypes.Network_Vlan{},
	}

	if privateVlanID > 0 {
		primaryBackendNetworkComponent.NetworkVlan.Id = &privateVlanID
	}

	// Apply private subnet if provided
	if privateSubnet != "" {
		primarySubnetID, err := getSubnetID(privateSubnet, meta)
		if err != nil {
			return opts, fmt.Errorf("Error creating virtual guest: %s", err)
		}
		primaryBackendNetworkComponent.NetworkVlan.PrimarySubnetId = &primarySubnetID
	}

	if privateVlanID > 0 || privateSubnet != "" {
		opts.PrimaryBackendNetworkComponent = &primaryBackendNetworkComponent
	}

	if userData, ok := d.GetOk("user_metadata"); ok {
		opts.UserData = []datatypes.Virtual_Guest_Attribute{
			{
				Value: sl.String(userData.(string)),
			},
		}
	}

	// Get configured ssh_keys
	sshKeys := d.Get("ssh_key_ids").([]interface{})
	sshKeysLen := len(sshKeys)
	if sshKeysLen > 0 {
		opts.SshKeys = make([]datatypes.Security_Ssh_Key, 0, sshKeysLen)
		for _, sshKey := range sshKeys {
			opts.SshKeys = append(opts.SshKeys, datatypes.Security_Ssh_Key{
				Id: sl.Int(sshKey.(int)),
			})
		}
	}

	return opts, nil
}

func resourceIBMCloudInfraVirtualGuestCreate(d *schema.ResourceData, meta interface{}) error {
	service := services.GetVirtualGuestService(meta.(ClientSession).SoftLayerSession())
	sess := meta.(ClientSession).SoftLayerSession()

	opts, err := getVirtualGuestTemplateFromResourceData(d, meta)
	if err != nil {
		return err
	}

	log.Println("[INFO] Creating virtual machine")

	var id int
	var template datatypes.Container_Product_Order

	// Build an order template with a custom image.
	if opts.BlockDevices != nil && opts.BlockDeviceTemplateGroup != nil {
		bd := *opts.BlockDeviceTemplateGroup
		opts.BlockDeviceTemplateGroup = nil
		opts.OperatingSystemReferenceCode = sl.String("UBUNTU_LATEST")
		template, err = service.GenerateOrderTemplate(&opts)
		if err != nil {
			return fmt.Errorf("Error generating order template: %s", err)
		}

		// Remove temporary OS from actual order
		prices := make([]datatypes.Product_Item_Price, len(template.Prices))
		i := 0
		for _, p := range template.Prices {
			if !strings.Contains(*p.Item.Description, "Ubuntu") {
				prices[i] = p
				i++
			}
		}
		template.Prices = prices[:i]

		template.ImageTemplateId = sl.Int(d.Get("image_id").(int))
		template.VirtualGuests[0].BlockDeviceTemplateGroup = &bd
		template.VirtualGuests[0].OperatingSystemReferenceCode = nil
	} else {
		// Build an order template with os_reference_code
		template, err = service.GenerateOrderTemplate(&opts)
		if err != nil {
			return fmt.Errorf("Error generating order template: %s", err)
		}
	}

	// Add an IPv6 price item
	privateNetworkOnly := d.Get("private_network_only").(bool)

	if d.Get("ipv6_enabled").(bool) {
		if privateNetworkOnly {
			return fmt.Errorf("Unable to configure a public IPv6 address with a private_network_only option")
		}

		ipv6Items, err := services.GetProductPackageService(sess).
			Id(*template.PackageId).
			Mask("id,capacity,description,units,keyName,prices[id,categories[id,name,categoryCode]]").
			Filter(filter.Build(filter.Path("items.keyName").Eq("1_IPV6_ADDRESS"))).
			GetItems()
		if err != nil {
			return fmt.Errorf("Error generating order template: %s", err)
		}
		if len(ipv6Items) == 0 {
			return fmt.Errorf("No product items matching 1_IPV6_ADDRESS could be found")
		}

		template.Prices = append(template.Prices,
			datatypes.Product_Item_Price{
				Id: ipv6Items[0].Prices[0].Id,
			},
		)
	}

	// Configure secondary IPs
	secondaryIPCount := d.Get("secondary_ip_count").(int)
	if secondaryIPCount > 0 {
		if privateNetworkOnly {
			return fmt.Errorf("Unable to configure public secondary addresses with a private_network_only option")
		}
		staticIPItems, err := services.GetProductPackageService(sess).
			Id(*template.PackageId).
			Mask("id,capacity,description,units,keyName,prices[id,categories[id,name,categoryCode]]").
			Filter(filter.Build(filter.Path("items.keyName").Eq(strconv.Itoa(secondaryIPCount) + "_PUBLIC_IP_ADDRESSES"))).
			GetItems()
		if err != nil {
			return fmt.Errorf("Error generating order template: %s", err)
		}
		if len(staticIPItems) == 0 {
			return fmt.Errorf("No product items matching %d_PUBLIC_IP_ADDRESSES could be found", secondaryIPCount)
		}

		template.Prices = append(template.Prices,
			datatypes.Product_Item_Price{
				Id: staticIPItems[0].Prices[0].Id,
			},
		)
	}

	// GenerateOrderTemplate omits UserData, subnet, and maxSpeed, so configure virtual_guest.
	template.VirtualGuests[0] = opts

	order := &datatypes.Container_Product_Order_Virtual_Guest{
		Container_Product_Order_Hardware_Server: datatypes.Container_Product_Order_Hardware_Server{Container_Product_Order: template},
	}

	orderService := services.GetProductOrderService(sess)
	receipt, err := orderService.PlaceOrder(order, sl.Bool(false))
	if err != nil {
		return fmt.Errorf("Error ordering virtual guest: %s", err)
	}
	id = *receipt.OrderDetails.VirtualGuests[0].Id

	d.SetId(fmt.Sprintf("%d", id))

	log.Printf("[INFO] Virtual Machine ID: %s", d.Id())

	// Set tags
	tags := getTags(d)
	if tags != "" {
		//Try setting only when it is non empty as we are creating virtual guest
		err = setGuestTags(id, tags, meta)
		if err != nil {
			return err
		}
	}

	// wait for machine availability

	_, err = WaitForVirtualGuestAvailable(d, meta)

	if err != nil {
		return fmt.Errorf(
			"Error waiting for virtual machine (%s) to become ready: %s", d.Id(), err)
	}

	return resourceIBMCloudInfraVirtualGuestRead(d, meta)
}

func resourceIBMCloudInfraVirtualGuestRead(d *schema.ResourceData, meta interface{}) error {
	service := services.GetVirtualGuestService(meta.(ClientSession).SoftLayerSession())

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return fmt.Errorf("Not a valid ID, must be an integer: %s", err)
	}

	result, err := service.Id(id).Mask(
		"hostname,domain,startCpus,maxMemory,dedicatedAccountHostOnlyFlag," +
			"primaryIpAddress,primaryBackendIpAddress,privateNetworkOnlyFlag," +
			"hourlyBillingFlag,localDiskFlag," +
			"userData[value],tagReferences[id,tag[name]]," +
			"datacenter[id,name,longName]," +
			"primaryNetworkComponent[networkVlan[id]," +
			"primaryVersion6IpAddressRecord[subnet,guestNetworkComponentBinding[ipAddressId]]," +
			"primaryIpAddressRecord[subnet,guestNetworkComponentBinding[ipAddressId]]]," +
			"primaryBackendNetworkComponent[networkVlan[id]," +
			"primaryIpAddressRecord[subnet,guestNetworkComponentBinding[ipAddressId]]]",
	).GetObject()

	if err != nil {
		return fmt.Errorf("Error retrieving virtual guest: %s", err)
	}

	d.Set("hostname", *result.Hostname)
	d.Set("domain", *result.Domain)

	if result.Datacenter != nil {
		d.Set("datacenter", *result.Datacenter.Name)
	}

	d.Set(
		"network_speed",
		sl.Grab(
			result,
			"PrimaryBackendNetworkComponent.MaxSpeed",
			d.Get("network_speed").(int),
		),
	)
	d.Set("cores", *result.StartCpus)
	d.Set("memory", *result.MaxMemory)
	d.Set("dedicated_acct_host_only", *result.DedicatedAccountHostOnlyFlag)
	if result.PrimaryIpAddress != nil {
		d.Set("has_public_ip", *result.PrimaryIpAddress != "")
		d.Set("ipv4_address", *result.PrimaryIpAddress)
	}
	d.Set("ipv4_address_private", *result.PrimaryBackendIpAddress)
	if result.PrimaryNetworkComponent.PrimaryIpAddressRecord != nil {
		d.Set("ip_address_id", *result.PrimaryNetworkComponent.PrimaryIpAddressRecord.GuestNetworkComponentBinding.IpAddressId)
	}
	d.Set("ip_address_id_private",
		*result.PrimaryBackendNetworkComponent.PrimaryIpAddressRecord.GuestNetworkComponentBinding.IpAddressId)
	d.Set("private_network_only", *result.PrivateNetworkOnlyFlag)
	d.Set("hourly_billing", *result.HourlyBillingFlag)
	d.Set("local_disk", *result.LocalDiskFlag)

	if result.PrimaryNetworkComponent.NetworkVlan != nil {
		d.Set("public_vlan_id", *result.PrimaryNetworkComponent.NetworkVlan.Id)
	}

	d.Set("private_vlan_id", *result.PrimaryBackendNetworkComponent.NetworkVlan.Id)

	if result.PrimaryNetworkComponent.PrimaryIpAddressRecord != nil {
		publicSubnet := result.PrimaryNetworkComponent.PrimaryIpAddressRecord.Subnet
		d.Set(
			"public_subnet",
			fmt.Sprintf("%s/%d", *publicSubnet.NetworkIdentifier, *publicSubnet.Cidr),
		)
	}

	privateSubnet := result.PrimaryBackendNetworkComponent.PrimaryIpAddressRecord.Subnet
	d.Set(
		"private_subnet",
		fmt.Sprintf("%s/%d", *privateSubnet.NetworkIdentifier, *privateSubnet.Cidr),
	)

	d.Set("ipv6_enabled", false)
	if result.PrimaryNetworkComponent.PrimaryVersion6IpAddressRecord != nil {
		d.Set("ipv6_enabled", true)
		d.Set("ipv6_address", *result.PrimaryNetworkComponent.PrimaryVersion6IpAddressRecord.IpAddress)
		d.Set("ipv6_address_id", *result.PrimaryNetworkComponent.PrimaryVersion6IpAddressRecord.GuestNetworkComponentBinding.IpAddressId)
		publicSubnet := result.PrimaryNetworkComponent.PrimaryVersion6IpAddressRecord.Subnet
		d.Set(
			"public_ipv6_subnet",
			fmt.Sprintf("%s/%d", *publicSubnet.NetworkIdentifier, *publicSubnet.Cidr),
		)
	}

	userData := result.UserData
	if userData != nil && len(userData) > 0 {
		data, err := base64.StdEncoding.DecodeString(*userData[0].Value)
		if err != nil {
			log.Printf("Can't base64 decode user data %s. error: %s", *userData[0].Value, err)
			d.Set("user_metadata", *userData[0].Value)
		} else {
			d.Set("user_metadata", string(data))
		}
	}

	tagReferences := result.TagReferences
	tagReferencesLen := len(tagReferences)
	if tagReferencesLen > 0 {
		tags := make([]string, 0, tagReferencesLen)
		for _, tagRef := range tagReferences {
			tags = append(tags, *tagRef.Tag.Name)
		}
		d.Set("tags", tags)
	}

	// Set connection info
	connInfo := map[string]string{"type": "ssh"}
	if !*result.PrivateNetworkOnlyFlag && result.PrimaryIpAddress != nil {
		connInfo["host"] = *result.PrimaryIpAddress
	} else {
		connInfo["host"] = *result.PrimaryBackendIpAddress
	}
	d.SetConnInfo(connInfo)

	// Read secondary IP addresses
	d.Set("secondary_ip_addresses", nil)
	if result.PrimaryIpAddress != nil {
		secondarySubnetResult, err := services.GetAccountService(meta.(ClientSession).SoftLayerSession()).
			Mask("ipAddresses[id,ipAddress],subnetType").
			Filter(filter.Build(filter.Path("publicSubnets.endPointIpAddress.ipAddress").Eq(*result.PrimaryIpAddress))).
			GetPublicSubnets()
		if err != nil {
			log.Printf("Error getting secondary Ip addresses: %s", err)
		}

		secondaryIps := make([]string, 0)
		for _, subnet := range secondarySubnetResult {
			// Count static secondary ip addresses.
			if *subnet.SubnetType == staticIPRouted {
				for _, ipAddressObj := range subnet.IpAddresses {
					secondaryIps = append(secondaryIps, *ipAddressObj.IpAddress)
				}
			}
		}
		if len(secondaryIps) > 0 {
			d.Set("secondary_ip_addresses", secondaryIps)
			d.Set("secondary_ip_count", len(secondaryIps))
		}
	}

	return nil
}

func resourceIBMCloudInfraVirtualGuestUpdate(d *schema.ResourceData, meta interface{}) error {
	sess := meta.(ClientSession).SoftLayerSession()
	service := services.GetVirtualGuestService(sess)

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return fmt.Errorf("Not a valid ID, must be an integer: %s", err)
	}

	result, err := service.Id(id).GetObject()
	if err != nil {
		return fmt.Errorf("Error retrieving virtual guest: %s", err)
	}

	// Update "hostname" and "domain" fields if present and changed
	// Those are the only fields, which could be updated
	if d.HasChange("hostname") || d.HasChange("domain") {
		result.Hostname = sl.String(d.Get("hostname").(string))
		result.Domain = sl.String(d.Get("domain").(string))

		_, err = service.Id(id).EditObject(&result)

		if err != nil {
			return fmt.Errorf("Couldn't update virtual guest: %s", err)
		}
	}

	// Set user data if provided and not empty
	if d.HasChange("user_metadata") {
		_, err := service.Id(id).SetUserMetadata([]string{d.Get("user_metadata").(string)})
		if err != nil {
			return fmt.Errorf("Couldn't update user data for virtual guest: %s", err)
		}
	}

	// Update tags
	if d.HasChange("tags") {
		tags := getTags(d)
		err := setGuestTags(id, tags, meta)
		if err != nil {
			return err
		}
	}

	// Upgrade "cores", "memory" and "network_speed" if provided and changed
	upgradeOptions := map[string]float64{}
	if d.HasChange("cores") {
		upgradeOptions[product.CPUCategoryCode] = float64(d.Get("cores").(int))
	}

	if d.HasChange("memory") {
		memoryInMB := float64(d.Get("memory").(int))

		// Convert memory to GB, as softlayer only allows to upgrade RAM in Gigs
		// Must be already validated at this step
		upgradeOptions[product.MemoryCategoryCode] = float64(int(memoryInMB / 1024))
	}

	if d.HasChange("network_speed") {
		upgradeOptions[product.NICSpeedCategoryCode] = float64(d.Get("network_speed").(int))
	}

	if len(upgradeOptions) > 0 {
		_, err = virtual.UpgradeVirtualGuest(sess, &result, upgradeOptions)
		if err != nil {
			return fmt.Errorf("Couldn't upgrade virtual guest: %s", err)
		}

		// Wait for softlayer to start upgrading...
		_, err = WaitForUpgradeTransactionsToAppear(d, meta)

		// Wait for upgrade transactions to finish
		_, err = WaitForNoActiveTransactions(d, meta)

		return err
	}

	return resourceIBMCloudInfraVirtualGuestRead(d, meta)
}

func resourceIBMCloudInfraVirtualGuestDelete(d *schema.ResourceData, meta interface{}) error {
	service := services.GetVirtualGuestService(meta.(ClientSession).SoftLayerSession())

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return fmt.Errorf("Not a valid ID, must be an integer: %s", err)
	}

	_, err = WaitForNoActiveTransactions(d, meta)

	if err != nil {
		return fmt.Errorf("Error deleting virtual guest, couldn't wait for zero active transactions: %s", err)
	}

	ok, err := service.Id(id).DeleteObject()

	if err != nil {
		return fmt.Errorf("Error deleting virtual guest: %s", err)
	}

	if !ok {
		return fmt.Errorf(
			"API reported it was unsuccessful in removing the virtual guest '%d'", id)
	}

	return nil
}

//genID generates a random string to be used for the optional
//hostname
func genID() (interface{}, error) {
	numBytes := 8
	bytes := make([]byte, numBytes)
	n, err := rand.Reader.Read(bytes)
	if err != nil {
		return nil, err
	}

	if n != numBytes {
		return nil, errors.New("generated insufficient random bytes")
	}

	hexStr := hex.EncodeToString(bytes)
	return fmt.Sprintf("terraformed-%s", hexStr), nil
}

// WaitForUpgradeTransactionsToAppear Wait for upgrade transactions
func WaitForUpgradeTransactionsToAppear(d *schema.ResourceData, meta interface{}) (interface{}, error) {
	log.Printf("Waiting for server (%s) to have upgrade transactions", d.Id())

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return nil, fmt.Errorf("The instance ID %s must be numeric", d.Id())
	}
	stateConf := &resource.StateChangeConf{
		Pending: []string{"retry", pendingUpgrade},
		Target:  []string{inProgressUpgrade},
		Refresh: func() (interface{}, string, error) {
			service := services.GetVirtualGuestService(meta.(ClientSession).SoftLayerSession())
			transactions, err := service.Id(id).GetActiveTransactions()
			if err != nil {
				if apiErr, ok := err.(sl.Error); ok && apiErr.StatusCode == 404 {
					return nil, "", fmt.Errorf("Couldn't fetch active transactions: %s", err)
				}
				return false, "retry", nil
			}
			for _, transaction := range transactions {
				if strings.Contains(*transaction.TransactionStatus.Name, upgradeTransaction) {
					return transactions, inProgressUpgrade, nil
				}
			}
			return transactions, pendingUpgrade, nil
		},
		Timeout:    10 * time.Minute,
		Delay:      5 * time.Second,
		MinTimeout: 5 * time.Second,
	}

	return stateConf.WaitForState()
}

// WaitForNoActiveTransactions Wait for no active transactions
func WaitForNoActiveTransactions(d *schema.ResourceData, meta interface{}) (interface{}, error) {
	log.Printf("Waiting for server (%s) to have zero active transactions", d.Id())
	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return nil, fmt.Errorf("The instance ID %s must be numeric", d.Id())
	}
	stateConf := &resource.StateChangeConf{
		Pending: []string{"retry", activeTransaction},
		Target:  []string{idleTransaction},
		Refresh: func() (interface{}, string, error) {
			service := services.GetVirtualGuestService(meta.(ClientSession).SoftLayerSession())
			transactions, err := service.Id(id).GetActiveTransactions()
			if err != nil {
				if apiErr, ok := err.(sl.Error); ok && apiErr.StatusCode == 404 {
					return nil, "", fmt.Errorf("Couldn't get active transactions: %s", err)
				}
				return false, "retry", nil
			}
			if len(transactions) == 0 {
				return transactions, idleTransaction, nil
			}
			return transactions, activeTransaction, nil
		},
		Timeout:    time.Duration(d.Get("wait_time_minutes").(int)) * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 10 * time.Second,
	}

	return stateConf.WaitForState()
}

// WaitForVirtualGuestAvailable Waits for virtual guest creation
func WaitForVirtualGuestAvailable(d *schema.ResourceData, meta interface{}) (interface{}, error) {
	log.Printf("Waiting for server (%s) to be available.", d.Id())
	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return nil, fmt.Errorf("The instance ID %s must be numeric", d.Id())
	}
	sess := meta.(ClientSession).SoftLayerSession()
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"retry", virtualGuestProvisioning},
		Target:     []string{virtualGuestAvailable},
		Refresh:    virtualGuestStateRefreshFunc(sess, id, d),
		Timeout:    time.Duration(d.Get("wait_time_minutes").(int)) * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 10 * time.Second,
	}

	return stateConf.WaitForState()
}

func virtualGuestStateRefreshFunc(sess *session.Session, instanceID int, d *schema.ResourceData) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		// Check active transactions
		publicNetwork := !d.Get("private_network_only").(bool)
		service := services.GetVirtualGuestService(sess)
		result, err := service.Id(instanceID).Mask("activeTransaction,primaryBackendIpAddress,primaryIpAddress").GetObject()
		if err != nil {
			if apiErr, ok := err.(sl.Error); ok && apiErr.StatusCode == 404 {
				return nil, "", fmt.Errorf("Error retrieving virtual guest: %s", err)
			}
			return false, "retry", nil
		}
		// Check active transactions
		log.Println("Checking active transactions.")
		if result.ActiveTransaction != nil {
			return result, virtualGuestProvisioning, nil
		}

		// Check Primary IP address availability.
		log.Println("Checking primary backend IP address.")
		if result.PrimaryBackendIpAddress == nil {
			return result, virtualGuestProvisioning, nil
		}

		log.Println("Checking primary IP address.")
		if publicNetwork && result.PrimaryIpAddress == nil {
			return result, virtualGuestProvisioning, nil
		}

		// Check Secondary IP address availability.
		if d.Get("secondary_ip_count").(int) > 0 {
			log.Println("Refreshing secondary IPs state.")
			secondarySubnetResult, err := services.GetAccountService(sess).
				Mask("ipAddresses[id,ipAddress]").
				Filter(filter.Build(filter.Path("publicSubnets.endPointIpAddress.virtualGuest.id").Eq(d.Id()))).
				GetPublicSubnets()
			if err != nil {
				return nil, "", fmt.Errorf("Error retrieving secondary ip address: %s", err)
			}
			if len(secondarySubnetResult) == 0 {
				return result, virtualGuestProvisioning, nil
			}
		}

		return result, virtualGuestAvailable, nil
	}
}

func resourceIBMCloudInfraVirtualGuestExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	service := services.GetVirtualGuestService(meta.(ClientSession).SoftLayerSession())
	guestID, err := strconv.Atoi(d.Id())
	if err != nil {
		return false, fmt.Errorf("Not a valid ID, must be an integer: %s", err)
	}

	result, err := service.Id(guestID).GetObject()
	if err != nil {
		if apiErr, ok := err.(sl.Error); ok {
			if apiErr.StatusCode == 404 {
				return false, nil
			}
		}
		return false, fmt.Errorf("Error communicating with the API: %s", err)
	}

	return result.Id != nil && *result.Id == guestID, nil
}

func getTags(d *schema.ResourceData) string {
	tagSet := d.Get("tags").(*schema.Set)

	if tagSet.Len() == 0 {
		return ""
	}

	tags := make([]string, 0, tagSet.Len())
	for _, elem := range tagSet.List() {
		tag := elem.(string)
		tags = append(tags, tag)
	}
	return strings.Join(tags, ",")
}

func setGuestTags(id int, tags string, meta interface{}) error {
	service := services.GetVirtualGuestService(meta.(ClientSession).SoftLayerSession())
	_, err := service.Id(id).SetTags(sl.String(tags))
	if err != nil {
		return fmt.Errorf("Could not set tags on virtual guest %d", id)
	}
	return nil
}

func validateSecondaryIPCount(v interface{}, k string) (ws []string, errors []error) {
	value := v.(int)
	if !(value == 4 || value == 8) {
		errors = append(errors, fmt.Errorf("secondary_ip_count must be either 4 or 8. Provided value is %d", value))
	}
	return
}

func validateNetworkSpeed(v interface{}, k string) (ws []string, errors []error) {
	value := v.(int)
	switch value {
	case 10:
	case 100:
	case 1000:
	default:
		errors = append(errors, fmt.Errorf("network_speed permissible values are 10, 100 and 1000. The values are in Mbps. Provided value is %d", value))
	}
	return
}
