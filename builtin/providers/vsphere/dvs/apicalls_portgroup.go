package dvs

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/builtin/providers/vsphere/helpers"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"golang.org/x/net/context"
)

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
	return helpers.WaitForTaskEnd(task, "Could not complete Destroy - Task failed: %+v")
}
