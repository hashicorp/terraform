package vsphere

import (
	"fmt"
	"log"

	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"golang.org/x/net/context"
)

// Collector models the PropertyCollector managed object.
//
// For more information, see:
// http://pubs.vmware.com/vsphere-55/index.jsp#com.vmware.wssdk.apiref.doc/vmodl.query.PropertyCollector.html
//
type StoragePodDataStore struct {
	name           string
	template       string
	storagePodName string
	clone          bool

	ConfigSpecsNetwork []types.BaseVirtualDeviceConfigSpec
	ResourcePool       *object.ResourcePool
	// TODO Do we need this?
	// HostSystem         *object.HostSystem
	Folder         *object.Folder
	VirtualMachine *object.VirtualMachine
	DataCenter     *object.Datacenter
	StoragePod     *object.StoragePod
}

// Based of of govc clone.go
// Get the recommended StoragePod datastore
func (spds *StoragePodDataStore) findRecommendedStoragePodDataStore(client *vim25.Client) (datastore *object.Datastore, err error) {

	folderref := spds.Folder.Reference()
	poolref := spds.ResourcePool.Reference()

	relocateSpec := types.VirtualMachineRelocateSpec{
		DeviceChange: spds.ConfigSpecsNetwork,
		Folder:       &folderref,
		Pool:         &poolref,
	}

	// TODO do we need this
	// govc is using this
	// if pds.HostSystem != nil {
	// 	hostref := pds.HostSystem.Reference()
	// 	relocateSpec.Host = &hostref
	// }

	hasTemplate := false
	if spds.template != "" {
		hasTemplate = true
	}
	cloneSpec := &types.VirtualMachineCloneSpec{
		Location: relocateSpec,
		PowerOn:  false,
		Template: hasTemplate,
	}

	sp, err := spds.findStoragePod(client)
	if err != nil {
		log.Printf("[ERROR] Couldn't find storage pod '%s'.  %s", spds.storagePodName, err)
		return nil, err
	}
	storagePod := sp.Reference()

	// Build pod selection spec from config spec
	podSelectionSpec := types.StorageDrsPodSelectionSpec{
		StoragePod: &storagePod,
	}

	// Get the virtual machine reference
	vmref := spds.VirtualMachine.Reference()

	// Build the placement spec

	var spec string
	if spds.clone {
		spec = string(types.StoragePlacementSpecPlacementTypeClone)
	} else {
		spec = string(types.StoragePlacementSpecPlacementTypeReconfigure)
	}

	// TODO does this support update??
	storagePlacementSpec := types.StoragePlacementSpec{
		Folder:           &folderref,
		Vm:               &vmref,
		CloneName:        spds.name,
		CloneSpec:        cloneSpec,
		PodSelectionSpec: podSelectionSpec,
		Type:             spec,
	}
	log.Printf("[DEBUG] storage placement spec, %v", storagePlacementSpec)

	datastore, err = spds.findRecommendedDatastore(client, storagePlacementSpec)
	if err != nil {
		log.Printf("[ERROR] Couldn't find datastore %s", err)
		return nil, err
	}
	log.Printf("[DEBUG] Found datastore: %v", datastore)
	return datastore, nil
}

func (spds *StoragePodDataStore) findRecommendedDatastore(client *vim25.Client, sps types.StoragePlacementSpec) (*object.Datastore, error) {

	var datastore *object.Datastore
	log.Printf("[DEBUG] findDatastore: StoragePlacementSpec: %#v\n", sps)

	srm := object.NewStorageResourceManager(client)
	rds, err := srm.RecommendDatastores(context.TODO(), sps)
	if err != nil {
		return nil, err
	}
	log.Printf("[DEBUG] findDatastore: recommendDatastores: %#v\n", rds)

	// Get the recommendations
	recommendations := rds.Recommendations
	if len(recommendations) == 0 {
		log.Printf("[ERROR] no recommendations for datastore")
		return nil, fmt.Errorf("no recommendations for datastore")
	}

	spa := rds.Recommendations[0].Action[0].(*types.StoragePlacementAction)
	var mds mo.Datastore

	p := property.DefaultCollector(client)

	err = p.RetrieveOne(context.TODO(), spa.Destination, []string{"name"}, &mds)
	if err != nil {
		return nil, err
	}

	datastore = object.NewDatastore(client, spa.Destination)
	datastore.InventoryPath = mds.Name
	log.Printf("[DEBUG] findDatastore: datastore: %#v", datastore)

	return datastore, nil
}

// find the Default or named Storage Pod
func (spds *StoragePodDataStore) findStoragePod(client *vim25.Client) (sp *object.StoragePod, err error) {

	finder := find.NewFinder(client, true)
	if spds.DataCenter != nil {
		finder.SetDatacenter(spds.DataCenter)
	}

	if spds.storagePodName != "" {
		log.Printf("[DEBUG] looking for DataStore Cluster")
		sp, err = finder.DatastoreCluster(context.TODO(), spds.storagePodName)
		if err != nil {
			log.Printf("[ERROR] Couldn't find datastore cluster %v.  %s", spds.storagePodName, err)
			return nil, err
		}
	} else {
		// TODO this does not seem to be working ... wth
		sp, err = finder.DefaultDatastoreCluster(context.TODO())
		if err != nil {
			log.Printf("[ERROR] Couldn't find default datastore cluster %s", err)
			return nil, err
		}
	}

	log.Printf("[DEBUG] Found datastore cluster: %v", sp)
	return sp, nil
}

// Not actually used.  Holding on to the code
func (spds *StoragePodDataStore) buildStoragePlacementSpecClone(c *vim25.Client) types.StoragePlacementSpec {
	vmr := spds.VirtualMachine.Reference()
	vmfr := spds.Folder.Reference()
	rpr := spds.ResourcePool.Reference()
	spr := spds.StoragePod.Reference()

	var o mo.VirtualMachine
	err := spds.VirtualMachine.Properties(context.TODO(), vmr, []string{"datastore"}, &o)
	if err != nil {
		return types.StoragePlacementSpec{}
	}
	ds := object.NewDatastore(c, o.Datastore[0])
	log.Printf("[DEBUG] findDatastore: datastore: %#v\n", ds)

	devices, err := spds.VirtualMachine.Device(context.TODO())
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

// buildStoragePlacementSpecCreate builds StoragePlacementSpec for create action.
func (spds *StoragePodDataStore) findDataStoreSpecCreate(c *vim25.Client, configSpec types.VirtualMachineConfigSpec) (datastore *object.Datastore, err error) {

	spds.StoragePod, err = spds.findStoragePod(c)

	if err != nil {
		log.Printf("[ERROR] Couldn't find datastore cluster %v.  %s", spds.storagePodName, err)
		return nil, err
	}

	vmfr := spds.Folder.Reference()
	rpr := spds.ResourcePool.Reference()
	spr := spds.StoragePod.Reference()

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
	datastore, err = spds.findRecommendedDatastore(c, sps)
	if err != nil {
		log.Printf("[ERROR] Couldn't find datastore %s", err)
		return nil, err
	}
	return datastore, nil
}
