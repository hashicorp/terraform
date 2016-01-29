/*
Copyright (c) 2014-2015 VMware, Inc. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package vm

import (
	"flag"
	"fmt"

	"github.com/vmware/govmomi/govc/cli"
	"github.com/vmware/govmomi/govc/flags"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/types"
	"golang.org/x/net/context"
)

type create struct {
	*flags.ClientFlag
	*flags.DatacenterFlag
	*flags.DatastoreFlag
	*flags.ResourcePoolFlag
	*flags.HostSystemFlag
	*flags.NetworkFlag

	memory     int
	cpus       int
	guestID    string
	link       bool
	on         bool
	force      bool
	iso        string
	disk       string
	controller string

	Client       *vim25.Client
	Datacenter   *object.Datacenter
	Datastore    *object.Datastore
	ResourcePool *object.ResourcePool
	HostSystem   *object.HostSystem
}

func init() {
	cli.Register("vm.create", &create{})
}

func (cmd *create) Register(ctx context.Context, f *flag.FlagSet) {
	cmd.ClientFlag, ctx = flags.NewClientFlag(ctx)
	cmd.ClientFlag.Register(ctx, f)

	cmd.DatacenterFlag, ctx = flags.NewDatacenterFlag(ctx)
	cmd.DatacenterFlag.Register(ctx, f)

	cmd.DatastoreFlag, ctx = flags.NewDatastoreFlag(ctx)
	cmd.DatastoreFlag.Register(ctx, f)

	cmd.ResourcePoolFlag, ctx = flags.NewResourcePoolFlag(ctx)
	cmd.ResourcePoolFlag.Register(ctx, f)

	cmd.HostSystemFlag, ctx = flags.NewHostSystemFlag(ctx)
	cmd.HostSystemFlag.Register(ctx, f)

	cmd.NetworkFlag, ctx = flags.NewNetworkFlag(ctx)
	cmd.NetworkFlag.Register(ctx, f)

	f.IntVar(&cmd.memory, "m", 1024, "Size in MB of memory")
	f.IntVar(&cmd.cpus, "c", 1, "Number of CPUs")
	f.StringVar(&cmd.guestID, "g", "otherGuest", "Guest OS")
	f.BoolVar(&cmd.link, "link", true, "Link specified disk")
	f.BoolVar(&cmd.on, "on", true, "Power on VM. Default is true if -disk argument is given.")
	f.BoolVar(&cmd.force, "force", false, "Create VM if vmx already exists")
	f.StringVar(&cmd.iso, "iso", "", "Path to ISO")
	f.StringVar(&cmd.controller, "disk.controller", "scsi", "Disk controller type")
	f.StringVar(&cmd.disk, "disk", "", "Disk path name")
}

func (cmd *create) Process(ctx context.Context) error {
	if err := cmd.ClientFlag.Process(ctx); err != nil {
		return err
	}
	if err := cmd.DatacenterFlag.Process(ctx); err != nil {
		return err
	}
	if err := cmd.DatastoreFlag.Process(ctx); err != nil {
		return err
	}
	if err := cmd.ResourcePoolFlag.Process(ctx); err != nil {
		return err
	}
	if err := cmd.HostSystemFlag.Process(ctx); err != nil {
		return err
	}
	if err := cmd.NetworkFlag.Process(ctx); err != nil {
		return err
	}
	return nil
}

func (cmd *create) Run(ctx context.Context, f *flag.FlagSet) error {
	var err error

	if len(f.Args()) != 1 {
		return flag.ErrHelp
	}

	cmd.Client, err = cmd.ClientFlag.Client()
	if err != nil {
		return err
	}

	cmd.Datacenter, err = cmd.DatacenterFlag.Datacenter()
	if err != nil {
		return err
	}

	cmd.Datastore, err = cmd.DatastoreFlag.Datastore()
	if err != nil {
		return err
	}

	cmd.HostSystem, err = cmd.HostSystemFlag.HostSystemIfSpecified()
	if err != nil {
		return err
	}

	if cmd.HostSystem != nil {
		if cmd.ResourcePool, err = cmd.HostSystem.ResourcePool(context.TODO()); err != nil {
			return err
		}
	} else {
		// -host is optional
		if cmd.ResourcePool, err = cmd.ResourcePoolFlag.ResourcePool(); err != nil {
			return err
		}
	}

	for _, file := range []*string{&cmd.iso, &cmd.disk} {
		if *file != "" {
			_, err = cmd.Datastore.Stat(context.TODO(), *file)
			if err != nil {
				return err
			}
		}
	}

	task, err := cmd.createVM(f.Arg(0))
	if err != nil {
		return err
	}

	info, err := task.WaitForResult(context.TODO(), nil)
	if err != nil {
		return err
	}

	vm := object.NewVirtualMachine(cmd.Client, info.Result.(types.ManagedObjectReference))

	if err := cmd.addDevices(vm); err != nil {
		return err
	}

	if cmd.on {
		task, err := vm.PowerOn(context.TODO())
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

func (cmd *create) addDevices(vm *object.VirtualMachine) error {
	devices, err := vm.Device(context.TODO())
	if err != nil {
		return err
	}

	var add []types.BaseVirtualDevice

	if cmd.disk != "" {
		controller, err := devices.FindDiskController(cmd.controller)
		if err != nil {
			return err
		}

		disk := devices.CreateDisk(controller, cmd.Datastore.Path(cmd.disk))

		if cmd.link {
			disk = devices.ChildDisk(disk)
		}

		add = append(add, disk)
	}

	if cmd.iso != "" {
		ide, err := devices.FindIDEController("")
		if err != nil {
			return err
		}

		cdrom, err := devices.CreateCdrom(ide)
		if err != nil {
			return err
		}

		add = append(add, devices.InsertIso(cdrom, cmd.Datastore.Path(cmd.iso)))
	}

	netdev, err := cmd.NetworkFlag.Device()
	if err != nil {
		return err
	}

	add = append(add, netdev)

	return vm.AddDevice(context.TODO(), add...)
}

func (cmd *create) createVM(name string) (*object.Task, error) {
	spec := types.VirtualMachineConfigSpec{
		Name:     name,
		GuestId:  cmd.guestID,
		Files:    &types.VirtualMachineFileInfo{VmPathName: fmt.Sprintf("[%s]", cmd.Datastore.Name())},
		NumCPUs:  cmd.cpus,
		MemoryMB: int64(cmd.memory),
	}

	if !cmd.force {
		vmxPath := fmt.Sprintf("%s/%s.vmx", name, name)

		_, err := cmd.Datastore.Stat(context.TODO(), vmxPath)
		if err == nil {
			dsPath := cmd.Datastore.Path(vmxPath)
			return nil, fmt.Errorf("File %s already exists", dsPath)
		}
	}

	if cmd.controller != "ide" {
		scsi, err := object.SCSIControllerTypes().CreateSCSIController(cmd.controller)
		if err != nil {
			return nil, err
		}

		spec.DeviceChange = append(spec.DeviceChange, &types.VirtualDeviceConfigSpec{
			Operation: types.VirtualDeviceConfigSpecOperationAdd,
			Device:    scsi,
		})
	}

	folders, err := cmd.Datacenter.Folders(context.TODO())
	if err != nil {
		return nil, err
	}

	return folders.VmFolder.CreateVM(context.TODO(), spec, cmd.ResourcePool, cmd.HostSystem)
}
