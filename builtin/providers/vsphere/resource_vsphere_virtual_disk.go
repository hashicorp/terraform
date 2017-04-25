package vsphere

import (
	"fmt"
	"log"

	"errors"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
	"golang.org/x/net/context"
	"path"
)

type virtualDisk struct {
	size        int
	vmdkPath    string
	initType    string
	adapterType string
	datacenter  string
	datastore   string
}

// Define VirtualDisk args
func resourceVSphereVirtualDisk() *schema.Resource {
	return &schema.Resource{
		Create: resourceVSphereVirtualDiskCreate,
		Read:   resourceVSphereVirtualDiskRead,
		Delete: resourceVSphereVirtualDiskDelete,

		Schema: map[string]*schema.Schema{
			// Size in GB
			"size": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true, //TODO Can this be optional (resize)?
			},

			"vmdk_path": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true, //TODO Can this be optional (move)?
			},

			"type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "eagerZeroedThick",
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if value != "thin" && value != "eagerZeroedThick" && value != "lazy" {
						errors = append(errors, fmt.Errorf(
							"only 'thin', 'eagerZeroedThick', and 'lazy' are supported values for 'type'"))
					}
					return
				},
			},

			"adapter_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "ide",
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if value != "ide" && value != "busLogic" && value != "lsiLogic" {
						errors = append(errors, fmt.Errorf(
							"only 'ide', 'busLogic', and 'lsiLogic' are supported values for 'adapter_type'"))
					}
					return
				},
			},

			"datacenter": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"datastore": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceVSphereVirtualDiskCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Creating Virtual Disk")
	client := meta.(*govmomi.Client)

	vDisk := virtualDisk{
		size: d.Get("size").(int),
	}

	if v, ok := d.GetOk("vmdk_path"); ok {
		vDisk.vmdkPath = v.(string)
	}

	if v, ok := d.GetOk("type"); ok {
		vDisk.initType = v.(string)
	}

	if v, ok := d.GetOk("adapter_type"); ok {
		vDisk.adapterType = v.(string)
	}

	if v, ok := d.GetOk("datacenter"); ok {
		vDisk.datacenter = v.(string)
	}

	if v, ok := d.GetOk("datastore"); ok {
		vDisk.datastore = v.(string)
	}

	finder := find.NewFinder(client.Client, true)

	dc, err := getDatacenter(client, d.Get("datacenter").(string))
	if err != nil {
		return fmt.Errorf("Error finding Datacenter: %s: %s", vDisk.datacenter, err)
	}
	finder = finder.SetDatacenter(dc)

	ds, err := getDatastore(finder, vDisk.datastore)
	if err != nil {
		return fmt.Errorf("Error finding Datastore: %s: %s", vDisk.datastore, err)
	}

	err = createHardDisk(client, vDisk.size, ds.Path(vDisk.vmdkPath), vDisk.initType, vDisk.adapterType, vDisk.datacenter)
	if err != nil {
		return err
	}

	d.SetId(ds.Path(vDisk.vmdkPath))
	log.Printf("[DEBUG] Virtual Disk id: %v", ds.Path(vDisk.vmdkPath))

	return resourceVSphereVirtualDiskRead(d, meta)
}

func resourceVSphereVirtualDiskRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Reading virtual disk.")
	client := meta.(*govmomi.Client)

	vDisk := virtualDisk{
		size: d.Get("size").(int),
	}

	if v, ok := d.GetOk("vmdk_path"); ok {
		vDisk.vmdkPath = v.(string)
	}

	if v, ok := d.GetOk("type"); ok {
		vDisk.initType = v.(string)
	}

	if v, ok := d.GetOk("adapter_type"); ok {
		vDisk.adapterType = v.(string)
	}

	if v, ok := d.GetOk("datacenter"); ok {
		vDisk.datacenter = v.(string)
	}

	if v, ok := d.GetOk("datastore"); ok {
		vDisk.datastore = v.(string)
	}

	dc, err := getDatacenter(client, d.Get("datacenter").(string))
	if err != nil {
		return err
	}

	finder := find.NewFinder(client.Client, true)
	finder = finder.SetDatacenter(dc)

	ds, err := finder.Datastore(context.TODO(), d.Get("datastore").(string))
	if err != nil {
		return err
	}

	ctx := context.TODO()
	b, err := ds.Browser(ctx)
	if err != nil {
		return err
	}

	// `Datastore.Stat` does not allow to query `VmDiskFileQuery`. Instead, we
	// search the datastore manually.
	spec := types.HostDatastoreBrowserSearchSpec{
		Query: []types.BaseFileQuery{&types.VmDiskFileQuery{Details: &types.VmDiskFileQueryFlags{
			CapacityKb: true,
			DiskType:   true,
		}}},
		Details: &types.FileQueryFlags{
			FileSize:     true,
			FileType:     true,
			Modification: true,
			FileOwner:    types.NewBool(true),
		},
		MatchPattern: []string{path.Base(vDisk.vmdkPath)},
	}

	dsPath := ds.Path(path.Dir(vDisk.vmdkPath))
	task, err := b.SearchDatastore(context.TODO(), dsPath, &spec)

	if err != nil {
		log.Printf("[DEBUG] resourceVSphereVirtualDiskRead - could not search datastore for: %v", vDisk.vmdkPath)
		return err
	}

	info, err := task.WaitForResult(context.TODO(), nil)
	if err != nil {
		if info == nil || info.Error != nil {
			_, ok := info.Error.Fault.(*types.FileNotFound)
			if ok {
				log.Printf("[DEBUG] resourceVSphereVirtualDiskRead - could not find: %v", vDisk.vmdkPath)
				d.SetId("")
				return nil
			}
		}

		log.Printf("[DEBUG] resourceVSphereVirtualDiskRead - could not search datastore for: %v", vDisk.vmdkPath)
		return err
	}

	res := info.Result.(types.HostDatastoreBrowserSearchResults)
	log.Printf("[DEBUG] num results: %d", len(res.File))
	if len(res.File) == 0 {
		d.SetId("")
		log.Printf("[DEBUG] resourceVSphereVirtualDiskRead - could not find: %v", vDisk.vmdkPath)
		return nil
	}

	if len(res.File) != 1 {
		return errors.New("Datastore search did not return exactly one result")
	}

	fileInfo := res.File[0]
	log.Printf("[DEBUG] resourceVSphereVirtualDiskRead - fileinfo: %#v", fileInfo)
	size := fileInfo.(*types.VmDiskFileInfo).CapacityKb / 1024 / 1024

	d.SetId(vDisk.vmdkPath)

	d.Set("size", size)
	d.Set("vmdk_path", vDisk.vmdkPath)
	d.Set("datacenter", d.Get("datacenter"))
	d.Set("datastore", d.Get("datastore"))
	// Todo collect and write type info

	return nil

}

func resourceVSphereVirtualDiskDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*govmomi.Client)

	vDisk := virtualDisk{}

	if v, ok := d.GetOk("vmdk_path"); ok {
		vDisk.vmdkPath = v.(string)
	}
	if v, ok := d.GetOk("datastore"); ok {
		vDisk.datastore = v.(string)
	}

	dc, err := getDatacenter(client, d.Get("datacenter").(string))
	if err != nil {
		return err
	}

	finder := find.NewFinder(client.Client, true)
	finder = finder.SetDatacenter(dc)

	ds, err := getDatastore(finder, vDisk.datastore)
	if err != nil {
		return err
	}

	diskPath := ds.Path(vDisk.vmdkPath)

	virtualDiskManager := object.NewVirtualDiskManager(client.Client)

	task, err := virtualDiskManager.DeleteVirtualDisk(context.TODO(), diskPath, dc)
	if err != nil {
		return err
	}

	_, err = task.WaitForResult(context.TODO(), nil)
	if err != nil {
		log.Printf("[INFO] Failed to delete disk:  %v", err)
		return err
	}

	log.Printf("[INFO] Deleted disk: %v", diskPath)
	d.SetId("")
	return nil
}

// createHardDisk creates a new Hard Disk.
func createHardDisk(client *govmomi.Client, size int, diskPath string, diskType string, adapterType string, dc string) error {
	var vDiskType string
	switch diskType {
	case "thin":
		vDiskType = "thin"
	case "eagerZeroedThick":
		vDiskType = "eagerZeroedThick"
	case "lazy":
		vDiskType = "preallocated"
	}

	virtualDiskManager := object.NewVirtualDiskManager(client.Client)
	spec := &types.FileBackedVirtualDiskSpec{
		VirtualDiskSpec: types.VirtualDiskSpec{
			AdapterType: adapterType,
			DiskType:    vDiskType,
		},
		CapacityKb: int64(1024 * 1024 * size),
	}
	datacenter, err := getDatacenter(client, dc)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] Disk spec: %v", spec)

	task, err := virtualDiskManager.CreateVirtualDisk(context.TODO(), diskPath, datacenter, spec)
	if err != nil {
		return err
	}

	_, err = task.WaitForResult(context.TODO(), nil)
	if err != nil {
		log.Printf("[INFO] Failed to create disk:  %v", err)
		return err
	}
	log.Printf("[INFO] Created disk.")

	return nil
}
