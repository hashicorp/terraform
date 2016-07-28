package dvs

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
)

// name format for MapVMDVPG: datacenter, switch name, port name, vm name

type mapVMDVPGID struct {
	datacenter    string
	switchName    string
	portgroupName string
	vmName        string
}

/* Functions for MapVMDVPG */

func resourceVSphereMapVMDVPGCreate(d *schema.ResourceData, meta interface{}) error {
	var errs []error
	var err error
	var params *dvs_map_vm_dvpg
	// start by getting the DVS
	client, err := getGovmomiClient(meta)
	if err != nil {
		errs = append(errs, err)
	}
	params, err = parseMapVMDVPG(d)
	if err != nil {
		errs = append(errs, err)
		goto EndCondition
	}
	if err = params.createMapVMDVPG(client); err != nil {
		errs = append(errs, err)
	}
	// end
EndCondition:
	if len(errs) > 0 {
		return fmt.Errorf("Errors in MapVMDVPG.Create: %+v", errs)
	}
	d.SetId(params.getID())
	return nil
}

func resourceVSphereMapVMDVPGRead(d *schema.ResourceData, meta interface{}) error {
	// read the state of said DVPG using just its Id and set the d object
	// values accordingly

	var errs []error

	client, err := getGovmomiClient(meta)
	if err != nil {
		errs = append(errs, err)
	}
	// load the state from vSphere and provide the hydrated object.
	idObj, err := parseMapVMDVPGID(d.Id())
	if err != nil {
		errs = append(errs, fmt.Errorf("Cannot parse MapVMDVPGPGID… %+v", err))
	}
	if len(errs) > 0 {
		return fmt.Errorf("There are errors in MapVMDVPGRead. Cannot proceed.\n%+v", errs)
	}

	mapdvspgObject, err := loadMapVMDVPG(client, idObj.datacenter, idObj.switchName, idObj.portgroupName, idObj.vmName)
	if err != nil {
		errs = append(errs, fmt.Errorf("Cannot load MapVMDVPG %+v: %+v", err, err))
	}
	if len(errs) > 0 { // we cannot load the DVPG for a reason
		log.Printf("[ERROR] Cannot load MapVMDVPG %+v", mapdvspgObject)
		return fmt.Errorf("Errors in MapVMDVPGRead: %+v", errs)
	}
	// now just populate the ResourceData
	return unparseMapVMDVPG(d, mapdvspgObject)
}

func resourceVSphereMapVMDVPGUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceVSphereMapVMDVPGDelete(d *schema.ResourceData, meta interface{}) error {
	var errs []error

	client, err := getGovmomiClient(meta)
	if err != nil {
		errs = append(errs, err)
	}
	// load the state from vSphere and provide the hydrated object.
	idObj, err := parseMapVMDVPGID(d.Id())
	if err != nil {
		errs = append(errs, fmt.Errorf("Cannot parse MapVMDVPGPGID… %+v", err))
	}
	mapdvspgObject, err := loadMapVMDVPG(client, idObj.datacenter, idObj.switchName, idObj.portgroupName, idObj.vmName)
	if err != nil {
		errs = append(errs, fmt.Errorf("Cannot load MapVMDVPG %+v: %+v", err, err))
	}
	if len(errs) > 0 { // we cannot load the DVPG for a reason
		log.Printf("[ERROR] Cannot load MapVMDVPG %+v", mapdvspgObject)
		return fmt.Errorf("Errors in MapVMDVPGRead: %+v", errs)
	}
	// now just populate the ResourceData
	if err = mapdvspgObject.deleteMapVMDVPG(client); err != nil {
		return err
	}
	d.SetId("")
	return nil
}

func (d *dvs_map_vm_dvpg) getID() string {
	portgroupID, err := parseDVPGID(d.portgroup)
	if err != nil {
		return "!!ERROR!!"
	}
	return fmt.Sprintf(
		mapvmdvpg_name_format, portgroupID.datacenter,
		portgroupID.switchName, portgroupID.name, d.vm)
}

/* parse a MapVMDVPG to its struct */
func parseMapVMDVPG(d *schema.ResourceData) (*dvs_map_vm_dvpg, error) {
	o := dvs_map_vm_dvpg{}
	if v, ok := d.GetOk("vm"); ok {
		o.vm = v.(string)
	}
	if v, ok := d.GetOk("nic_label"); ok {
		o.nicLabel = v.(string)
	}
	if v, ok := d.GetOk("portgroup"); ok {
		o.portgroup = v.(string)
	}
	return &o, nil
}

// take a dvs_map_vm_dvpg and put its contents into the ResourceData.
func unparseMapVMDVPG(d *schema.ResourceData, in *dvs_map_vm_dvpg) error {
	var errs []error
	fieldsMap := map[string]interface{}{
		"nic_label": in.nicLabel,
		"portgroup": in.portgroup,
		"vm":        in.vm,
	}
	// set values
	for fieldName, fieldValue := range fieldsMap {
		if err := d.Set(fieldName, fieldValue); err != nil {
			errs = append(errs, fmt.Errorf("%s invalid: %s", fieldName, fieldValue))
		}
	}
	// handle errors
	if len(errs) > 0 {
		return fmt.Errorf("Errors in unparseDVPG: invalid resource definition!\n%+v", errs)
	}
	return nil
}
