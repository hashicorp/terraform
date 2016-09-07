package vsphere

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/soap"
	"golang.org/x/net/context"
)

type file struct {
	sourceDatacenter  string
	datacenter        string
	sourceDatastore   string
	datastore         string
	sourceFile        string
	destinationFile   string
	createDirectories bool
	copyFile          bool
}

func resourceVSphereFile() *schema.Resource {
	return &schema.Resource{
		Create: resourceVSphereFileCreate,
		Read:   resourceVSphereFileRead,
		Update: resourceVSphereFileUpdate,
		Delete: resourceVSphereFileDelete,

		Schema: map[string]*schema.Schema{
			"datacenter": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"source_datacenter": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"datastore": {
				Type:     schema.TypeString,
				Required: true,
			},

			"source_datastore": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"source_file": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"destination_file": {
				Type:     schema.TypeString,
				Required: true,
			},

			"create_directories": {
				Type:     schema.TypeBool,
				Optional: true,
			},
		},
	}
}

func resourceVSphereFileCreate(d *schema.ResourceData, meta interface{}) error {

	log.Printf("[DEBUG] creating file: %#v", d)
	client := meta.(*govmomi.Client)

	f := file{}

	if v, ok := d.GetOk("source_datacenter"); ok {
		f.sourceDatacenter = v.(string)
		f.copyFile = true
	}

	if v, ok := d.GetOk("datacenter"); ok {
		f.datacenter = v.(string)
	}

	if v, ok := d.GetOk("source_datastore"); ok {
		f.sourceDatastore = v.(string)
		f.copyFile = true
	}

	if v, ok := d.GetOk("datastore"); ok {
		f.datastore = v.(string)
	} else {
		return fmt.Errorf("datastore argument is required")
	}

	if v, ok := d.GetOk("source_file"); ok {
		f.sourceFile = v.(string)
	} else {
		return fmt.Errorf("source_file argument is required")
	}

	if v, ok := d.GetOk("destination_file"); ok {
		f.destinationFile = v.(string)
	} else {
		return fmt.Errorf("destination_file argument is required")
	}

	if v, ok := d.GetOk("create_directories"); ok {
		f.createDirectories = v.(bool)
	}

	err := createFile(client, &f)
	if err != nil {
		return err
	}

	d.SetId(fmt.Sprintf("[%v] %v/%v", f.datastore, f.datacenter, f.destinationFile))
	log.Printf("[INFO] Created file: %s", f.destinationFile)

	return resourceVSphereFileRead(d, meta)
}

func createFile(client *govmomi.Client, f *file) error {

	finder := find.NewFinder(client.Client, true)

	dc, err := finder.Datacenter(context.TODO(), f.datacenter)
	if err != nil {
		return fmt.Errorf("error %s", err)
	}
	finder = finder.SetDatacenter(dc)

	ds, err := getDatastore(finder, f.datastore)
	if err != nil {
		return fmt.Errorf("error %s", err)
	}

	if f.copyFile {
		// Copying file from withing vSphere
		source_dc, err := finder.Datacenter(context.TODO(), f.sourceDatacenter)
		if err != nil {
			return fmt.Errorf("error %s", err)
		}
		finder = finder.SetDatacenter(dc)

		source_ds, err := getDatastore(finder, f.sourceDatastore)
		if err != nil {
			return fmt.Errorf("error %s", err)
		}

		fm := object.NewFileManager(client.Client)
		if f.createDirectories {
			directoryPathIndex := strings.LastIndex(f.destinationFile, "/")
			path := f.destinationFile[0:directoryPathIndex]
			err = fm.MakeDirectory(context.TODO(), ds.Path(path), dc, true)
			if err != nil {
				return fmt.Errorf("error %s", err)
			}
		}
		task, err := fm.CopyDatastoreFile(context.TODO(), source_ds.Path(f.sourceFile), source_dc, ds.Path(f.destinationFile), dc, true)

		if err != nil {
			return fmt.Errorf("error %s", err)
		}

		_, err = task.WaitForResult(context.TODO(), nil)
		if err != nil {
			return fmt.Errorf("error %s", err)
		}

	} else {
		// Uploading file to vSphere
		dsurl, err := ds.URL(context.TODO(), dc, f.destinationFile)
		if err != nil {
			return fmt.Errorf("error %s", err)
		}

		p := soap.DefaultUpload
		err = client.Client.UploadFile(f.sourceFile, dsurl, &p)
		if err != nil {
			return fmt.Errorf("error %s", err)
		}
	}

	return nil
}

func resourceVSphereFileRead(d *schema.ResourceData, meta interface{}) error {

	log.Printf("[DEBUG] reading file: %#v", d)
	f := file{}

	if v, ok := d.GetOk("source_datacenter"); ok {
		f.sourceDatacenter = v.(string)
	}

	if v, ok := d.GetOk("datacenter"); ok {
		f.datacenter = v.(string)
	}

	if v, ok := d.GetOk("source_datastore"); ok {
		f.sourceDatastore = v.(string)
	}

	if v, ok := d.GetOk("datastore"); ok {
		f.datastore = v.(string)
	} else {
		return fmt.Errorf("datastore argument is required")
	}

	if v, ok := d.GetOk("source_file"); ok {
		f.sourceFile = v.(string)
	} else {
		return fmt.Errorf("source_file argument is required")
	}

	if v, ok := d.GetOk("destination_file"); ok {
		f.destinationFile = v.(string)
	} else {
		return fmt.Errorf("destination_file argument is required")
	}

	client := meta.(*govmomi.Client)
	finder := find.NewFinder(client.Client, true)

	dc, err := finder.Datacenter(context.TODO(), f.datacenter)
	if err != nil {
		return fmt.Errorf("error %s", err)
	}
	finder = finder.SetDatacenter(dc)

	ds, err := getDatastore(finder, f.datastore)
	if err != nil {
		return fmt.Errorf("error %s", err)
	}

	_, err = ds.Stat(context.TODO(), f.destinationFile)
	if err != nil {
		log.Printf("[DEBUG] resourceVSphereFileRead - stat failed on: %v", f.destinationFile)
		d.SetId("")

		_, ok := err.(object.DatastoreNoSuchFileError)
		if !ok {
			return err
		}
	}

	return nil
}

func resourceVSphereFileUpdate(d *schema.ResourceData, meta interface{}) error {

	log.Printf("[DEBUG] updating file: %#v", d)

	if d.HasChange("destination_file") || d.HasChange("datacenter") || d.HasChange("datastore") {
		// File needs to be moved, get old and new destination changes
		var oldDataceneter, newDatacenter, oldDatastore, newDatastore, oldDestinationFile, newDestinationFile string
		if d.HasChange("datacenter") {
			tmpOldDataceneter, tmpNewDatacenter := d.GetChange("datacenter")
			oldDataceneter = tmpOldDataceneter.(string)
			newDatacenter = tmpNewDatacenter.(string)
		} else {
			if v, ok := d.GetOk("datacenter"); ok {
				oldDataceneter = v.(string)
				newDatacenter = oldDataceneter
			}
		}
		if d.HasChange("datastore") {
			tmpOldDatastore, tmpNewDatastore := d.GetChange("datastore")
			oldDatastore = tmpOldDatastore.(string)
			newDatastore = tmpNewDatastore.(string)
		} else {
			oldDatastore = d.Get("datastore").(string)
			newDatastore = oldDatastore
		}
		if d.HasChange("destination_file") {
			tmpOldDestinationFile, tmpNewDestinationFile := d.GetChange("destination_file")
			oldDestinationFile = tmpOldDestinationFile.(string)
			newDestinationFile = tmpNewDestinationFile.(string)
		} else {
			oldDestinationFile = d.Get("destination_file").(string)
			newDestinationFile = oldDestinationFile
		}

		// Get old and new dataceter and datastore
		client := meta.(*govmomi.Client)
		dcOld, err := getDatacenter(client, oldDataceneter)
		if err != nil {
			return err
		}
		dcNew, err := getDatacenter(client, newDatacenter)
		if err != nil {
			return err
		}
		finder := find.NewFinder(client.Client, true)
		finder = finder.SetDatacenter(dcOld)
		dsOld, err := getDatastore(finder, oldDatastore)
		if err != nil {
			return fmt.Errorf("error %s", err)
		}
		finder = finder.SetDatacenter(dcNew)
		dsNew, err := getDatastore(finder, newDatastore)
		if err != nil {
			return fmt.Errorf("error %s", err)
		}

		// Move file between old/new dataceter, datastore and path (destination_file)
		fm := object.NewFileManager(client.Client)
		task, err := fm.MoveDatastoreFile(context.TODO(), dsOld.Path(oldDestinationFile), dcOld, dsNew.Path(newDestinationFile), dcNew, true)
		if err != nil {
			return err
		}
		_, err = task.WaitForResult(context.TODO(), nil)
		if err != nil {
			return err
		}
	}

	return nil
}

func resourceVSphereFileDelete(d *schema.ResourceData, meta interface{}) error {

	log.Printf("[DEBUG] deleting file: %#v", d)
	f := file{}

	if v, ok := d.GetOk("datacenter"); ok {
		f.datacenter = v.(string)
	}

	if v, ok := d.GetOk("datastore"); ok {
		f.datastore = v.(string)
	} else {
		return fmt.Errorf("datastore argument is required")
	}

	if v, ok := d.GetOk("source_file"); ok {
		f.sourceFile = v.(string)
	} else {
		return fmt.Errorf("source_file argument is required")
	}

	if v, ok := d.GetOk("destination_file"); ok {
		f.destinationFile = v.(string)
	} else {
		return fmt.Errorf("destination_file argument is required")
	}

	client := meta.(*govmomi.Client)

	err := deleteFile(client, &f)
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}

func deleteFile(client *govmomi.Client, f *file) error {

	dc, err := getDatacenter(client, f.datacenter)
	if err != nil {
		return err
	}

	finder := find.NewFinder(client.Client, true)
	finder = finder.SetDatacenter(dc)

	ds, err := getDatastore(finder, f.datastore)
	if err != nil {
		return fmt.Errorf("error %s", err)
	}

	fm := object.NewFileManager(client.Client)
	task, err := fm.DeleteDatastoreFile(context.TODO(), ds.Path(f.destinationFile), dc)
	if err != nil {
		return err
	}

	_, err = task.WaitForResult(context.TODO(), nil)
	if err != nil {
		return err
	}
	return nil
}

// getDatastore gets datastore object
func getDatastore(f *find.Finder, ds string) (*object.Datastore, error) {

	if ds != "" {
		dso, err := f.Datastore(context.TODO(), ds)
		return dso, err
	} else {
		dso, err := f.DefaultDatastore(context.TODO())
		return dso, err
	}
}
