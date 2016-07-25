package dvs

import (
	"fmt"
	"log"

	"github.com/davecgh/go-spew/spew" // debug dependency
	"github.com/hashicorp/terraform/builtin/providers/vsphere/helpers"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
	"golang.org/x/net/context"
)

// load a DVS
func loadDVS(c *govmomi.Client, datacenter, dvsPath string, output *dvs) error {
	output.datacenter = datacenter
	err := output.loadDVS(c, datacenter, dvsPath)
	return err
}

// load map between host and DVS
func loadMapHostDVS(switchName string, hostmember types.DistributedVirtualSwitchHostMember) (out *dvs_map_host_dvs, err error) {
	h := hostmember.Config.Host
	hostObj, casted := h.Value, true
	if !casted {
		err = fmt.Errorf("Could not cast Host to mo.HostSystem")
		return
	}
	backingInfosObj := hostmember.Config.Backing

	backingInfos, casted := backingInfosObj.(*types.DistributedVirtualSwitchHostMemberPnicBacking)
	if !casted {
		err = fmt.Errorf("Could not cast Host to mo.HostSystem")
		return
	}
	for _, pnic := range backingInfos.PnicSpec {
		out.nicName = append(out.nicName, pnic.PnicDevice)
	}
	out.switchName = switchName
	out.hostName = hostObj
	return
}

// load a DVPG
func loadDVPG(client *govmomi.Client, datacenter, switchName, name string) (*dvs_port_group, error) {
	dvpg := dvs_port_group{}

	err := dvpg.loadDVPG(client, datacenter, switchName, name, &dvpg)
	return &dvpg, err
}

// load a map between DVPG and VM
func loadMapVMDVPG(client *govmomi.Client, datacenter, switchName, portgroup, vmPath string) (out *dvs_map_vm_dvpg, err error) {
	return out.loadMapVMDVPG(client, datacenter, switchName, portgroup, vmPath)
}

// Host manipulation functions

func getHost(c *govmomi.Client, datacenter, hostPath string) (*object.HostSystem, error) {
	dc, _, err := getDCAndFolders(c, datacenter)
	if err != nil {
		return nil, fmt.Errorf("Could not get DC and folders: %+v", err)
	}
	finder := find.NewFinder(c.Client, true)
	finder.SetDatacenter(dc)
	host, err := finder.HostSystem(context.TODO(), hostPath)
	if err != nil {
		return nil, fmt.Errorf("Cannot find HostSystem %s: %+v", hostPath, err)
	}
	return host, nil
}

// VM manipulation functions

func getVirtualMachine(c *govmomi.Client, datacenter, vmPath string) (*object.VirtualMachine, error) {
	var finder *find.Finder
	var errs []error
	var err error
	var dc *object.Datacenter
	//var folders *object.DatacenterFolders
	var vm *object.VirtualMachine
	dc, _, err = getDCAndFolders(c, datacenter)
	if err != nil {
		errs = append(errs, err)
		goto EndPosition
	}
	finder = find.NewFinder(c.Client, true)
	finder.SetDatacenter(dc)
	vm, err = finder.VirtualMachine(context.TODO(), vmPath)
	if err != nil {
		errs = append(errs, err)
		goto EndPosition
	}
EndPosition:
	if len(errs) > 0 {
		err = fmt.Errorf("Errors in getVirtualMachine: %+v", errs)
	}
	return vm, err
}

// device manipulation functions

func getDeviceByName(c *govmomi.Client, vm *object.VirtualMachine, deviceName string) (*types.BaseVirtualDevice, error) {
	devices, err := vm.Device(context.TODO())
	if err != nil {
		return nil, err
	}
	out := devices.Find(deviceName)
	if out == nil {
		return nil, fmt.Errorf("Could not get device named %v\n", deviceName)
	}

	return &out, nil
}

// VEth manipulation functions

func getVEthByName(c *govmomi.Client, vm *object.VirtualMachine, deviceName string) (types.BaseVirtualEthernetCard, error) {
	dev, err := getDeviceByName(c, vm, deviceName)
	if err != nil {
		return nil, err
	}
	if dev == nil {
		return nil, fmt.Errorf("Cannot return VEth: %T:%+v", err, err)
	}
	return (*dev).(types.BaseVirtualEthernetCard), nil
	/*vc := (*dev).(types.BaseVirtualEthernetCard).GetVirtualEthernetCard()
	if vmx2, casted := (*dev).(*types.VirtualVmxnet2); casted {
		return vmx2.GetVirtualEthernetCard(), "vmx2", nil
	} else if vmx3, casted := (*dev).(*types.VirtualVmxnet3); casted {
		return vmx3.GetVirtualEthernetCard(), "vmx3", nil
	} else if vmx, casted := (*dev).(*types.VirtualVmxnet); casted {
		return vmx.GetVirtualEthernetCard(), "vmx", nil
	} else if e1000e, casted := (*dev).(*types.VirtualE1000e); casted {
		return e1000e.GetVirtualEthernetCard(), "e1000e", nil
	} else if e1000, casted := (*dev).(*types.VirtualE1000); casted {
		return e1000.GetVirtualEthernetCard(), "e1000", nil
	} else if pcnet, casted := (*dev).(*types.VirtualPCNet32); casted {
		return pcnet.GetVirtualEthernetCard(), "pcnet32", nil
	} else if sriov, casted := (*dev).(*types.VirtualSriovEthernetCard); casted {
		return sriov.GetVirtualEthernetCard(), "sriov", nil
	}
	return vc, "unknown", nil
	*/
}

func setVethDeviceAndBackingInEthChange(devchange *types.VirtualDeviceConfigSpec, veth types.BaseVirtualEthernetCard, cbk types.BaseVirtualDeviceBackingInfo) error {
	switch t := veth.(type) {
	case *types.VirtualVmxnet3:
		devchange.Device = t
		t.Backing = cbk
	case *types.VirtualVmxnet2:
		devchange.Device = t
		t.Backing = cbk
	case *types.VirtualE1000:
		devchange.Device = t
		t.Backing = cbk
	case *types.VirtualE1000e:
		devchange.Device = t
		t.Backing = cbk
	case *types.VirtualPCNet32:
		devchange.Device = t
		t.Backing = cbk
	default:
		return fmt.Errorf("Incorrect veth type! %T", veth)
	}
	return nil
}

// buildVEthDeviceChange veth *types.VirtualEthernetCard
func buildVEthDeviceChange(c *govmomi.Client, veth types.BaseVirtualEthernetCard, portgroup *dvs_port_group, optype types.VirtualDeviceConfigSpecOperation) (*types.VirtualDeviceConfigSpec, error) {
	devChange := types.VirtualDeviceConfigSpec{}
	cbk := types.VirtualEthernetCardDistributedVirtualPortBackingInfo{}

	properties, err := portgroup.getProperties(c)
	if err != nil {
		return nil, fmt.Errorf("Cannot get portgroup properties: %+v", err)
	}
	switchID, err := parseDVSID(portgroup.switchId)
	if err != nil {
		return nil, fmt.Errorf("Cannot parse switchID: %+v", err)
	}
	dvsObj := dvs{}
	err = loadDVS(c, switchID.datacenter, switchID.path, &dvsObj)
	if err != nil {
		return nil, fmt.Errorf("Cannot get switch: %+v", err)
	}
	dvsProps, err := dvsObj.getProperties(c)
	if err != nil {
		return nil, fmt.Errorf("Cannot get dvs properties: %+v", err)
	}

	cbk.Port = types.DistributedVirtualSwitchPortConnection{
		PortgroupKey: properties.Key,
		SwitchUuid:   dvsProps.Uuid,
	}
	devChange.Operation = optype // `should be add, remove or edit`
	setVethDeviceAndBackingInEthChange(&devChange, veth, &cbk)
	return &devChange, nil
}

// bind a VEth and a portgroup → change the VEth so it is bound to one port in the portgroup.
func bindVEthAndPortgroup(c *govmomi.Client, vm *object.VirtualMachine, veth types.BaseVirtualEthernetCard, portgroup *dvs_port_group) error {
	// use a VirtualMachineConfigSpec.deviceChange (VirtualDeviceConfigSpec[])
	conf := types.VirtualMachineConfigSpec{}
	devspec, err := buildVEthDeviceChange(c, veth, portgroup, types.VirtualDeviceConfigSpecOperationEdit)
	if err != nil {
		return err
	}
	devspec.Device.GetVirtualDevice().Connectable = &(types.VirtualDeviceConnectInfo{
		Connected: true,
	})

	conf.DeviceChange = []types.BaseVirtualDeviceConfigSpec{devspec}

	log.Printf("\n\n\nHere comes the debug\n")
	spew.Dump("VM to be reconfigured", vm)
	task, err := vm.Reconfigure(context.TODO(), conf)
	if err != nil {
		spew.Dump("Error\n\n", err, "\n\n")
		return err
	}
	return helpers.WaitForTaskEnd(task, "Cannot complete vm.Reconfigure: %+v")
}

// unbind a VEth and a Portgroup
func unbindVEthAndPortgroup(c *govmomi.Client, vm *object.VirtualMachine, veth types.BaseVirtualEthernetCard, portgroup *dvs_port_group) error {
	// use a VirtualMachineConfigSpec.deviceChange (VirtualDeviceConfigSpec[])
	conf := types.VirtualMachineConfigSpec{}
	devspec, err := buildVEthDeviceChange(c, veth, portgroup, types.VirtualDeviceConfigSpecOperationEdit)
	if err != nil {
		return err
	}
	devspec.Device.GetVirtualDevice().Connectable = &(types.VirtualDeviceConnectInfo{
		Connected: false,
	})

	conf.DeviceChange = []types.BaseVirtualDeviceConfigSpec{devspec}

	task, err := vm.Reconfigure(context.TODO(), conf)
	if err != nil {
		return err
	}
	return helpers.WaitForTaskEnd(task, "Cannot complete vm.Reconfigure: %+v")
}

func vmDebug(c *govmomi.Client, vm *object.VirtualMachine) {
	return
}
