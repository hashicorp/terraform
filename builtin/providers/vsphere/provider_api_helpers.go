package vsphere

import (
	"fmt"
	"log"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"golang.org/x/net/context"
)

type api_helper struct {
	controller    string
	link          bool
	disk          string
	iso           string
	isoDatastore  *object.Datastore
	diskDatastore *object.Datastore

	// Only set if the disk argument is a byte size, which means the disk
	// doesn't exist yet and should be created
	diskByteSize int64

	Client       *vim25.Client
	StoragePod   *object.StoragePod
	ResourcePool *object.ResourcePool
	Folder       *object.Folder
	//Datacenter   *object.Datacenter
	//Datastore    *object.Datastore
}

func (api_helper *api_helper) Init(client *vim25.Client, sp *object.StoragePod, rp *object.ResourcePool,
	fo *object.Folder) {
	api_helper.Client = client
	api_helper.StoragePod = sp
	api_helper.ResourcePool = rp
	api_helper.Folder = fo
}

func (cmd *api_helper) addStorage(devices object.VirtualDeviceList) (object.VirtualDeviceList, error) {
	if cmd.controller != "ide" {
		scsi, err := devices.CreateSCSIController(cmd.controller)
		if err != nil {
			return nil, err
		}

		devices = append(devices, scsi)
	}

	// If controller is specified to be IDE or if an ISO is specified, add IDE controller.
	if cmd.controller == "ide" || cmd.iso != "" {
		ide, err := devices.CreateIDEController()
		if err != nil {
			return nil, err
		}

		devices = append(devices, ide)
	}

	if cmd.diskByteSize != 0 {
		controller, err := devices.FindDiskController(cmd.controller)
		if err != nil {
			return nil, err
		}

		disk := &types.VirtualDisk{
			VirtualDevice: types.VirtualDevice{
				Key: devices.NewKey(),
				Backing: &types.VirtualDiskFlatVer2BackingInfo{
					DiskMode:        string(types.VirtualDiskModePersistent),
					ThinProvisioned: types.NewBool(true),
				},
			},
			CapacityInKB: cmd.diskByteSize / 1024,
		}

		devices.AssignController(disk, controller)
		devices = append(devices, disk)
	} else if cmd.disk != "" {
		controller, err := devices.FindDiskController(cmd.controller)
		if err != nil {
			return nil, err
		}

		ds := cmd.diskDatastore.Reference()
		path := cmd.diskDatastore.Path(cmd.disk)
		disk := devices.CreateDisk(controller, ds, path)

		if cmd.link {
			disk = devices.ChildDisk(disk)
		}

		devices = append(devices, disk)
	}

	if cmd.iso != "" {
		ide, err := devices.FindIDEController("")
		if err != nil {
			return nil, err
		}

		cdrom, err := devices.CreateCdrom(ide)
		if err != nil {
			return nil, err
		}

		cdrom = devices.InsertIso(cdrom, cmd.isoDatastore.Path(cmd.iso))
		devices = append(devices, cdrom)
	}

	return devices, nil
}

func (api_helper *api_helper) recommendDatastore(ctx context.Context, spec *types.VirtualMachineConfigSpec) (*object.Datastore, error) {
	sp := api_helper.StoragePod.Reference()

	// Build pod selection spec from config spec
	podSelectionSpec := types.StorageDrsPodSelectionSpec{
		StoragePod: &sp,
	}

	// Keep list of disks that need to be placed
	var disks []*types.VirtualDisk

	// Collect disks eligible for placement
	for _, deviceConfigSpec := range spec.DeviceChange {
		s := deviceConfigSpec.GetVirtualDeviceConfigSpec()
		if s.Operation != types.VirtualDeviceConfigSpecOperationAdd {
			continue
		}

		if s.FileOperation != types.VirtualDeviceConfigSpecFileOperationCreate {
			continue
		}

		d, ok := s.Device.(*types.VirtualDisk)
		if !ok {
			continue
		}

		podConfigForPlacement := types.VmPodConfigForPlacement{
			StoragePod: sp,
			Disk: []types.PodDiskLocator{
				{
					DiskId:          d.Key,
					DiskBackingInfo: d.Backing,
				},
			},
		}

		podSelectionSpec.InitialVmConfig = append(podSelectionSpec.InitialVmConfig, podConfigForPlacement)
		disks = append(disks, d)
	}

	sps := types.StoragePlacementSpec{
		Type:             string(types.StoragePlacementSpecPlacementTypeCreate),
		ResourcePool:     types.NewReference(api_helper.ResourcePool.Reference()),
		PodSelectionSpec: podSelectionSpec,
		ConfigSpec:       spec,
	}

	srm := object.NewStorageResourceManager(api_helper.Client)
	result, err := srm.RecommendDatastores(ctx, sps)
	if err != nil {
		log.Printf("[ERROR] unable to find recommenedDataStores: %#v", sps)
		return nil, err
	}

	// Use result to pin disks to recommended datastores
	recs := result.Recommendations
	if len(recs) == 0 {
		return nil, fmt.Errorf("no recommendations")
	}

	ds := recs[0].Action[0].(*types.StoragePlacementAction).Destination

	var mds mo.Datastore
	err = property.DefaultCollector(api_helper.Client).RetrieveOne(ctx, ds, []string{"name"}, &mds)
	if err != nil {
		log.Printf("[ERROR] unable to find DefaultCollector: %#v", ds)
		return nil, err
	}

	datastore := object.NewDatastore(api_helper.Client, ds)
	datastore.InventoryPath = mds.Name

	// Apply recommendation to eligible disks
	for _, disk := range disks {
		backing := disk.Backing.(*types.VirtualDiskFlatVer2BackingInfo)
		backing.Datastore = &ds
	}

	return datastore, nil
}
