package vsphere

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"golang.org/x/net/context"
	"log"
)

func resourceVSphereRevertSnapshot() *schema.Resource {
	return &schema.Resource{
		Create: resourceVSphereSnapshotRevert,
		Delete: resourceVSphereSnapshotDummyDelete,
		Read:   resourceVSphereSnapshotDummyRead,

		Schema: map[string]*schema.Schema{
			"datacenter": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"vm_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: schema.EnvDefaultFunc("VSPHERE_VM_NAME", nil),
			},
			"folder": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: schema.EnvDefaultFunc("VSPHERE_VM_FOLDER", nil),
			},
			"snapshot_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"suppress_power_on": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceVSphereSnapshotRevert(d *schema.ResourceData, meta interface{}) error {
	vm, err := findVM(d, meta)
	if err != nil {
		return fmt.Errorf("Error while getting the VirtualMachine :%s", err)
	}
	task, err := vm.RevertToSnapshot(context.TODO(), d.Get("snapshot_name").(string), d.Get("suppress_power_on").(bool))
	if err != nil {
		log.Printf("[ERROR] Error While Creating the Task for Revert Snapshot: %v", err)
		return fmt.Errorf("Error While Creating the Task for Revert Snapshot: %s", err)
	}
	log.Printf("[INFO] Task created for Revert Snapshot: %v", task)

	err = task.Wait(context.TODO())
	if err != nil {
		log.Printf("[ERROR] Error While waiting for the Task of Revert Snapshot: %v", err)
		return fmt.Errorf("Error While waiting for the Task of Revert Snapshot: %s", err)
	}
	log.Printf("[INFO] Revert Snapshot completed %v", d.Get("snapshot_name").(string))
	d.SetId(d.Get("snapshot_name").(string))
	return nil

}

func resourceVSphereSnapshotDummyRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceVSphereSnapshotDummyDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}
