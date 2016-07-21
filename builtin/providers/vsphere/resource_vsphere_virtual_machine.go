//go:generate constarray -type=controllerType,nicType,provisioningType
//go:generate stringalias -type=controllerType,nicType,provisioningType

package vsphere

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"

	"time"

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

var DiskControllerTypes = []controllerType{
	controllerTypeSCSI,
	controllerTypeLSIParallel,

	"scsi",
	"scsi-lsi-parallel",
	"scsi-buslogic",
	"scsi-paravirtual",
	"scsi-lsi-sas",
	"ide",
}

type nicType string

const (
	nicTypeE1000   nicType = "e1000"
	nicTypeVmxnet3 nicType = "vmxnet3"
)

type controllerType string

const (
	controllerTypeSCSI        controllerType = "scsi"
	controllerTypeLSIParallel controllerType = "scsi-lsi-parallel"
	controllerTypeLSISAS      controllerType = "scsi-lsi-sas"
	controllerTypeBuslogic    controllerType = "scsi-buslogic"
	controllerTypePV          controllerType = "scsi-paravirtual"
	controllerTypeIDE         controllerType = "ide"
)

type provisioningType string

const (
	provisioningTypeEager     provisioningType = "eager_zeroed"
	provisioningTypeThin      provisioningType = "thin"
	provisioningTypeThickLazy provisioningType = "thick_lazy"
)

type networkInterface struct {
	deviceName       string
	label            string
	ipv4Address      string
	ipv4PrefixLength int
	ipv4Gateway      string
	ipv6Address      string
	ipv6PrefixLength int
	ipv6Gateway      string
	adapterType      nicType // TODO: Make "adapter_type" argument
	macAddress       string
}

type hardDisk struct {
	name       string
	size       int64
	iops       int64
	initType   provisioningType
	vmdkPath   string
	controller controllerType
	bootable   bool
}

//Additional options Vsphere can use clones of windows machines
type windowsOptConfig struct {
	productKey         string
	adminPassword      string
	domainUser         string
	domain             string
	domainUserPassword string
}

type cdrom struct {
	datastore string
	path      string
}

type memoryAllocation struct {
	reservation int64
}

type virtualMachine struct {
	name                  string
	folder                string
	datacenter            string
	cluster               string
	resourcePool          string
	datastore             string
	vcpu                  int32
	memoryMb              int64
	memoryAllocation      memoryAllocation
	template              string
	guestID               types.VirtualMachineGuestOsIdentifier
	networkInterfaces     []networkInterface
	hardDisks             []hardDisk
	cdroms                []cdrom
	domain                string
	timeZone              string
	dnsSuffixes           []string
	dnsServers            []string
	hasBootableVmdk       bool
	linkedClone           bool
	skipCustomization     bool
	enableDiskUUID        bool
	windowsOptionalConfig windowsOptConfig
	customConfigurations  map[string](types.AnyType)
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
		Update: resourceVSphereVirtualMachineUpdate,
		Delete: resourceVSphereVirtualMachineDelete,

		SchemaVersion: 1,
		MigrateState:  resourceVSphereVirtualMachineMigrateState,

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
			},

			"memory": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},

			"memory_reservation": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
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

			"linked_clone": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},
			"gateway": &schema.Schema{
				Type:       schema.TypeString,
				Optional:   true,
				ForceNew:   true,
				Deprecated: "Please use network_interface.ipv4_gateway",
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

			"skip_customization": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Default:  false,
			},

			"enable_disk_uuid": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Default:  false,
			},

			"uuid": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"custom_configuration_parameters": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},

			"guest_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"windows_opt_config": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"product_key": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"admin_password": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"domain_user": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"domain": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"domain_user_password": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
					},
				},
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

						"ipv4_gateway": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"ipv6_address": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"ipv6_prefix_length": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},

						"ipv6_gateway": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"adapter_type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"mac_address": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
					},
				},
			},

			"disk": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"uuid": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},

						"key": &schema.Schema{
							Type:     schema.TypeInt,
							Computed: true,
						},

						"template": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"type": &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Default:      provisioningTypeEager,
							ValidateFunc: validatorFromValue("type", provisioningTypeValuesAsInterface),
						},

						"datastore": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"size": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},

						"name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"iops": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},

						"vmdk": &schema.Schema{
							// TODO: Add ValidateFunc to confirm path exists
							Type:     schema.TypeString,
							Optional: true,
						},

						"bootable": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},

						"keep_on_remove": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},

						"controller_type": &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "scsi",
							ValidateFunc: validatorFromValue("controller_type", controllerTypeValuesAsInterface),
						},
					},
				},
			},

			"cdrom": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"datastore": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},

						"path": &schema.Schema{
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

	vm := virtualMachine{}
	populateVMStruct(d, &vm)

	err := vm.setupVirtualMachine(client)
	if err != nil {
		return err
	}

	d.SetId(vm.Path())
	log.Printf("[INFO] Created virtual machine: %s", d.Id())

	return resourceVSphereVirtualMachineRead(d, meta)
}

func resourceVSphereVirtualMachineUpdate(d *schema.ResourceData, meta interface{}) error {
	// flag if changes have to be applied
	hasChanges := false
	// flag if changes have to be done when powered off
	rebootRequired := false

	// make config spec
	configSpec := types.VirtualMachineConfigSpec{}

	if d.HasChange("vcpu") {
		configSpec.NumCPUs = int32(d.Get("vcpu").(int))
		hasChanges = true
		rebootRequired = true
	}

	if d.HasChange("memory") {
		configSpec.MemoryMB = int64(d.Get("memory").(int))
		hasChanges = true
		rebootRequired = true
	}

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

	if d.HasChange("disk") {
		hasChanges = true
		oldDisks, newDisks := d.GetChange("disk")
		oldDiskSet := oldDisks.(*schema.Set)
		newDiskSet := newDisks.(*schema.Set)

		addedDisks := newDiskSet.Difference(oldDiskSet)
		removedDisks := oldDiskSet.Difference(newDiskSet)

		// Removed disks
		for _, diskRaw := range removedDisks.List() {
			if disk, ok := diskRaw.(map[string]interface{}); ok {
				devices, err := vm.Device(context.TODO())
				if err != nil {
					return fmt.Errorf("[ERROR] Update Remove Disk - Could not get virtual device list: %v", err)
				}
				virtualDisk := devices.FindByKey(int32(disk["key"].(int)))

				keep := false
				if v, ok := disk["keep_on_remove"].(bool); ok {
					keep = v
				}

				err = vm.RemoveDevice(context.TODO(), keep, virtualDisk)
				if err != nil {
					return fmt.Errorf("[ERROR] Update Remove Disk - Error removing disk: %v", err)
				}
			}
		}
		// Added disks
		for _, diskRaw := range addedDisks.List() {
			if disk, ok := diskRaw.(map[string]interface{}); ok {

				var datastore *object.Datastore
				if disk["datastore"] == "" {
					datastore, err = finder.DefaultDatastore(context.TODO())
					if err != nil {
						return fmt.Errorf("[ERROR] Update Remove Disk - Error finding datastore: %v", err)
					}
				} else {
					datastore, err = finder.Datastore(context.TODO(), disk["datastore"].(string))
					if err != nil {
						log.Printf("[ERROR] Couldn't find datastore %v.  %s", disk["datastore"].(string), err)
						return err
					}
				}

				var size int64
				if disk["size"] == 0 {
					size = 0
				} else {
					size = int64(disk["size"].(int))
				}
				iops := int64(disk["iops"].(int))
				controller_type := disk["controller_type"].(controllerType)

				var mo mo.VirtualMachine
				vm.Properties(context.TODO(), vm.Reference(), []string{"summary", "config"}, &mo)

				var diskPath string
				switch {
				case disk["vmdk"] != "":
					diskPath = disk["vmdk"].(string)
				case disk["name"] != "":
					snapshotFullDir := mo.Config.Files.SnapshotDirectory
					split := strings.Split(snapshotFullDir, " ")
					if len(split) != 2 {
						return fmt.Errorf("[ERROR] createVirtualMachine - failed to split snapshot directory: %v", snapshotFullDir)
					}
					vmWorkingPath := split[1]
					diskPath = vmWorkingPath + disk["name"].(string)
				default:
					return fmt.Errorf("[ERROR] resourceVSphereVirtualMachineUpdate - Neither vmdk path nor vmdk name was given")
				}

				var initType string
				if disk["type"] != "" {
					initType = disk["type"].(string)
				} else {
					initType = "thin"
				}

				log.Printf("[INFO] Attaching disk: %v", diskPath)
				err = addHardDisk(vm, size, iops, initType, datastore, diskPath, controller_type)
				if err != nil {
					log.Printf("[ERROR] Add Hard Disk Failed: %v", err)
					return err
				}
			}
			if err != nil {
				return err
			}
		}
	}

	// do nothing if there are no changes
	if !hasChanges {
		return nil
	}

	log.Printf("[DEBUG] virtual machine config spec: %v", configSpec)

	if rebootRequired {
		log.Printf("[INFO] Shutting down virtual machine: %s", d.Id())

		task, err := vm.PowerOff(context.TODO())
		if err != nil {
			return err
		}

		err = task.Wait(context.TODO())
		if err != nil {
			return err
		}
	}

	log.Printf("[INFO] Reconfiguring virtual machine: %s", d.Id())

	task, err := vm.Reconfigure(context.TODO(), configSpec)
	if err != nil {
		log.Printf("[ERROR] %s", err)
	}

	err = task.Wait(context.TODO())
	if err != nil {
		log.Printf("[ERROR] %s", err)
	}

	if rebootRequired {
		task, err = vm.PowerOn(context.TODO())
		if err != nil {
			return err
		}

		err = task.Wait(context.TODO())
		if err != nil {
			log.Printf("[ERROR] %s", err)
		}
	}

	return resourceVSphereVirtualMachineRead(d, meta)
}

func resourceVSphereVirtualMachineRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] virtual machine resource data: %#v", d)
	var err error
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
	if err = waitForIP(vm, time.Second*10); err != nil {
		return err
	}

	var mvm mo.VirtualMachine
	collector := property.DefaultCollector(client.Client)
	if err = collector.RetrieveOne(context.TODO(), vm.Reference(), []string{"guest", "summary", "datastore", "config"}, &mvm); err != nil {
		return err
	}

	log.Printf("[DEBUG] Datacenter - %#v", dc)
	log.Printf("[DEBUG] mvm.Summary.Config - %#v", mvm.Summary.Config)
	log.Printf("[DEBUG] mvm.Summary.Config - %#v", mvm.Config)
	log.Printf("[DEBUG] mvm.Guest.Net - %#v", mvm.Guest.Net)

	d.Set("datacenter", dc)
	d.Set("memory", mvm.Summary.Config.MemorySizeMB)
	d.Set("memory_reservation", mvm.Summary.Config.MemoryReservation)
	d.Set("cpu", mvm.Summary.Config.NumCpu)
	d.Set("uuid", mvm.Summary.Config.Uuid)

	if err = populateResourceDataDisks(d, mvm.Config.Hardware.Device); err != nil {
		return err
	}
	if err = populateResourceDataNetwork(d, &mvm); err != nil {
		return err
	}
	if err = populateResourceDataDatastore(d, &mvm, collector); err != nil {
		return err
	}
	return nil
}

func resourceVSphereVirtualMachineDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*govmomi.Client)
	var err error
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
	devices, err := vm.Device(context.TODO())
	if err != nil {
		log.Printf("[DEBUG] resourceVSphereVirtualMachineDelete - Failed to get device list: %v", err)
		return err
	}

	log.Printf("[INFO] Deleting virtual machine: %s", d.Id())
	if err = shutdownVM(vm); err != nil {
		return err
	}

	// Safely eject any disks the user marked as keep_on_remove
	if vL, ok := d.GetOk("disk"); ok {
		if diskSet, ok := vL.(*schema.Set); ok {
			if err = unmountDisks(vm, &devices, diskSet.List()); err != nil {
				return err
			}
		}
	}

	if err = destroyVM(vm); err != nil {
		return err
	}
	d.SetId("")
	return nil
}

// addHardDisk adds a new Hard Disk to the VirtualMachine.
func addHardDisk(vm *object.VirtualMachine, size, iops int64, diskType provisioningType, datastore *object.Datastore, diskPath string, controller_type controllerType) error {
	var err error
	devices, err := vm.Device(context.TODO())
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] vm devices: %#v\n", devices)

	var controller types.BaseVirtualController
	switch controller_type {
	case controllerTypeSCSI:
		controller, err = devices.FindDiskController(string(controller_type))
	case controllerTypeLSIParallel:
		controller = devices.PickController(&types.VirtualLsiLogicController{})
	case controllerTypeBuslogic:
		controller = devices.PickController(&types.VirtualBusLogicController{})
	case controllerTypePV:
		controller = devices.PickController(&types.ParaVirtualSCSIController{})
	case controllerTypeLSISAS:
		controller = devices.PickController(&types.VirtualLsiLogicSASController{})
	case controllerTypeIDE:
		controller, err = devices.FindDiskController(string(controller_type))
	default:
		return fmt.Errorf("[ERROR] Unsupported disk controller provided: %v", controller_type)
	}

	if err != nil || controller == nil {
		// Check if max number of scsi controller are already used
		diskControllers := getSCSIControllers(devices)
		if len(diskControllers) >= 4 {
			return fmt.Errorf("[ERROR] Maximum number of SCSI controllers created")
		}

		log.Printf("[DEBUG] Couldn't find a %v controller.  Creating one..", controller_type)

		var c types.BaseVirtualDevice
		switch controller_type {
		case controllerTypeSCSI:
			// Create scsi controller
			c, err = devices.CreateSCSIController("scsi")
			if err != nil {
				return fmt.Errorf("[ERROR] Failed creating SCSI controller: %v", err)
			}
		case controllerTypeLSIParallel:
			// Create scsi controller
			c, err = devices.CreateSCSIController("lsilogic")
			if err != nil {
				return fmt.Errorf("[ERROR] Failed creating SCSI controller: %v", err)
			}
		case controllerTypeBuslogic:
			// Create scsi controller
			c, err = devices.CreateSCSIController("buslogic")
			if err != nil {
				return fmt.Errorf("[ERROR] Failed creating SCSI controller: %v", err)
			}
		case controllerTypePV:
			// Create scsi controller
			c, err = devices.CreateSCSIController("pvscsi")
			if err != nil {
				return fmt.Errorf("[ERROR] Failed creating SCSI controller: %v", err)
			}
		case controllerTypeLSISAS:
			// Create scsi controller
			c, err = devices.CreateSCSIController("lsilogic-sas")
			if err != nil {
				return fmt.Errorf("[ERROR] Failed creating SCSI controller: %v", err)
			}
		case controllerTypeIDE:
			// Create ide controller
			c, err = devices.CreateIDEController()
			if err != nil {
				return fmt.Errorf("[ERROR] Failed creating IDE controller: %v", err)
			}
		default:
			return fmt.Errorf("[ERROR] Unsupported disk controller provided: %v", controller_type)
		}

		vm.AddDevice(context.TODO(), c)
		// Update our devices list
		devices, err = vm.Device(context.TODO())
		if err != nil {
			return err
		}
		controller = devices.PickController(c.(types.BaseVirtualController))
		if controller == nil {
			log.Printf("[ERROR] Could not find the new %v controller", controller_type)
			return fmt.Errorf("Could not find the new %v controller", controller_type)
		}
	}

	log.Printf("[DEBUG] disk controller: %#v\n", controller)

	// TODO Check if diskPath & datastore exist
	// If diskPath is not specified, pass empty string to CreateDisk()
	if diskPath == "" {
		return fmt.Errorf("[ERROR] addHardDisk - No path proided")
	} else {
		diskPath = datastore.Path(diskPath)
	}
	diskPath = fmt.Sprintf("[%v] %v", datastore.Name(), diskPath)
	log.Printf("[DEBUG] addHardDisk - diskPath: %v", diskPath)
	disk := devices.CreateDisk(controller, datastore.Reference(), diskPath)

	if strings.Contains(controller_type, "scsi") {
		unitNumber, err := getNextUnitNumber(devices, controller)
		if err != nil {
			return err
		}
		*disk.UnitNumber = unitNumber
	}

	existing := devices.SelectByBackingInfo(disk.Backing)
	log.Printf("[DEBUG] disk: %#v\n", disk)

	if len(existing) != 0 {
		log.Printf("[DEBUG] addHardDisk: Disk already present.\n")
		return nil
	}
	disk.CapacityInKB = int64(size * 1024 * 1024)
	if iops != 0 {
		disk.StorageIOAllocation = &types.StorageIOAllocationInfo{
			Limit: iops,
		}
	}
	backing := disk.Backing.(*types.VirtualDiskFlatVer2BackingInfo)

	if diskType == provisioningTypeEager {
		// eager zeroed thick virtual disk
		backing.ThinProvisioned = types.NewBool(false)
		backing.EagerlyScrub = types.NewBool(true)
	} else if diskType == provisioningTypeThin {
		// thin provisioned virtual disk
		backing.ThinProvisioned = types.NewBool(true)
	} else if diskType == provisioningTypeThickLazy {
		// thin provisioned virtual disk
		backing.ThinProvisioned = types.NewBool(false)
		backing.EagerlyScrub = types.NewBool(false)
	}

	log.Printf("[DEBUG] addHardDisk: %#v\n", disk)
	log.Printf("[DEBUG] addHardDisk capacity: %#v\n", disk.CapacityInKB)

	return vm.AddDevice(context.TODO(), disk)
}

func getSCSIControllers(vmDevices object.VirtualDeviceList) []*types.VirtualController {
	// get virtual scsi controllers of all supported types
	var scsiControllers []*types.VirtualController
	for _, device := range vmDevices {
		devType := vmDevices.Type(device)
		switch devType {
		case "scsi", "lsilogic", "buslogic", "pvscsi", "lsilogic-sas":
			if c, ok := device.(types.BaseVirtualController); ok {
				scsiControllers = append(scsiControllers, c.GetVirtualController())
			}
		}
	}
	return scsiControllers
}

func getNextUnitNumber(devices object.VirtualDeviceList, c types.BaseVirtualController) (int32, error) {
	key := c.GetVirtualController().Key

	var unitNumbers [16]bool
	unitNumbers[7] = true

	for _, device := range devices {
		d := device.GetVirtualDevice()

		if d.ControllerKey == key {
			if d.UnitNumber != nil {
				unitNumbers[*d.UnitNumber] = true
			}
		}
	}
	for i, taken := range unitNumbers {
		if !taken {
			return int32(i), nil
		}
	}
	return -1, fmt.Errorf("[ERROR] getNextUnitNumber - controller is full")
}

// addCdrom adds a new virtual cdrom drive to the VirtualMachine and attaches an image (ISO) to it from a datastore path.
func addCdrom(client *govmomi.Client, vm *object.VirtualMachine, datacenter *object.Datacenter, datastore, path string) error {
	devices, err := vm.Device(context.TODO())
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] vm devices: %#v", devices)

	var controller *types.VirtualIDEController
	controller, err = devices.FindIDEController("")
	if err != nil {
		log.Printf("[DEBUG] Couldn't find a ide controller.  Creating one..")

		var c types.BaseVirtualDevice
		c, err := devices.CreateIDEController()
		if err != nil {
			return fmt.Errorf("[ERROR] Failed creating IDE controller: %v", err)
		}

		if v, ok := c.(*types.VirtualIDEController); ok {
			controller = v
		} else {
			return fmt.Errorf("[ERROR] Controller type could not be asserted")
		}
		vm.AddDevice(context.TODO(), c)
		// Update our devices list
		devices, err := vm.Device(context.TODO())
		if err != nil {
			return err
		}
		controller, err = devices.FindIDEController("")
		if err != nil {
			log.Printf("[ERROR] Could not find the new disk IDE controller: %v", err)
			return err
		}
	}
	log.Printf("[DEBUG] ide controller: %#v", controller)

	c, err := devices.CreateCdrom(controller)
	if err != nil {
		return err
	}

	finder := find.NewFinder(client.Client, true)
	finder = finder.SetDatacenter(datacenter)
	ds, err := getDatastore(finder, datastore)
	if err != nil {
		return err
	}

	c = devices.InsertIso(c, ds.Path(path))
	log.Printf("[DEBUG] addCdrom: %#v", c)

	return vm.AddDevice(context.TODO(), c)
}

// buildNetworkDevice builds VirtualDeviceConfigSpec for Network Device.
func buildNetworkDevice(f *find.Finder, label string, adapterType nicType, macAddress string) (*types.VirtualDeviceConfigSpec, error) {
	network, err := f.Network(context.TODO(), "*"+label)
	if err != nil {
		return nil, err
	}

	backing, err := network.EthernetCardBackingInfo(context.TODO())
	if err != nil {
		return nil, err
	}

	var addressType string
	if macAddress == "" {
		addressType = string(types.VirtualEthernetCardMacTypeGenerated)
	} else {
		addressType = string(types.VirtualEthernetCardMacTypeManual)
	}

	switch adapterType {
	case nicTypeVmxnet3:
		return &types.VirtualDeviceConfigSpec{
			Operation: types.VirtualDeviceConfigSpecOperationAdd,
			Device: &types.VirtualVmxnet3{
				VirtualVmxnet: types.VirtualVmxnet{
					VirtualEthernetCard: types.VirtualEthernetCard{
						VirtualDevice: types.VirtualDevice{
							Key:     -1,
							Backing: backing,
						},
						AddressType: addressType,
						MacAddress:  macAddress,
					},
				},
			},
		}, nil
	case nicTypeE1000:
		return &types.VirtualDeviceConfigSpec{
			Operation: types.VirtualDeviceConfigSpecOperationAdd,
			Device: &types.VirtualE1000{
				VirtualEthernetCard: types.VirtualEthernetCard{
					VirtualDevice: types.VirtualDevice{
						Key:     -1,
						Backing: backing,
					},
					AddressType: addressType,
					MacAddress:  macAddress,
				},
			},
		}, nil
	default:
		return nil, fmt.Errorf("Invalid network adapter type.")
	}
}

// buildVMRelocateSpec builds VirtualMachineRelocateSpec to set a place for a new VirtualMachine.
func buildVMRelocateSpec(rp *object.ResourcePool, ds *object.Datastore, vm *object.VirtualMachine, linkedClone bool, initType provisioningType) (types.VirtualMachineRelocateSpec, error) {
	var key int32
	var moveType string
	if linkedClone {
		moveType = "createNewChildDiskBacking"
	} else {
		moveType = "moveAllDiskBackingsAndDisallowSharing"
	}
	log.Printf("[DEBUG] relocate type: [%s]", moveType)

	devices, err := vm.Device(context.TODO())
	if err != nil {
		return types.VirtualMachineRelocateSpec{}, err
	}
	for _, d := range devices {
		if devices.Type(d) == "disk" {
			key = int32(d.GetVirtualDevice().Key)
		}
	}

	isThin := initType == provisioningTypeThin
	eagerScrub := initType == provisioningTypeEager
	rpr := rp.Reference()
	dsr := ds.Reference()
	return types.VirtualMachineRelocateSpec{
		Datastore:    &dsr,
		Pool:         &rpr,
		DiskMoveType: moveType,
		Disk: []types.VirtualMachineRelocateSpecDiskLocator{
			{
				Datastore: dsr,
				DiskBackingInfo: &types.VirtualDiskFlatVer2BackingInfo{
					DiskMode:        "persistent",
					ThinProvisioned: types.NewBool(isThin),
					EagerlyScrub:    types.NewBool(eagerScrub),
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
func buildStoragePlacementSpecCreate(f *object.DatacenterFolders, rp *object.ResourcePool, storagePod object.StoragePod, configSpec *types.VirtualMachineConfigSpec) types.StoragePlacementSpec {
	vmfr := f.VmFolder.Reference()
	rpr := rp.Reference()
	spr := storagePod.Reference()

	sps := types.StoragePlacementSpec{
		Type:       "create",
		ConfigSpec: configSpec,
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

	var key int32
	for _, d := range devices.SelectByType((*types.VirtualDisk)(nil)) {
		key = int32(d.GetVirtualDevice().Key)
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
					{
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

func (vm *virtualMachine) setupVirtualMachine(c *govmomi.Client) error {
	dc, err := getDatacenter(c, vm.datacenter)

	if err != nil {
		return err
	}
	finder := find.NewFinder(c.Client, true)
	finder = finder.SetDatacenter(dc)

	// spawn the VM
	newVM, networkDevices, networkConfigs, datastore, templateMo, err := vm.makeExist(c, dc, finder)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] new vm: %v", newVM)

	// Create the cdroms if needed.
	if err = createCdroms(newVM, vm.cdroms); err != nil {
		return err
	}
	// add the hard disks
	if err = vm.addHardDisks(newVM, datastore); err != nil {
		return err
	}

	// perform the basic customization
	if err = vm.doCustomization(newVM, templateMo, networkConfigs); err != nil {
		return err
	}

	// add the network devices
	if err = vmAddNetworkDevices(newVM, networkDevices); err != nil {
		return err
	}

	// boot the VM
	if vm.hasBootableVmdk || vm.template != "" {
		if err = powerOnVM(newVM); err != nil {
			return err
		}
	}
	return nil
}

// createCdroms is a helper function to attach virtual cdrom devices (and their attached disk images) to a virtual IDE controller.
func createCdroms(vm *object.VirtualMachine, cdroms []cdrom) error {
	log.Printf("[DEBUG] add cdroms: %v", cdroms)
	for _, cd := range cdroms {
		log.Printf("[DEBUG] add cdrom (datastore): %v", cd.datastore)
		log.Printf("[DEBUG] add cdrom (cd path): %v", cd.path)
		err := addCdrom(vm, cd.datastore, cd.path)
		if err != nil {
			return err
		}
	}

	return nil
}

func getResourcePool(vm *virtualMachine, finder *find.Finder) (*object.ResourcePool, error) {
	var resourcePool *object.ResourcePool
	var err error
	if vm.resourcePool == "" {
		if vm.cluster == "" {
			resourcePool, err = finder.DefaultResourcePool(context.TODO())
			if err != nil {
				return nil, err
			}
		} else {
			resourcePool, err = finder.ResourcePool(context.TODO(), "*"+vm.cluster+"/Resources")
			if err != nil {
				return nil, err
			}
		}
		return resourcePool, nil
	}
	resourcePool, err = finder.ResourcePool(context.TODO(), vm.resourcePool)
	if err != nil {
		return nil, err
	}
	return resourcePool, nil
}

func getProperFolder(vm *virtualMachine, c *govmomi.Client, defaultFolder *object.Folder) (*object.Folder, error) {
	folder := defaultFolder
	if len(vm.folder) > 0 {
		si := object.NewSearchIndex(c.Client)
		folderRef, err := si.FindByInventoryPath(
			context.TODO(), fmt.Sprintf("%v/vm/%v", vm.datacenter, vm.folder))
		if err != nil {
			return nil, fmt.Errorf("Error reading folder %s: %s", vm.folder, err)
		} else if folderRef == nil {
			return nil, fmt.Errorf("Cannot find folder %s", vm.folder)
		} else {
			folder = folderRef.(*object.Folder)
		}
	}
	return folder, nil
}

func vmAddNetworkDevices(vm *object.VirtualMachine, networkDevices []types.BaseVirtualDeviceConfigSpec) error {
	devices, err := vm.Device(context.TODO())
	if err != nil {
		log.Printf("[DEBUG] devices can't be found")
		return err
	}

	for _, dvc := range devices {
		// Issue 3559/3560: Delete all ethernet devices to add the correct ones later
		if devices.Type(dvc) != "ethernet" {
			continue
		}
		if err = vm.RemoveDevice(context.TODO(), false, dvc); err != nil {
			return err
		}
	}
	// Add Network devices
	for _, dvc := range networkDevices {
		if err = vm.AddDevice(context.TODO(), dvc.GetVirtualDeviceConfigSpec().Device); err != nil {
			return err
		}
	}
	return nil
}

func (vm *virtualMachine) doCustomization(vmObj *object.VirtualMachine, templateMo *mo.VirtualMachine, networkConfigs []types.CustomizationAdapterMapping) error {
	if vm.skipCustomization || vm.template == "" {
		return nil
	}
	identityOptions, err := vm.prepareCustomization(templateMo)
	if err != nil {
		return err
	}

	// create CustomizationSpec
	customSpec := types.CustomizationSpec{
		Identity: identityOptions,
		GlobalIPSettings: types.CustomizationGlobalIPSettings{
			DnsSuffixList: vm.dnsSuffixes,
			DnsServerList: vm.dnsServers,
		},
		NicSettingMap: networkConfigs,
	}
	log.Printf("[DEBUG] custom spec: %v", customSpec)

	log.Printf("[DEBUG] VM customization starting")
	taskb, err := vmObj.Customize(context.TODO(), customSpec)
	if err != nil {
		return err
	}
	if _, err = taskb.WaitForResult(context.TODO(), nil); err != nil {
		return err
	}
	log.Printf("[DEBUG] VM customization finished")
	return nil
}

func (vm *virtualMachine) prepareCustomization(templateMo *mo.VirtualMachine) (types.BaseCustomizationIdentitySettings, error) {
	if !strings.HasPrefix(templateMo.Config.GuestId, "win") {
		return &types.CustomizationLinuxPrep{
			HostName: &types.CustomizationFixedName{
				Name: strings.Split(vm.name, ".")[0],
			},
			Domain:     vm.domain,
			TimeZone:   vm.timeZone,
			HwClockUTC: types.NewBool(true),
		}, nil
	}
	var timeZone int
	if vm.timeZone == "Etc/UTC" {
		vm.timeZone = "085"
	}
	timeZone, err := strconv.Atoi(vm.timeZone)
	if err != nil {
		return nil, fmt.Errorf("Error converting TimeZone: %s", err)
	}

	guiUnattended := types.CustomizationGuiUnattended{
		AutoLogon:      false,
		AutoLogonCount: 1,
		TimeZone:       int32(timeZone),
	}

	customIdentification := types.CustomizationIdentification{}

	userData := types.CustomizationUserData{
		ComputerName: &types.CustomizationFixedName{
			Name: strings.Split(vm.name, ".")[0],
		},
		ProductId: vm.windowsOptionalConfig.productKey,
		FullName:  "terraform",
		OrgName:   "terraform",
	}

	if vm.windowsOptionalConfig.domainUserPassword != "" && vm.windowsOptionalConfig.domainUser != "" && vm.windowsOptionalConfig.domain != "" {
		customIdentification.DomainAdminPassword = &types.CustomizationPassword{
			PlainText: true,
			Value:     vm.windowsOptionalConfig.domainUserPassword,
		}
		customIdentification.DomainAdmin = vm.windowsOptionalConfig.domainUser
		customIdentification.JoinDomain = vm.windowsOptionalConfig.domain
	}

	if vm.windowsOptionalConfig.adminPassword != "" {
		guiUnattended.Password = &types.CustomizationPassword{
			PlainText: true,
			Value:     vm.windowsOptionalConfig.adminPassword,
		}
	}

	return &types.CustomizationSysprep{
		GuiUnattended:  guiUnattended,
		Identification: customIdentification,
		UserData:       userData,
	}, nil
}

func (vm *virtualMachine) addHardDisks(vmObj *object.VirtualMachine, datastore *object.Datastore) error {
	var vmMo *mo.VirtualMachine
	var err error
	vmObj.Properties(context.TODO(), vmObj.Reference(), []string{"summary", "config"}, vmMo)
	firstDisk := 0
	if vm.template != "" {
		firstDisk++
	}
	for i := firstDisk; i < len(vm.hardDisks); i++ {
		log.Printf("[DEBUG] disk index: %v", i)

		var diskPath string
		switch {
		case vm.hardDisks[i].vmdkPath != "":
			diskPath = vm.hardDisks[i].vmdkPath
		case vm.hardDisks[i].name != "":
			snapshotFullDir := vmMo.Config.Files.SnapshotDirectory
			split := strings.Split(snapshotFullDir, " ")
			if len(split) != 2 {
				return fmt.Errorf("[ERROR] setupVirtualMachine - failed to split snapshot directory: %v", snapshotFullDir)
			}
			vmWorkingPath := split[1]
			diskPath = vmWorkingPath + vm.hardDisks[i].name
		default:
			return fmt.Errorf("[ERROR] setupVirtualMachine - Neither vmdk path nor vmdk name was given: %#v", vm.hardDisks[i])
		}

		err = addHardDisk(vmObj, vm.hardDisks[i].size, vm.hardDisks[i].iops, vm.hardDisks[i].initType, datastore, diskPath, vm.hardDisks[i].controller)
		if err != nil {
			return err
		}
	}
	return nil
}

func (vm *virtualMachine) performConfigCoherenceTests() error {
	if vm.template != "" && vm.guestID != "" {
		return fmt.Errorf("Cannot enforce guestID if template is set aswell")
	}
	return nil
}

func (vm *virtualMachine) bootstrapConfigSpec() *types.VirtualMachineConfigSpec {
	configSpec := types.VirtualMachineConfigSpec{
		Name:              vm.name,
		NumCPUs:           vm.vcpu,
		NumCoresPerSocket: 1,
		MemoryMB:          vm.memoryMb,
		MemoryAllocation: &types.ResourceAllocationInfo{
			Reservation: vm.memoryAllocation.reservation,
		},
		Flags: &types.VirtualMachineFlagInfo{
			DiskUuidEnabled: &vm.enableDiskUUID,
		},
	}
	if vm.template == "" {
		if vm.guestID == "" {
			configSpec.GuestId = "otherLinux64Guest"
		} else {
			configSpec.GuestId = string(vm.guestID)
		}
	}
	return &configSpec
}

func (vm *virtualMachine) getTemplate(finder *find.Finder) (template *object.VirtualMachine, templateMo *mo.VirtualMachine, err error) {
	templateMo = &mo.VirtualMachine{}
	if vm.template != "" {
		template, err = finder.VirtualMachine(context.TODO(), vm.template)
		if err != nil {
			return
		}
		log.Printf("[DEBUG] template: %#v", template)

		err = template.Properties(context.TODO(), template.Reference(), []string{"parent", "config.template", "config.guestId", "resourcePool", "snapshot", "guest.toolsVersionStatus2", "config.guestFullName"}, templateMo)
		if err != nil {
			return
		}
	}
	return
}

func (vm *virtualMachine) extraConfig(configSpec *types.VirtualMachineConfigSpec) error {
	// make extraConfig
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
	return nil
}

func (vm *virtualMachine) setupDatastore(c *govmomi.Client, resourcePool *object.ResourcePool, template *object.VirtualMachine, dcFolders *object.DatacenterFolders, configSpec *types.VirtualMachineConfigSpec, finder *find.Finder) (*object.Datastore, *mo.Datastore, error) {
	var err error
	var datastore *object.Datastore
	var mds *mo.Datastore
	if vm.datastore == "" {
		datastore, err = finder.DefaultDatastore(context.TODO())
		if err != nil {
			return nil, nil, err
		}
		return datastore, nil, nil
	}
	datastore, err = finder.Datastore(context.TODO(), vm.datastore)
	if err != nil {
		// TODO: datastore cluster support in govmomi finder function
		d, err := getDatastoreObject(c, dcFolders, vm.datastore)
		if err != nil {
			return nil, nil, err
		}

		if d.Type == "StoragePod" {
			sp := object.StoragePod{
				Folder: object.NewFolder(c.Client, d),
			}

			var sps types.StoragePlacementSpec
			if vm.template != "" {
				sps = buildStoragePlacementSpecClone(c, dcFolders, template, resourcePool, sp)
			} else {
				sps = buildStoragePlacementSpecCreate(dcFolders, resourcePool, sp, configSpec)
			}

			datastore, err = findDatastore(c, sps)
			if err != nil {
				return nil, nil, err
			}
		} else {
			datastore = object.NewDatastore(c.Client, d)
		}
	}
	if err = datastore.Properties(context.TODO(), datastore.Reference(), []string{"name"}, mds); err != nil {
		return nil, nil, err
	}

	log.Printf("[DEBUG] datastore: %#v", datastore)
	log.Printf("[DEBUG] mds: %#v", mds)

	return datastore, mds, nil
}

func (vm *virtualMachine) setupNetwork(finder *find.Finder) ([]types.BaseVirtualDeviceConfigSpec, []types.CustomizationAdapterMapping, error) {
	networkDevices := []types.BaseVirtualDeviceConfigSpec{}
	networkConfigs := []types.CustomizationAdapterMapping{}
	for _, network := range vm.networkInterfaces {
		// network device
		var networkDeviceType nicType
		if network.adapterType != "" {
			networkDeviceType = network.adapterType
		} else if vm.template == "" {
			networkDeviceType = nicTypeE1000
		} else {
			networkDeviceType = nicTypeVmxnet3
		}

		nd, err := buildNetworkDevice(finder, network.label, networkDeviceType, network.macAddress)
		if err != nil {
			return nil, nil, err
		}
		log.Printf("[DEBUG] network device: %+v", nd.Device)
		networkDevices = append(networkDevices, nd)

		if vm.template == "" {
			continue
		}
		// work for customization (only relevant for templated VMs)
		var ipSetting types.CustomizationIPSettings
		if network.ipv4Address == "" {
			ipSetting.Ip = &types.CustomizationDhcpIpGenerator{}
		} else {
			if network.ipv4PrefixLength == 0 {
				return nil, nil, fmt.Errorf("Error: ipv4_prefix_length argument is empty.")
			}
			m := net.CIDRMask(network.ipv4PrefixLength, 32)
			sm := net.IPv4(m[0], m[1], m[2], m[3])
			subnetMask := sm.String()
			log.Printf("[DEBUG] ipv4 gateway: %v\n", network.ipv4Gateway)
			log.Printf("[DEBUG] ipv4 address: %v\n", network.ipv4Address)
			log.Printf("[DEBUG] ipv4 prefix length: %v\n", network.ipv4PrefixLength)
			log.Printf("[DEBUG] ipv4 subnet mask: %v\n", subnetMask)
			ipSetting.Gateway = []string{
				network.ipv4Gateway,
			}
			ipSetting.Ip = &types.CustomizationFixedIp{
				IpAddress: network.ipv4Address,
			}
			ipSetting.SubnetMask = subnetMask
		}

		ipv6Spec := &types.CustomizationIPSettingsIpV6AddressSpec{}
		if network.ipv6Address == "" {
			ipv6Spec.Ip = []types.BaseCustomizationIpV6Generator{
				&types.CustomizationDhcpIpV6Generator{},
			}
		} else {
			log.Printf("[DEBUG] ipv6 gateway: %v\n", network.ipv6Gateway)
			log.Printf("[DEBUG] ipv6 address: %v\n", network.ipv6Address)
			log.Printf("[DEBUG] ipv6 prefix length: %v\n", network.ipv6PrefixLength)

			ipv6Spec.Ip = []types.BaseCustomizationIpV6Generator{
				&types.CustomizationFixedIpV6{
					IpAddress:  network.ipv6Address,
					SubnetMask: int32(network.ipv6PrefixLength),
				},
			}
			ipv6Spec.Gateway = []string{network.ipv6Gateway}
		}
		ipSetting.IpV6Spec = ipv6Spec

		// network config
		config := types.CustomizationAdapterMapping{
			Adapter: ipSetting,
		}
		networkConfigs = append(networkConfigs, config)
	}
	log.Printf("[DEBUG] network devices: %#v", networkDevices)
	log.Printf("[DEBUG] network configs: %#v", networkConfigs)
	return networkDevices, networkConfigs, nil
}

func (vm *virtualMachine) makeExist(c *govmomi.Client, dc *object.Datacenter, finder *find.Finder) (*object.VirtualMachine, []types.BaseVirtualDeviceConfigSpec, []types.CustomizationAdapterMapping, *object.Datastore, *mo.VirtualMachine, error) {
	template, templateMo, err := vm.getTemplate(finder)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	resourcePool, err := getResourcePool(vm, finder)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	log.Printf("[DEBUG] resource pool: %#v", resourcePool)

	dcFolders, err := dc.Folders(context.TODO())
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	log.Printf("[DEBUG] folder: %#v", vm.folder)

	folder, err := getProperFolder(vm, c, dcFolders.VmFolder)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	if err = vm.performConfigCoherenceTests(); err != nil {
		return nil, nil, nil, nil, nil, err
	}
	// make config spec
	configSpec := vm.bootstrapConfigSpec()
	log.Printf("[DEBUG] virtual machine config spec: %v", configSpec)

	if err = vm.extraConfig(configSpec); err != nil {
		return nil, nil, nil, nil, nil, err
	}

	datastore, mds, err := vm.setupDatastore(c, resourcePool, template, dcFolders, configSpec, finder)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	// network
	networkDevices, networkConfigs, err := vm.setupNetwork(finder)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	if vm.template == "" {
		err = createVMWithoutTemplate(configSpec, mds, resourcePool, folder)
	} else {
		err = createVMWithTemplate(vm.name, vm.linkedClone, vm.hardDisks[0].initType, template, templateMo, configSpec, resourcePool, datastore, folder)
	}
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	newVM, err := finder.VirtualMachine(context.TODO(), vm.Path())
	return newVM, networkDevices, networkConfigs, datastore, templateMo, err
}

// utility functions for object.VirtualMachine objects

func shutdownVM(vm *object.VirtualMachine) error {
	state, err := vm.PowerState(context.TODO())
	if err != nil {
		return err
	}
	if state == types.VirtualMachinePowerStatePoweredOn {
		task, err := vm.PowerOff(context.TODO())
		if err != nil {
			return err
		}

		err = task.Wait(context.TODO())
		if err != nil {
			return err
		}
	}
	return nil
}

func unmountDisks(vm *object.VirtualMachine, devices *object.VirtualDeviceList, diskSet []interface{}) error {
	var err error
	for _, value := range diskSet {
		disk := value.(map[string]interface{})
		v, ok := disk["keep_on_remove"].(bool)
		if !ok || !v {
			continue
		}
		log.Printf("[DEBUG] not destroying %v", disk["name"])
		virtualDisk := devices.FindByKey(int32(disk["key"].(int)))
		err = vm.RemoveDevice(context.TODO(), true, virtualDisk)
		if err != nil {
			log.Printf("[ERROR] Update Remove Disk - Error removing disk: %v", err)
			return err
		}
	}
	return nil
}

func destroyVM(vm *object.VirtualMachine) error {
	task, err := vm.Destroy(context.TODO())
	if err != nil {
		return err
	}

	err = task.Wait(context.TODO())
	if err != nil {
		return err
	}
	return nil
}

func powerOnVM(vm *object.VirtualMachine) error {
	var err error
	vm.PowerOn(context.TODO())
	err = vm.WaitForPowerState(context.TODO(), types.VirtualMachinePowerStatePoweredOn)
	if err != nil {
		return err
	}
	return nil
}

// waitForIP tests if a machine is up and if it is up waits for its IP to show up
func waitForIP(vm *object.VirtualMachine, timeout time.Duration) error {
	state, err := vm.PowerState(context.TODO())
	if err != nil {
		return err
	}

	if state == types.VirtualMachinePowerStatePoweredOn {
		// wait for interfaces to appear
		_, err = vm.WaitForNetIP(context.TODO(), true)
		if err != nil {
			return err
		}
	}
	return nil
}

// validator helper function
func validatorFromValue(fieldName string, possibleValuesF func() []interface{}) func(v interface{}, k string) (ws []string, errors []error) {
	possibleValuesI := possibleValuesF()
	possibleValues := make([]stringer, len(possibleValuesI))
	var ok bool
	for i, v := range possibleValuesI {
		possibleValues[i], ok = v.(stringer)
		if !ok {
			log.Panicf("not a stringer %[1]T %[1]v", v)
		}
	}
	return func(v interface{}, k string) (ws []string, errors []error) {
		value := v.(string)
		found := false
		for _, t := range possibleValues {
			if t.String() == value {
				found = true
				break
			}
		}
		if !found {
			errors = append(errors, fmt.Errorf(
				"Supported values for '%s' are %v", fieldName, JoinStringer(possibleValues, ", ")))
		}
		return
	}
}

// populateVMStruct populates a VM struct from a ResourceData
func populateVMStruct(d *schema.ResourceData, vm *virtualMachine) error {
	vm.name = d.Get("name").(string)
	vm.vcpu = int32(d.Get("vcpu").(int))
	vm.memoryMb = int64(d.Get("memory").(int))
	vm.memoryAllocation = memoryAllocation{
		reservation: int64(d.Get("memory_reservation").(int)),
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

	if v, ok := d.GetOk("domain"); ok {
		vm.domain = v.(string)
	}

	if v, ok := d.GetOk("time_zone"); ok {
		vm.timeZone = v.(string)
	}

	if v, ok := d.GetOk("guest_id"); ok {
		vm.guestID = v.(types.VirtualMachineGuestOsIdentifier)
	}

	if v, ok := d.GetOk("linked_clone"); ok {
		vm.linkedClone = v.(bool)
	}

	if v, ok := d.GetOk("skip_customization"); ok {
		vm.skipCustomization = v.(bool)
	}

	if v, ok := d.GetOk("enable_disk_uuid"); ok {
		vm.enableDiskUUID = v.(bool)
	}

	if raw, ok := d.GetOk("dns_suffixes"); ok {
		for _, v := range raw.([]interface{}) {
			vm.dnsSuffixes = append(vm.dnsSuffixes, v.(string))
		}
	} else {
		vm.dnsSuffixes = DefaultDNSSuffixes
	}

	// Create the cdroms if needed.
	if err := createCdroms(c, newVM, dc, vm.cdroms); err != nil {
	if raw, ok := d.GetOk("dns_servers"); ok {
		for _, v := range raw.([]interface{}) {
			vm.dnsServers = append(vm.dnsServers, v.(string))
		}
	} else {
		vm.dnsServers = DefaultDNSServers
	}
	if err := populateVMStructConfigParams(d, vm); err != nil {
		return err
	}
	if err := populateVMStructNetwork(d, vm); err != nil {
		return err
	}
	if err := populateVMStructWindowsOptConfig(d, vm); err != nil {
		return err
	}
	if err := populateVMStructDisk(d, vm); err != nil {
		return err
	}
	if err := populateVMStructCdrom(d, vm); err != nil {
		return err
	}
	return nil
}

// helper functions for populateVMStruct

func populateVMStructConfigParams(d *schema.ResourceData, vm *virtualMachine) error {
	vL, ok := d.GetOk("custom_configuration_parameters")
	if !ok {
		return nil
	}
	customConfigs, ok := vL.(map[string]interface{})
	if !ok {
		return nil
	}
	custom := make(map[string]types.AnyType)
	for k, v := range customConfigs {
		custom[k] = v
	}
	vm.customConfigurations = custom
	log.Printf("[DEBUG] custom_configuration_parameters init: %v", vm.customConfigurations)
	return nil
}

func populateVMStructNetwork(d *schema.ResourceData, vm *virtualMachine) error {
	vL, ok := d.GetOk("network_interface")
	if !ok {
		return nil
	}
	networks := make([]networkInterface, len(vL.([]interface{})))
	for i, v := range vL.([]interface{}) {
		network := v.(map[string]interface{})
		networks[i].label = network["label"].(string)
		if v, ok := network["ip_address"].(string); ok && v != "" {
			networks[i].ipv4Address = v
		}
		if v, ok := d.GetOk("gateway"); ok {
			networks[i].ipv4Gateway = v.(string)
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
		if v, ok := network["ipv4_gateway"].(string); ok && v != "" {
			networks[i].ipv4Gateway = v
		}
		if v, ok := network["ipv6_address"].(string); ok && v != "" {
			networks[i].ipv6Address = v
		}
		if v, ok := network["ipv6_prefix_length"].(int); ok && v != 0 {
			networks[i].ipv6PrefixLength = v
		}
		if v, ok := network["ipv6_gateway"].(string); ok && v != "" {
			networks[i].ipv6Gateway = v
		}
		if v, ok := network["mac_address"].(string); ok && v != "" {
			networks[i].macAddress = v
		}
		if v, ok := network["adapter_type"].(nicType); ok && v != "" {
			networks[i].adapterType = v
		}
	}
	vm.networkInterfaces = networks
	log.Printf("[DEBUG] network_interface init: %v", networks)
	return nil
}

func populateVMStructWindowsOptConfig(d *schema.ResourceData, vm *virtualMachine) error {
	vL, ok := d.GetOk("windows_opt_config")
	if !ok {
		return nil
	}
	var winOpt windowsOptConfig
	customConfigs := (vL.([]interface{}))[0].(map[string]interface{})
	if v, ok := customConfigs["admin_password"].(string); ok && v != "" {
		winOpt.adminPassword = v
	}
	if v, ok := customConfigs["domain"].(string); ok && v != "" {
		winOpt.domain = v
	}
	if v, ok := customConfigs["domain_user"].(string); ok && v != "" {
		winOpt.domainUser = v
	}
	if v, ok := customConfigs["product_key"].(string); ok && v != "" {
		winOpt.productKey = v
	}
	if v, ok := customConfigs["domain_user_password"].(string); ok && v != "" {
		winOpt.domainUserPassword = v
	}
	vm.windowsOptionalConfig = winOpt
	log.Printf("[DEBUG] windows config init: %v", winOpt)
	return nil
}

func populateVMStructDisk(d *schema.ResourceData, vm *virtualMachine) error {
	vL, ok := d.GetOk("disk")
	if !ok {
		return nil
	}

	diskSet, ok := vL.(*schema.Set)
	if !ok {
		return nil
	}

	disks := []hardDisk{}
	hasBootableDisk := false
	for _, value := range diskSet.List() {
		disk := value.(map[string]interface{})
		newDisk := hardDisk{}

		if v, ok := disk["template"].(string); ok && v != "" {
			if v, ok := disk["name"].(string); ok && v != "" {
				return fmt.Errorf("Cannot specify name of a template")
			}
			vm.template = v
			if hasBootableDisk {
				return fmt.Errorf("[ERROR] Only one bootable disk or template may be given")
			}
			hasBootableDisk = true
		}

		if v, ok := disk["type"].(provisioningType); ok && v != "" {
			newDisk.initType = v
		}

		if v, ok := disk["datastore"].(string); ok && v != "" {
			vm.datastore = v
		}

		if v, ok := disk["size"].(int); ok && v != 0 {
			if v, ok := disk["template"].(string); ok && v != "" {
				return fmt.Errorf("Cannot specify size of a template")
			}

			if v, ok := disk["name"].(string); ok && v != "" {
				newDisk.name = v
			} else {
				return fmt.Errorf("[ERROR] Disk name must be provided when creating a new disk")
			}

			newDisk.size = int64(v)
		}

		if v, ok := disk["iops"].(int); ok && v != 0 {
			newDisk.iops = int64(v)
		}

		if v, ok := disk["controller_type"].(controllerType); ok && v != "" {
			newDisk.controller = v
		}

		if vVmdk, ok := disk["vmdk"].(string); ok && vVmdk != "" {
			if v, ok := disk["template"].(string); ok && v != "" {
				return fmt.Errorf("Cannot specify a vmdk for a template")
			}
			if v, ok := disk["size"].(string); ok && v != "" {
				return fmt.Errorf("Cannot specify size of a vmdk")
			}
			if v, ok := disk["name"].(string); ok && v != "" {
				return fmt.Errorf("Cannot specify name of a vmdk")
			}
			if vBootable, ok := disk["bootable"].(bool); ok {
				hasBootableDisk = true
				newDisk.bootable = vBootable
				vm.hasBootableVmdk = vBootable
			}
			newDisk.vmdkPath = vVmdk
		}
		// Preserves order so bootable disk is first
		if newDisk.bootable == true || disk["template"] != "" {
			disks = append([]hardDisk{newDisk}, disks...)
		} else {
			disks = append(disks, newDisk)
		}
	}
	vm.hardDisks = disks
	log.Printf("[DEBUG] disk init: %v", disks)
	return nil
}

func populateVMStructCdrom(d *schema.ResourceData, vm *virtualMachine) error {
	vL, ok := d.GetOk("cdrom")
	if !ok {
		return nil
	}
	cdroms := make([]cdrom, len(vL.([]interface{})))
	for i, v := range vL.([]interface{}) {
		c := v.(map[string]interface{})
		if v, ok := c["datastore"].(string); ok && v != "" {
			cdroms[i].datastore = v
		} else {
			return fmt.Errorf("Datastore argument must be specified when attaching a cdrom image.")
		}
		if v, ok := c["path"].(string); ok && v != "" {
			cdroms[i].path = v
		} else {
			return fmt.Errorf("Path argument must be specified when attaching a cdrom image.")
		}
	}
	vm.cdroms = cdroms
	log.Printf("[DEBUG] cdrom init: %v", cdroms)
	return nil
}

// helper functions to spawn a VM
func createVMWithoutTemplate(configSpec *types.VirtualMachineConfigSpec, mds *mo.Datastore, resourcePool *object.ResourcePool, folder *object.Folder) error {
	var task *object.Task
	scsi, err := object.SCSIControllerTypes().CreateSCSIController("scsi")
	if err != nil {
		log.Printf("[ERROR] %s", err)
	}

	configSpec.DeviceChange = append(configSpec.DeviceChange, &types.VirtualDeviceConfigSpec{
		Operation: types.VirtualDeviceConfigSpecOperationAdd,
		Device:    scsi,
	})

	configSpec.Files = &types.VirtualMachineFileInfo{VmPathName: fmt.Sprintf("[%s]", mds.Name)}

	task, err = folder.CreateVM(context.TODO(), *configSpec, resourcePool, nil)
	if err != nil {
		log.Printf("[ERROR] %s", err)
	}

	err = task.Wait(context.TODO())
	if err != nil {
		log.Printf("[ERROR] %s", err)
	}
	return nil

}

func createVMWithTemplate(name string, linkedClone bool, diskInitType provisioningType, template *object.VirtualMachine, templateMo *mo.VirtualMachine, configSpec *types.VirtualMachineConfigSpec, resourcePool *object.ResourcePool, datastore *object.Datastore, folder *object.Folder) error {
	var task *object.Task
	relocateSpec, err := buildVMRelocateSpec(resourcePool, datastore, template, linkedClone, diskInitType)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] relocate spec: %v", relocateSpec)

	// make vm clone spec
	cloneSpec := types.VirtualMachineCloneSpec{
		Location: relocateSpec,
		Template: false,
		Config:   configSpec,
		PowerOn:  false,
	}
	if linkedClone {
		if templateMo.Snapshot == nil {
			return fmt.Errorf("`linkedClone=true`, but image VM has no snapshots")
		}
		cloneSpec.Snapshot = templateMo.Snapshot.CurrentSnapshot
	}
	log.Printf("[DEBUG] clone spec: %v", cloneSpec)

	task, err = template.Clone(context.TODO(), folder, name, cloneSpec)
	if err != nil {
		return err
	}

	err = task.Wait(context.TODO())
	if err != nil {
		log.Printf("[ERROR] %s", err)
	}
	return nil
}

// helper functions to parse vSphere-side objects into a ResourceData

func populateResourceDataDisks(d *schema.ResourceData, devices []types.BaseVirtualDevice) error {
	var err error
	var disks []map[string]interface{}
	templateDisk := make(map[string]interface{}, 1)
	for _, device := range devices {
		vd, ok := device.(*types.VirtualDisk)
		if !ok {
			continue
		}

		virtualDevice := vd.GetVirtualDevice()

		backingInfo := virtualDevice.Backing
		var diskFullPath string
		var diskUUID string
		if v, ok := backingInfo.(*types.VirtualDiskFlatVer2BackingInfo); ok {
			diskFullPath = v.FileName
			diskUUID = v.Uuid
		} else if v, ok := backingInfo.(*types.VirtualDiskSparseVer2BackingInfo); ok {
			diskFullPath = v.FileName
			diskUUID = v.Uuid
		}
		log.Printf("[DEBUG] resourceVSphereVirtualMachineRead - Analyzing disk: %v", diskFullPath)

		// Separate datastore and path
		diskFullPathSplit := strings.Split(diskFullPath, " ")
		if len(diskFullPathSplit) != 2 {
			return fmt.Errorf("[ERROR] Failed trying to parse disk path: %v", diskFullPath)
		}
		diskPath := diskFullPathSplit[1]
		// Isolate filename
		diskNameSplit := strings.Split(diskPath, "/")
		diskName := diskNameSplit[len(diskNameSplit)-1]
		// Remove possible extension
		diskName = strings.Split(diskName, ".")[0]

		prevDisks, ok := d.GetOk("disk")
		if !ok {
			continue
		}
		prevDiskSet, ok := prevDisks.(*schema.Set)
		if !ok {
			continue
		}
		for _, v := range prevDiskSet.List() {
			prevDisk := v.(map[string]interface{})
			// We're guaranteed only one template disk.  Passing value directly through since templates should be immutable
			if prevDisk["template"] != "" {
				if len(templateDisk) == 0 {
					templateDisk = prevDisk
					disks = append(disks, templateDisk)
					break
				}
			}

			// It is enforced that prevDisk["name"] should only be set in the case
			// of creating a new disk for the user.
			// size case:  name was set by user, compare parsed filename from mo.filename (without path or .vmdk extension) with name
			// vmdk case:  compare prevDisk["vmdk"] and mo.Filename
			if diskName == prevDisk["name"] || diskPath == prevDisk["vmdk"] {

				prevDisk["key"] = virtualDevice.Key
				prevDisk["uuid"] = diskUUID

				disks = append(disks, prevDisk)
				break
			}
		}
		log.Printf("[DEBUG] disks: %#v", disks)
	}
	if err = d.Set("disk", disks); err != nil {
		return err
	}
	return nil
}

func populateResourceDataNIC(devices []types.GuestNicInfo, networkInterfaces []map[string]interface{}) error {
	for _, v := range devices {
		if v.DeviceConfigId < 0 {
			continue
		}
		log.Printf("[DEBUG] v.Network - %#v", v.Network)
		networkInterface := make(map[string]interface{})
		networkInterface["label"] = v.Network
		networkInterface["mac_address"] = v.MacAddress
		for _, ip := range v.IpConfig.IpAddress {
			p := net.ParseIP(ip.IpAddress)
			if p.To4() != nil {
				log.Printf("[DEBUG] p.String - %#v", p.String())
				log.Printf("[DEBUG] ip.PrefixLength - %#v", ip.PrefixLength)
				networkInterface["ipv4_address"] = p.String()
				networkInterface["ipv4_prefix_length"] = ip.PrefixLength
			} else if p.To16() != nil {
				log.Printf("[DEBUG] p.String - %#v", p.String())
				log.Printf("[DEBUG] ip.PrefixLength - %#v", ip.PrefixLength)
				networkInterface["ipv6_address"] = p.String()
				networkInterface["ipv6_prefix_length"] = ip.PrefixLength
			}
			log.Printf("[DEBUG] networkInterface: %#v", networkInterface)
		}
		log.Printf("[DEBUG] networkInterface: %#v", networkInterface)
		networkInterfaces = append(networkInterfaces, networkInterface)
	}
	return nil
}

func populateResourceDataIPStack(devices []types.GuestStackInfo, networkInterfaces []map[string]interface{}) error {
	if devices == nil {
		return nil
	}
	for _, v := range devices {
		if v.IpRouteConfig == nil || v.IpRouteConfig.IpRoute == nil {
			continue
		}
		for _, route := range v.IpRouteConfig.IpRoute {
			if route.Gateway.Device == "" {
				continue
			}
			gatewaySetting := ""
			if route.Network == "::" {
				gatewaySetting = "ipv6_gateway"
			} else if route.Network == "0.0.0.0" {
				gatewaySetting = "ipv4_gateway"
			}
			if gatewaySetting == "" {
				continue
			}
			deviceID, err := strconv.Atoi(route.Gateway.Device)
			if err != nil {
				log.Printf("[WARN] error at processing %s of device id %#v: %#v", gatewaySetting, route.Gateway.Device, err)
				continue
			}
			log.Printf("[DEBUG] %s of device id %d: %s", gatewaySetting, deviceID, route.Gateway.IpAddress)
			networkInterfaces[deviceID][gatewaySetting] = route.Gateway.IpAddress
		}
	}
	return nil
}

func populateResourceDataNetwork(d *schema.ResourceData, mvm *mo.VirtualMachine) error {
	var err error
	var networkInterfaces []map[string]interface{}
	populateResourceDataNIC(mvm.Guest.Net, networkInterfaces)
	populateResourceDataIPStack(mvm.Guest.IpStack, networkInterfaces)
	log.Printf("[DEBUG] networkInterfaces: %#v", networkInterfaces)
	if err = d.Set("network_interface", networkInterfaces); err != nil {
		return fmt.Errorf("Invalid network interfaces to set: %#v", networkInterfaces)
	}
	// get primary IP address as set
	if len(networkInterfaces) == 0 {
		return nil
	}
	if _, ok := networkInterfaces[0]["ipv4_address"]; !ok {
		return nil
	}
	log.Printf("[DEBUG] ip address: %v", networkInterfaces[0]["ipv4_address"].(string))
	d.SetConnInfo(map[string]string{
		"type": "ssh",
		"host": networkInterfaces[0]["ipv4_address"].(string),
	})
	return nil
}

func populateResourceDataDatastore(d *schema.ResourceData, mvm *mo.VirtualMachine, collector *property.Collector) error {
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
	d.Set("datastore", rootDatastore)
	return nil
}
