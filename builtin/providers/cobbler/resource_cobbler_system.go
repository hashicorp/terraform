package cobbler

import (
	"bytes"
	"fmt"
	"log"
	"sync"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	cobbler "github.com/jtopjian/cobblerclient"
)

var systemSyncLock sync.Mutex

func resourceSystem() *schema.Resource {
	return &schema.Resource{
		Create: resourceSystemCreate,
		Read:   resourceSystemRead,
		Update: resourceSystemUpdate,
		Delete: resourceSystemDelete,

		Schema: map[string]*schema.Schema{
			"boot_files": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"comment": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"enable_gpxe": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"fetchable_files": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"gateway": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"hostname": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"image": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"interface": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"cnames": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},

						"dhcp_tag": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"dns_name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"bonding_opts": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"bridge_opts": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"gateway": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"interface_type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"interface_master": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"ip_address": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"ipv6_address": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"ipv6_secondaries": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},

						"ipv6_mtu": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"ipv6_static_routes": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},

						"ipv6_default_gateway": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"mac_address": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"management": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Computed: true,
						},

						"netmask": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"static": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Computed: true,
						},

						"static_routes": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},

						"virt_bridge": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
					},
				},
				Set: resourceSystemInterfaceHash,
			},

			"ipv6_default_device": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"kernel_options": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"kernel_options_post": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"kickstart": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"ks_meta": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"ldap_enabled": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"ldap_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"mgmt_classes": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"mgmt_parameters": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"monit_enabled": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"name_servers_search": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"name_servers": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"netboot_enabled": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"owners": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"power_address": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"power_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"power_pass": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"power_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"power_user": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"profile": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"proxy": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"redhat_management_key": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"redhat_management_server": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"status": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"template_files": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"template_remote_kickstarts": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},

			"virt_auto_boot": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"virt_file_size": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"virt_cpus": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"virt_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"virt_path": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"virt_pxe_boot": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},

			"virt_ram": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"virt_disk_driver": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceSystemCreate(d *schema.ResourceData, meta interface{}) error {
	systemSyncLock.Lock()
	defer systemSyncLock.Unlock()

	config := meta.(*Config)

	// Create a cobblerclient.System struct
	system := buildSystem(d)

	// Attempt to create the System
	log.Printf("[DEBUG] Cobbler System: Create Options: %#v", system)
	newSystem, err := config.cobblerClient.CreateSystem(system)
	if err != nil {
		return fmt.Errorf("Cobbler System: Error Creating: %s", err)
	}

	// Build cobblerclient.Interface structs
	interfaces := buildSystemInterfaces(d.Get("interface").(*schema.Set))

	// Add each interface to the system
	for interfaceName, interfaceInfo := range interfaces {
		log.Printf("[DEBUG] Cobbler System Interface %#v: %#v", interfaceName, interfaceInfo)
		if err := newSystem.CreateInterface(interfaceName, interfaceInfo); err != nil {
			return fmt.Errorf("Cobbler System: Error adding Interface %s to %s: %s", interfaceName, newSystem.Name, err)
		}
	}

	log.Printf("[DEBUG] Cobbler System: Created System: %#v", newSystem)
	d.SetId(newSystem.Name)

	log.Printf("[DEBUG] Cobbler System: syncing system")
	if err := config.cobblerClient.Sync(); err != nil {
		return fmt.Errorf("Cobbler System: Error syncing system: %s", err)
	}

	return resourceSystemRead(d, meta)
}

func resourceSystemRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Retrieve the system entry from Cobbler
	system, err := config.cobblerClient.GetSystem(d.Id())
	if err != nil {
		return fmt.Errorf("Cobbler System: Error Reading (%s): %s", d.Id(), err)
	}

	// Set all fields
	d.Set("boot_files", system.BootFiles)
	d.Set("comment", system.Comment)
	d.Set("enable_gpxe", system.EnableGPXE)
	d.Set("fetchable_files", system.FetchableFiles)
	d.Set("gateway", system.Gateway)
	d.Set("hostname", system.Hostname)
	d.Set("image", system.Image)
	d.Set("ipv6_default_device", system.IPv6DefaultDevice)
	d.Set("kernel_options", system.KernelOptions)
	d.Set("kernel_options_post", system.KernelOptionsPost)
	d.Set("kickstart", system.Kickstart)
	d.Set("ks_meta", system.KSMeta)
	d.Set("ldap_enabled", system.LDAPEnabled)
	d.Set("ldap_type", system.LDAPType)
	d.Set("mgmt_classes", system.MGMTClasses)
	d.Set("mgmt_parameters", system.MGMTParameters)
	d.Set("monit_enabled", system.MonitEnabled)
	d.Set("name_servers_search", system.NameServersSearch)
	d.Set("name_servers", system.NameServers)
	d.Set("netboot_enabled", system.NetbootEnabled)
	d.Set("owners", system.Owners)
	d.Set("power_address", system.PowerAddress)
	d.Set("power_id", system.PowerID)
	d.Set("power_pass", system.PowerPass)
	d.Set("power_type", system.PowerType)
	d.Set("power_user", system.PowerUser)
	d.Set("profile", system.Profile)
	d.Set("proxy", system.Proxy)
	d.Set("redhat_management_key", system.RedHatManagementKey)
	d.Set("redhat_management_server", system.RedHatManagementServer)
	d.Set("status", system.Status)
	d.Set("template_files", system.TemplateFiles)
	d.Set("template_remote_kickstarts", system.TemplateRemoteKickstarts)
	d.Set("virt_auto_boot", system.VirtAutoBoot)
	d.Set("virt_file_size", system.VirtFileSize)
	d.Set("virt_cpus", system.VirtCPUs)
	d.Set("virt_type", system.VirtType)
	d.Set("virt_path", system.VirtPath)
	d.Set("virt_pxe_boot", system.VirtPXEBoot)
	d.Set("virt_ram", system.VirtRam)
	d.Set("virt_disk_driver", system.VirtDiskDriver)

	// Get all interfaces that the System has
	allInterfaces, err := system.GetInterfaces()
	if err != nil {
		return fmt.Errorf("Cobbler System %s: Error getting interfaces: %s", system.Name, err)
	}

	// Build a generic map array with the interface attributes
	var systemInterfaces []map[string]interface{}
	for interfaceName, interfaceInfo := range allInterfaces {
		iface := make(map[string]interface{})
		iface["name"] = interfaceName
		iface["cnames"] = interfaceInfo.CNAMEs
		iface["dhcp_tag"] = interfaceInfo.DHCPTag
		iface["dns_name"] = interfaceInfo.DNSName
		iface["bonding_opts"] = interfaceInfo.BondingOpts
		iface["bridge_opts"] = interfaceInfo.BridgeOpts
		iface["gateway"] = interfaceInfo.Gateway
		iface["interface_type"] = interfaceInfo.InterfaceType
		iface["interface_master"] = interfaceInfo.InterfaceMaster
		iface["ip_address"] = interfaceInfo.IPAddress
		iface["ipv6_address"] = interfaceInfo.IPv6Address
		iface["ipv6_secondaries"] = interfaceInfo.IPv6Secondaries
		iface["ipv6_mtu"] = interfaceInfo.IPv6MTU
		iface["ipv6_static_routes"] = interfaceInfo.IPv6StaticRoutes
		iface["ipv6_default_gateway"] = interfaceInfo.IPv6DefaultGateway
		iface["mac_address"] = interfaceInfo.MACAddress
		iface["management"] = interfaceInfo.Management
		iface["netmask"] = interfaceInfo.Netmask
		iface["static"] = interfaceInfo.Static
		iface["static_Routes"] = interfaceInfo.StaticRoutes
		iface["virt_bridge"] = interfaceInfo.VirtBridge
		systemInterfaces = append(systemInterfaces, iface)
	}

	d.Set("interface", systemInterfaces)

	return nil
}

func resourceSystemUpdate(d *schema.ResourceData, meta interface{}) error {
	systemSyncLock.Lock()
	defer systemSyncLock.Unlock()

	config := meta.(*Config)

	// Retrieve the existing system entry from Cobbler
	system, err := config.cobblerClient.GetSystem(d.Id())
	if err != nil {
		return fmt.Errorf("Cobbler System: Error Reading (%s): %s", d.Id(), err)
	}

	// Get a list of the old interfaces
	currentInterfaces, err := system.GetInterfaces()
	if err != nil {
		return fmt.Errorf("Error getting interfaces: %s", err)
	}
	log.Printf("[DEBUG] Cobbler System Interfaces: %#v", currentInterfaces)

	// Create a new cobblerclient.System struct with the new information
	newSystem := buildSystem(d)

	// Attempt to update the system with new information
	log.Printf("[DEBUG] Cobbler System: Updating System (%s) with options: %+v", d.Id(), system)
	err = config.cobblerClient.UpdateSystem(&newSystem)
	if err != nil {
		return fmt.Errorf("Cobbler System: Error Updating (%s): %s", d.Id(), err)
	}

	if d.HasChange("interface") {
		oldInterfaces, newInterfaces := d.GetChange("interface")
		oldInterfacesSet := oldInterfaces.(*schema.Set)
		newInterfacesSet := newInterfaces.(*schema.Set)
		interfacesToRemove := oldInterfacesSet.Difference(newInterfacesSet)

		oldIfaces := buildSystemInterfaces(interfacesToRemove)
		newIfaces := buildSystemInterfaces(newInterfacesSet)

		for interfaceName, interfaceInfo := range oldIfaces {
			if _, ok := newIfaces[interfaceName]; !ok {
				// Interface does not exist in the new set,
				// so it has been removed from terraform.
				log.Printf("[DEBUG] Cobbler System: Deleting Interface %#v: %#v", interfaceName, interfaceInfo)

				if err := system.DeleteInterface(interfaceName); err != nil {
					return fmt.Errorf("Cobbler System: Error deleting Interface %s to %s: %s", interfaceName, system.Name, err)

				}
			}
		}

		// Modify interfaces that have changed
		for interfaceName, interfaceInfo := range newIfaces {
			log.Printf("[DEBUG] Cobbler System: New Interface %#v: %#v", interfaceName, interfaceInfo)

			if err := system.CreateInterface(interfaceName, interfaceInfo); err != nil {
				return fmt.Errorf("Cobbler System: Error adding Interface %s to %s: %s", interfaceName, system.Name, err)

			}
		}
	}

	log.Printf("[DEBUG] Cobbler System: syncing system")
	if err := config.cobblerClient.Sync(); err != nil {
		return fmt.Errorf("Cobbler System: Error syncing system: %s", err)
	}

	return resourceSystemRead(d, meta)
}

func resourceSystemDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Attempt to delete the system
	if err := config.cobblerClient.DeleteSystem(d.Id()); err != nil {
		return fmt.Errorf("Cobbler System: Error Deleting (%s): %s", d.Id(), err)
	}

	return nil
}

// buildSystem builds a cobblerclient.System out of the Terraform attributes
func buildSystem(d *schema.ResourceData) cobbler.System {
	mgmtClasses := []string{}
	for _, i := range d.Get("mgmt_classes").([]interface{}) {
		mgmtClasses = append(mgmtClasses, i.(string))
	}

	nameServersSearch := []string{}
	for _, i := range d.Get("name_servers_search").([]interface{}) {
		nameServersSearch = append(nameServersSearch, i.(string))
	}

	nameServers := []string{}
	for _, i := range d.Get("name_servers").([]interface{}) {
		nameServers = append(nameServers, i.(string))
	}

	owners := []string{}
	for _, i := range d.Get("owners").([]interface{}) {
		owners = append(owners, i.(string))
	}

	system := cobbler.System{
		BootFiles:                d.Get("boot_files").(string),
		Comment:                  d.Get("comment").(string),
		EnableGPXE:               d.Get("enable_gpxe").(bool),
		FetchableFiles:           d.Get("fetchable_files").(string),
		Gateway:                  d.Get("gateway").(string),
		Hostname:                 d.Get("hostname").(string),
		Image:                    d.Get("image").(string),
		IPv6DefaultDevice:        d.Get("ipv6_default_device").(string),
		KernelOptions:            d.Get("kernel_options").(string),
		KernelOptionsPost:        d.Get("kernel_options_post").(string),
		Kickstart:                d.Get("kickstart").(string),
		KSMeta:                   d.Get("ks_meta").(string),
		LDAPEnabled:              d.Get("ldap_enabled").(bool),
		LDAPType:                 d.Get("ldap_type").(string),
		MGMTClasses:              mgmtClasses,
		MGMTParameters:           d.Get("mgmt_parameters").(string),
		MonitEnabled:             d.Get("monit_enabled").(bool),
		Name:                     d.Get("name").(string),
		NameServersSearch:        nameServersSearch,
		NameServers:              nameServers,
		NetbootEnabled:           d.Get("netboot_enabled").(bool),
		Owners:                   owners,
		PowerAddress:             d.Get("power_address").(string),
		PowerID:                  d.Get("power_id").(string),
		PowerPass:                d.Get("power_pass").(string),
		PowerType:                d.Get("power_type").(string),
		PowerUser:                d.Get("power_user").(string),
		Profile:                  d.Get("profile").(string),
		Proxy:                    d.Get("proxy").(string),
		RedHatManagementKey:      d.Get("redhat_management_key").(string),
		RedHatManagementServer:   d.Get("redhat_management_server").(string),
		Status:                   d.Get("status").(string),
		TemplateFiles:            d.Get("template_files").(string),
		TemplateRemoteKickstarts: d.Get("template_remote_kickstarts").(int),
		VirtAutoBoot:             d.Get("virt_auto_boot").(string),
		VirtFileSize:             d.Get("virt_file_size").(string),
		VirtCPUs:                 d.Get("virt_cpus").(string),
		VirtType:                 d.Get("virt_type").(string),
		VirtPath:                 d.Get("virt_path").(string),
		VirtPXEBoot:              d.Get("virt_pxe_boot").(int),
		VirtRam:                  d.Get("virt_ram").(string),
		VirtDiskDriver:           d.Get("virt_disk_driver").(string),
	}

	return system
}

// buildSystemInterface builds a cobblerclient.Interface out of the Terraform attributes
func buildSystemInterfaces(systemInterfaces *schema.Set) cobbler.Interfaces {
	interfaces := make(cobbler.Interfaces)
	rawInterfaces := systemInterfaces.List()
	for _, rawInterface := range rawInterfaces {
		rawInterfaceMap := rawInterface.(map[string]interface{})

		cnames := []string{}
		for _, i := range rawInterfaceMap["cnames"].([]interface{}) {
			cnames = append(cnames, i.(string))
		}

		ipv6Secondaries := []string{}
		for _, i := range rawInterfaceMap["ipv6_secondaries"].([]interface{}) {
			ipv6Secondaries = append(ipv6Secondaries, i.(string))
		}

		ipv6StaticRoutes := []string{}
		for _, i := range rawInterfaceMap["ipv6_static_routes"].([]interface{}) {
			ipv6StaticRoutes = append(ipv6StaticRoutes, i.(string))
		}

		staticRoutes := []string{}
		for _, i := range rawInterfaceMap["static_routes"].([]interface{}) {
			staticRoutes = append(staticRoutes, i.(string))
		}

		interfaceName := rawInterfaceMap["name"].(string)
		interfaces[interfaceName] = cobbler.Interface{
			CNAMEs:             cnames,
			DHCPTag:            rawInterfaceMap["dhcp_tag"].(string),
			DNSName:            rawInterfaceMap["dns_name"].(string),
			BondingOpts:        rawInterfaceMap["bonding_opts"].(string),
			BridgeOpts:         rawInterfaceMap["bridge_opts"].(string),
			Gateway:            rawInterfaceMap["gateway"].(string),
			InterfaceType:      rawInterfaceMap["interface_type"].(string),
			InterfaceMaster:    rawInterfaceMap["interface_master"].(string),
			IPAddress:          rawInterfaceMap["ip_address"].(string),
			IPv6Address:        rawInterfaceMap["ipv6_address"].(string),
			IPv6Secondaries:    ipv6Secondaries,
			IPv6MTU:            rawInterfaceMap["ipv6_mtu"].(string),
			IPv6StaticRoutes:   ipv6StaticRoutes,
			IPv6DefaultGateway: rawInterfaceMap["ipv6_default_gateway"].(string),
			MACAddress:         rawInterfaceMap["mac_address"].(string),
			Management:         rawInterfaceMap["management"].(bool),
			Netmask:            rawInterfaceMap["netmask"].(string),
			Static:             rawInterfaceMap["static"].(bool),
			StaticRoutes:       staticRoutes,
			VirtBridge:         rawInterfaceMap["virt_bridge"].(string),
		}
	}

	return interfaces
}

func resourceSystemInterfaceHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%s", m["name"].(string)))

	if v, ok := m["cnames"]; ok {
		for _, x := range v.([]interface{}) {
			buf.WriteString(fmt.Sprintf("%v-", x.(string)))
		}
	}

	if v, ok := m["dhcp_tag"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	if v, ok := m["dns_name"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	if v, ok := m["bonding_opts"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	if v, ok := m["bridge_opts"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	if v, ok := m["gateway"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	if v, ok := m["interface_type"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	if v, ok := m["interface_master"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	if v, ok := m["ip_address"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	if v, ok := m["ipv6_address"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	if v, ok := m["ipv6_secondaries"]; ok {
		for _, x := range v.([]interface{}) {
			buf.WriteString(fmt.Sprintf("%v-", x.(string)))
		}
	}

	if v, ok := m["ipv6_mtu"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	if v, ok := m["ipv6_static_routes"]; ok {
		for _, x := range v.([]interface{}) {
			buf.WriteString(fmt.Sprintf("%v-", x.(string)))
		}
	}

	if v, ok := m["ipv6_default_gateway"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	if v, ok := m["mac_address"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	if v, ok := m["management"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(bool)))
	}

	if v, ok := m["netmask"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	if v, ok := m["static"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(bool)))
	}

	if v, ok := m["static_Routes"]; ok {
		for _, x := range v.([]interface{}) {
			buf.WriteString(fmt.Sprintf("%v-", x.(string)))
		}
	}

	if v, ok := m["virt_bridge"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	return hashcode.String(buf.String())
}
