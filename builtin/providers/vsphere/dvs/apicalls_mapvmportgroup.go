package dvs

import (
	"fmt"
	"log"

	"github.com/davecgh/go-spew/spew"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/vim25/types"
	"golang.org/x/net/context"
)

// dvs_map_vm_dvpg methods

func (m *dvs_map_vm_dvpg) loadMapVMDVPG(c *govmomi.Client, datacenter, switchName, portgroup, vmPath string) (out *dvs_map_vm_dvpg, err error) {
	var errs []error
	vmObj, err := getVirtualMachine(c, datacenter, vmPath)
	if err != nil {
		errs = append(errs, err)
	}
	pgID, err := parseDVPGID(portgroup)
	if err != nil {
		errs = append(errs, err)
	}

	dvpgObj, err := loadDVPG(c, pgID.datacenter, pgID.switchName, pgID.name)
	if err != nil {
		errs = append(errs, err)
	}
	dvpgProps, err := dvpgObj.getProperties(c)
	if err != nil {
		errs = append(errs, err)
	}
	devs, err := vmObj.Device(context.TODO())
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		goto EndStatement
	}
	for _, dev := range devs {
		switch dev.(type) {
		case (types.BaseVirtualEthernetCard):
			log.Printf("Veth")
			d2 := dev.(types.BaseVirtualEthernetCard)
			back, casted := d2.GetVirtualEthernetCard().Backing.(*types.VirtualEthernetCardDistributedVirtualPortBackingInfo)
			if !casted {
				errs = append(errs, fmt.Errorf("Cannot cast Veth, aborting…"))
				goto EndStatement
			}
			if back.Port.PortgroupKey == dvpgProps.Key {
				out.nicLabel = d2.GetVirtualEthernetCard().VirtualDevice.DeviceInfo.GetDescription().Label
				break
			}
		default:
			log.Printf("Type not implemented: %T %+v\n", dev, dev)
		}
	}
	out.portgroup = portgroup
	out.vm = vmPath

EndStatement:
	if len(errs) > 0 {
		return nil, fmt.Errorf("Errors in loadMapVMDVPG: %+v", errs)
	}
	return out, err
}

func (m *dvs_map_vm_dvpg) createMapVMDVPG(c *govmomi.Client) error {
	var errs []error
	portgroupID, err := parseDVPGID(m.portgroup)
	if err != nil {
		errs = append(errs, err)
	}
	// get VM and NIC
	log.Printf("Boo:\n\n[%+v]\n[%+v]\n[%+v]\n\n\n", c, portgroupID, m)
	vm, err := getVirtualMachine(c, portgroupID.datacenter, m.vm)
	if err != nil {
		errs = append(errs, err)
		return err
	}
	veth, err := getVEthByName(c, vm, m.nicLabel)
	if err != nil {
		errs = append(errs, err)
		return err
	}
	portgroup, err := loadDVPG(c, portgroupID.datacenter, portgroupID.switchName, portgroupID.name)
	if err != nil {
		errs = append(errs, err)
	}

	// update backing informations of the VEth so it connects to the Portgroup
	switch veth.(type) {
	case types.BaseVirtualEthernetCard:
		veth2 := veth.(types.BaseVirtualEthernetCard)
		err = bindVEthAndPortgroup(c, vm, veth2, portgroup)
		if err != nil {
			errs = append(errs, err)
		}
	default:
		errs = append(errs, fmt.Errorf("Cannot handle type %T", veth))
	}
	// end
	if len(errs) > 0 {
		spew.Dump("Errors!", errs)
		return fmt.Errorf("Errors in createMapVMDVPG: {\n%+v\n}\n", errs)
	}
	return nil
}

func (m *dvs_map_vm_dvpg) deleteMapVMDVPG(c *govmomi.Client) error {
	var errs []error
	portgroupID, err := parseDVPGID(m.portgroup)
	if err != nil {
		errs = append(errs, err)
	}
	// get VM and NIC
	vm, err := getVirtualMachine(c, portgroupID.datacenter, m.vm)
	if err != nil {
		errs = append(errs, err)
		return err
	}
	veth, err := getVEthByName(c, vm, m.nicLabel)
	if err != nil {
		errs = append(errs, err)
	}
	portgroup, err := loadDVPG(c, portgroupID.datacenter, portgroupID.switchName, portgroupID.name)
	if err != nil {
		errs = append(errs, err)
	}

	// update backing informations of the VEth so it connects to the Portgroup
	err = unbindVEthAndPortgroup(c, vm, veth, portgroup)
	if err != nil {
		errs = append(errs, err)
	}
	// end
	if len(errs) > 0 {
		return fmt.Errorf("Errors in deleteMapVMDVPG: %+v", errs)
	}
	return nil

}
