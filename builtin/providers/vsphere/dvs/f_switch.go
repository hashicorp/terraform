package dvs

import "github.com/hashicorp/terraform/helper/schema"
import "log"
import "fmt"
import "strconv"

// name format for DVS: datacenter, name

type dvsID struct {
	datacenter string
	path       string
}

/* functions for DistributedVirtualSwitch */

func (d *dvs) getID() string {
	return fmt.Sprintf(dvs_name_format, d.datacenter, d.getFullName())
}

func resourceVSphereDVSCreate(d *schema.ResourceData, meta interface{}) error {
	// this creates a DVS

	client, err := getGovmomiClient(meta)
	if err != nil {
		return err
	}
	item := dvs{}
	err = parseDVS(d, &item)
	if err != nil {
		return fmt.Errorf("Cannot parseDVS %+v: %+v", d, err)
	}
	err = item.createSwitch(client)
	if err != nil {
		return fmt.Errorf("Cannot createSwitch: %+v", err)
	}
	d.SetId(item.getID())
	d.Set("full_path", item.getFullName())
	return nil
}

func resourceVSphereDVSRead(d *schema.ResourceData, meta interface{}) error {

	var errs []error
	client, err := getGovmomiClient(meta)
	if err != nil {
		errs = append(errs, err)
	}

	// load the state from vSphere and provide the hydrated object.
	resourceID, err := parseDVSID(d.Id())
	if err != nil {
		errs = append(errs, fmt.Errorf("Cannot parse DVSID… %+v", err))
	}
	if len(errs) > 0 {
		log.Panicf("There are errors in DVSRead. Cannot proceed.\n%+v", errs)
	}
	dvsObject := dvs{}
	err = loadDVS(client, resourceID.datacenter, resourceID.path, &dvsObject)
	if err != nil {
		errs = append(errs, fmt.Errorf("Cannot read DVS %+v: %+v", resourceID, err))
	}
	if len(errs) > 0 { // we cannot load the DVS for a reason so
		log.Printf("[ERROR] Cannot load DVS %+v", resourceID)
		return errs[0]
	}
	// now the state is loaded so we should return
	return unparseDVS(d, &dvsObject)
}

func resourceVSphereDVSUpdate(d *schema.ResourceData, meta interface{}) error {

	/* client, err := getGovmomiClient(meta)
	if err != nil {
		return err
	}

	// detect the different changes in the object and perform needed updates
	*/
	client, err := getGovmomiClient(meta)
	if err != nil {
		return err
	}
	dvsObject := dvs{}
	err = parseDVS(d, &dvsObject)
	if err != nil {
		return fmt.Errorf("Cannot parse DVS")
	}
	hasChanges := false
	updatableFields := []string{"datacenter", "extension_key", "description",
		"switch_ip_address", "num_standalone_ports", "contact", "switch_usage_policy"}
	for _, f := range updatableFields {
		if d.HasChange(f) {
			hasChanges = true
			break
		}
	}
	if hasChanges {
		dvsObject.updateDVS(client)
	}
	return resourceVSphereDVSRead(d, meta)
}

func resourceVSphereDVSDelete(d *schema.ResourceData, meta interface{}) error {
	var errs []error
	var err error
	var resourceID *dvsID
	var dvsObject dvs

	client, err := getGovmomiClient(meta)
	if err != nil {
		return err
	}

	// remove the object and its dependencies in vSphere
	// use Destroy_Task
	resourceID, err = parseDVSID(d.Id())
	if err != nil {
		errs = append(errs, fmt.Errorf("Cannot parse DVSID… %+v", err))
		goto EndCondition
	}
	dvsObject = dvs{}
	err = loadDVS(client, resourceID.datacenter, resourceID.path, &dvsObject)
	if err != nil {
		errs = append(errs, fmt.Errorf("Cannot loadDVS… %+v", err))
		goto EndCondition
	}
	err = dvsObject.Destroy(client)
	if err != nil {
		errs = append(errs, err)
		goto EndCondition
	}
	// then remove object from the datastore.
	d.SetId("")

EndCondition:
	if len(errs) > 0 {
		return fmt.Errorf("There are errors in DVSRead. Cannot proceed.\n%+v", errs)
	}
	return nil
}

// parsers

// parse a provided Terraform config into a dvs struct
func parseDVS(d *schema.ResourceData, out *dvs) error {
	var f = out
	if v, ok := d.GetOk("name"); ok {
		f.name = v.(string)
	}
	if v, ok := d.GetOk("folder"); ok {
		f.folder = v.(string)
	}
	if v, ok := d.GetOk("datacenter"); ok {
		f.datacenter = v.(string)
	}
	if v, ok := d.GetOk("extension_key"); ok {
		f.extensionKey = v.(string)
	}
	if v, ok := d.GetOk("description"); ok {
		f.description = v.(string)
	}
	if v, ok := d.GetOk("switch_ip_address"); ok {
		f.switchIPAddress = v.(string)
	}
	if v, ok := d.GetOk("num_standalone_ports"); ok {
		f.numStandalonePorts = v.(int)
	}
	// contact
	if s, ok := d.GetOk("contact"); ok {
		vmap, casted := s.(map[string]interface{})
		if !casted {
			return fmt.Errorf("Cannot cast contact as a string map. Contact: %+v", s)
		}
		f.contact.name = vmap["name"].(string)
		f.contact.infos = vmap["infos"].(string)
	}
	if s, ok := d.GetOk("switch_usage_policy"); ok {
		vmap, casted := s.(map[string]interface{})
		if !casted {
			return fmt.Errorf("Cannot cast switch_usage_policy as a string map. Contact: %+v", s)
		}
		f.switchUsagePolicy.autoPreinstallAllowed = vmap["auto_preinstall_allowed"].(bool)
		f.switchUsagePolicy.autoUpgradeAllowed = vmap["auto_upgrade_allowed"].(bool)
		f.switchUsagePolicy.partialUpgradeAllowed = vmap["partial_upgrade_allowed"].(bool)
	}

	return nil
}

func unparseDVS(d *schema.ResourceData, in *dvs) error {
	var errs []error
	// define the contents - this means map the stuff to what Terraform expects
	fieldsMap := map[string]interface{}{
		"name":                 in.name,
		"folder":               in.folder,
		"datacenter":           in.datacenter,
		"extension_key":        in.extensionKey,
		"description":          in.description,
		"switch_ip_address":    in.switchIPAddress,
		"num_standalone_ports": in.numStandalonePorts,
		"contact": map[string]interface{}{
			"name":  in.contact.name,
			"infos": in.contact.infos,
		},
		"switch_usage_policy": map[string]interface{}{
			"auto_preinstall_allowed": strconv.FormatBool(in.switchUsagePolicy.autoPreinstallAllowed),
			"auto_upgrade_allowed":    strconv.FormatBool(in.switchUsagePolicy.autoUpgradeAllowed),
			"partial_upgrade_allowed": strconv.FormatBool(in.switchUsagePolicy.partialUpgradeAllowed),
		},
		"full_path": in.getFullName(),
	}
	// set values
	for fieldName, fieldValue := range fieldsMap {
		if err := d.Set(fieldName, fieldValue); err != nil {
			errs = append(errs, fmt.Errorf("[[%s] invalid: [%s]]: %s", fieldName, fieldValue, err))
		}
	}
	// handle errors
	if len(errs) > 0 {
		return fmt.Errorf("Errors in unparseDVS: invalid resource definition!\n%+v", errs)
	}
	return nil
}

func (d *dvs) getFullName() string {

	return fmt.Sprintf("%s/%s", d.folder, d.name)
}
