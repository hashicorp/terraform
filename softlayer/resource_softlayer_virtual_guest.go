package softlayer

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"encoding/base64"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	datatypes "github.com/maximilien/softlayer-go/data_types"
	"github.com/maximilien/softlayer-go/softlayer"
	"math"
	"strings"
)

func resourceSoftLayerVirtualGuest() *schema.Resource {
	return &schema.Resource{
		Create: resourceSoftLayerVirtualGuestCreate,
		Read:   resourceSoftLayerVirtualGuestRead,
		Update: resourceSoftLayerVirtualGuestUpdate,
		Delete: resourceSoftLayerVirtualGuestDelete,
		Exists: resourceSoftLayerVirtualGuestExists,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"domain": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"image": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"hourly_billing": &schema.Schema{
				Type:     schema.TypeBool,
				Required: true,
				ForceNew: true,
			},

			"private_network_only": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},

			"region": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"cpu": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				// TODO: This fields for now requires recreation, because currently for some reason SoftLayer resets "dedicated_acct_host_only"
				// TODO: flag to false, while upgrading CPUs. That problem is reported to SoftLayer team. "ForceNew" can be set back
				// TODO: to false as soon as it is fixed at their side. Also corresponding test for virtual guest upgrade will be uncommented.
				ForceNew: true,
			},

			"ram": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					memoryInMB := float64(v.(int))

					// Validate memory to match gigs format
					remaining := math.Mod(memoryInMB, 1024)
					if remaining > 0 {
						suggested := math.Ceil(memoryInMB/1024) * 1024
						errors = append(errors, fmt.Errorf(
							"Invalid 'ram' value %d megabytes, must be a multiple of 1024 (e.g. use %d)", int(memoryInMB), int(suggested)))
					}

					return
				},
			},

			"dedicated_acct_host_only": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"frontend_vlan_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"backend_vlan_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"disks": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeInt},
			},

			"public_network_speed": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  1000,
			},

			"ipv4_address": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"ipv4_address_private": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"ssh_keys": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeInt},
			},

			"user_data": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"local_disk": &schema.Schema{
				Type:     schema.TypeBool,
				Required: true,
				ForceNew: true,
			},

			"post_install_script_uri": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  nil,
				ForceNew: true,
			},

			"block_device_template_group_gid": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func getNameForBlockDevice(i int) string {
	// skip 1, which is reserved for the swap disk.
	// so we get 0, 2, 3, 4, 5 ...
	if i == 0 {
		return "0"
	} else {
		return strconv.Itoa(i + 1)
	}
}

func getBlockDevices(d *schema.ResourceData) []datatypes.BlockDevice {
	numBlocks := d.Get("disks.#").(int)
	if numBlocks == 0 {
		return nil
	} else {
		blocks := make([]datatypes.BlockDevice, 0, numBlocks)
		for i := 0; i < numBlocks; i++ {
			blockRef := fmt.Sprintf("disks.%d", i)
			name := getNameForBlockDevice(i)
			capacity := d.Get(blockRef).(int)
			block := datatypes.BlockDevice{
				Device: name,
				DiskImage: datatypes.DiskImage{
					Capacity: capacity,
				},
			}
			blocks = append(blocks, block)
		}
		return blocks
	}
}

func resourceSoftLayerVirtualGuestCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client).virtualGuestService
	if client == nil {
		return fmt.Errorf("The client was nil.")
	}

	dc := datatypes.Datacenter{
		Name: d.Get("region").(string),
	}

	networkComponent := datatypes.NetworkComponents{
		MaxSpeed: d.Get("public_network_speed").(int),
	}

	privateNetworkOnly := d.Get("private_network_only").(bool)
	opts := datatypes.SoftLayer_Virtual_Guest_Template{
		Hostname:               d.Get("name").(string),
		Domain:                 d.Get("domain").(string),
		HourlyBillingFlag:      d.Get("hourly_billing").(bool),
		PrivateNetworkOnlyFlag: privateNetworkOnly,
		Datacenter:             dc,
		StartCpus:              d.Get("cpu").(int),
		MaxMemory:              d.Get("ram").(int),
		NetworkComponents:      []datatypes.NetworkComponents{networkComponent},
		BlockDevices:           getBlockDevices(d),
		LocalDiskFlag:          d.Get("local_disk").(bool),
		PostInstallScriptUri:   d.Get("post_install_script_uri").(string),
	}

	if dedicatedAcctHostOnly, ok := d.GetOk("dedicated_acct_host_only"); ok {
		opts.DedicatedAccountHostOnlyFlag = dedicatedAcctHostOnly.(bool)
	}

	if globalIdentifier, ok := d.GetOk("block_device_template_group_gid"); ok {
		opts.BlockDeviceTemplateGroup = &datatypes.BlockDeviceTemplateGroup{
			GlobalIdentifier: globalIdentifier.(string),
		}
	}

	if operatingSystemReferenceCode, ok := d.GetOk("image"); ok {
		opts.OperatingSystemReferenceCode = operatingSystemReferenceCode.(string)
	}

	// Apply frontend VLAN if provided
	if param, ok := d.GetOk("frontend_vlan_id"); ok {
		frontendVlanId, err := strconv.Atoi(param.(string))
		if err != nil {
			return fmt.Errorf("Not a valid frontend ID, must be an integer: %s", err)
		}
		opts.PrimaryNetworkComponent = &datatypes.PrimaryNetworkComponent{
			NetworkVlan: datatypes.NetworkVlan{Id: (frontendVlanId)},
		}
	}

	// Apply backend VLAN if provided
	if param, ok := d.GetOk("backend_vlan_id"); ok {
		backendVlanId, err := strconv.Atoi(param.(string))
		if err != nil {
			return fmt.Errorf("Not a valid backend ID, must be an integer: %s", err)
		}
		opts.PrimaryBackendNetworkComponent = &datatypes.PrimaryBackendNetworkComponent{
			NetworkVlan: datatypes.NetworkVlan{Id: (backendVlanId)},
		}
	}

	if userData, ok := d.GetOk("user_data"); ok {
		opts.UserData = []datatypes.UserData{
			datatypes.UserData{
				Value: userData.(string),
			},
		}
	}

	// Get configured ssh_keys
	ssh_keys := d.Get("ssh_keys.#").(int)
	if ssh_keys > 0 {
		opts.SshKeys = make([]datatypes.SshKey, 0, ssh_keys)
		for i := 0; i < ssh_keys; i++ {
			key := fmt.Sprintf("ssh_keys.%d", i)
			id := d.Get(key).(int)
			sshKey := datatypes.SshKey{
				Id: id,
			}
			opts.SshKeys = append(opts.SshKeys, sshKey)
		}
	}

	log.Printf("[INFO] Creating virtual machine")

	guest, err := client.CreateObject(opts)

	if err != nil {
		return fmt.Errorf("Error creating virtual guest: %s", err)
	}

	d.SetId(fmt.Sprintf("%d", guest.Id))

	log.Printf("[INFO] Virtual Machine ID: %s", d.Id())

	// wait for machine availability
	_, err = WaitForNoActiveTransactions(d, meta)

	if err != nil {
		return fmt.Errorf(
			"Error waiting for virtual machine (%s) to become ready: %s", d.Id(), err)
	}

	if !privateNetworkOnly {
		_, err = WaitForPublicIpAvailable(d, meta)
		if err != nil {
			return fmt.Errorf(
				"Error waiting for virtual machine (%s) public ip to become ready: %s", d.Id(), err)
		}
	}

	return resourceSoftLayerVirtualGuestRead(d, meta)
}

func resourceSoftLayerVirtualGuestRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client).virtualGuestService
	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return fmt.Errorf("Not a valid ID, must be an integer: %s", err)
	}
	result, err := client.GetObject(id)
	if err != nil {
		return fmt.Errorf("Error retrieving virtual guest: %s", err)
	}

	d.Set("name", result.Hostname)
	d.Set("domain", result.Domain)
	if result.Datacenter != nil {
		d.Set("region", result.Datacenter.Name)
	}
	d.Set("public_network_speed", result.NetworkComponents[0].MaxSpeed)
	d.Set("cpu", result.StartCpus)
	d.Set("ram", result.MaxMemory)
	d.Set("dedicated_acct_host_only", result.DedicatedAccountHostOnlyFlag)
	d.Set("has_public_ip", result.PrimaryIpAddress != "")
	d.Set("ipv4_address", result.PrimaryIpAddress)
	d.Set("ipv4_address_private", result.PrimaryBackendIpAddress)
	d.Set("private_network_only", result.PrivateNetworkOnlyFlag)
	d.Set("hourly_billing", result.HourlyBillingFlag)
	d.Set("local_disk", result.LocalDiskFlag)
	d.Set("frontend_vlan_id", result.PrimaryNetworkComponent.NetworkVlan.Id)
	d.Set("backend_vlan_id", result.PrimaryBackendNetworkComponent.NetworkVlan.Id)

	userData := result.UserData
	if userData != nil && len(userData) > 0 {
		data, err := base64.StdEncoding.DecodeString(userData[0].Value)
		if err != nil {
			log.Printf("Can't base64 decode user data %s. error: %s", userData, err)
			d.Set("user_data", userData)
		} else {
			d.Set("user_data", string(data))
		}
	}

	return nil
}

func resourceSoftLayerVirtualGuestUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client).virtualGuestService
	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return fmt.Errorf("Not a valid ID, must be an integer: %s", err)
	}
	result, err := client.GetObject(id)
	if err != nil {
		return fmt.Errorf("Error retrieving virtual guest: %s", err)
	}

	// Update "name" and "domain" fields if present and changed
	// Those are the only fields, which could be updated
	if d.HasChange("name") || d.HasChange("domain") {
		result.Hostname = d.Get("name").(string)
		result.Domain = d.Get("domain").(string)

		_, err = client.EditObject(id, result)

		if err != nil {
			return fmt.Errorf("Couldn't update virtual guest: %s", err)
		}
	}

	// Set user data if provided and not empty
	if d.HasChange("user_data") {
		client.SetMetadata(id, d.Get("user_data").(string))
	}

	// Upgrade "cpu", "ram" and "nic_speed" if provided and changed
	upgradeOptions := softlayer.UpgradeOptions{}
	if d.HasChange("cpu") {
		upgradeOptions.Cpus = d.Get("cpu").(int)
	}
	if d.HasChange("ram") {
		memoryInMB := float64(d.Get("ram").(int))

		// Convert memory to GB, as softlayer only allows to upgrade RAM in Gigs
		// Must be already validated at this step
		upgradeOptions.MemoryInGB = int(memoryInMB / 1024)
	}
	if d.HasChange("public_network_speed") {
		upgradeOptions.NicSpeed = d.Get("public_network_speed").(int)
	}

	started, err := client.UpgradeObject(id, &upgradeOptions)
	if err != nil {
		return fmt.Errorf("Couldn't upgrade virtual guest: %s", err)
	}

	if started {
		// Wait for softlayer to start upgrading...
		_, err = WaitForUpgradeTransactionsToAppear(d, meta)

		// Wait for upgrade transactions to finish
		_, err = WaitForNoActiveTransactions(d, meta)
	}

	return err
}

func resourceSoftLayerVirtualGuestDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client).virtualGuestService
	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return fmt.Errorf("Not a valid ID, must be an integer: %s", err)
	}

	_, err = WaitForNoActiveTransactions(d, meta)

	if err != nil {
		return fmt.Errorf("Error deleting virtual guest, couldn't wait for zero active transactions: %s", err)
	}

	_, err = client.DeleteObject(id)

	if err != nil {
		return fmt.Errorf("Error deleting virtual guest: %s", err)
	}

	return nil
}

func WaitForUpgradeTransactionsToAppear(d *schema.ResourceData, meta interface{}) (interface{}, error) {

	log.Printf("Waiting for server (%s) to have upgrade transactions", d.Id())

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return nil, fmt.Errorf("The instance ID %s must be numeric", d.Id())
	}

	stateConf := &resource.StateChangeConf{
		Pending: []string{"pending_upgrade"},
		Target:  []string{"upgrade_started"},
		Refresh: func() (interface{}, string, error) {
			client := meta.(*Client).virtualGuestService
			transactions, err := client.GetActiveTransactions(id)
			if err != nil {
				return nil, "", fmt.Errorf("Couldn't fetch active transactions: %s", err)
			}
			for _, transaction := range transactions {
				if strings.Contains(transaction.TransactionStatus.Name, "UPGRADE") {
					return transactions, "upgrade_started", nil
				}
			}
			return transactions, "pending_upgrade", nil
		},
		Timeout:    5 * time.Minute,
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	return stateConf.WaitForState()
}

func WaitForPublicIpAvailable(d *schema.ResourceData, meta interface{}) (interface{}, error) {
	log.Printf("Waiting for server (%s) to get a public IP", d.Id())

	stateConf := &resource.StateChangeConf{
		Pending: []string{"", "unavailable"},
		Target:  []string{"available"},
		Refresh: func() (interface{}, string, error) {
			fmt.Println("Refreshing server state...")
			client := meta.(*Client).virtualGuestService
			id, err := strconv.Atoi(d.Id())
			if err != nil {
				return nil, "", fmt.Errorf("Not a valid ID, must be an integer: %s", err)
			}
			result, err := client.GetObject(id)
			if err != nil {
				return nil, "", fmt.Errorf("Error retrieving virtual guest: %s", err)
			}
			if result.PrimaryIpAddress == "" {
				return result, "unavailable", nil
			} else {
				return result, "available", nil
			}
		},
		Timeout:    30 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	return stateConf.WaitForState()
}

func WaitForNoActiveTransactions(d *schema.ResourceData, meta interface{}) (interface{}, error) {
	log.Printf("Waiting for server (%s) to have zero active transactions", d.Id())
	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return nil, fmt.Errorf("The instance ID %s must be numeric", d.Id())
	}

	stateConf := &resource.StateChangeConf{
		Pending: []string{"", "active"},
		Target:  []string{"idle"},
		Refresh: func() (interface{}, string, error) {
			client := meta.(*Client).virtualGuestService
			transactions, err := client.GetActiveTransactions(id)
			if err != nil {
				return nil, "", fmt.Errorf("Couldn't get active transactions: %s", err)
			}
			if len(transactions) == 0 {
				return transactions, "idle", nil
			} else {
				return transactions, "active", nil
			}
		},
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	return stateConf.WaitForState()
}

func resourceSoftLayerVirtualGuestExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*Client).virtualGuestService

	if client == nil {
		return false, fmt.Errorf("The client was nil.")
	}

	guestId, err := strconv.Atoi(d.Id())
	if err != nil {
		return false, fmt.Errorf("Not a valid ID, must be an integer: %s", err)
	}

	result, err := client.GetObject(guestId)
	return result.Id == guestId && err == nil, nil
}
