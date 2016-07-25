package dvs

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/builtin/providers/vsphere/helpers"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"golang.org/x/net/context"
)

// methods for dvs objects

func (d *dvs) makeDVSCreateSpec() types.DVSCreateSpec {
	return types.DVSCreateSpec{
		ConfigSpec: d.makeDVSConfigSpec(),
	}
}

func (d *dvs) makeDVSConfigSpec() *types.DVSConfigSpec {
	return &types.DVSConfigSpec{
		Contact: &types.DVSContactInfo{
			Contact: d.contact.infos,
			Name:    d.contact.name,
		},
		ExtensionKey:       d.extensionKey,
		Description:        d.description,
		Name:               d.name,
		NumStandalonePorts: int32(d.numStandalonePorts),
		Policy: &types.DVSPolicy{
			AutoPreInstallAllowed: &d.switchUsagePolicy.autoPreinstallAllowed,
			AutoUpgradeAllowed:    &d.switchUsagePolicy.autoUpgradeAllowed,
			PartialUpgradeAllowed: &d.switchUsagePolicy.partialUpgradeAllowed,
		},
		SwitchIpAddress: d.switchIPAddress,
	}
}

func (d *dvs) getDCAndFolders(c *govmomi.Client) (*object.Datacenter, *object.DatacenterFolders, error) {
	datacenter, err := helpers.GetDatacenter(c, d.datacenter)
	if err != nil {
		return nil, nil, fmt.Errorf("Cannot get datacenter from %+v [%+v]", d, err)
	}

	// get Network Folder from datacenter
	dcFolders, err := datacenter.Folders(context.TODO())
	if err != nil {
		return nil, nil, fmt.Errorf("Cannot get folders for datacenter %+v [%+v]", datacenter, err)
	}
	return datacenter, dcFolders, nil
}

func (d *dvs) addHost(c *govmomi.Client, host string, nicNames []string) error {
	dvsItem, err := d.getDVS(c, d.getFullName())
	dvsStruct := mo.VmwareDistributedVirtualSwitch{}
	if err = dvsItem.Properties(
		context.TODO(),
		dvsItem.Reference(),
		[]string{"capability", "config", "networkResourcePool", "portgroup", "summary", "uuid"},
		&dvsStruct); err != nil {
		return fmt.Errorf("Could not get properties for %s", dvsItem)
	}
	config := dvsStruct.Config.GetDVSConfigInfo()
	hostObj, err := getHost(c, d.datacenter, host)
	if err != nil {
		return fmt.Errorf("Could not get host %s: %+v", host, err)
	}
	hostref := hostObj.Reference()

	var pnicSpecs []types.DistributedVirtualSwitchHostMemberPnicSpec
	for _, nic := range nicNames {
		pnicSpecs = append(
			pnicSpecs,
			types.DistributedVirtualSwitchHostMemberPnicSpec{
				PnicDevice: nic,
			})
	}
	newHost := types.DistributedVirtualSwitchHostMemberConfigSpec{
		Host:      hostref,
		Operation: "add",
		Backing: &types.DistributedVirtualSwitchHostMemberPnicBacking{
			PnicSpec: pnicSpecs,
		},
	}
	configSpec := types.DVSConfigSpec{
		ConfigVersion: config.ConfigVersion,
		Host:          []types.DistributedVirtualSwitchHostMemberConfigSpec{newHost},
	}
	task, err := dvsItem.Reconfigure(context.TODO(), &configSpec)
	if err != nil {
		return fmt.Errorf("Could not reconfigure the DVS: %+v", err)
	}
	return helpers.WaitForTaskEnd(task, "Could not reconfigure the DVS: %+v")
}

func (d *dvs) createSwitch(c *govmomi.Client) error {
	/*_, folders, err := d.getDCAndFolders(c)
	if err != nil {
		return fmt.Errorf("Could not get datacenter and  folders: %+v", err)
	}
	*/
	folder, err := changeFolder(c, d.datacenter, "network", d.folder)
	if err != nil {
		return fmt.Errorf("Cannot get to folder %v/%v: %+v", d.datacenter, d.folder, err)
	}

	// using Network Folder, create the DVSCreateSpec (pretty much a mapping of the config)
	spec := d.makeDVSCreateSpec()
	task, err := folder.CreateDVS(context.TODO(), spec)
	if err != nil {
		return fmt.Errorf("[CreateSwitch.CreateDVS] Could not create the DVS with spec\n\t%+v\nError: %T %+v\n", spec.ConfigSpec, err, err)
	}

	log.Printf("Started creation of switch: %s\n", time.Now())
	if err := helpers.WaitForTaskEnd(task, "[CreateSwitch.WaitForResult] Could not create the DVS: %+v"); err != nil {
		log.Printf("Failed creation of switch: %s\n", time.Now())
		return fmt.Errorf("[CreateSwitch.WaitForResult] Could not create the DVS with spec\n\t%+v\n\tError: %+v\n\tFolder: %+v", spec.ConfigSpec, err, folder)
	}
	log.Printf("Success creation of switch: %s\n", time.Now())
	return nil
}

// get a DVS from its name and populate the DVS with its infos
func (d *dvs) getDVS(c *govmomi.Client, dvsPath string) (*object.VmwareDistributedVirtualSwitch, error) {
	var res object.NetworkReference
	datacenter, _, err := d.getDCAndFolders(c)
	if err != nil {
		return nil, fmt.Errorf("Could not get datacenter and folders: %+v", err)
	}
	finder := find.NewFinder(c.Client, true)
	finder.SetDatacenter(datacenter)

	res, err = finder.Network(context.TODO(), dvsPath)
	if err != nil {
		return nil, fmt.Errorf("Cannot get DVS %s: %+v", dvsPath, err)
	}
	castedobj, casted := res.(*object.DistributedVirtualSwitch)
	if !casted {
		return nil, fmt.Errorf("Oops! Object %s is not a DVS but a %T", res, res)
	}
	return &(object.VmwareDistributedVirtualSwitch{*castedobj}), nil
}

// load a DVS and populate the struct with it
func (d *dvs) loadDVS(c *govmomi.Client, datacenter, dvsName string) error {
	var dvsMo mo.VmwareDistributedVirtualSwitch
	dvsobj, err := d.getDVS(c, dvsName)
	if err != nil {
		return err
	}
	folder, switchName := dirAndFile(dvsName)
	// retrieve the DVS properties

	err = dvsobj.Properties(
		context.TODO(),
		dvsobj.Reference(),
		[]string{"capability", "config", "networkResourcePool", "portgroup", "summary", "uuid"},
		&dvsMo)
	if err != nil {
		return fmt.Errorf("Could not retrieve properties: %+v", err)
	}
	// populate the struct from the data
	dvsci := dvsMo.Config.GetDVSConfigInfo()
	d.folder = folder
	d.contact.infos = dvsci.Contact.Contact
	d.contact.name = dvsci.Contact.Name
	d.description = dvsci.Description
	d.extensionKey = dvsci.ExtensionKey
	d.name = switchName
	d.numStandalonePorts = int(dvsci.NumStandalonePorts)
	d.switchIPAddress = dvsci.SwitchIpAddress
	d.switchUsagePolicy.autoPreinstallAllowed = *dvsci.Policy.AutoPreInstallAllowed
	d.switchUsagePolicy.autoUpgradeAllowed = *dvsci.Policy.AutoUpgradeAllowed
	d.switchUsagePolicy.partialUpgradeAllowed = *dvsci.Policy.PartialUpgradeAllowed
	// return nil: no error

	return nil
}

func (d *dvs) getDVSHostMembers(c *govmomi.Client) (out map[string]*dvs_map_host_dvs, err error) {
	properties, err := d.getProperties(c)
	if err != nil {
		return
	}
	hostInfos := properties.Config.GetDVSConfigInfo().Host
	// now we need to populate out
	for _, hostmember := range hostInfos {
		mapObj, err := loadMapHostDVS(d.getFullName(), hostmember)
		if err != nil {
			return nil, err
		}
		out[mapObj.hostName] = mapObj
	}
	return
}

func (d *dvs) updateDVS(c *govmomi.Client) error {
	updateSpec := d.makeDVSConfigSpec()
	props, err := d.getProperties(c)
	if err != nil {
		return fmt.Errorf("updateDVS::Could not get DVS properties")
	}
	updateSpec.ConfigVersion = props.Config.GetDVSConfigInfo().ConfigVersion
	dvsObj, err := d.getDVS(c, d.getFullName())
	if err != nil {
		return fmt.Errorf("updateDVS::Could not updateDVS")
	}
	dvsObj.Reconfigure(context.TODO(), updateSpec)
	return nil
}

func (d *dvs) getProperties(c *govmomi.Client) (out *mo.VmwareDistributedVirtualSwitch, err error) {
	dvsMo := mo.VmwareDistributedVirtualSwitch{}
	dvsobj, err := d.getDVS(c, d.getFullName())
	if err != nil {
		return nil, fmt.Errorf("Error in getProperties: [%T]%+v", err, err)
	}
	return &dvsMo, dvsobj.Properties(
		context.TODO(),
		dvsobj.Reference(),
		[]string{"capability", "config", "networkResourcePool", "portgroup", "summary", "uuid", "name"},
		&dvsMo)
}

func (d *dvs) Destroy(c *govmomi.Client) error {
	dvsO, err := d.getDVS(c, d.getFullName())
	if err != nil {
		return fmt.Errorf("Cannot call Destroy - cannot get object: %+v", err)
	}

	task, err := dvsO.Destroy(context.TODO())
	if err != nil {
		return fmt.Errorf("Cannot call Destroy - underlying Destroy NOK: %+v", err)
	}
	return helpers.WaitForTaskEnd(task, "Could not complete Destroy - Task failed: %+v")
}
