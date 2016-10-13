package vsphere

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"testing"

	"path/filepath"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"golang.org/x/net/context"
)

///////
// Various ENV vars are used to setup these tests. Look for `os.Getenv`
///////

// Base setup function to check that a template, and nic information is set
// TODO needs some TLC - determine exactly how we want to do this
func testBasicPreCheck(t *testing.T) {

	testAccPreCheck(t)

	if v := os.Getenv("VSPHERE_TEMPLATE"); v == "" {
		t.Fatal("env variable VSPHERE_TEMPLATE must be set for acceptance tests")
	}

	if v := os.Getenv("VSPHERE_IPV4_GATEWAY"); v == "" {
		t.Fatal("env variable VSPHERE_IPV4_GATEWAY must be set for acceptance tests")
	}

	if v := os.Getenv("VSPHERE_IPV4_ADDRESS"); v == "" {
		t.Fatal("env variable VSPHERE_IPV4_ADDRESS must be set for acceptance tests")
	}

	if v := os.Getenv("VSPHERE_NETWORK_LABEL"); v == "" {
		t.Fatal("env variable VSPHERE_NETWORK_LABEL must be set for acceptance tests")
	}
}

////
// Collects optional env vars used in the tests
////
func setupBaseVars() (string, string) {
	var locationOpt string
	var datastoreOpt string

	if v := os.Getenv("VSPHERE_DATACENTER"); v != "" {
		locationOpt += fmt.Sprintf("    datacenter = \"%s\"\n", v)
	}
	if v := os.Getenv("VSPHERE_CLUSTER"); v != "" {
		locationOpt += fmt.Sprintf("    cluster = \"%s\"\n", v)
	}
	if v := os.Getenv("VSPHERE_RESOURCE_POOL"); v != "" {
		locationOpt += fmt.Sprintf("    resource_pool = \"%s\"\n", v)
	}
	if v := os.Getenv("VSPHERE_DATASTORE"); v != "" {
		datastoreOpt = fmt.Sprintf("        datastore = \"%s\"\n", v)
	}

	return locationOpt, datastoreOpt
}

////
// Structs and funcs used with DHCP data template
////
type TestDHCPBodyData struct {
	template     string
	locationOpt  string
	datastoreOpt string
	label        string
}

func (body TestDHCPBodyData) parseDHCPTemplateConfigWithTemplate(template string) string {
	return fmt.Sprintf(
		template,
		body.locationOpt,
		body.label,
		body.datastoreOpt,
		body.template,
	)

}

const testAccCheckVSphereTemplate_dhcp = `
%s
  vcpu = 2
  memory = 1024
  network_interface {
    label = "%s"
  }
  disk {
%s
    template = "%s"
  }
}
`

// replaces data in the above template
func (body TestDHCPBodyData) parseDHCPTemplateConfig() string {
	return fmt.Sprintf(
		testAccCheckVSphereTemplate_dhcp,
		body.locationOpt,
		body.label,
		body.datastoreOpt,
		body.template,
	)
}

func (body TestDHCPBodyData) testSprintfDHCPTemplateBodySecondArgDynamic(template string, arg string) string {
	return fmt.Sprintf(
		template,
		body.locationOpt,
		arg,
		body.label,
		body.datastoreOpt,
		body.template,
	)
}

// returns variables that are used in DHCP tests
func setupTemplateFuncDHCPData() TestDHCPBodyData {

	locationOpt, datastoreOpt := setupBaseVars()
	data := TestDHCPBodyData{
		template:     os.Getenv("VSPHERE_TEMPLATE"),
		label:        os.Getenv("VSPHERE_NETWORK_LABEL_DHCP"),
		locationOpt:  locationOpt,
		datastoreOpt: datastoreOpt,
	}
	// log.Printf("[DEBUG] basic vars= %v", data)
	return data

}

////
// Structs and funcs used with static ip data templates
////
type TemplateBasicBodyVars struct {
	locationOpt   string
	label         string
	ipv4IpAddress string
	ipv4Prefix    string
	ipv4Gateway   string
	datastoreOpt  string
	template      string
}

// Takes a base template that has seven "%s" values in it, used by most fixed ip
// tests
func (body TemplateBasicBodyVars) testSprintfTemplateBody(template string) string {

	return fmt.Sprintf(
		template,
		body.locationOpt,
		body.label,
		body.ipv4IpAddress,
		body.ipv4Prefix,
		body.ipv4Gateway,
		body.datastoreOpt,
		body.template,
	)
}

// setups variables used by fixed ip tests
func setupTemplateBasicBodyVars() TemplateBasicBodyVars {

	locationOpt, datastoreOpt := setupBaseVars()
	prefix := os.Getenv("VSPHERE_IPV4_PREFIX")
	if prefix == "" {
		prefix = "24"
	}
	data := TemplateBasicBodyVars{
		template:      os.Getenv("VSPHERE_TEMPLATE"),
		ipv4Gateway:   os.Getenv("VSPHERE_IPV4_GATEWAY"),
		label:         os.Getenv("VSPHERE_NETWORK_LABEL"),
		ipv4IpAddress: os.Getenv("VSPHERE_IPV4_ADDRESS"),
		ipv4Prefix:    prefix,
		locationOpt:   locationOpt,
		datastoreOpt:  datastoreOpt,
	}
	// log.Printf("[DEBUG] basic vars= %v", data)
	return data
}

////
// Basic data to create series of testing functions
////
type TestFuncData struct {
	vm         virtualMachine
	label      string
	vmName     string
	vmResource string
	numDisks   string
	numCPU     string
	mem        string
}

// returns TestCheckFunc's that are used in many of our tests
// mem defaults to 1024
// cpu defaults to 2
// disks defatuls to 1
// vmResource defaults to "terraform-test"
// vmName defaults to "vsphere_virtual_machine.foo
func (test TestFuncData) testCheckFuncBasic() (
	resource.TestCheckFunc, resource.TestCheckFunc, resource.TestCheckFunc, resource.TestCheckFunc,
	resource.TestCheckFunc, resource.TestCheckFunc, resource.TestCheckFunc, resource.TestCheckFunc) {
	//log.Printf("[DEBUG] data= %v", test)
	mem := test.mem
	if mem == "" {
		mem = "1024"
	}
	cpu := test.numCPU
	if cpu == "" {
		cpu = "2"
	}
	disks := test.numDisks
	if disks == "" {
		disks = "1"
	}
	res := test.vmResource
	if res == "" {
		res = "terraform-test"
	}
	vmName := test.vmName
	if vmName == "" {
		vmName = "vsphere_virtual_machine.foo"
	}
	return testAccCheckVSphereVirtualMachineExists(vmName, &test.vm),
		resource.TestCheckResourceAttr(vmName, "name", res),
		resource.TestCheckResourceAttr(vmName, "vcpu", cpu),
		resource.TestMatchResourceAttr(vmName, "uuid", regexp.MustCompile("[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}")),
		resource.TestCheckResourceAttr(vmName, "memory", mem),
		resource.TestCheckResourceAttr(vmName, "disk.#", disks),
		resource.TestCheckResourceAttr(vmName, "network_interface.#", "1"),
		resource.TestCheckResourceAttr(vmName, "network_interface.0.label", test.label)
}

const testAccCheckVSphereVirtualMachineConfig_really_basic = `
resource "vsphere_virtual_machine" "foo" {
    name = "terraform-test"
` + testAccTemplateBasicBodyWithEnd

// WARNING this is one of the base templates.  You change this and you will
// be impacting multiple tests
const testAccTemplateBasicBody = `
%s
    vcpu = 2
    memory = 1024
    network_interface {
        label = "%s"
        ipv4_address = "%s"
        ipv4_prefix_length = %s
        ipv4_gateway = "%s"
    }
     disk {
%s
        template = "%s"
        iops = 500
    }
`
const testAccTemplateBasicBodyWithEnd = testAccTemplateBasicBody + `
}`

func TestAccVSphereVirtualMachine_basic(t *testing.T) {
	var vm virtualMachine
	basic_vars := setupTemplateBasicBodyVars()
	config := basic_vars.testSprintfTemplateBody(testAccCheckVSphereVirtualMachineConfig_really_basic)

	log.Printf("[DEBUG] template= %s", testAccCheckVSphereVirtualMachineConfig_really_basic)
	log.Printf("[DEBUG] template config= %s", config)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testBasicPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereVirtualMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					TestFuncData{vm: vm, label: basic_vars.label}.testCheckFuncBasic(),
				),
			},
		},
	})
}

const testAccCheckVSphereVirtualMachineConfig_debug = `
provider "vsphere" {
  client_debug = true
}

` + testAccCheckVSphereVirtualMachineConfig_really_basic

func TestAccVSphereVirtualMachine_client_debug(t *testing.T) {
	var vm virtualMachine
	basic_vars := setupTemplateBasicBodyVars()
	config := basic_vars.testSprintfTemplateBody(testAccCheckVSphereVirtualMachineConfig_debug)

	log.Printf("[DEBUG] template= %s", testAccCheckVSphereVirtualMachineConfig_debug)
	log.Printf("[DEBUG] template config= %s", config)

	test_exists, test_name, test_cpu, test_uuid, test_mem, test_num_disk, test_num_of_nic, test_nic_label :=
		TestFuncData{vm: vm, label: basic_vars.label}.testCheckFuncBasic()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testBasicPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereVirtualMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					test_exists, test_name, test_cpu, test_uuid, test_mem, test_num_disk, test_num_of_nic, test_nic_label,
					testAccCheckDebugExists(),
				),
			},
		},
	})
}

const testAccCheckVSphereVirtualMachineConfig_diskSCSICapacity = `
resource "vsphere_virtual_machine" "scsiCapacity" {
    name = "terraform-test"
` + testAccTemplateBasicBody + `
    disk {
        size = 1
        controller_type = "scsi-paravirtual"
        name = "one"
    }
    disk {
        size = 1
        controller_type = "scsi-paravirtual"
        name = "two"
    }
	disk {
        size = 1
        controller_type = "scsi-paravirtual"
        name = "three"
    }
	disk {
        size = 1
        controller_type = "scsi-paravirtual"
        name = "four"
    }
	disk {
        size = 1
        controller_type = "scsi-paravirtual"
        name = "five"
    }
	disk {
        size = 1
        controller_type = "scsi-paravirtual"
        name = "six"
    }
	disk {
        size = 1
        controller_type = "scsi-paravirtual"
        name = "seven"
    }
}
`

func TestAccVSphereVirtualMachine_diskSCSICapacity(t *testing.T) {
	var vm virtualMachine
	basic_vars := setupTemplateBasicBodyVars()
	config := basic_vars.testSprintfTemplateBody(testAccCheckVSphereVirtualMachineConfig_diskSCSICapacity)

	vmName := "vsphere_virtual_machine.scsiCapacity"

	test_exists, test_name, test_cpu, test_uuid, test_mem, test_num_disk, test_num_of_nic, test_nic_label :=
		TestFuncData{vm: vm, label: basic_vars.label, vmName: vmName, numDisks: "8"}.testCheckFuncBasic()

	log.Printf("[DEBUG] template= %s", testAccCheckVSphereVirtualMachineConfig_diskSCSICapacity)
	log.Printf("[DEBUG] template config= %s", config)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereVirtualMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					test_exists, test_name, test_cpu, test_uuid, test_mem, test_num_disk, test_num_of_nic, test_nic_label,
				),
			},
		},
	})
}

const testAccCheckVSphereVirtualMachineConfig_initTypeEager = `
resource "vsphere_virtual_machine" "thickEagerZero" {
    name = "terraform-test"
` + testAccTemplateBasicBody + `
    disk {
		size = 1
		iops = 500
		controller_type = "scsi"
		name = "one"
    }
    disk {
		size = 1
		controller_type = "ide"
		type = "eager_zeroed"
		name = "two"
    }
}
`

func TestAccVSphereVirtualMachine_diskInitTypeEager(t *testing.T) {
	var vm virtualMachine
	basic_vars := setupTemplateBasicBodyVars()
	config := basic_vars.testSprintfTemplateBody(testAccCheckVSphereVirtualMachineConfig_initTypeEager)

	vmName := "vsphere_virtual_machine.thickEagerZero"

	test_exists, test_name, test_cpu, test_uuid, test_mem, test_num_disk, test_num_of_nic, test_nic_label :=
		TestFuncData{vm: vm, label: basic_vars.label, vmName: vmName, numDisks: "3"}.testCheckFuncBasic()

	log.Printf("[DEBUG] template= %s", testAccCheckVSphereVirtualMachineConfig_initTypeEager)
	log.Printf("[DEBUG] template config= %s", config)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereVirtualMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					test_exists, test_name, test_cpu, test_uuid, test_mem, test_num_disk, test_num_of_nic, test_nic_label,
					// FIXME dynmically calculate the hashes
					resource.TestCheckResourceAttr(vmName, "disk.294918912.type", "eager_zeroed"),
					resource.TestCheckResourceAttr(vmName, "disk.294918912.controller_type", "ide"),
					resource.TestCheckResourceAttr(vmName, "disk.1380467090.controller_type", "scsi"),
				),
			},
		},
	})
}

const testAccCheckVSphereVirtualMachineConfig_initTypeLazy = `
resource "vsphere_virtual_machine" "lazy" {
    name = "terraform-test"
` + testAccTemplateBasicBody + `
    disk {
		size = 1
		iops = 500
		controller_type = "scsi"
		name = "one"
    }
    disk {
		size = 1
		controller_type = "ide"
		type = "lazy"
		name = "two"
    }
}
`

func TestAccVSphereVirtualMachine_diskInitTypeLazy(t *testing.T) {
	var vm virtualMachine
	basic_vars := setupTemplateBasicBodyVars()
	config := basic_vars.testSprintfTemplateBody(testAccCheckVSphereVirtualMachineConfig_initTypeLazy)

	vmName := "vsphere_virtual_machine.lazy"

	test_exists, test_name, test_cpu, test_uuid, test_mem, test_num_disk, test_num_of_nic, test_nic_label :=
		TestFuncData{vm: vm, label: basic_vars.label, vmName: vmName, numDisks: "3"}.testCheckFuncBasic()

	log.Printf("[DEBUG] template= %s", testAccCheckVSphereVirtualMachineConfig_initTypeLazy)
	log.Printf("[DEBUG] template config= %s", config)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereVirtualMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					test_exists, test_name, test_cpu, test_uuid, test_mem, test_num_disk, test_num_of_nic, test_nic_label,
					// FIXME dynmically calculate the hashes
					resource.TestCheckResourceAttr(vmName, "disk.692719290.type", "lazy"),
					resource.TestCheckResourceAttr(vmName, "disk.692719290.controller_type", "ide"),
					resource.TestCheckResourceAttr(vmName, "disk.531766495.controller_type", "scsi"),
				),
			},
		},
	})
}

const testAccCheckVSphereVirtualMachineConfig_dhcp = `
resource "vsphere_virtual_machine" "bar" {
    name = "terraform-test"
`

func TestAccVSphereVirtualMachine_dhcp(t *testing.T) {
	var vm virtualMachine
	data := setupTemplateFuncDHCPData()
	config := testAccCheckVSphereVirtualMachineConfig_dhcp + data.parseDHCPTemplateConfigWithTemplate(testAccCheckVSphereTemplate_dhcp)
	log.Printf("[DEBUG] template= %s", testAccCheckVSphereVirtualMachineConfig_dhcp+testAccCheckVSphereTemplate_dhcp)
	log.Printf("[DEBUG] config= %s", config)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereVirtualMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					TestFuncData{vm: vm, label: data.label, vmName: "vsphere_virtual_machine.bar"}.testCheckFuncBasic(),
				),
			},
		},
	})
}

const testAccCheckVSphereVirtualMachineConfig_custom_configs = `
resource "vsphere_virtual_machine" "car" {
    name = "terraform-test-custom"
    custom_configuration_parameters {
      "foo" = "bar"
      "car" = "ferrari"
      "num" = 42
    }
	enable_disk_uuid = true
`

func TestAccVSphereVirtualMachine_custom_configs(t *testing.T) {

	var vm virtualMachine
	data := setupTemplateFuncDHCPData()
	config := testAccCheckVSphereVirtualMachineConfig_custom_configs + data.parseDHCPTemplateConfigWithTemplate(testAccCheckVSphereTemplate_dhcp)
	vmName := "vsphere_virtual_machine.car"
	res := "terraform-test-custom"

	test_exists, test_name, test_cpu, test_uuid, test_mem, test_num_disk, test_num_of_nic, test_nic_label :=
		TestFuncData{vm: vm, label: data.label, vmName: vmName, vmResource: res}.testCheckFuncBasic()

	log.Printf("[DEBUG] template= %s", testAccCheckVSphereVirtualMachineConfig_custom_configs+testAccCheckVSphereTemplate_dhcp)
	log.Printf("[DEBUG] config= %s", config)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereVirtualMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					test_exists, test_name, test_cpu, test_uuid, test_mem, test_num_disk, test_num_of_nic, test_nic_label,
					testAccCheckVSphereVirtualMachineExistsHasCustomConfig(vmName, &vm),
					resource.TestCheckResourceAttr(vmName, "custom_configuration_parameters.foo", "bar"),
					resource.TestCheckResourceAttr(vmName, "custom_configuration_parameters.car", "ferrari"),
					resource.TestCheckResourceAttr(vmName, "custom_configuration_parameters.num", "42"),
					resource.TestCheckResourceAttr(vmName, "enable_disk_uuid", "true"),
				),
			},
		},
	})
}

const testAccCheckVSphereVirtualMachineConfig_createInFolder = `
resource "vsphere_virtual_machine" "folder" {
    name = "terraform-test-folder"
    folder = "%s"
`

func TestAccVSphereVirtualMachine_createInExistingFolder(t *testing.T) {
	var vm virtualMachine
	datacenter := os.Getenv("VSPHERE_DATACENTER")

	folder := "tf_test_cpureateInExistingFolder"

	data := setupTemplateFuncDHCPData()
	config := fmt.Sprintf(testAccCheckVSphereVirtualMachineConfig_createInFolder,
		folder,
	) + data.parseDHCPTemplateConfig()

	log.Printf("[DEBUG] template= %s", testAccCheckVSphereVirtualMachineConfig_createInFolder)
	log.Printf("[DEBUG] template config= %s", config)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckVSphereVirtualMachineDestroy,
			removeVSphereFolder(datacenter, folder, ""),
		),
		Steps: []resource.TestStep{
			resource.TestStep{
				PreConfig: func() { createVSphereFolder(datacenter, folder) },
				Config:    config,
				Check: resource.ComposeTestCheckFunc(
					TestFuncData{vm: vm, label: data.label, vmName: "vsphere_virtual_machine.folder", vmResource: "terraform-test-folder"}.testCheckFuncBasic(),
				),
			},
		},
	})
}

const testAccCheckVSphereVirtualMachineConfig_createWithFolder = `
resource "vsphere_folder" "with_folder" {
	path = "%s"
%s
}
resource "vsphere_virtual_machine" "with_folder" {
    name = "terraform-test-with-folder"
    folder = "${vsphere_folder.with_folder.path}"
`

func TestAccVSphereVirtualMachine_createWithFolder(t *testing.T) {
	var vm virtualMachine
	var folderLocationOpt string
	var f folder

	if v := os.Getenv("VSPHERE_DATACENTER"); v != "" {
		folderLocationOpt = fmt.Sprintf("    datacenter = \"%s\"\n", v)
	}

	folder := "tf_test_cpureateWithFolder"

	data := setupTemplateFuncDHCPData()
	vmName := "vsphere_virtual_machine.with_folder"
	test_exists, test_name, test_cpu, test_uuid, test_mem, test_num_disk, test_num_of_nic, test_nic_label :=
		TestFuncData{vm: vm, label: data.label, vmName: vmName, vmResource: "terraform-test-with-folder"}.testCheckFuncBasic()

	config := fmt.Sprintf(testAccCheckVSphereVirtualMachineConfig_createWithFolder,
		folder,
		folderLocationOpt,
	) + data.parseDHCPTemplateConfig()

	log.Printf("[DEBUG] template= %s", testAccCheckVSphereVirtualMachineConfig_createWithFolder+testAccCheckVSphereTemplate_dhcp)
	log.Printf("[DEBUG] template config= %s", config)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckVSphereVirtualMachineDestroy,
			testAccCheckVSphereFolderDestroy,
		),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					test_exists, test_name, test_cpu, test_uuid, test_mem, test_num_disk, test_num_of_nic, test_nic_label,
					testAccCheckVSphereFolderExists(vmName, &f),
					resource.TestCheckResourceAttr(vmName, "folder", folder),
				),
			},
		},
	})
}

const testAccCheckVsphereVirtualMachineConfig_cdrom = `
resource "vsphere_virtual_machine" "with_cdrom" {
    name = "terraform-test-with-cdrom"
    cdrom {
        datastore = "%s"
        path = "%s"
    }
`

func TestAccVSphereVirtualMachine_createWithCdrom(t *testing.T) {
	var vm virtualMachine

	// FIXME check that these exist
	cdromDatastore := os.Getenv("VSPHERE_CDROM_DATASTORE")
	cdromPath := os.Getenv("VSPHERE_CDROM_PATH")
	vmName := "vsphere_virtual_machine.with_cdrom"

	data := setupTemplateFuncDHCPData()
	test_exists, test_name, test_cpu, test_uuid, test_mem, test_num_disk, test_num_of_nic, test_nic_label :=
		TestFuncData{vm: vm, label: data.label, vmName: vmName, vmResource: "terraform-test-with-cdrom"}.testCheckFuncBasic()

	config := fmt.Sprintf(
		testAccCheckVsphereVirtualMachineConfig_cdrom,
		cdromDatastore,
		cdromPath,
	) + data.parseDHCPTemplateConfig()

	log.Printf("[DEBUG] template= %s", testAccCheckVsphereVirtualMachineConfig_cdrom)
	log.Printf("[DEBUG] template config= %s", config)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereVirtualMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					test_exists, test_name, test_cpu, test_uuid, test_mem, test_num_disk, test_num_of_nic, test_nic_label,
					//resource.TestCheckResourceAttr(
					//	"vsphere_virtual_machine.with_cdrom", "disk.4088143748.template", template),
					resource.TestCheckResourceAttr(vmName, "cdrom.#", "1"),
					resource.TestCheckResourceAttr(vmName, "cdrom.0.datastore", cdromDatastore),
					resource.TestCheckResourceAttr(vmName, "cdrom.0.path", cdromPath),
				),
			},
		},
	})
}

const testAccCheckVSphereVirtualMachineConfig_withExistingVmdk = `
resource "vsphere_virtual_machine" "with_existing_vmdk" {
    name = "terraform-test-with-existing-vmdk"
%s
    vcpu = 2
    memory = 1024
    network_interface {
        label = "%s"
    }
    disk {
%s
        vmdk = "%s"
		bootable = true
    }
    disk {
        size = 1
        iops = 500
		name = "one"
    }
}
`

func TestAccVSphereVirtualMachine_createWithExistingVmdk(t *testing.T) {
	var vm virtualMachine
	vmdk_path := os.Getenv("VSPHERE_VMDK_PATH")

	data := setupTemplateFuncDHCPData()
	config := fmt.Sprintf(
		testAccCheckVSphereVirtualMachineConfig_withExistingVmdk,
		data.locationOpt,
		data.label,
		data.datastoreOpt,
		vmdk_path,
	)
	log.Printf("[DEBUG] template= %s", testAccCheckVSphereVirtualMachineConfig_withExistingVmdk)
	log.Printf("[DEBUG] template config= %s", config)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereVirtualMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					TestFuncData{vm: vm, label: data.label, vmName: "vsphere_virtual_machine.with_existing_vmdk",
						vmResource: "terraform-test-with-existing-vmdk", numDisks: "2"}.testCheckFuncBasic(),
					//resource.TestCheckResourceAttr(
					//	"vsphere_virtual_machine.with_existing_vmdk", "disk.2393891804.vmdk", vmdk_path),
					//resource.TestCheckResourceAttr(
					//	"vsphere_virtual_machine.with_existing_vmdk", "disk.2393891804.bootable", "true"),
				),
			},
		},
	})
}

const testAccCheckVSphereVirtualMachineConfig_updateMemory = `
resource "vsphere_virtual_machine" "bar" {
    name = "terraform-test"
%s
    vcpu = 2
    memory = %s
    network_interface {
        label = "%s"
    }
    disk {
%s
      template = "%s"
    }
}
`

func TestAccVSphereVirtualMachine_updateMemory(t *testing.T) {
	var vm virtualMachine
	data := setupTemplateFuncDHCPData()

	log.Printf("[DEBUG] template= %s", testAccCheckVSphereVirtualMachineConfig_updateMemory)

	config := data.testSprintfDHCPTemplateBodySecondArgDynamic(testAccCheckVSphereVirtualMachineConfig_updateMemory, "1024")
	log.Printf("[DEBUG] template config= %s", config)

	configUpdate := data.testSprintfDHCPTemplateBodySecondArgDynamic(testAccCheckVSphereVirtualMachineConfig_updateMemory, "2048")
	log.Printf("[DEBUG] template configUpdate= %s", configUpdate)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereVirtualMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					TestFuncData{vm: vm, label: data.label, vmName: "vsphere_virtual_machine.bar"}.testCheckFuncBasic(),
				),
			},
			resource.TestStep{
				Config: configUpdate,
				Check: resource.ComposeTestCheckFunc(
					TestFuncData{vm: vm, label: data.label, mem: "2048", vmName: "vsphere_virtual_machine.bar"}.testCheckFuncBasic(),
				),
			},
		},
	})
}

const testAccCheckVSphereVirtualMachineConfig_updateVcpu = `
resource "vsphere_virtual_machine" "bar" {
    name = "terraform-test"
%s
    vcpu = %s
    memory = 1024
    network_interface {
        label = "%s"
    }
    disk {
%s
        template = "%s"
    }
}
`

func TestAccVSphereVirtualMachine_updateVcpu(t *testing.T) {
	var vm virtualMachine
	data := setupTemplateFuncDHCPData()
	log.Printf("[DEBUG] template= %s", testAccCheckVSphereVirtualMachineConfig_updateVcpu)

	config := data.testSprintfDHCPTemplateBodySecondArgDynamic(testAccCheckVSphereVirtualMachineConfig_updateVcpu, "2")
	log.Printf("[DEBUG] template config= %s", config)

	configUpdate := data.testSprintfDHCPTemplateBodySecondArgDynamic(testAccCheckVSphereVirtualMachineConfig_updateVcpu, "4")
	log.Printf("[DEBUG] template configUpdate= %s", configUpdate)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereVirtualMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					TestFuncData{vm: vm, label: data.label, vmName: "vsphere_virtual_machine.bar"}.testCheckFuncBasic(),
				),
			},
			resource.TestStep{
				Config: configUpdate,
				Check: resource.ComposeTestCheckFunc(
					TestFuncData{vm: vm, label: data.label, vmName: "vsphere_virtual_machine.bar", numCPU: "4"}.testCheckFuncBasic(),
				),
			},
		},
	})
}

const testAccCheckVSphereVirtualMachineConfig_ipv6 = `
resource "vsphere_virtual_machine" "ipv6" {
    name = "terraform-test-ipv6"
%s
    vcpu = 2
    memory = 1024
    network_interface {
        label = "%s"
        %s
        ipv6_address = "%s"
        ipv6_prefix_length = 64
        ipv6_gateway = "%s"
    }
    disk {
%s
        template = "%s"
        iops = 500
    }
    disk {
        size = 1
        iops = 500
		name = "one"
    }
}
`

func TestAccVSphereVirtualMachine_ipv4Andipv6(t *testing.T) {
	var vm virtualMachine
	data := setupTemplateBasicBodyVars()
	log.Printf("[DEBUG] template= %s", testAccCheckVSphereVirtualMachineConfig_ipv6)

	vmName := "vsphere_virtual_machine.ipv6"

	test_exists, test_name, test_cpu, test_uuid, test_mem, test_num_disk, test_num_of_nic, test_nic_label :=
		TestFuncData{vm: vm, label: data.label, vmName: vmName, numDisks: "2", vmResource: "terraform-test-ipv6"}.testCheckFuncBasic()

	// FIXME test for this or warn??
	ipv6Address := os.Getenv("VSPHERE_IPV6_ADDRESS")
	ipv6Gateway := os.Getenv("VSPHERE_IPV6_GATEWAY")

	ipv4Settings := fmt.Sprintf(`
		ipv4_address = "%s"
        ipv4_prefix_length = %s
        ipv4_gateway = "%s"
	`, data.ipv4IpAddress, data.ipv4Prefix, data.ipv4Gateway)

	config := fmt.Sprintf(
		testAccCheckVSphereVirtualMachineConfig_ipv6,
		data.locationOpt,
		data.label,
		ipv4Settings,
		ipv6Address,
		ipv6Gateway,
		data.datastoreOpt,
		data.template,
	)

	log.Printf("[DEBUG] template config= %s", config)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereVirtualMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					test_exists, test_name, test_cpu, test_uuid, test_mem, test_num_disk, test_num_of_nic, test_nic_label,
					resource.TestCheckResourceAttr(vmName, "network_interface.0.ipv4_address", data.ipv4IpAddress),
					resource.TestCheckResourceAttr(vmName, "network_interface.0.ipv4_gateway", data.ipv4Gateway),
					resource.TestCheckResourceAttr(vmName, "network_interface.0.ipv6_address", ipv6Address),
					resource.TestCheckResourceAttr(vmName, "network_interface.0.ipv6_gateway", ipv6Gateway),
				),
			},
		},
	})
}

func TestAccVSphereVirtualMachine_ipv6Only(t *testing.T) {
	var vm virtualMachine
	data := setupTemplateBasicBodyVars()
	log.Printf("[DEBUG] template= %s", testAccCheckVSphereVirtualMachineConfig_ipv6)

	vmName := "vsphere_virtual_machine.ipv6"

	test_exists, test_name, test_cpu, test_uuid, test_mem, test_num_disk, test_num_of_nic, test_nic_label :=
		TestFuncData{vm: vm, label: data.label, vmName: vmName, numDisks: "2", vmResource: "terraform-test-ipv6"}.testCheckFuncBasic()

	// Checks for this will be handled when this code is merged with https://github.com/hashicorp/terraform/pull/7575.
	ipv6Address := os.Getenv("VSPHERE_IPV6_ADDRESS")
	ipv6Gateway := os.Getenv("VSPHERE_IPV6_GATEWAY")

	config := fmt.Sprintf(
		testAccCheckVSphereVirtualMachineConfig_ipv6,
		data.locationOpt,
		data.label,
		"",
		ipv6Address,
		ipv6Gateway,
		data.datastoreOpt,
		data.template,
	)

	log.Printf("[DEBUG] template config= %s", config)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereVirtualMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					test_exists, test_name, test_cpu, test_uuid, test_mem, test_num_disk, test_num_of_nic, test_nic_label,
					resource.TestCheckResourceAttr(vmName, "network_interface.0.ipv6_address", ipv6Address),
					resource.TestCheckResourceAttr(vmName, "network_interface.0.ipv6_gateway", ipv6Gateway),
				),
			},
		},
	})
}

const testAccCheckVSphereVirtualMachineConfig_updateAddDisks = `
resource "vsphere_virtual_machine" "foo" {
    name = "terraform-test"
` + testAccTemplateBasicBody + `
    disk {
        size = 1
        iops = 500
        name = "one"
%s
    }
	disk {
        size = 1
        iops = 500
        name = "two"
%s
    }
	disk {
        size = 1
        iops = 500
        name = "three"
%s
    }
}
`
const testAccCheckVSphereVirtualMachineConfig_basic = `
resource "vsphere_virtual_machine" "foo" {
    name = "terraform-test"
` + testAccTemplateBasicBody + `
    disk {
        size = 1
        iops = 500
	name = "one"
    }
}
`

func TestAccVSphereVirtualMachine_updateDisks(t *testing.T) {
	var vm virtualMachine
	basic_vars := setupTemplateBasicBodyVars()
	config_basic := basic_vars.testSprintfTemplateBody(testAccCheckVSphereVirtualMachineConfig_basic)

	log.Printf("[DEBUG] template= %s", testAccCheckVSphereVirtualMachineConfig_basic)
	log.Printf("[DEBUG] template config= %s", config_basic)

	config_add := fmt.Sprintf(
		testAccCheckVSphereVirtualMachineConfig_updateAddDisks,
		basic_vars.locationOpt,
		basic_vars.label,
		basic_vars.ipv4IpAddress,
		basic_vars.ipv4Prefix,
		basic_vars.ipv4Gateway,
		basic_vars.datastoreOpt,
		basic_vars.template,
		basic_vars.datastoreOpt,
		basic_vars.datastoreOpt,
		basic_vars.datastoreOpt,
	)

	log.Printf("[DEBUG] template= %s", testAccCheckVSphereVirtualMachineConfig_basic)
	log.Printf("[DEBUG] template config= %s", config_add)

	config_del := basic_vars.testSprintfTemplateBody(testAccCheckVSphereVirtualMachineConfig_really_basic)

	log.Printf("[DEBUG] template= %s", testAccCheckVSphereVirtualMachineConfig_really_basic)
	log.Printf("[DEBUG] template config= %s", config_del)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereVirtualMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config_basic,
				Check: resource.ComposeTestCheckFunc(
					TestFuncData{vm: vm, label: basic_vars.label, numDisks: "2"}.testCheckFuncBasic(),
				),
			},
			resource.TestStep{
				Config: config_add,
				Check: resource.ComposeTestCheckFunc(
					TestFuncData{vm: vm, label: basic_vars.label, numDisks: "4"}.testCheckFuncBasic(),
				),
			},
			resource.TestStep{
				Config: config_del,
				Check: resource.ComposeTestCheckFunc(
					TestFuncData{vm: vm, label: basic_vars.label, numDisks: "1"}.testCheckFuncBasic(),
				),
			},
		},
	})
}

const testAccCheckVSphereVirtualMachineConfig_mac_address = `
resource "vsphere_virtual_machine" "mac_address" {
    name = "terraform-mac-address"
%s
    vcpu = 2
    memory = 1024
    network_interface {
        label = "%s"
        mac_address = "%s"
    }
    disk {
%s
        template = "%s"
    }
}
`

// VSPHERE_NETWORK_MAC_ADDRESS needs to be set to run TestAccVSphereVirtualMachine_mac_address
// use a basic NIC MAC address like 6:5c:89:2b:a0:64
func testMacPreCheck(t *testing.T) {

	testBasicPreCheck(t)

	// TODO should start do parse values to ensure they are correct
	// for instance
	//  func ParseMAC(s string) (hw HardwareAddr, err error)
	if v := os.Getenv("VSPHERE_NETWORK_MAC_ADDRESS"); v == "" {
		t.Fatal("env variable VSPHERE_NETWORK_MAC_ADDRESS must be set for this acceptance test")
	}
}

// test new mac address feature
func TestAccVSphereVirtualMachine_mac_address(t *testing.T) {
	var vm virtualMachine
	data := setupTemplateFuncDHCPData()
	vmName := "vsphere_virtual_machine.mac_address"

	macAddress := os.Getenv("VSPHERE_NETWORK_MAC_ADDRESS")

	log.Printf("[DEBUG] template= %s", testAccCheckVSphereVirtualMachineConfig_mac_address)
	config := fmt.Sprintf(
		testAccCheckVSphereVirtualMachineConfig_mac_address,
		data.locationOpt,
		data.label,
		macAddress,
		data.datastoreOpt,
		data.template,
	)
	log.Printf("[DEBUG] template config= %s", config)

	test_exists, test_name, test_cpu, test_uuid, test_mem, test_num_disk, test_num_of_nic, test_nic_label :=
		TestFuncData{vm: vm, label: data.label, vmName: vmName, numDisks: "1", vmResource: "terraform-mac-address"}.testCheckFuncBasic()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testMacPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereVirtualMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					test_exists, test_name, test_cpu, test_uuid, test_mem, test_num_disk, test_num_of_nic, test_nic_label,
					resource.TestCheckResourceAttr(vmName, "network_interface.0.mac_address", macAddress),
				),
			},
		},
	})
}

func testAccCheckVSphereVirtualMachineDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*govmomi.Client)
	finder := find.NewFinder(client.Client, true)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "vsphere_virtual_machine" {
			continue
		}

		dc, err := finder.Datacenter(context.TODO(), rs.Primary.Attributes["datacenter"])
		if err != nil {
			return fmt.Errorf("error %s", err)
		}

		dcFolders, err := dc.Folders(context.TODO())
		if err != nil {
			return fmt.Errorf("error %s", err)
		}

		folder := dcFolders.VmFolder
		if len(rs.Primary.Attributes["folder"]) > 0 {
			si := object.NewSearchIndex(client.Client)
			folderRef, err := si.FindByInventoryPath(
				context.TODO(), fmt.Sprintf("%v/vm/%v", rs.Primary.Attributes["datacenter"], rs.Primary.Attributes["folder"]))
			if err != nil {
				return err
			} else if folderRef != nil {
				folder = folderRef.(*object.Folder)
			}
		}

		v, err := object.NewSearchIndex(client.Client).FindChild(context.TODO(), folder, rs.Primary.Attributes["name"])

		if v != nil {
			return fmt.Errorf("Record still exists")
		}
	}

	return nil
}

func testAccCheckVSphereVirtualMachineExistsHasCustomConfig(n string, vm *virtualMachine) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client := testAccProvider.Meta().(*govmomi.Client)
		finder := find.NewFinder(client.Client, true)

		dc, err := finder.Datacenter(context.TODO(), rs.Primary.Attributes["datacenter"])
		if err != nil {
			return fmt.Errorf("error %s", err)
		}

		dcFolders, err := dc.Folders(context.TODO())
		if err != nil {
			return fmt.Errorf("error %s", err)
		}

		_, err = object.NewSearchIndex(client.Client).FindChild(context.TODO(), dcFolders.VmFolder, rs.Primary.Attributes["name"])
		if err != nil {
			return fmt.Errorf("error %s", err)
		}

		finder = finder.SetDatacenter(dc)
		instance, err := finder.VirtualMachine(context.TODO(), rs.Primary.Attributes["name"])
		if err != nil {
			return fmt.Errorf("error %s", err)
		}

		var mvm mo.VirtualMachine

		collector := property.DefaultCollector(client.Client)

		if err := collector.RetrieveOne(context.TODO(), instance.Reference(), []string{"config.extraConfig"}, &mvm); err != nil {
			return fmt.Errorf("error %s", err)
		}

		var configMap = make(map[string]types.AnyType)
		if mvm.Config != nil && mvm.Config.ExtraConfig != nil && len(mvm.Config.ExtraConfig) > 0 {
			for _, v := range mvm.Config.ExtraConfig {
				value := v.GetOptionValue()
				configMap[value.Key] = value.Value
			}
		} else {
			return fmt.Errorf("error no ExtraConfig")
		}

		if configMap["foo"] == nil {
			return fmt.Errorf("error no ExtraConfig for 'foo'")
		}

		if configMap["foo"] != "bar" {
			return fmt.Errorf("error ExtraConfig 'foo' != bar")
		}

		if configMap["car"] == nil {
			return fmt.Errorf("error no ExtraConfig for 'car'")
		}

		if configMap["car"] != "ferrari" {
			return fmt.Errorf("error ExtraConfig 'car' != ferrari")
		}

		if configMap["num"] == nil {
			return fmt.Errorf("error no ExtraConfig for 'num'")
		}

		// todo this should be an int, getting back a string
		if configMap["num"] != "42" {
			return fmt.Errorf("error ExtraConfig 'num' != 42")
		}
		*vm = virtualMachine{
			name: rs.Primary.ID,
		}

		return nil
	}
}

func testAccCheckDebugExists() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if _, err := os.Stat(filepath.Join(os.Getenv("HOME"), ".govmomi")); os.IsNotExist(err) {
			return fmt.Errorf("Debug logs not found")
		}

		return nil
	}

}
func testAccCheckVSphereVirtualMachineExists(n string, vm *virtualMachine) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if n == "" {
			return fmt.Errorf("No vm name passed in")
		}
		if vm == nil {
			return fmt.Errorf("No vm obj passed in")
		}
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client := testAccProvider.Meta().(*govmomi.Client)
		finder := find.NewFinder(client.Client, true)

		dc, err := finder.Datacenter(context.TODO(), rs.Primary.Attributes["datacenter"])
		if err != nil {
			return fmt.Errorf("error %s", err)
		}

		dcFolders, err := dc.Folders(context.TODO())
		if err != nil {
			return fmt.Errorf("error %s", err)
		}

		folder := dcFolders.VmFolder
		if len(rs.Primary.Attributes["folder"]) > 0 {
			si := object.NewSearchIndex(client.Client)
			folderRef, err := si.FindByInventoryPath(
				context.TODO(), fmt.Sprintf("%v/vm/%v", rs.Primary.Attributes["datacenter"], rs.Primary.Attributes["folder"]))
			if err != nil {
				return err
			} else if folderRef != nil {
				folder = folderRef.(*object.Folder)
			}
		}

		_, err = object.NewSearchIndex(client.Client).FindChild(context.TODO(), folder, rs.Primary.Attributes["name"])

		*vm = virtualMachine{
			name: rs.Primary.ID,
		}

		return nil
	}
}

const testAccCheckVSphereVirtualMachineConfig_keepOnRemove = `
resource "vsphere_virtual_machine" "keep_disk" {
    name = "terraform-test"
` + testAccTemplateBasicBody + `
    disk {
        size = 1
        iops = 500
	controller_type = "scsi"
	name = "one"
	keep_on_remove = true
    }
}
`

func TestAccVSphereVirtualMachine_keepOnRemove(t *testing.T) {
	var vm virtualMachine
	basic_vars := setupTemplateBasicBodyVars()
	config := basic_vars.testSprintfTemplateBody(testAccCheckVSphereVirtualMachineConfig_keepOnRemove)
	var datastore string
	if v := os.Getenv("VSPHERE_DATASTORE"); v != "" {
		datastore = v
	}
	var datacenter string
	if v := os.Getenv("VSPHERE_DATACENTER"); v != "" {
		datacenter = v
	}

	vmName := "vsphere_virtual_machine.keep_disk"
	test_exists, test_name, test_cpu, test_uuid, test_mem, test_num_disk, test_num_of_nic, test_nic_label :=
		TestFuncData{vm: vm, label: basic_vars.label, vmName: vmName, numDisks: "2"}.testCheckFuncBasic()

	log.Printf("[DEBUG] template= %s", testAccCheckVSphereVirtualMachineConfig_keepOnRemove)
	log.Printf("[DEBUG] template config= %s", config)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereVirtualMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					test_exists, test_name, test_cpu, test_uuid, test_mem, test_num_disk, test_num_of_nic, test_nic_label,
				),
			},
			resource.TestStep{
				Config: " ",
				Check:  checkForDisk(datacenter, datastore, "terraform-test", "one.vmdk", true, true),
			},
		},
	})
}

const testAccVSphereVirtualMachine_DetachUnknownDisks = `
resource "vsphere_virtual_machine" "detach_unkown_disks" {
    name = "terraform-test"
` + testAccTemplateBasicBody + `
    detach_unknown_disks_on_delete = true
    disk {
        size = 1
        iops = 500
	controller_type = "scsi"
	name = "one"
	keep_on_remove = true
    }
    disk {
        size = 2
        iops = 500
	controller_type = "scsi"
	name = "two"
	keep_on_remove = false
    }
    disk {
        size = 3
        iops = 500
	controller_type = "scsi"
	name = "three"
	keep_on_remove = true
    }
}
`

func TestAccVSphereVirtualMachine_DetachUnknownDisks(t *testing.T) {
	var vm virtualMachine
	basic_vars := setupTemplateBasicBodyVars()
	config := basic_vars.testSprintfTemplateBody(testAccVSphereVirtualMachine_DetachUnknownDisks)
	var datastore string
	if v := os.Getenv("VSPHERE_DATASTORE"); v != "" {
		datastore = v
	}
	var datacenter string
	if v := os.Getenv("VSPHERE_DATACENTER"); v != "" {
		datacenter = v
	}

	vmName := "vsphere_virtual_machine.detach_unkown_disks"
	test_exists, test_name, test_cpu, test_uuid, test_mem, test_num_disk, test_num_of_nic, test_nic_label :=
		TestFuncData{vm: vm, label: basic_vars.label, vmName: vmName, numDisks: "4"}.testCheckFuncBasic()

	log.Printf("[DEBUG] template= %s", testAccVSphereVirtualMachine_DetachUnknownDisks)
	log.Printf("[DEBUG] template config= %s", config)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereVirtualMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					test_exists, test_name, test_cpu, test_uuid, test_mem, test_num_disk, test_num_of_nic, test_nic_label,
				),
			},
			resource.TestStep{
				PreConfig: func() {
					createAndAttachDisk(t, "terraform-test", 1, datastore, "terraform-test/tf_custom_disk.vmdk", "lazy", "scsi", datacenter)
				},
				Config: " ",
				Check: resource.ComposeTestCheckFunc(
					checkForDisk(datacenter, datastore, "terraform-test", "one.vmdk", true, false),
					checkForDisk(datacenter, datastore, "terraform-test", "two.vmdk", false, false),
					checkForDisk(datacenter, datastore, "terraform-test", "three.vmdk", true, false),
					checkForDisk(datacenter, datastore, "terraform-test", "tf_custom_disk.vmdk", true, true),
				),
			},
		},
	})
}

func createAndAttachDisk(t *testing.T, vmName string, size int, datastore string, diskPath string, diskType string, adapterType string, datacenter string) {
	client := testAccProvider.Meta().(*govmomi.Client)
	finder := find.NewFinder(client.Client, true)

	dc, err := finder.Datacenter(context.TODO(), datacenter)
	if err != nil {
		log.Printf("[ERROR] finding Datacenter %s: %v", datacenter, err)
		t.Fail()
		return
	}
	finder = finder.SetDatacenter(dc)
	ds, err := getDatastore(finder, datastore)
	if err != nil {
		log.Printf("[ERROR] getDatastore %s: %v", datastore, err)
		t.Fail()
		return
	}
	vm, err := finder.VirtualMachine(context.TODO(), vmName)
	if err != nil {
		log.Printf("[ERROR] finding VM %s: %v", vmName, err)
		t.Fail()
		return
	}
	err = addHardDisk(vm, int64(size), int64(0), diskType, ds, diskPath, adapterType)
	if err != nil {
		log.Printf("[ERROR] addHardDisk: %v", err)
		t.Fail()
		return
	}
}

func vmCleanup(dc *object.Datacenter, ds *object.Datastore, vmName string) error {
	client := testAccProvider.Meta().(*govmomi.Client)
	fileManager := object.NewFileManager(client.Client)
	task, err := fileManager.DeleteDatastoreFile(context.TODO(), ds.Path(vmName), dc)
	if err != nil {
		log.Printf("[ERROR] checkForDisk - Couldn't delete vm folder '%v': %v", vmName, err)
		return err
	}

	_, err = task.WaitForResult(context.TODO(), nil)
	if err != nil {
		log.Printf("[ERROR] checForDisk - Failed while deleting vm folder '%v': %v", vmName, err)
		return err
	}
	return nil
}

func checkForDisk(datacenter string, datastore string, vmName string, path string, exists bool, cleanup bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*govmomi.Client)
		finder := find.NewFinder(client.Client, true)

		dc, err := getDatacenter(client, datacenter)
		if err != nil {
			return err
		}
		finder.SetDatacenter(dc)

		ds, err := finder.Datastore(context.TODO(), datastore)
		if err != nil {
			log.Printf("[ERROR] checkForDisk - Couldn't find Datastore '%v': %v", datastore, err)
			return err
		}

		diskPath := vmName + "/" + path

		_, err = ds.Stat(context.TODO(), diskPath)
		if err != nil && exists {
			log.Printf("[ERROR] checkForDisk - Couldn't stat file '%v': %v", diskPath, err)
			return err
		} else if err == nil && !exists {
			errorMessage := fmt.Sprintf("checkForDisk - disk %s still exists", diskPath)
			err = vmCleanup(dc, ds, vmName)
			if err != nil {
				return fmt.Errorf("[ERROR] %s, cleanup also failed: %v", errorMessage, err)
			}
			return fmt.Errorf("[ERROR] %s", errorMessage)
		}

		if !cleanup || !exists {
			return nil
		}

		err = vmCleanup(dc, ds, vmName)
		if err != nil {
			return fmt.Errorf("[ERROR] cleanup failed: %v", err)
		}

		return nil
	}
}
