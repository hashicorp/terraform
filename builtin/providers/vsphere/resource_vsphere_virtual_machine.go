package vsphere

import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"golang.org/x/net/context"
)

var DefaultDNSSuffixes = []string{
	"vsphere.local",
}

var DefaultDNSServers = []string{
	"8.8.8.8",
	"8.8.4.4",
}

type networkInterface struct {
	deviceName       string
	label            string
	ipv4Address      string
	ipv4PrefixLength int
	ipv6Address      string
	ipv6PrefixLength int
	adapterType      string // TODO: Make "adapter_type" argument
}

type hardDisk struct {
	size     int64
	iops     int64
	initType string
}

type cdRom struct {
	isoPath string
}

type virtualMachine struct {
	name                 string
	folder               string
	datacenter           string
	cluster              string
	resourcePool         string
	datastore            string
	vcpu                 int
	memoryMb             int64
	template             string
	networkInterfaces    []networkInterface
	hardDisks            []hardDisk
	cdRoms               []cdRom
	gateway              string
	domain               string
	timeZone             string
	dnsSuffixes          []string
	dnsServers           []string
	customConfigurations map[string](types.AnyType)
}

func (v virtualMachine) Path() string {
	return vmPath(v.folder, v.name)
}

func vmPath(folder string, name string) string {
	var path string
	if len(folder) > 0 {
		path += folder + "/"
	}
	return path + name
}

func resourceVSphereVirtualMachine() *schema.Resource {
	return &schema.Resource{
		Create: resourceVSphereVirtualMachineCreate,
		Read:   resourceVSphereVirtualMachineRead,
		Delete: resourceVSphereVirtualMachineDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"folder": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"vcpu": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},

			"memory": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},

			"datacenter": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"cluster": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"resource_pool": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"gateway": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"domain": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "vsphere.local",
			},

			"time_zone": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "Etc/UTC",
			},

			"dns_suffixes": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				ForceNew: true,
			},

			"dns_servers": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				ForceNew: true,
			},

			"custom_configuration_parameters": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},

			"network_interface": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"label": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},

						"ip_address": &schema.Schema{
							Type:       schema.TypeString,
							Optional:   true,
							Computed:   true,
							Deprecated: "Please use ipv4_address",
						},

						"subnet_mask": &schema.Schema{
							Type:       schema.TypeString,
							Optional:   true,
							Computed:   true,
							Deprecated: "Please use ipv4_prefix_length",
						},

						"ipv4_address": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"ipv4_prefix_length": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},

						// TODO: Imprement ipv6 parameters to be optional
						"ipv6_address": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
							ForceNew: true,
						},

						"ipv6_prefix_length": &schema.Schema{
							Type:     schema.TypeInt,
							Computed: true,
							ForceNew: true,
						},

						"adapter_type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},

			"disk": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"template": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
							Default:  "eager_zeroed",
							ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
								value := v.(string)
								if value != "thin" && value != "eager_zeroed" {
									errors = append(errors, fmt.Errorf(
										"only 'thin' and 'eager_zeroed' are supported values for 'type'"))
								}
								return
							},
						},

						"datastore": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"size": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: true,
						},

						"iops": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},

			"boot_delay": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},

			"cdrom": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"iso_path": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
					},
				},
			},
		},
	}
}

func resourceVSphereVirtualMachineCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*govmomi.Client)

	vm := virtualMachine{
		name:     d.Get("name").(string),
		vcpu:     d.Get("vcpu").(int),
		memoryMb: int64(d.Get("memory").(int)),
	}

	if v, ok := d.GetOk("folder"); ok {
		vm.folder = v.(string)
	}

	if v, ok := d.GetOk("datacenter"); ok {
		vm.datacenter = v.(string)
	}

	if v, ok := d.GetOk("cluster"); ok {
		vm.cluster = v.(string)
	}

	if v, ok := d.GetOk("resource_pool"); ok {
		vm.resourcePool = v.(string)
	}

	if v, ok := d.GetOk("gateway"); ok {
		vm.gateway = v.(string)
	}

	if v, ok := d.GetOk("domain"); ok {
		vm.domain = v.(string)
	}

	if v, ok := d.GetOk("time_zone"); ok {
		vm.timeZone = v.(string)
	}

	if raw, ok := d.GetOk("dns_suffixes"); ok {
		for _, v := range raw.([]interface{}) {
			vm.dnsSuffixes = append(vm.dnsSuffixes, v.(string))
		}
	} else {
		vm.dnsSuffixes = DefaultDNSSuffixes
	}

	if raw, ok := d.GetOk("dns_servers"); ok {
		for _, v := range raw.([]interface{}) {
			vm.dnsServers = append(vm.dnsServers, v.(string))
		}
	} else {
		vm.dnsServers = DefaultDNSServers
	}

	if vL, ok := d.GetOk("custom_configuration_parameters"); ok {
		if custom_configs, ok := vL.(map[string]interface{}); ok {
			custom := make(map[string]types.AnyType)
			for k, v := range custom_configs {
				custom[k] = v
			}
			vm.customConfigurations = custom
			log.Printf("[DEBUG] custom_configuration_parameters init: %v", vm.customConfigurations)
		}
	}

	if vL, ok := d.GetOk("network_interface"); ok {
		networks := make([]networkInterface, len(vL.([]interface{})))
		for i, v := range vL.([]interface{}) {
			network := v.(map[string]interface{})
			networks[i].label = network["label"].(string)
			if v, ok := network["ip_address"].(string); ok && v != "" {
				networks[i].ipv4Address = v
			}
			if v, ok := network["subnet_mask"].(string); ok && v != "" {
				ip := net.ParseIP(v).To4()
				if ip != nil {
					mask := net.IPv4Mask(ip[0], ip[1], ip[2], ip[3])
					pl, _ := mask.Size()
					networks[i].ipv4PrefixLength = pl
				} else {
					return fmt.Errorf("subnet_mask parameter is invalid.")
				}
			}
			if v, ok := network["ipv4_address"].(string); ok && v != "" {
				networks[i].ipv4Address = v
			}
			if v, ok := network["ipv4_prefix_length"].(int); ok && v != 0 {
				networks[i].ipv4PrefixLength = v
			}
		}
		vm.networkInterfaces = networks
		log.Printf("[DEBUG] network_interface init: %v", networks)
	}

	if vL, ok := d.GetOk("cdrom"); ok {
		cdroms := make([]cdRom, len(vL.([]interface{})))
		for i, v := range vL.([]interface{}) {
			cdrom := v.(map[string]interface{})
			if v, ok := cdrom["iso_path"].(string); ok && v != "" {
				cdroms[i].isoPath = v
			}
		}
		vm.cdRoms = cdroms
		log.Printf("[DEBUG] cdrom init: %v", cdroms)
	}

	if vL, ok := d.GetOk("disk"); ok {
		disks := make([]hardDisk, len(vL.([]interface{})))
		for i, v := range vL.([]interface{}) {
			disk := v.(map[string]interface{})
			if i == 0 {
				if v, ok := disk["template"].(string); ok && v != "" {
					vm.template = v
				} else {
					if v, ok := disk["size"].(int); ok && v != 0 {
						disks[i].size = int64(v)
					} else {
						return fmt.Errorf("If template argument is not specified, size argument is required.")
					}
				}
				if v, ok := disk["datastore"].(string); ok && v != "" {
					vm.datastore = v
				}
			} else {
				if v, ok := disk["size"].(int); ok && v != 0 {
					disks[i].size = int64(v)
				} else {
					return fmt.Errorf("Size argument is required.")
				}

			}
			if v, ok := disk["iops"].(int); ok && v != 0 {
				disks[i].iops = int64(v)
			}
			if v, ok := disk["type"].(string); ok && v != "" {
				disks[i].initType = v
			}
		}
		vm.hardDisks = disks
		log.Printf("[DEBUG] disk init: %v", disks)
	}

	if vm.template != "" {
		err := vm.deployVirtualMachine(client)
		if err != nil {
			return err
		}
	} else {
		err := vm.createVirtualMachine(client)
		if err != nil {
			return err
		}
	}

	if _, ok := d.GetOk("network_interface.0.ipv4_address"); !ok {
		if v, ok := d.GetOk("boot_delay"); ok {
			stateConf := &resource.StateChangeConf{
				Pending:    []string{"pending"},
				Target:     []string{"active"},
				Refresh:    waitForNetworkingActive(client, vm.datacenter, vm.Path()),
				Timeout:    600 * time.Second,
				Delay:      time.Duration(v.(int)) * time.Second,
				MinTimeout: 2 * time.Second,
			}

			_, err := stateConf.WaitForState()
			if err != nil {
				return err
			}
		}
	}

	if ip, ok := d.GetOk("network_interface.0.ipv4_address"); ok {
		d.SetConnInfo(map[string]string{
			"host": ip.(string),
		})
	} else {
		log.Printf("[DEBUG] Could not get IP address for %s", d.Id())
	}

	d.SetId(vm.Path())
	log.Printf("[INFO] Created virtual machine: %s", d.Id())

	return resourceVSphereVirtualMachineRead(d, meta)
}

func resourceVSphereVirtualMachineRead(d *schema.ResourceData, meta interface{}) error {

	log.Printf("[DEBUG] reading virtual machine: %#v", d)
	client := meta.(*govmomi.Client)
	dc, err := getDatacenter(client, d.Get("datacenter").(string))
	if err != nil {
		return err
	}
	finder := find.NewFinder(client.Client, true)
	finder = finder.SetDatacenter(dc)

	vm, err := finder.VirtualMachine(context.TODO(), d.Id())
	if err != nil {
		d.SetId("")
		return nil
	}

	var mvm mo.VirtualMachine

	collector := property.DefaultCollector(client.Client)
	if err := collector.RetrieveOne(context.TODO(), vm.Reference(), []string{"guest", "summary", "datastore"}, &mvm); err != nil {
		return err
	}

	log.Printf("[DEBUG] %#v", dc)
	log.Printf("[DEBUG] %#v", mvm.Summary.Config)
	log.Printf("[DEBUG] %#v", mvm.Guest.Net)

	networkInterfaces := make([]map[string]interface{}, 0)
	for _, v := range mvm.Guest.Net {
		if v.DeviceConfigId >= 0 {
			log.Printf("[DEBUG] %#v", v.Network)
			networkInterface := make(map[string]interface{})
			networkInterface["label"] = v.Network
			for _, ip := range v.IpConfig.IpAddress {
				p := net.ParseIP(ip.IpAddress)
				if p.To4() != nil {
					log.Printf("[DEBUG] %#v", p.String())
					log.Printf("[DEBUG] %#v", ip.PrefixLength)
					networkInterface["ipv4_address"] = p.String()
					networkInterface["ipv4_prefix_length"] = ip.PrefixLength
				} else if p.To16() != nil {
					log.Printf("[DEBUG] %#v", p.String())
					log.Printf("[DEBUG] %#v", ip.PrefixLength)
					networkInterface["ipv6_address"] = p.String()
					networkInterface["ipv6_prefix_length"] = ip.PrefixLength
				}
				log.Printf("[DEBUG] networkInterface: %#v", networkInterface)
			}
			log.Printf("[DEBUG] networkInterface: %#v", networkInterface)
			networkInterfaces = append(networkInterfaces, networkInterface)
		}
	}
	log.Printf("[DEBUG] networkInterfaces: %#v", networkInterfaces)
	err = d.Set("network_interface", networkInterfaces)
	if err != nil {
		return fmt.Errorf("Invalid network interfaces to set: %#v", networkInterfaces)
	}

	var rootDatastore string
	for _, v := range mvm.Datastore {
		var md mo.Datastore
		if err := collector.RetrieveOne(context.TODO(), v, []string{"name", "parent"}, &md); err != nil {
			return err
		}
		if md.Parent.Type == "StoragePod" {
			var msp mo.StoragePod
			if err := collector.RetrieveOne(context.TODO(), *md.Parent, []string{"name"}, &msp); err != nil {
				return err
			}
			rootDatastore = msp.Name
			log.Printf("[DEBUG] %#v", msp.Name)
		} else {
			rootDatastore = md.Name
			log.Printf("[DEBUG] %#v", md.Name)
		}
		break
	}

	d.Set("datacenter", dc)
	d.Set("memory", mvm.Summary.Config.MemorySizeMB)
	d.Set("cpu", mvm.Summary.Config.NumCpu)
	d.Set("datastore", rootDatastore)

	return nil
}

func resourceVSphereVirtualMachineDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*govmomi.Client)
	dc, err := getDatacenter(client, d.Get("datacenter").(string))
	if err != nil {
		return err
	}
	finder := find.NewFinder(client.Client, true)
	finder = finder.SetDatacenter(dc)

	vm, err := finder.VirtualMachine(context.TODO(), vmPath(d.Get("folder").(string), d.Get("name").(string)))
	if err != nil {
		return err
	}

	log.Printf("[INFO] Deleting virtual machine: %s", d.Id())

	task, err := vm.PowerOff(context.TODO())
	if err != nil {
		return err
	}

	err = task.Wait(context.TODO())
	if err != nil {
		return err
	}

	task, err = vm.Destroy(context.TODO())
	if err != nil {
		return err
	}

	err = task.Wait(context.TODO())
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}

func waitForNetworkingActive(client *govmomi.Client, datacenter, name string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		dc, err := getDatacenter(client, datacenter)
		if err != nil {
			log.Printf("[ERROR] %#v", err)
			return nil, "", err
		}
		finder := find.NewFinder(client.Client, true)
		finder = finder.SetDatacenter(dc)

		vm, err := finder.VirtualMachine(context.TODO(), name)
		if err != nil {
			log.Printf("[ERROR] %#v", err)
			return nil, "", err
		}

		var mvm mo.VirtualMachine
		collector := property.DefaultCollector(client.Client)
		if err := collector.RetrieveOne(context.TODO(), vm.Reference(), []string{"summary"}, &mvm); err != nil {
			log.Printf("[ERROR] %#v", err)
			return nil, "", err
		}

		if mvm.Summary.Guest.IpAddress != "" {
			log.Printf("[DEBUG] IP address with DHCP: %v", mvm.Summary.Guest.IpAddress)
			return mvm.Summary, "active", err
		} else {
			log.Printf("[DEBUG] Waiting for IP address")
			return nil, "pending", err
		}
	}
}

// addCdRom adds a new CD Rom to the VirtualMachine
func addCdRom(vm *object.VirtualMachine, datastore *object.Datastore, isoFilepath string) error {
	devices, err := vm.Device(context.TODO())

	ide, err := devices.FindIDEController("")
	if err != nil {
		return err
	}

	cdrom, err := devices.CreateCdrom(ide)
	if err != nil {
		return err
	}

	return vm.AddDevice(context.TODO(), devices.InsertIso(cdrom, datastore.Path(isoFilepath)))
}

// addHardDisk adds a new Hard Disk to the VirtualMachine.
func addHardDisk(vm *object.VirtualMachine, size, iops int64, diskType string) error {
	devices, err := vm.Device(context.TODO())
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] vm devices: %#v\n", devices)

	controller, err := devices.FindDiskController("scsi")
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] disk controller: %#v\n", controller)

	disk := devices.CreateDisk(controller, "")
	existing := devices.SelectByBackingInfo(disk.Backing)
	log.Printf("[DEBUG] disk: %#v\n", disk)

	if len(existing) == 0 {
		disk.CapacityInKB = int64(size * 1024 * 1024)
		if iops != 0 {
			disk.StorageIOAllocation = &types.StorageIOAllocationInfo{
				Limit: iops,
			}
		}
		backing := disk.Backing.(*types.VirtualDiskFlatVer2BackingInfo)

		if diskType == "eager_zeroed" {
			// eager zeroed thick virtual disk
			backing.ThinProvisioned = types.NewBool(false)
			backing.EagerlyScrub = types.NewBool(true)
		} else if diskType == "thin" {
			// thin provisioned virtual disk
			backing.ThinProvisioned = types.NewBool(true)
		}

		log.Printf("[DEBUG] addHardDisk: %#v\n", disk)
		log.Printf("[DEBUG] addHardDisk: %#v\n", disk.CapacityInKB)

		return vm.AddDevice(context.TODO(), disk)
	} else {
		log.Printf("[DEBUG] addHardDisk: Disk already present.\n")

		return nil
	}
}

// buildNetworkDevice builds VirtualDeviceConfigSpec for Network Device.
func buildNetworkDevice(f *find.Finder, label, adapterType string) (*types.VirtualDeviceConfigSpec, error) {
	network, err := f.Network(context.TODO(), "*"+label)
	if err != nil {
		return nil, err
	}

	backing, err := network.EthernetCardBackingInfo(context.TODO())
	if err != nil {
		return nil, err
	}

	if adapterType == "vmxnet3" {
		return &types.VirtualDeviceConfigSpec{
			Operation: types.VirtualDeviceConfigSpecOperationAdd,
			Device: &types.VirtualVmxnet3{
				VirtualVmxnet: types.VirtualVmxnet{
					VirtualEthernetCard: types.VirtualEthernetCard{
						VirtualDevice: types.VirtualDevice{
							Key:     -1,
							Backing: backing,
						},
						AddressType: string(types.VirtualEthernetCardMacTypeGenerated),
					},
				},
			},
		}, nil
	} else if adapterType == "e1000" {
		return &types.VirtualDeviceConfigSpec{
			Operation: types.VirtualDeviceConfigSpecOperationAdd,
			Device: &types.VirtualE1000{
				VirtualEthernetCard: types.VirtualEthernetCard{
					VirtualDevice: types.VirtualDevice{
						Key:     -1,
						Backing: backing,
					},
					AddressType: string(types.VirtualEthernetCardMacTypeGenerated),
				},
			},
		}, nil
	} else {
		return nil, fmt.Errorf("Invalid network adapter type.")
	}
}

// buildVMRelocateSpec builds VirtualMachineRelocateSpec to set a place for a new VirtualMachine.
func buildVMRelocateSpec(rp *object.ResourcePool, ds *object.Datastore, vm *object.VirtualMachine, initType string) (types.VirtualMachineRelocateSpec, error) {
	var key int

	devices, err := vm.Device(context.TODO())
	if err != nil {
		return types.VirtualMachineRelocateSpec{}, err
	}
	for _, d := range devices {
		if devices.Type(d) == "disk" {
			key = d.GetVirtualDevice().Key
		}
	}

	isThin := initType == "thin"
	rpr := rp.Reference()
	dsr := ds.Reference()
	return types.VirtualMachineRelocateSpec{
		Datastore: &dsr,
		Pool:      &rpr,
		Disk: []types.VirtualMachineRelocateSpecDiskLocator{
			types.VirtualMachineRelocateSpecDiskLocator{
				Datastore: dsr,
				DiskBackingInfo: &types.VirtualDiskFlatVer2BackingInfo{
					DiskMode:        "persistent",
					ThinProvisioned: types.NewBool(isThin),
					EagerlyScrub:    types.NewBool(!isThin),
				},
				DiskId: key,
			},
		},
	}, nil
}

// getDatastoreObject gets datastore object.
func getDatastoreObject(client *govmomi.Client, f *object.DatacenterFolders, name string) (types.ManagedObjectReference, error) {
	s := object.NewSearchIndex(client.Client)
	ref, err := s.FindChild(context.TODO(), f.DatastoreFolder, name)
	if err != nil {
		return types.ManagedObjectReference{}, err
	}
	if ref == nil {
		return types.ManagedObjectReference{}, fmt.Errorf("Datastore '%s' not found.", name)
	}
	log.Printf("[DEBUG] getDatastoreObject: reference: %#v", ref)
	return ref.Reference(), nil
}

// buildStoragePlacementSpecCreate builds StoragePlacementSpec for create action.
func buildStoragePlacementSpecCreate(f *object.DatacenterFolders, rp *object.ResourcePool, storagePod object.StoragePod, configSpec types.VirtualMachineConfigSpec) types.StoragePlacementSpec {
	vmfr := f.VmFolder.Reference()
	rpr := rp.Reference()
	spr := storagePod.Reference()

	sps := types.StoragePlacementSpec{
		Type:       "create",
		ConfigSpec: &configSpec,
		PodSelectionSpec: types.StorageDrsPodSelectionSpec{
			StoragePod: &spr,
		},
		Folder:       &vmfr,
		ResourcePool: &rpr,
	}
	log.Printf("[DEBUG] findDatastore: StoragePlacementSpec: %#v\n", sps)
	return sps
}

// buildStoragePlacementSpecClone builds StoragePlacementSpec for clone action.
func buildStoragePlacementSpecClone(c *govmomi.Client, f *object.DatacenterFolders, vm *object.VirtualMachine, rp *object.ResourcePool, storagePod object.StoragePod) types.StoragePlacementSpec {
	vmr := vm.Reference()
	vmfr := f.VmFolder.Reference()
	rpr := rp.Reference()
	spr := storagePod.Reference()

	var o mo.VirtualMachine
	err := vm.Properties(context.TODO(), vmr, []string{"datastore"}, &o)
	if err != nil {
		return types.StoragePlacementSpec{}
	}
	ds := object.NewDatastore(c.Client, o.Datastore[0])
	log.Printf("[DEBUG] findDatastore: datastore: %#v\n", ds)

	devices, err := vm.Device(context.TODO())
	if err != nil {
		return types.StoragePlacementSpec{}
	}

	var key int
	for _, d := range devices.SelectByType((*types.VirtualDisk)(nil)) {
		key = d.GetVirtualDevice().Key
		log.Printf("[DEBUG] findDatastore: virtual devices: %#v\n", d.GetVirtualDevice())
	}

	sps := types.StoragePlacementSpec{
		Type: "clone",
		Vm:   &vmr,
		PodSelectionSpec: types.StorageDrsPodSelectionSpec{
			StoragePod: &spr,
		},
		CloneSpec: &types.VirtualMachineCloneSpec{
			Location: types.VirtualMachineRelocateSpec{
				Disk: []types.VirtualMachineRelocateSpecDiskLocator{
					types.VirtualMachineRelocateSpecDiskLocator{
						Datastore:       ds.Reference(),
						DiskBackingInfo: &types.VirtualDiskFlatVer2BackingInfo{},
						DiskId:          key,
					},
				},
				Pool: &rpr,
			},
			PowerOn:  false,
			Template: false,
		},
		CloneName: "dummy",
		Folder:    &vmfr,
	}
	return sps
}

// findDatastore finds Datastore object.
func findDatastore(c *govmomi.Client, sps types.StoragePlacementSpec) (*object.Datastore, error) {
	var datastore *object.Datastore
	log.Printf("[DEBUG] findDatastore: StoragePlacementSpec: %#v\n", sps)

	srm := object.NewStorageResourceManager(c.Client)
	rds, err := srm.RecommendDatastores(context.TODO(), sps)
	if err != nil {
		return nil, err
	}
	log.Printf("[DEBUG] findDatastore: recommendDatastores: %#v\n", rds)

	spa := rds.Recommendations[0].Action[0].(*types.StoragePlacementAction)
	datastore = object.NewDatastore(c.Client, spa.Destination)
	log.Printf("[DEBUG] findDatastore: datastore: %#v", datastore)

	return datastore, nil
}

// createVirtualMachine creates a new VirtualMachine.
func (vm *virtualMachine) createVirtualMachine(c *govmomi.Client) error {
	dc, err := getDatacenter(c, vm.datacenter)

	if err != nil {
		return err
	}
	finder := find.NewFinder(c.Client, true)
	finder = finder.SetDatacenter(dc)

	var resourcePool *object.ResourcePool
	if vm.resourcePool == "" {
		if vm.cluster == "" {
			resourcePool, err = finder.DefaultResourcePool(context.TODO())
			if err != nil {
				return err
			}
		} else {
			resourcePool, err = finder.ResourcePool(context.TODO(), "*"+vm.cluster+"/Resources")
			if err != nil {
				return err
			}
		}
	} else {
		resourcePool, err = finder.ResourcePool(context.TODO(), vm.resourcePool)
		if err != nil {
			return err
		}
	}
	log.Printf("[DEBUG] resource pool: %#v", resourcePool)

	dcFolders, err := dc.Folders(context.TODO())
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] folder: %#v", vm.folder)
	folder := dcFolders.VmFolder
	if len(vm.folder) > 0 {
		si := object.NewSearchIndex(c.Client)
		folderRef, err := si.FindByInventoryPath(
			context.TODO(), fmt.Sprintf("%v/vm/%v", vm.datacenter, vm.folder))
		if err != nil {
			return fmt.Errorf("Error reading folder %s: %s", vm.folder, err)
		} else if folderRef == nil {
			return fmt.Errorf("Cannot find folder %s", vm.folder)
		} else {
			folder = folderRef.(*object.Folder)
		}
	}

	// network
	networkDevices := []types.BaseVirtualDeviceConfigSpec{}
	for _, network := range vm.networkInterfaces {
		// network device
		nd, err := buildNetworkDevice(finder, network.label, "e1000")
		if err != nil {
			return err
		}
		networkDevices = append(networkDevices, nd)
	}

	// make config spec
	configSpec := types.VirtualMachineConfigSpec{
		GuestId:           "otherLinux64Guest",
		Name:              vm.name,
		NumCPUs:           vm.vcpu,
		NumCoresPerSocket: 1,
		MemoryMB:          vm.memoryMb,
		DeviceChange:      networkDevices,
	}
	log.Printf("[DEBUG] virtual machine config spec: %v", configSpec)

	// make ExtraConfig
	log.Printf("[DEBUG] virtual machine Extra Config spec start")
	if len(vm.customConfigurations) > 0 {
		var ov []types.BaseOptionValue
		for k, v := range vm.customConfigurations {
			key := k
			value := v
			o := types.OptionValue{
				Key:   key,
				Value: &value,
			}
			log.Printf("[DEBUG] virtual machine Extra Config spec: %s,%s", k, v)
			ov = append(ov, &o)
		}
		configSpec.ExtraConfig = ov
		log.Printf("[DEBUG] virtual machine Extra Config spec: %v", configSpec.ExtraConfig)
	}

	var datastore *object.Datastore
	if vm.datastore == "" {
		datastore, err = finder.DefaultDatastore(context.TODO())
		if err != nil {
			return err
		}
	} else {
		datastore, err = finder.Datastore(context.TODO(), vm.datastore)
		if err != nil {
			// TODO: datastore cluster support in govmomi finder function
			d, err := getDatastoreObject(c, dcFolders, vm.datastore)
			if err != nil {
				return err
			}

			if d.Type == "StoragePod" {
				sp := object.StoragePod{
					Folder: object.NewFolder(c.Client, d),
				}
				sps := buildStoragePlacementSpecCreate(dcFolders, resourcePool, sp, configSpec)
				datastore, err = findDatastore(c, sps)
				if err != nil {
					return err
				}
			} else {
				datastore = object.NewDatastore(c.Client, d)
			}
		}
	}

	log.Printf("[DEBUG] datastore: %#v", datastore)

	var mds mo.Datastore
	if err = datastore.Properties(context.TODO(), datastore.Reference(), []string{"name"}, &mds); err != nil {
		return err
	}
	log.Printf("[DEBUG] datastore: %#v", mds.Name)
	scsi, err := object.SCSIControllerTypes().CreateSCSIController("scsi")
	if err != nil {
		log.Printf("[ERROR] %s", err)
	}

	configSpec.DeviceChange = append(configSpec.DeviceChange, &types.VirtualDeviceConfigSpec{
		Operation: types.VirtualDeviceConfigSpecOperationAdd,
		Device:    scsi,
	})
	configSpec.Files = &types.VirtualMachineFileInfo{VmPathName: fmt.Sprintf("[%s]", mds.Name)}

	task, err := folder.CreateVM(context.TODO(), configSpec, resourcePool, nil)
	if err != nil {
		log.Printf("[ERROR] %s", err)
	}

	err = task.Wait(context.TODO())
	if err != nil {
		log.Printf("[ERROR] %s", err)
	}

	newVM, err := finder.VirtualMachine(context.TODO(), vm.Path())
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] new vm: %v", newVM)

	log.Printf("[DEBUG] add hard disk: %v", vm.hardDisks)
	for _, hd := range vm.hardDisks {
		log.Printf("[DEBUG] add hard disk: %v", hd.size)
		log.Printf("[DEBUG] add hard disk: %v", hd.iops)
		err = addHardDisk(newVM, hd.size, hd.iops, "thin")
		if err != nil {
			return err
		}
	}

	for _, cd := range vm.cdRoms {
		log.Printf("[DEBUG] add cdrom iso: %v", cd.isoPath)
		err = addCdRom(newVM, datastore, cd.isoPath)
		if err != nil {
			return err
		}
	}
	return nil
}

// deployVirtualMachine deploys a new VirtualMachine.
func (vm *virtualMachine) deployVirtualMachine(c *govmomi.Client) error {
	dc, err := getDatacenter(c, vm.datacenter)
	if err != nil {
		return err
	}
	finder := find.NewFinder(c.Client, true)
	finder = finder.SetDatacenter(dc)

	template, err := finder.VirtualMachine(context.TODO(), vm.template)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] template: %#v", template)

	var resourcePool *object.ResourcePool
	if vm.resourcePool == "" {
		if vm.cluster == "" {
			resourcePool, err = finder.DefaultResourcePool(context.TODO())
			if err != nil {
				return err
			}
		} else {
			resourcePool, err = finder.ResourcePool(context.TODO(), "*"+vm.cluster+"/Resources")
			if err != nil {
				return err
			}
		}
	} else {
		resourcePool, err = finder.ResourcePool(context.TODO(), vm.resourcePool)
		if err != nil {
			return err
		}
	}
	log.Printf("[DEBUG] resource pool: %#v", resourcePool)

	dcFolders, err := dc.Folders(context.TODO())
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] folder: %#v", vm.folder)
	folder := dcFolders.VmFolder
	if len(vm.folder) > 0 {
		si := object.NewSearchIndex(c.Client)
		folderRef, err := si.FindByInventoryPath(
			context.TODO(), fmt.Sprintf("%v/vm/%v", vm.datacenter, vm.folder))
		if err != nil {
			return fmt.Errorf("Error reading folder %s: %s", vm.folder, err)
		} else if folderRef == nil {
			return fmt.Errorf("Cannot find folder %s", vm.folder)
		} else {
			folder = folderRef.(*object.Folder)
		}
	}

	var datastore *object.Datastore
	if vm.datastore == "" {
		datastore, err = finder.DefaultDatastore(context.TODO())
		if err != nil {
			return err
		}
	} else {
		datastore, err = finder.Datastore(context.TODO(), vm.datastore)
		if err != nil {
			// TODO: datastore cluster support in govmomi finder function
			d, err := getDatastoreObject(c, dcFolders, vm.datastore)
			if err != nil {
				return err
			}

			if d.Type == "StoragePod" {
				sp := object.StoragePod{
					Folder: object.NewFolder(c.Client, d),
				}
				sps := buildStoragePlacementSpecClone(c, dcFolders, template, resourcePool, sp)

				datastore, err = findDatastore(c, sps)
				if err != nil {
					return err
				}
			} else {
				datastore = object.NewDatastore(c.Client, d)
			}
		}
	}
	log.Printf("[DEBUG] datastore: %#v", datastore)

	relocateSpec, err := buildVMRelocateSpec(resourcePool, datastore, template, vm.hardDisks[0].initType)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] relocate spec: %v", relocateSpec)

	// network
	networkDevices := []types.BaseVirtualDeviceConfigSpec{}
	networkConfigs := []types.CustomizationAdapterMapping{}
	for _, network := range vm.networkInterfaces {
		// network device
		nd, err := buildNetworkDevice(finder, network.label, "vmxnet3")
		if err != nil {
			return err
		}
		networkDevices = append(networkDevices, nd)

		// TODO: IPv6 support
		var ipSetting types.CustomizationIPSettings
		if network.ipv4Address == "" {
			ipSetting = types.CustomizationIPSettings{
				Ip: &types.CustomizationDhcpIpGenerator{},
			}
		} else {
			if network.ipv4PrefixLength == 0 {
				return fmt.Errorf("Error: ipv4_prefix_length argument is empty.")
			}
			m := net.CIDRMask(network.ipv4PrefixLength, 32)
			sm := net.IPv4(m[0], m[1], m[2], m[3])
			subnetMask := sm.String()
			log.Printf("[DEBUG] gateway: %v", vm.gateway)
			log.Printf("[DEBUG] ipv4 address: %v", network.ipv4Address)
			log.Printf("[DEBUG] ipv4 prefix length: %v", network.ipv4PrefixLength)
			log.Printf("[DEBUG] ipv4 subnet mask: %v", subnetMask)
			ipSetting = types.CustomizationIPSettings{
				Gateway: []string{
					vm.gateway,
				},
				Ip: &types.CustomizationFixedIp{
					IpAddress: network.ipv4Address,
				},
				SubnetMask: subnetMask,
			}
		}

		// network config
		config := types.CustomizationAdapterMapping{
			Adapter: ipSetting,
		}
		networkConfigs = append(networkConfigs, config)
	}
	log.Printf("[DEBUG] network configs: %v", networkConfigs[0].Adapter)

	// make config spec
	configSpec := types.VirtualMachineConfigSpec{
		NumCPUs:           vm.vcpu,
		NumCoresPerSocket: 1,
		MemoryMB:          vm.memoryMb,
	}
	log.Printf("[DEBUG] virtual machine config spec: %v", configSpec)

	log.Printf("[DEBUG] starting extra custom config spec: %v", vm.customConfigurations)

	// make ExtraConfig
	if len(vm.customConfigurations) > 0 {
		var ov []types.BaseOptionValue
		for k, v := range vm.customConfigurations {
			key := k
			value := v
			o := types.OptionValue{
				Key:   key,
				Value: &value,
			}
			ov = append(ov, &o)
		}
		configSpec.ExtraConfig = ov
		log.Printf("[DEBUG] virtual machine Extra Config spec: %v", configSpec.ExtraConfig)
	}

	// create CustomizationSpec
	customSpec := types.CustomizationSpec{
		Identity: &types.CustomizationLinuxPrep{
			HostName: &types.CustomizationFixedName{
				Name: strings.Split(vm.name, ".")[0],
			},
			Domain:     vm.domain,
			TimeZone:   vm.timeZone,
			HwClockUTC: types.NewBool(true),
		},
		GlobalIPSettings: types.CustomizationGlobalIPSettings{
			DnsSuffixList: vm.dnsSuffixes,
			DnsServerList: vm.dnsServers,
		},
		NicSettingMap: networkConfigs,
	}
	log.Printf("[DEBUG] custom spec: %v", customSpec)

	// make vm clone spec
	cloneSpec := types.VirtualMachineCloneSpec{
		Location: relocateSpec,
		Template: false,
		Config:   &configSpec,
		PowerOn:  false,
	}
	log.Printf("[DEBUG] clone spec: %v", cloneSpec)

	task, err := template.Clone(context.TODO(), folder, vm.name, cloneSpec)
	if err != nil {
		return err
	}

	_, err = task.WaitForResult(context.TODO(), nil)
	if err != nil {
		return err
	}

	newVM, err := finder.VirtualMachine(context.TODO(), vm.Path())
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] new vm: %v", newVM)

	devices, err := newVM.Device(context.TODO())
	if err != nil {
		log.Printf("[DEBUG] Template devices can't be found")
		return err
	}

	for _, dvc := range devices {
		// Issue 3559/3560: Delete all ethernet devices to add the correct ones later
		if devices.Type(dvc) == "ethernet" {
			err := newVM.RemoveDevice(context.TODO(), dvc)
			if err != nil {
				return err
			}
		}
	}
	// Add Network devices
	for _, dvc := range networkDevices {
		err := newVM.AddDevice(
			context.TODO(), dvc.GetVirtualDeviceConfigSpec().Device)
		if err != nil {
			return err
		}
	}

	taskb, err := newVM.Customize(context.TODO(), customSpec)
	if err != nil {
		return err
	}

	_, err = taskb.WaitForResult(context.TODO(), nil)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG]VM customization finished")

	for i := 1; i < len(vm.hardDisks); i++ {
		err = addHardDisk(newVM, vm.hardDisks[i].size, vm.hardDisks[i].iops, vm.hardDisks[i].initType)
		if err != nil {
			return err
		}
	}
	log.Printf("[DEBUG] virtual machine config spec: %v", configSpec)

	for _, cd := range vm.cdRoms {
		log.Printf("[DEBUG] add cdrom iso: %v", cd.isoPath)
		err = addCdRom(newVM, datastore, cd.isoPath)
		if err != nil {
			return err
		}
	}

	newVM.PowerOn(context.TODO())

	ip, err := newVM.WaitForIP(context.TODO())
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] ip address: %v", ip)

	return nil
}
