package vsphere

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"golang.org/x/net/context"
)

type dvs_port_group struct {
	name           string
	switchId       string
	defaultVLAN    int
	vlanRanges     []dvs_port_range
	pgType         string
	description    string
	autoExpand     bool
	numPorts       int
	portNameFormat string
	policy         struct {
		allowBlockOverride         bool
		allowLivePortMoving        bool
		allowNetworkRPOverride     bool
		portConfigResetDisconnect  bool
		allowShapingOverride       bool
		allowTrafficFilterOverride bool
		allowVendorConfigOverride  bool
	}
}

// name format for DVPG: datacenter, switch name, name

type dvPGID struct {
	datacenter string
	switchName string
	name       string
}

type dvs_port_range struct {
	start int
	end   int
}

// ResourceVSphereDVPG exposes the DVPG resource
func ResourceVSphereDVPG() *schema.Resource {
	return &schema.Resource{
		Create: resourceVSphereDVPGCreate,
		Read:   resourceVSphereDVPGRead,
		Update: resourceVSphereDVPGUpdate,
		Delete: resourceVSphereDVPGDelete,
		Schema: resourceVSphereDVPGSchema(),
	}
}

/* functions for DistributedVirtualPortgroup */
func resourceVSphereDVPGSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"name": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"switch_id": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"default_vlan": &schema.Schema{
			Type:     schema.TypeInt,
			Optional: true,
		},
		"vlan_range": &schema.Schema{
			Type:     schema.TypeSet,
			Optional: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"start": &schema.Schema{
						Type:     schema.TypeInt,
						Required: true,
					},
					"end": &schema.Schema{
						Type:     schema.TypeInt,
						Required: true,
					},
				},
			},
		},
		"datacenter": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"type": &schema.Schema{
			Type:        schema.TypeString,
			Required:    true,
			Description: "earlyBinding|ephemeral",
		},
		"description": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
		},
		"auto_expand": &schema.Schema{
			Type:     schema.TypeBool,
			Optional: true,
		},
		"num_ports": &schema.Schema{
			Type:     schema.TypeInt,
			Optional: true,
		},
		"port_name_format": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
		},
		"policy": &schema.Schema{
			Type:     schema.TypeSet,
			Computed: true,
			Optional: true,
			MaxItems: 1,
			Set:      _setDVPGPolicy,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"allow_block_override": &schema.Schema{
						Type:     schema.TypeBool,
						Optional: true,
						Default:  false,
					},
					"allow_live_port_moving": &schema.Schema{
						Type:     schema.TypeBool,
						Optional: true,
						Default:  false,
					},
					"allow_network_resources_pool_override": &schema.Schema{
						Type:     schema.TypeBool,
						Optional: true,
						Default:  false,
					},
					"port_config_reset_disconnect": &schema.Schema{
						Type:     schema.TypeBool,
						Optional: true,
						Default:  true,
					},
					"allow_shaping_override": &schema.Schema{
						Type:     schema.TypeBool,
						Optional: true,
						Default:  false,
					},
					"allow_traffic_filter_override": &schema.Schema{
						Type:     schema.TypeBool,
						Optional: true,
						Default:  false,
					},
					"allow_vendor_config_override": &schema.Schema{
						Type:     schema.TypeBool,
						Optional: true,
						Default:  false,
					},
				},
			},
		},
		"full_path": &schema.Schema{
			Type:     schema.TypeString,
			Computed: true,
		},
	}
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
	log.Printf("[DEBUG] policyItems read: %#v", policyItems)
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

// portgroup methods

func (p *dvs_port_group) getVmomiDVPG(c *govmomi.Client, datacenter, switchPath, name string) (*object.DistributedVirtualPortgroup, error) {
	datacenterO, _, err := getDCAndFolders(c, datacenter)
	if err != nil {
		return nil, fmt.Errorf("Could not get datacenter and folders: %+v", err)
	}
	finder := find.NewFinder(c.Client, true)
	finder.SetDatacenter(datacenterO)
	pgPath := fmt.Sprintf("%s/%s", dirname(switchPath), name)
	res, err := finder.Network(context.TODO(), pgPath)
	if err != nil {
		return nil, fmt.Errorf("Cannot get DVPG %s: %+v", pgPath, err)
	}
	castedobj, casted := res.(*object.DistributedVirtualPortgroup)
	if !casted {
		return nil, fmt.Errorf("Cannot cast %s to DVPG", pgPath)
	}
	return castedobj, nil
}

// load a DVPG and populate the passed struct with
func (p *dvs_port_group) loadDVPG(client *govmomi.Client, datacenter, switchName, name string, out *dvs_port_group) error {
	var pgmoObj mo.DistributedVirtualPortgroup
	pgObj, err := p.getVmomiDVPG(client, datacenter, switchName, name)
	if err != nil {
		return fmt.Errorf("[ERROR] Could not get pgObj: %+v", err)
	}
	sfolderName, sname := dirAndFile(switchName)
	dvsObj := dvs{
		datacenter: datacenter,
		folder:     sfolderName,
		name:       sname,
	}
	// tokensSwitch := strings.Split(switchName, "/")
	// folder := strings.Join(tokensSwitch[:len(tokensSwitch)-1], "/")
	err = pgObj.Properties(
		context.TODO(),
		pgObj.Reference(),
		[]string{"config", "key", "portKeys"},
		&pgmoObj)
	if err != nil {
		return fmt.Errorf("Could not retrieve properties: %+v", err)
	}
	policy := pgmoObj.Config.Policy.GetDVPortgroupPolicy()
	out.description = pgmoObj.Config.Description
	out.name = pgmoObj.Config.Name
	out.numPorts = int(pgmoObj.Config.NumPorts)
	out.autoExpand = *pgmoObj.Config.AutoExpand
	out.pgType = pgmoObj.Config.Type
	out.policy.allowBlockOverride = policy.BlockOverrideAllowed
	out.policy.allowLivePortMoving = policy.LivePortMovingAllowed
	out.policy.allowNetworkRPOverride = *policy.NetworkResourcePoolOverrideAllowed
	out.policy.allowShapingOverride = policy.ShapingOverrideAllowed
	out.policy.allowTrafficFilterOverride = *policy.TrafficFilterOverrideAllowed
	out.policy.allowVendorConfigOverride = policy.VendorConfigOverrideAllowed
	out.policy.portConfigResetDisconnect = policy.PortConfigResetAtDisconnect
	out.portNameFormat = pgmoObj.Config.PortNameFormat
	out.switchId = dvsObj.getID()
	vlans := pgmoObj.Config.DefaultPortConfig.(*types.VMwareDVSPortSetting).Vlan

	switch vlans.(type) {
	case *types.VmwareDistributedVirtualSwitchPvlanSpec:
		v := vlans.(*types.VmwareDistributedVirtualSwitchPvlanSpec)
		out.defaultVLAN = int(v.PvlanId)
	case *types.VmwareDistributedVirtualSwitchTrunkVlanSpec:
		v := vlans.(*types.VmwareDistributedVirtualSwitchTrunkVlanSpec)
		for _, r := range v.VlanId {
			out.vlanRanges = append(out.vlanRanges, dvs_port_range{
				start: int(r.Start),
				end:   int(r.End),
			})
		}
	case *types.VmwareDistributedVirtualSwitchVlanIdSpec:
		v := vlans.(*types.VmwareDistributedVirtualSwitchVlanIdSpec)
		out.defaultVLAN = int(v.VlanId)

	}
	return nil
}

func (p *dvs_port_group) makeDVPGConfigSpec() types.DVPortgroupConfigSpec {
	a := types.DVPortgroupPolicy{
		BlockOverrideAllowed:               p.policy.allowBlockOverride,
		LivePortMovingAllowed:              p.policy.allowLivePortMoving,
		NetworkResourcePoolOverrideAllowed: &p.policy.allowNetworkRPOverride,
		PortConfigResetAtDisconnect:        p.policy.portConfigResetDisconnect,
		ShapingOverrideAllowed:             p.policy.allowShapingOverride,
		TrafficFilterOverrideAllowed:       &p.policy.allowTrafficFilterOverride,
		VendorConfigOverrideAllowed:        p.policy.allowVendorConfigOverride,
	}

	dpc := types.VMwareDVSPortSetting{
		Vlan: &(types.VmwareDistributedVirtualSwitchVlanIdSpec{

			VlanId: int32(p.defaultVLAN),
		}),
	}
	dpcTrunk := types.VmwareDistributedVirtualSwitchTrunkVlanSpec{}

	for _, i := range p.vlanRanges {
		dpcTrunk.VlanId = append(dpcTrunk.VlanId, types.NumericRange{
			Start: int32(i.start),
			End:   int32(i.end),
		})
	}
	if len(p.vlanRanges) > 0 {
		dpc.Vlan = &dpcTrunk
	}
	return types.DVPortgroupConfigSpec{
		AutoExpand:        &p.autoExpand,
		Description:       p.description,
		Name:              p.name,
		NumPorts:          int32(p.numPorts),
		PortNameFormat:    p.portNameFormat,
		Type:              "earlyBinding",
		Policy:            &a,
		DefaultPortConfig: &dpc,
	}
}

func (p *dvs_port_group) createPortgroup(c *govmomi.Client) error {
	createSpec := p.makeDVPGConfigSpec()

	switchID, err := parseDVSID(p.switchId) // here we get the datacenter ID aswell

	if err != nil {
		return fmt.Errorf("Cannot parse switchID %s. %+v", p.switchId, err)
	}
	dvsObj := dvs{}

	err = loadDVS(c, switchID.datacenter, switchID.path, &dvsObj)
	if err != nil {
		return fmt.Errorf("Cannot loadDVS %+v: %+v", switchID, err)
	}
	log.Printf("DVS is now: %+v", dvsObj)
	dvsMo, err := dvsObj.getDVS(c, dvsObj.getFullName())
	if err != nil {
		return fmt.Errorf("Cannot getDVS: %+v", err)
	}
	task, err := createDVPortgroup(c, dvsMo, createSpec)

	_, err = task.WaitForResult(context.TODO(), nil)
	if err != nil {

		return fmt.Errorf("Could not create the DVPG: %+v", err)
	}
	return nil
}

func (p *dvs_port_group) updatePortgroup(c *govmomi.Client) error {
	updateSpec := p.makeDVPGConfigSpec()
	vmomi, err := p.getVmomiObject(c)
	if err != nil {
		return err
	}
	mo, err := p.getProperties(c)
	if err != nil {
		return err
	}
	updateSpec.ConfigVersion = mo.Config.ConfigVersion

	task, err := updateDVPortgroup(c, vmomi, updateSpec)

	_, err = task.WaitForResult(context.TODO(), nil)
	if err != nil {
		return fmt.Errorf("Could not update the DVPG: %+v", err)
	}
	return nil

}

func (p *dvs_port_group) deletePortgroup(c *govmomi.Client) error {
	return p.Destroy(c)
}

func (p *dvs_port_group) getProperties(c *govmomi.Client) (*mo.DistributedVirtualPortgroup, error) {

	dvspgMo := mo.DistributedVirtualPortgroup{}
	switchID, err := parseDVSID(p.switchId)
	if err != nil {
		return nil, err
	}

	dvspgobj, err := p.getVmomiDVPG(c, switchID.datacenter, switchID.path, p.name)
	if err != nil {
		return nil, err
	}
	return &dvspgMo, dvspgobj.Properties(
		context.TODO(),
		dvspgobj.Reference(),
		[]string{"config", "key", "portKeys"},
		&dvspgMo)
}

func (p *dvs_port_group) getVmomiObject(c *govmomi.Client) (*object.DistributedVirtualPortgroup, error) {
	switchID, err := parseDVSID(p.switchId)
	if err != nil {
		return nil, err
	}
	dvpg, err := p.getVmomiDVPG(c, switchID.datacenter, switchID.path, p.name)
	return dvpg, err
}

func (p *dvs_port_group) Destroy(c *govmomi.Client) error {
	switchID, err := parseDVSID(p.switchId)
	if err != nil {
		return err
	}
	dvpg, err := p.getVmomiDVPG(c, switchID.datacenter, switchID.path, p.name)
	if err != nil {
		return fmt.Errorf("Cannot call Destroy - cannot get object: %+v", err)
	}

	task, err := dvpg.Destroy(context.TODO())
	if err != nil {
		return fmt.Errorf("Cannot call Destroy - underlying Destroy NOK: %+v", err)
	}
	return waitForTaskEnd(task, "Could not complete Destroy - Task failed: %+v")
}

// functions

// load a DVPG
func loadDVPG(client *govmomi.Client, datacenter, switchName, name string) (*dvs_port_group, error) {
	dvpg := dvs_port_group{}

	err := dvpg.loadDVPG(client, datacenter, switchName, name, &dvpg)
	return &dvpg, err
}

// utility function

func _setDVPGPolicy(v interface{}) int {
	asmap := v.(map[string]interface{})
	components := []string{"allow_block_override", "allow_live_port_moving", "allow_network_resources_pool_override", "port_config_reset_disconnect", "allow_shaping_override", "allow_traffic_filter_override", "allow_vendor_config_override"}
	h := ""
	for _, i := range components {
		k, ok := asmap[i]
		if !ok {
			h += "unset"
			continue
		}
		h += fmt.Sprintf("%v-", k)
	}
	return schema.HashString(h)
}
