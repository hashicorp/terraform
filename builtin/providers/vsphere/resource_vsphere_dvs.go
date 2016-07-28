package vsphere

import (
	"fmt"
	"log"
	"time"

	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"golang.org/x/net/context"
)

type dvs struct {
	name         string
	folder       string
	datacenter   string
	extensionKey string
	description  string
	contact      struct {
		name  string
		infos string
	}
	switchUsagePolicy struct {
		autoPreinstallAllowed bool
		autoUpgradeAllowed    bool
		partialUpgradeAllowed bool
	}
	switchIPAddress    string
	numStandalonePorts int
}

// ResourceVSphereDVS exposes the DVS resource
func ResourceVSphereDVS() *schema.Resource {
	return &schema.Resource{
		Create: resourceVSphereDVSCreate,
		Read:   resourceVSphereDVSRead,
		Update: resourceVSphereDVSUpdate,
		Delete: resourceVSphereDVSDelete,
		Schema: resourceVSphereDVSSchema(),
	}
}

func resourceVSphereDVSSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"name": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
			// ForceNew:		true,
		},
		"folder": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
			// ForceNew:		true,
		},
		"datacenter": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
			// ForceNew:		true,
		},
		"extension_key": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
		},
		"description": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
		},
		"contact": &schema.Schema{
			Type:     schema.TypeMap,
			Optional: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"name": &schema.Schema{
						Type:     schema.TypeString,
						Required: true,
					},
					"infos": &schema.Schema{
						Type:     schema.TypeString,
						Required: true,
					},
				},
			},
		},
		"switch_usage_policy": &schema.Schema{
			Type:     schema.TypeMap,
			Optional: true,
			Default: map[string]bool{
				"auto_preinstall_allowed": false,
				"auto_upgrade_allowed":    false,
				"partial_upgrade_allowed": false,
			},
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"auto_preinstall_allowed": &schema.Schema{
						Type:     schema.TypeBool,
						Optional: true,
						Default:  false,
					},
					"auto_upgrade_allowed": &schema.Schema{
						Type:     schema.TypeBool,
						Optional: true,
						Default:  false,
					},
					"partial_upgrade_allowed": &schema.Schema{
						Type:     schema.TypeBool,
						Optional: true,
						Default:  false,
					},
				},
			},
		},
		"switch_ip_address": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
		},
		"num_standalone_ports": &schema.Schema{
			Type:     schema.TypeInt,
			Optional: true,
		},
		"full_path": &schema.Schema{
			Type:     schema.TypeString,
			Computed: true,
		},
	}
}

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
	datacenter, err := getDatacenter(c, d.datacenter)
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
	return waitForTaskEnd(task, "Could not reconfigure the DVS: %+v")
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
	if err := waitForTaskEnd(task, "[CreateSwitch.WaitForResult] Could not create the DVS: %+v"); err != nil {
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
	return waitForTaskEnd(task, "Could not complete Destroy - Task failed: %+v")
}

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

/** disabled (untested)
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


// */
