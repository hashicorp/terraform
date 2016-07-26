package dvs

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
)

// name format for DVPG: datacenter, switch name, name

type dvPGID struct {
	datacenter string
	switchName string
	name       string
}

/* functions for DistributedVirtualPortgroup */
func (p *dvs_port_group) getID() string {
	switchID, _ := parseDVSID(p.switchId)

	return fmt.Sprintf(dvpg_name_format, switchID.datacenter, switchID.path, p.name)
}

func (p *dvs_port_group) getFullPath() string {
	switchID, _ := parseDVSID(p.switchId)
	return fmt.Sprintf("%s/%s", dirname(switchID.path), p.name)
}

func resourceVSphereDVPGCreate(d *schema.ResourceData, meta interface{}) error {

	client, err := getGovmomiClient(meta)
	if err != nil {
		return err
	}
	item := dvs_port_group{}
	err = parseDVPG(d, &item)
	if err != nil {
		return fmt.Errorf("Cannot parseDVPG %+v: %+v", d, err)
	}
	err = item.createPortgroup(client)
	if err != nil {
		return fmt.Errorf("Cannot createPortgroup: %+v", err)
	}
	d.SetId(item.getID())
	d.Set("full_path", item.getFullPath())
	return nil
}

func resourceVSphereDVPGRead(d *schema.ResourceData, meta interface{}) error {
	var errs []error

	client, err := getGovmomiClient(meta)
	if err != nil {
		errs = append(errs, err)
	}

	// load the state from vSphere and provide the hydrated object.
	resourceID, err := parseDVPGID(d.Id())
	if err != nil {
		errs = append(errs, fmt.Errorf("Cannot parse DVSPGID… %+v", err))
	}
	if len(errs) > 0 {
		return fmt.Errorf("There are errors in DVPGRead. Cannot proceed.\n%+v", errs)
	}
	dvspgObject := dvs_port_group{}
	err = dvspgObject.loadDVPG(client, resourceID.datacenter, resourceID.switchName, resourceID.name, &dvspgObject)
	if err != nil {
		errs = append(errs, fmt.Errorf("Cannot read DVPG %+v: %+v", resourceID, err))
	}
	if len(errs) > 0 { // we cannot load the DVPG for a reason
		log.Printf("[ERROR] Cannot load DVPG %+v", resourceID)
		d.SetId("")
		log.Printf("[ERROR] Errors in DVPGRead: %+v", errs)
		return nil
	}
	return unparseDVPG(d, &dvspgObject)
}

func resourceVSphereDVPGUpdate(d *schema.ResourceData, meta interface{}) error {
	/*
		// now populate the object
		if err:=unparseDVPG(d, &dvspgObject); err != nil {
			log.Printf("[ERROR] Cannot populate DVPG: %+v", err)
			return err
		}
	*/
	item := dvs_port_group{}
	client, err := getGovmomiClient(meta)
	if err != nil {
		return err
	}
	err = parseDVPG(d, &item)
	if err != nil {
		return err
	}
	updatableFields := []string{"default_vlan", "vlan_range", "description",
		"auto_expand", "num_ports", "port_name_format", "policy"}
	hasChange := false
	for _, u := range updatableFields {
		if d.HasChange(u) {
			log.Printf("[DEBUG] DVPG %s has change on field %s", d.Id(), u)
			hasChange = true
			break
		}
	}
	if hasChange {
		item.updatePortgroup(client)
	}
	// now we shall update the State
	return resourceVSphereDVPGRead(d, meta)
}

func resourceVSphereDVPGDelete(d *schema.ResourceData, meta interface{}) error {
	//var errs []error
	/*client, err := getGovmomiClient(meta)
	if err != nil {
		errs = append(errs, err)
	}
	*/
	// use Destroy_Task
	//d.SetId("")
	var errs []error
	var err error
	var resourceID *dvPGID
	var dvpg *dvs_port_group

	client, err := getGovmomiClient(meta)
	if err != nil {
		return err
	}
	// remove the object and its dependencies in vSphere
	// use Destroy_Task
	resourceID, err = parseDVPGID(d.Id())

	if err != nil {
		errs = append(errs, fmt.Errorf("Cannot parse DVPGID… %+v", err))
		goto EndCondition
	}
	dvpg, err = loadDVPG(client, resourceID.datacenter, resourceID.switchName, resourceID.name)
	if err != nil {
		errs = append(errs, fmt.Errorf("Cannot loadDVPG… %+v", err))
		goto EndCondition
	}
	err = dvpg.Destroy(client)
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

// parse a DVPG ResourceData to a dvs_port_group struct
func parseDVPG(d *schema.ResourceData, out *dvs_port_group) error {
	o := out
	_, okvlan := d.GetOk("default_vlan")
	_, okrange := d.GetOk("vlan_range")
	if okvlan && okrange {
		return fmt.Errorf("Cannot set both default_vlan and vlan_range")
	}
	if v, ok := d.GetOk("name"); ok {
		o.name = v.(string)
	}
	if v, ok := d.GetOk("switch_id"); ok {
		o.switchId = v.(string)
	}
	if v, ok := d.GetOk("description"); ok {
		o.description = v.(string)
	}
	if v, ok := d.GetOk("auto_expand"); ok {
		o.autoExpand = v.(bool)
	}
	if v, ok := d.GetOk("num_ports"); ok {
		o.numPorts = v.(int)
	}
	if v, ok := d.GetOk("default_vlan"); ok {
		o.defaultVLAN = v.(int)
	}
	if v, ok := d.GetOk("port_name_format"); ok {
		o.portNameFormat = v.(string)
	}
	if a, ok := d.GetOk("vlan_range"); ok {
		alist, casted := a.(*schema.Set)
		if !casted {
			log.Panicf("Bad cast ☹: %+v %T", a, a)
		}
		for _, v := range alist.List() {
			vmap, casted := v.(map[string]interface{})
			if !casted {
				log.Panicf("Bad cast 2 ☹: %+v %T", v, v)
			}
			o.vlanRanges = append(
				o.vlanRanges,
				dvs_port_range{
					start: vmap["start"].(int),
					end:   vmap["end"].(int),
				})
		}
	}
	if s, ok := d.GetOk("policy"); ok {
		log.Printf("[DEBUG] Got dvpg.policy: %#v", s)
		setVal, ok := s.(*schema.Set)
		if !ok {
			return fmt.Errorf("Cannot cast policy as a Set with string map. See: %#v", s)
		}
		if len(setVal.List()) == 0 {
			return fmt.Errorf("No PG policy set")
		}
		vmap, casted := setVal.List()[0].(map[string]interface{})
		if !casted {
			return fmt.Errorf("Cannot cast policy as a map[string]interface{}. See: %#v", s)
		}
		o.policy.allowBlockOverride = vmap["allow_block_override"].(bool)
		o.policy.allowLivePortMoving = vmap["allow_live_port_moving"].(bool)
		o.policy.allowNetworkRPOverride = vmap["allow_network_resources_pool_override"].(bool)
		o.policy.portConfigResetDisconnect = vmap["port_config_reset_disconnect"].(bool)
		o.policy.allowShapingOverride = vmap["allow_shaping_override"].(bool)
		o.policy.allowTrafficFilterOverride = vmap["allow_traffic_filter_override"].(bool)
		o.policy.allowVendorConfigOverride = vmap["allow_vendor_config_override"].(bool)
		return nil
	}
	log.Printf("[DEBUG] Could not get dvpg.policy")
	o.policy.allowBlockOverride = false
	o.policy.allowLivePortMoving = false
	o.policy.allowNetworkRPOverride = false
	o.policy.portConfigResetDisconnect = true
	o.policy.allowShapingOverride = false
	o.policy.allowTrafficFilterOverride = false
	o.policy.allowVendorConfigOverride = false
	return nil
}

// fill a ResourceData using the provided DVPG
func unparseDVPG(d *schema.ResourceData, in *dvs_port_group) error {
	var errs []error
	// define the contents - this means map the stuff to what Terraform expects
	policyItems := map[string]interface{}{

		"allow_block_override":                  (in.policy.allowBlockOverride),
		"allow_live_port_moving":                (in.policy.allowLivePortMoving),
		"allow_network_resources_pool_override": (in.policy.allowNetworkRPOverride),
		"port_config_reset_disconnect":          (in.policy.portConfigResetDisconnect),
		"allow_shaping_override":                (in.policy.allowShapingOverride),
		"allow_traffic_filter_override":         (in.policy.allowTrafficFilterOverride),
		"allow_vendor_config_override":          (in.policy.allowVendorConfigOverride),
	}
	policyObj := &schema.Set{
		F: _setDVPGPolicy,
	}
	policyObj.Add(policyItems)
	fieldsMap := map[string]interface{}{
		"name":             in.name,
		"switch_id":        in.switchId,
		"description":      in.description,
		"default_vlan":     in.defaultVLAN,
		"auto_expand":      in.autoExpand,
		"num_ports":        in.numPorts,
		"port_name_format": in.portNameFormat,
		"policy":           policyObj,
		"full_path":        in.getFullPath(),
	}
	vlans := []map[string]interface{}{}
	for _, numPair := range in.vlanRanges {
		vlans = append(vlans, map[string]interface{}{
			"start": numPair.start,
			"end":   numPair.end,
		})
	}
	fieldsMap["vlan_range"] = vlans
	// set values
	for fieldName, fieldValue := range fieldsMap {
		if err := d.Set(fieldName, fieldValue); err != nil {
			errs = append(errs, fmt.Errorf("%s invalid: %s: %+v", fieldName, fieldValue, err))
		} else {
			log.Printf("[DEBUG] No error for setting field %s to %#v", fieldName, fieldValue)
		}
	}

	// handle errors
	if len(errs) > 0 {
		return fmt.Errorf("Errors in unparseDVPG: invalid resource definition!\n%+v", errs)
	}
	return nil
}
