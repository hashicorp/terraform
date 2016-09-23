package vsphere

import (
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/types"
	"golang.org/x/net/context"
)

/*
this part should be merged in govmomi (module object) when we get a chance to do so
*/

func createDVPortgroup(c *govmomi.Client, dvsRef object.NetworkReference, spec types.DVPortgroupConfigSpec) (*object.Task, error) {
	req := types.CreateDVPortgroup_Task{
		Spec: spec,
		This: dvsRef.Reference(),
	}
	res, err := methods.CreateDVPortgroup_Task(context.TODO(), c.Client, &req)
	if err != nil {
		return nil, err
	}
	return object.NewTask(c.Client, res.Returnval), nil
}

func updateDVPortgroup(c *govmomi.Client, dvpgRef object.NetworkReference, spec types.DVPortgroupConfigSpec) (*object.Task, error) {
	req := types.ReconfigureDVPortgroup_Task{
		Spec: spec,
		This: dvpgRef.Reference(),
	}
	res, err := methods.ReconfigureDVPortgroup_Task(context.TODO(), c.Client, &req)
	if err != nil {
		return nil, err
	}
	return object.NewTask(c.Client, res.Returnval), nil
}

/* end of what should be contributed to govmomi */
