package vsphere

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/soap"
	"golang.org/x/net/context"
)

type file struct {
	datacenter      string
	datastore       string
	sourceFile      string
	destinationFile string
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
				ForceNew: true,
			},

			"datastore": {
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
		},
	}
}

func resourceVSphereFileCreate(d *schema.ResourceData, meta interface{}) error {

	log.Printf("[DEBUG] creating file: %#v", d)
	client := meta.(*govmomi.Client)

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

	dsurl, err := ds.URL(context.TODO(), dc, f.destinationFile)
	if err != nil {
		return err
	}

	p := soap.DefaultUpload
	err = client.Client.UploadFile(f.sourceFile, dsurl, &p)
	if err != nil {
		return fmt.Errorf("error %s", err)
	}
	return nil
}

func resourceVSphereFileRead(d *schema.ResourceData, meta interface{}) error {

	log.Printf("[DEBUG] reading file: %#v", d)
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
		d.SetId("")
		return err
	}

	return nil
}

func resourceVSphereFileUpdate(d *schema.ResourceData, meta interface{}) error {

	log.Printf("[DEBUG] updating file: %#v", d)
	if d.HasChange("destination_file") {
		oldDestinationFile, newDestinationFile := d.GetChange("destination_file")
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
		task, err := fm.MoveDatastoreFile(context.TODO(), ds.Path(oldDestinationFile.(string)), dc, ds.Path(newDestinationFile.(string)), dc, true)
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
