package dvs

import "github.com/hashicorp/terraform/helper/schema"
import "fmt"

// name format for MapHostDVS: datacenter, DVS name, Host name, nic name
/* functions for MapHostDVS */

type mapHostDVSID struct {
	datacenter string
	switchName string
	hostName   string
}

func (d *dvs_map_host_dvs) getID() string {
	switchID, err := parseDVSID(d.switchName)
	if err != nil {
		return "!!ERROR!!"
	}
	return fmt.Sprintf(maphostdvs_name_format, switchID.datacenter, switchID.path, d.hostName)
}

func resourceVSphereMapHostDVSCreate(d *schema.ResourceData, meta interface{}) error {
	var errs []error
	// start by getting the DVS
	client, err := getGovmomiClient(meta)
	if err != nil {
		errs = append(errs, err)
	}
	sid := d.Get("switch").(string)
	nicNames := d.Get("nic_names").([]string)
	hostName := d.Get("host").(string)
	dvsID, err := parseDVSID(sid)
	if err != nil {
		errs = append(errs, err)
		return fmt.Errorf("Cannot parse switchID %s: %+v", sid, errs)
	}
	dvsObj := dvs{}
	if err := loadDVS(client, dvsID.datacenter, dvsID.path, &dvsObj); err != nil {
		return fmt.Errorf("Cannot load DVS %+v: %+v", dvsID, errs)
	}
	// now that we have a DVS, we may add a Host to it.
	if err := dvsObj.addHost(client, hostName, nicNames); err != nil {
		return fmt.Errorf("Cannot addHost: %+v", err)
	}
	// now set ID
	d.SetId(fmt.Sprintf(maphostdvs_name_format, dvsID.datacenter, dvsID.path, hostName))
	return nil
}

func resourceVSphereMapHostDVSRead(d *schema.ResourceData, meta interface{}) error {
	// fill the ResourceData from the actual online contents.
	// this should check whether the current mapping exists on the server
	// and fill the ResourceData
	var errs []error
	var ok bool
	var hostMembers map[string]*dvs_map_host_dvs
	var dvsObj dvs
	var maphostdvs *dvs_map_host_dvs
	client, err := getGovmomiClient(meta)
	if err != nil {
		errs = append(errs, err)
	}
	mapIDs := d.Id()
	mapID, err := parseMapHostDVSID(mapIDs)
	if err != nil {
		errs = append(errs, err)
		// this is a game stopper. skip to End.
		goto EndCondition
	}
	if err = loadDVS(client, mapID.datacenter, mapID.switchName, &dvsObj); err != nil {
		errs = append(errs, err)
		// this is a game stopper. skip to End.
		goto EndCondition
	}
	// we have a DVS. Fetch its Host mapping members and return the one
	// with right host.
	hostMembers, err = dvsObj.getDVSHostMembers(client)

	if err != nil {
		errs = append(errs, err)
		goto EndCondition
	}
	maphostdvs, ok = hostMembers[mapID.hostName]
	if !ok {
		errs = append(errs, fmt.Errorf("Could not get key %s from switch %s", mapID.hostName, mapID.switchName))
		d.SetId("")
		goto EndCondition
	}
	// now fill the ResourceData

EndCondition:
	// tear down and return
	if len(errs) > 0 {
		return fmt.Errorf("Errors in MapHostDVSRead: %+v", errs)
	}
	return unparseMapHostDVS(d, maphostdvs)
}

func resourceVSphereMapHostDVSDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}

/* parse a MapHostDVS to its struct */
func parseMapHostDVS(d *schema.ResourceData) (*dvs_map_host_dvs, error) {
	o := dvs_map_host_dvs{}
	if v, ok := d.GetOk("host"); ok {
		o.hostName = v.(string)
	}
	if v, ok := d.GetOk("switch"); ok {
		o.switchName = v.(string)
	}
	if v, ok := d.GetOk("nic_names"); ok {
		o.nicName = v.([]string)
	}
	return &o, nil
}

func unparseMapHostDVS(d *schema.ResourceData, in *dvs_map_host_dvs) error {
	var errs []error
	toSet := map[string]interface{}{
		"host":      in.hostName,
		"switch":    in.switchName,
		"nic_names": in.nicName,
	}
	for k, v := range toSet {
		err := d.Set(k, v)
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("Errors in unparseMapHostDVS: %+v", errs)
	}
	return nil
}
