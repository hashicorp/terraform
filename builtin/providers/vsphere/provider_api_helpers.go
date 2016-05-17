package vsphere

import (
	"fmt"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"golang.org/x/net/context"
)

type api_helper struct {
	Client       *vim25.Client
	StoragePod   *object.StoragePod
	ResourcePool *object.ResourcePool
	HostSystem   *object.HostSystem
	Folder       *object.Folder
	//Datacenter   *object.Datacenter
	//Datastore    *object.Datastore
}

/*
Based off of govc vm/create.go.  Now how do I use this ...
*/
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
