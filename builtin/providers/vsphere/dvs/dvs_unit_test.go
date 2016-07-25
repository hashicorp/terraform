/* test our govmomi wrappings to ensure they work properly */
package dvs

import (
	"fmt"
	"os"
	"testing"

	"github.com/vmware/govmomi"
)

var testParameters map[string]interface{}
var client *govmomi.Client

func init() {

	var err error
	client, err = getTestGovmomiClient()
	if err != nil {
		return
	}
}
func buildTestDVS(variant string) *dvs {
	dvsO := dvs{}
	dvsO.datacenter = testParameters["datacenter"].(string)
	dvsO.folder = testParameters["switchFolder"].(string)
	dvsO.name = fmt.Sprintf(testParameters["switchName"].(string), variant)
	dvsO.description = testParameters["switchDescription"].(string)
	dvsO.contact.infos = testParameters["contactInfos"].(string)
	dvsO.contact.name = testParameters["contactName"].(string)
	dvsO.numStandalonePorts = testParameters["numStandalonePorts"].(int)
	dvsO.switchUsagePolicy.autoPreinstallAllowed = true
	dvsO.switchUsagePolicy.autoUpgradeAllowed = true
	dvsO.switchUsagePolicy.partialUpgradeAllowed = false
	dvsO.switchIPAddress = testParameters["switchIPAddress"].(string)
	return &dvsO
}

func buildTestDVPG(variant string, dvsInfo *dvs) *dvs_port_group {
	dvpg := dvs_port_group{}
	dvpg.autoExpand = testParameters["dvpgAutoExpand"].(bool)
	dvpg.name = fmt.Sprintf(testParameters["portgroupName"].(string), variant)
	dvpg.numPorts = testParameters["portgroupPorts"].(int)
	dvpg.switchId = dvsInfo.getID()
	dvpg.description = testParameters["portgroupDescription"].(string)
	dvpg.pgType = "earlyBinding"
	dvpg.portNameFormat = "<dvsName>-<portIndex>"
	dvpg.defaultVLAN = 1337
	return &dvpg
}

func testCreateDVS(dvsObject *dvs, client *govmomi.Client) error {
	return dvsObject.createSwitch(client)
}

func testDeleteDVS(dvsObject *dvs, client *govmomi.Client) error {
	return dvsObject.Destroy(client)
}

func doCreateDVS(dvsO *dvs, t *testing.T) {
	var err error
	if err = testCreateDVS(dvsO, client); err != nil {
		t.Logf("[ERROR] Cannot create switch: %+v\n", err)
		t.Fail()
	}
	t.Log("Created DVS. Now getting props")

	props, err := dvsO.getProperties(client)
	if err != nil {
		t.Logf("Cannot retrieve DVS properties, failing: [%T]%+v\nProperties obj: [%T]%+v\n", err, err, props, props)

		t.Fail()
	} else {
		t.Log("Got properties. The DVS has been created")
	}
}

func doDeleteDVS(dvsO *dvs, t *testing.T) {
	if err := testDeleteDVS(dvsO, client); err != nil {
		t.Logf("[ERROR] Cannot delete switch: %+v\n", err)
		t.Fail()
	}
}

// Test DVS creation and destruction
func aaTestDVSCreationAndDestruction(t *testing.T) {
	// need:
	// datacenter name, host name 1, host name 2, switch path
	dvsO := buildTestDVS("test1")
	doCreateDVS(dvsO, t)
	doDeleteDVS(dvsO, t)
}

func testCreateDVPG(dvpg *dvs_port_group, client *govmomi.Client) error {
	return dvpg.createPortgroup(client)
}

func testDeleteDVPG(dvpg *dvs_port_group, client *govmomi.Client) error {
	return dvpg.deletePortgroup(client)
}

func doCreateDVPortgroup(dvpg *dvs_port_group, t *testing.T) {
	t.Logf("Create DVPG: %+v", dvpg)
	if err := testCreateDVPG(dvpg, client); err != nil {
		t.Logf("[ERROR] Cannot create portgroup: %+v\n", err)
		t.Fail()
	}
	t.Log("Created DVPG. Now getting props")

	props, err := dvpg.getProperties(client)
	if err != nil {
		t.Logf("Cannot retrieve DVPS properties, failing: [%T]%+v\nProperties obj: [%T]%+v\n", err, err, props, props)
		t.Fail()
	} else {
		t.Log("Got properties. The object is well created.")
	}
}

func doDeleteDVPortgroup(dvpg *dvs_port_group, t *testing.T) {
	t.Logf("Delete DVPG")
	if err := testDeleteDVPG(dvpg, client); err != nil {
		t.Logf("[ERROR] Cannot delete portgroup: %+v\n", err)
		t.Fail()
	} else {
		t.Log("Deleted DVPG ", dvpg)
	}
}

func aaTestPortgroupCreationAndDestruction(t *testing.T) {
	// need:
	// datacenter name, switch path, portgroup name
	dvsPath := "fillme"
	dvsO := dvs{}
	if err := loadDVS(client, testParameters["datacenter"].(string), dvsPath, &dvsO); err != nil {
		t.Logf("Could not load DVS with client %+v %+v %+v %+v: %+v\n", client, testParameters["datacenter"], dvsPath, dvsO, err)
		t.FailNow()
	}
	dvpg := buildTestDVPG("test2", &dvsO)
	doCreateDVPortgroup(dvpg, t)
	//doDeleteDVPortgroup(dvpg, t)
}

func buildTestMapVMDVPG(dvpg *dvs_port_group) *dvs_map_vm_dvpg {
	vmpth := vmPath(testParameters["vmFolder"].(string), testParameters["vmPath"].(string))
	o := dvs_map_vm_dvpg{}
	o.vm = vmpth
	o.nicLabel = testParameters["nicName"].(string)
	o.portgroup = dvpg.getID()
	return &o
}

func doCreateMapVMDVPG(mapvm *dvs_map_vm_dvpg, t *testing.T) {
	t.Logf("Creating MapVMDVPG: %+v", mapvm)
	if err := mapvm.createMapVMDVPG(client); err != nil {
		t.Logf("Could not create MapVMDVPG:\n%+v", err)
		t.Fail()
	} else {
		t.Log("Could create MapVMDVPG")
	}
}

func doDeleteMapVMDVPG(mapvm *dvs_map_vm_dvpg, t *testing.T) {
	t.Logf("Deleting MapVMDVPG: %+v", mapvm)
	if err := mapvm.deleteMapVMDVPG(client); err != nil {
		t.Logf("Could not delete MapVMDVPG:\n%+v", err)
		t.Fail()
	} else {
		t.Log("Could create MapVMDVPG")
	}
}

// Test VM-DVS binding creation and destruction
func aaTestVMDVSCreationAndDestruction(t *testing.T) {
	// need:
	// datacenter name, switch path, portgroup name, VM path name
	//dvsO := buildTestDVS("test3")
	dvsO := dvs{}
	dvsPath := "fillme"
	if err := loadDVS(client, testParameters["datacenter"].(string), dvsPath, &dvsO); err != nil {
		t.Logf("Could not load DVS with client %+v %+v %+v %+v: %+v\n", client, testParameters["datacenter"], dvsPath, dvsO, err)
		t.FailNow()
	}
	dvpg := buildTestDVPG("test3", &dvsO)
	mapvmdvpg := buildTestMapVMDVPG(dvpg)

	//doCreateDVS(dvsO, t)
	//doCreateDVPortgroup(dvpg, t)
	doCreateMapVMDVPG(mapvmdvpg, t)
	doDeleteMapVMDVPG(mapvmdvpg, t)
	//doDeleteDVPortgroup(dvpg, t)
}

// Test read DVS
func aaTestDVSRead(t *testing.T) {
	// need:
	// datacenter name, switch path
}

// Test read Portgroup
func aaTestPortgroupRead(t *testing.T) {
	// need:
	// datacenter name, switch path, portgroup name
}

// Test read VM-DVS binding
func aaTestVMDVSRead(t *testing.T) {
	// need:
	// datacenter name, switch path, portgroup name, VM path name
}

func init() {
	datacenter := os.Getenv("VSPHERE_TEST_DC")
	switchFolder := os.Getenv("VSPHERE_TEST_SWDIR")
	vmFolder := os.Getenv("VSPHERE_TEST_VMDIR")
	vmPath := os.Getenv("VSPHERE_TEST_VM")
	nicName := os.Getenv("VSPHERE_TEST_NIC")
	if datacenter == "" {
		datacenter = "vm-test-1"
	}
	if switchFolder == "" {
		switchFolder = "/DEVTESTS"
	}
	if vmFolder == "" {
		vmFolder = "/DEVTESTS"
	}
	if nicName == "" {
		nicName = "Network Adapter 1"
	}
	if vmPath == "" {
		vmPath = "TESTVM"
	}
	testParameters = make(map[string]interface{})
	testParameters["datacenter"] = datacenter
	testParameters["switchFolder"] = switchFolder
	testParameters["vmFolder"] = vmFolder
	testParameters["nicName"] = nicName
	testParameters["vmPath"] = vmPath
	testParameters["switchName"] = "DVSTEST-%s"
	testParameters["portgroupName"] = "PORTGROUPTEST1-%s"
	testParameters["switchDescription"] = "lorem test ipsum test"
	testParameters["portgroupDescription"] = "doler test sit amet test"
	testParameters["numStandalonePorts"] = 4
	testParameters["portgroupPorts"] = 16
	testParameters["contactInfos"] = "lorem test <test@example.invalid>"
	testParameters["contactName"] = "Lorem Test Ipsum Invalid"
	testParameters["switchIPAddress"] = "192.0.2.1"
	testParameters["dvpgAutoExpand"] = true
}
