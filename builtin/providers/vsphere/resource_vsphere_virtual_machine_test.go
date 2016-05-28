package vsphere

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"golang.org/x/net/context"
	"path/filepath"
)

// Base setup function to check that a template, and nic information is set
func testBasicPreCheck(t *testing.T) {

	testAccPreCheck(t)

	if v := os.Getenv("VSPHERE_TEMPLATE"); v == "" {
		t.Fatal("VSPHERE_TEMPLATE must be set for acceptance tests")
	}

	if v := os.Getenv("VSPHERE_IPV4_GATEWAY"); v == "" {
		t.Fatal("VSPHERE_IPV4_GATEWAY must be set for acceptance tests")
	}

	if v := os.Getenv("VSPHERE_IPV4_ADDRESS"); v == "" {
		t.Fatal("VSPHERE_IPV4_ADDRESS must be set for acceptance tests")
	}

	if v := os.Getenv("VSPHERE_NETWORK_LABEL"); v == "" {
		t.Fatal("VSPHERE_NETWORK_LABEL must be set for acceptance tests")
	}
}

// Collects optional env vars used in the tests
func setupBaseVars() (string, string) {
	// TODO refactor all these vars as a struct
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

// returns variables that are used in most tests
func setupBasicVars() (string, string, string, string) {
	return os.Getenv("VSPHERE_TEMPLATE"),
		os.Getenv("VSPHERE_IPV4_GATEWAY"),
		os.Getenv("VSPHERE_NETWORK_LABEL"),
		os.Getenv("VSPHERE_IPV4_ADDRESS")

}

// returns variables used in DHCP tests
func setupDHCPVars() (string, string) {
	return os.Getenv("VSPHERE_TEMPLATE"), os.Getenv("VSPHERE_NETWORK_LABEL_DHCP")
}

// Basic data
type TestFuncData struct {
	vm         virtualMachine
	label      string
	vmName     string
	vmResource string
	numDisks   string
	numCPU     string
	mem        string
}

// returncs TestCheckFunc's that are used in many of our tests
// mem defaults to 1024
// cpu defaults to 2
// disks defatuls to 1
// vmResource defaults to "terraform-test"
// vmName defaults to "vsphere_virtual_machine.foo
func (test TestFuncData) testCheckFuncBasic() (
	resource.TestCheckFunc, resource.TestCheckFunc, resource.TestCheckFunc, resource.TestCheckFunc,
	resource.TestCheckFunc, resource.TestCheckFunc, resource.TestCheckFunc) {
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
	return testAccCheckVSphereVirtualMachineExists(test.vmName, &test.vm),
		resource.TestCheckResourceAttr(test.vmName, "name", res),
		resource.TestCheckResourceAttr(test.vmName, "vcpu", cpu),
		resource.TestCheckResourceAttr(test.vmName, "memory", mem),
		resource.TestCheckResourceAttr(test.vmName, "disk.#", disks),
		resource.TestCheckResourceAttr(test.vmName, "network_interface.#", "1"),
		resource.TestCheckResourceAttr(test.vmName, "network_interface.0.label", test.label)
}

func TestAccVSphereVirtualMachine_basic(t *testing.T) {
	var vm virtualMachine

	locationOpt, datastoreOpt := setupBaseVars()
	template, gateway, label, ip_address := setupBasicVars()

	log.Printf("[DEBUG] template= %s", testAccCheckVSphereVirtualMachineConfig_really_basic)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testBasicPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereVirtualMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(
					testAccCheckVSphereVirtualMachineConfig_really_basic,
					locationOpt,
					label,
					ip_address,
					gateway,
					datastoreOpt,
					template,
				),
				Check: resource.ComposeTestCheckFunc(
					TestFuncData{vm: vm, label: label}.testCheckFuncBasic(),
				),
			},
		},
	})
}

func TestAccVSphereVirtualMachine_client_debug(t *testing.T) {
	var vm virtualMachine
	locationOpt, datastoreOpt := setupBaseVars()
	template, gateway, label, ip_address := setupBasicVars()

	test_exists, test_name, test_cpu, test_mem, test_num_disk, test_num_of_nic, test_nic_label :=
		TestFuncData{vm: vm, label: label}.testCheckFuncBasic()

	log.Printf("[DEBUG] template= %s", testAccCheckVSphereVirtualMachineConfig_debug)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testBasicPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereVirtualMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(
					testAccCheckVSphereVirtualMachineConfig_debug,
					locationOpt,
					label,
					ip_address,
					gateway,
					datastoreOpt,
					template,
				),
				Check: resource.ComposeTestCheckFunc(
					test_exists, test_name, test_cpu, test_mem, test_num_disk, test_num_of_nic, test_nic_label,
					testAccCheckDebugExists(),
				),
			},
		},
	})
}

func TestAccVSphereVirtualMachine_diskInitType(t *testing.T) {
	var vm virtualMachine
	locationOpt, datastoreOpt := setupBaseVars()
	template, gateway, label, ip_address := setupBasicVars()
	vmName := "vsphere_virtual_machine.thin"
	test_exists, test_name, test_cpu, test_mem, test_num_disk, test_num_of_nic, test_nic_label :=
		TestFuncData{vm: vm, label: label, vmName: vmName, numDisks: "3"}.testCheckFuncBasic()

	log.Printf("[DEBUG] template= %s", testAccCheckVSphereVirtualMachineConfig_initType)
	config := fmt.Sprintf(
		testAccCheckVSphereVirtualMachineConfig_initType,
		locationOpt,
		label,
		ip_address,
		gateway,
		datastoreOpt,
		template,
	)
	log.Printf("[DEBUG] template with config= %s", config)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereVirtualMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					test_exists, test_name, test_cpu, test_mem, test_num_disk, test_num_of_nic, test_nic_label,
					// FIXME dynmically calculate the hashes
					resource.TestCheckResourceAttr(vmName, "disk.294918912.type", "eager_zeroed"),
					resource.TestCheckResourceAttr(vmName, "disk.294918912.controller_type", "ide"),
					resource.TestCheckResourceAttr(vmName, "disk.1380467090.controller_type", "scsi"),
				),
			},
		},
	})
}

func TestAccVSphereVirtualMachine_dhcp(t *testing.T) {
	var vm virtualMachine
	locationOpt, datastoreOpt := setupBaseVars()
	template, label := setupDHCPVars()

	log.Printf("[DEBUG] template= %s", testAccCheckVSphereVirtualMachineConfig_debug)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereVirtualMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(
					testAccCheckVSphereVirtualMachineConfig_dhcp,
					locationOpt,
					label,
					datastoreOpt,
					template,
				),
				Check: resource.ComposeTestCheckFunc(
					TestFuncData{vm: vm, label: label, vmName: "vsphere_virtual_machine.bar"}.testCheckFuncBasic(),
				),
			},
		},
	})
}

func TestAccVSphereVirtualMachine_mac_address(t *testing.T) {
	var vm virtualMachine
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
	template := os.Getenv("VSPHERE_TEMPLATE")
	label := os.Getenv("VSPHERE_NETWORK_LABEL_DHCP")
	macAddress := os.Getenv("VSPHERE_NETWORK_MAC_ADDRESS")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereVirtualMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(
					testAccCheckVSphereVirtualMachineConfig_mac_address,
					locationOpt,
					label,
					macAddress,
					datastoreOpt,
					template,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVSphereVirtualMachineExists("vsphere_virtual_machine.mac_address", &vm),
					resource.TestCheckResourceAttr(
						"vsphere_virtual_machine.mac_address", "name", "terraform-mac-address"),
					resource.TestCheckResourceAttr(
						"vsphere_virtual_machine.mac_address", "vcpu", "2"),
					resource.TestCheckResourceAttr(
						"vsphere_virtual_machine.mac_address", "memory", "4096"),
					resource.TestCheckResourceAttr(
						"vsphere_virtual_machine.mac_address", "disk.#", "1"),
					resource.TestCheckResourceAttr(
						"vsphere_virtual_machine.mac_address", "disk.2166312600.template", template),
					resource.TestCheckResourceAttr(
						"vsphere_virtual_machine.mac_address", "network_interface.#", "1"),
					resource.TestCheckResourceAttr(
						"vsphere_virtual_machine.mac_address", "network_interface.0.label", label),
					resource.TestCheckResourceAttr(
						"vsphere_virtual_machine.mac_address", "network_interface.0.mac_address", macAddress),
				),
			},
		},
	})
}

func TestAccVSphereVirtualMachine_custom_configs(t *testing.T) {

	var vm virtualMachine
	locationOpt, datastoreOpt := setupBaseVars()
	template, label := setupDHCPVars()
	vmName := "vsphere_virtual_machine.car"
	test_exists, test_name, test_cpu, test_mem, test_num_disk, test_num_of_nic, test_nic_label :=
		TestFuncData{vm: vm, label: label, vmName: vmName}.testCheckFuncBasic()

	log.Printf("[DEBUG] template= %s", testAccCheckVSphereVirtualMachineConfig_custom_configs)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereVirtualMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(
					testAccCheckVSphereVirtualMachineConfig_custom_configs,
					locationOpt,
					label,
					datastoreOpt,
					template,
				),
				Check: resource.ComposeTestCheckFunc(
					test_exists, test_name, test_cpu, test_mem, test_num_disk, test_num_of_nic, test_nic_label,
					testAccCheckVSphereVirtualMachineExistsHasCustomConfig(vmName, &vm),
					resource.TestCheckResourceAttr(vmName, "custom_configuration_parameters.foo", "bar"),
					resource.TestCheckResourceAttr(vmName, "custom_configuration_parameters.car", "ferrari"),
					resource.TestCheckResourceAttr(vmName, "custom_configuration_parameters.num", "42"),
				),
			},
		},
	})
}

func TestAccVSphereVirtualMachine_createInExistingFolder(t *testing.T) {
	var vm virtualMachine
	var locationOpt string
	var datastoreOpt string
	var datacenter string

	folder := "tf_test_cpureateInExistingFolder"

	locationOpt, datastoreOpt = setupBaseVars()
	template, label := setupDHCPVars()

	log.Printf("[DEBUG] template= %s", testAccCheckVSphereVirtualMachineConfig_createInFolder)

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
				Config: fmt.Sprintf(
					testAccCheckVSphereVirtualMachineConfig_createInFolder,
					folder,
					locationOpt,
					label,
					datastoreOpt,
					template,
				),
				Check: resource.ComposeTestCheckFunc(
					TestFuncData{vm: vm, label: label, vmName: "vsphere_virtual_machine.folder", vmResource: "terraform-test-folder"}.testCheckFuncBasic(),
				),
			},
		},
	})
}

func TestAccVSphereVirtualMachine_createWithFolder(t *testing.T) {
	var vm virtualMachine
	var locationOpt string
	var folderLocationOpt string
	var datastoreOpt string
	var f folder

	folder := "tf_test_cpureateWithFolder"
	locationOpt, datastoreOpt = setupBaseVars()
	template, label := setupDHCPVars()
	vmName := "vsphere_virtual_machine.folder"
	test_exists, test_name, test_cpu, test_mem, test_num_disk, test_num_of_nic, test_nic_label :=
		TestFuncData{vm: vm, label: label, vmName: vmName, vmResource: "terraform-test-folder"}.testCheckFuncBasic()
	log.Printf("[DEBUG] template= %s", testAccCheckVSphereVirtualMachineConfig_createWithFolder)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckVSphereVirtualMachineDestroy,
			testAccCheckVSphereFolderDestroy,
		),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(
					testAccCheckVSphereVirtualMachineConfig_createWithFolder,
					folder,
					folderLocationOpt,
					locationOpt,
					label,
					datastoreOpt,
					template,
				),
				Check: resource.ComposeTestCheckFunc(
					test_exists, test_name, test_cpu, test_mem, test_num_disk, test_num_of_nic, test_nic_label,
					testAccCheckVSphereFolderExists(vmName, &f),
					resource.TestCheckResourceAttr(vmName, "folder", folder),
				),
			},
		},
	})
}

func TestAccVSphereVirtualMachine_createWithCdrom(t *testing.T) {
	var vm virtualMachine
	locationOpt, datastoreOpt := setupBaseVars()
	template, label := setupDHCPVars()
	cdromDatastore := os.Getenv("VSPHERE_CDROM_DATASTORE")
	cdromPath := os.Getenv("VSPHERE_CDROM_PATH")
	vmName := "vsphere_virtual_machine.with_cdrom"
	test_exists, test_name, test_cpu, test_mem, test_num_disk, test_num_of_nic, test_nic_label :=
		TestFuncData{vm: vm, label: label, vmName: vmName, vmResource: "terraform-test-with-cdrom"}.testCheckFuncBasic()

	log.Printf("[DEBUG] template= %s", testAccCheckVsphereVirtualMachineConfig_cdrom)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereVirtualMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(
					testAccCheckVsphereVirtualMachineConfig_cdrom,
					locationOpt,
					label,
					datastoreOpt,
					template,
					cdromDatastore,
					cdromPath,
				),
				Check: resource.ComposeTestCheckFunc(
					test_exists, test_name, test_cpu, test_mem, test_num_disk, test_num_of_nic, test_nic_label,
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

func TestAccVSphereVirtualMachine_createWithExistingVmdk(t *testing.T) {
	vmdk_path := os.Getenv("VSPHERE_VMDK_PATH")
	label := os.Getenv("VSPHERE_NETWORK_LABEL")

	var vm virtualMachine
	var locationOpt string
	var datastoreOpt string

	locationOpt, datastoreOpt = setupBaseVars()

	log.Printf("[DEBUG] template= %s", testAccCheckVSphereVirtualMachineConfig_withExistingVmdk)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereVirtualMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(
					testAccCheckVSphereVirtualMachineConfig_withExistingVmdk,
					locationOpt,
					label,
					datastoreOpt,
					vmdk_path,
				),
				Check: resource.ComposeTestCheckFunc(
					TestFuncData{vm: vm, label: label, vmName: "vsphere_virtual_machine.with_existing_vmdk",
						vmResource: "terraform-test-with-existing-vmdk"}.testCheckFuncBasic(),
					//resource.TestCheckResourceAttr(
					//	"vsphere_virtual_machine.with_existing_vmdk", "disk.2393891804.vmdk", vmdk_path),
					//resource.TestCheckResourceAttr(
					//	"vsphere_virtual_machine.with_existing_vmdk", "disk.2393891804.bootable", "true"),
				),
			},
		},
	})
}

func TestAccVSphereVirtualMachine_updateMemory(t *testing.T) {
	var vm virtualMachine
	var locationOpt string
	var datastoreOpt string

	locationOpt, datastoreOpt = setupBaseVars()
	template, label := setupDHCPVars()

	log.Printf("[DEBUG] template= %s", testAccCheckVSphereVirtualMachineConfig_updateMemory)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereVirtualMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(
					testAccCheckVSphereVirtualMachineConfig_updateMemory,
					locationOpt,
					"1024",
					label,
					datastoreOpt,
					template,
				),
				Check: resource.ComposeTestCheckFunc(
					TestFuncData{vm: vm, label: label, vmName: "vsphere_virtual_machine.bar"}.testCheckFuncBasic(),
				),
			},
			resource.TestStep{
				Config: fmt.Sprintf(
					testAccCheckVSphereVirtualMachineConfig_updateMemory,
					locationOpt,
					"2048",
					label,
					datastoreOpt,
					template,
				),
				Check: resource.ComposeTestCheckFunc(
					TestFuncData{vm: vm, label: label, mem: "2048", vmName: "vsphere_virtual_machine.bar"}.testCheckFuncBasic(),
				),
			},
		},
	})
}

func TestAccVSphereVirtualMachine_updateVcpu(t *testing.T) {
	var vm virtualMachine
	var locationOpt string
	var datastoreOpt string

	locationOpt, datastoreOpt = setupBaseVars()
	template, label := setupDHCPVars()

	log.Printf("[DEBUG] template= %s", testAccCheckVSphereVirtualMachineConfig_updateVcpu)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereVirtualMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(
					testAccCheckVSphereVirtualMachineConfig_updateVcpu,
					locationOpt,
					"2",
					label,
					datastoreOpt,
					template,
				),
				Check: resource.ComposeTestCheckFunc(
					TestFuncData{vm: vm, label: label, vmName: "vsphere_virtual_machine.bar"}.testCheckFuncBasic(),
				),
			},
			resource.TestStep{
				Config: fmt.Sprintf(
					testAccCheckVSphereVirtualMachineConfig_updateVcpu,
					locationOpt,
					"4",
					label,
					datastoreOpt,
					template,
				),
				Check: resource.ComposeTestCheckFunc(
					TestFuncData{vm: vm, label: label, vmName: "vsphere_virtual_machine.bar", numCPU: "4"}.testCheckFuncBasic(),
				),
			},
		},
	})
}

func TestAccVSphereVirtualMachine_ipv4Andipv6(t *testing.T) {
	var vm virtualMachine
	locationOpt, datastoreOpt := setupBaseVars()
	template, label := setupDHCPVars()
	vmName := "vsphere_virtual_machine.ipv4ipv6"
	test_exists, test_name, test_cpu, test_mem, test_num_disk, test_num_of_nic, test_nic_label :=
		TestFuncData{vm: vm, label: label, vmName: vmName, numDisks: "3", vmResource: "terraform-test-ipv4-ipv6"}.testCheckFuncBasic()
	ipv4Address := os.Getenv("VSPHERE_IPV4_ADDRESS")
	ipv4Gateway := os.Getenv("VSPHERE_IPV4_GATEWAY")
	ipv6Address := os.Getenv("VSPHERE_IPV6_ADDRESS")
	ipv6Gateway := os.Getenv("VSPHERE_IPV6_GATEWAY")

	log.Printf("[DEBUG] template= %s", testAccCheckVSphereVirtualMachineConfig_ipv4Andipv6)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereVirtualMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(
					testAccCheckVSphereVirtualMachineConfig_ipv4Andipv6,
					locationOpt,
					label,
					ipv4Address,
					ipv4Gateway,
					ipv6Address,
					ipv6Gateway,
					datastoreOpt,
					template,
				),
				Check: resource.ComposeTestCheckFunc(
					test_exists, test_name, test_cpu, test_mem, test_num_disk, test_num_of_nic, test_nic_label,
					resource.TestCheckResourceAttr(vmName, "network_interface.0.ipv4_address", ipv4Address),
					resource.TestCheckResourceAttr(vmName, "network_interface.0.ipv4_gateway", ipv4Gateway),
					resource.TestCheckResourceAttr(vmName, "network_interface.0.ipv6_address", ipv6Address),
					resource.TestCheckResourceAttr(vmName, "network_interface.0.ipv6_gateway", ipv6Gateway),
				),
			},
		},
	})
}

func TestAccVSphereVirtualMachine_updateDisks(t *testing.T) {
	var vm virtualMachine
	locationOpt, datastoreOpt := setupBaseVars()
	template, gateway, label, ip_address := setupBasicVars()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereVirtualMachineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(
					testAccCheckVSphereVirtualMachineConfig_basic,
					locationOpt,
					gateway,
					label,
					ip_address,
					gateway,
					datastoreOpt,
					template,
				),
				Check: resource.ComposeTestCheckFunc(
					TestFuncData{vm: vm, label: label, numDisks: "2"}.testCheckFuncBasic(),
				),
			},
			resource.TestStep{
				Config: fmt.Sprintf(
					testAccCheckVSphereVirtualMachineConfig_updateAddDisks,
					locationOpt,
					gateway,
					label,
					ip_address,
					gateway,
					datastoreOpt,
					template,
				),
				Check: resource.ComposeTestCheckFunc(
					TestFuncData{vm: vm, label: label, numDisks: "4"}.testCheckFuncBasic(),
				),
			},
			resource.TestStep{
				Config: fmt.Sprintf(
					testAccCheckVSphereVirtualMachineConfig_really_basic,
					locationOpt,
					gateway,
					label,
					ip_address,
					gateway,
					datastoreOpt,
					template,
				),
				Check: resource.ComposeTestCheckFunc(
					TestFuncData{vm: vm, label: label, numDisks: "1"}.testCheckFuncBasic(),
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

const testAccCheckVSphereVirtualMachineConfig_debug = `
provider "vsphere" {
  client_debug = true
}

` + testAccCheckVSphereVirtualMachineConfig_really_basic

const testAccTemplateBasicBody = `
%s
    vcpu = 2
    memory = 1024
    network_interface {
        label = "%s"
        ipv4_address = "%s"
        ipv4_prefix_length = 24
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

const testAccCheckVSphereVirtualMachineConfig_really_basic = `
resource "vsphere_virtual_machine" "foo" {
    name = "terraform-test"
` + testAccTemplateBasicBodyWithEnd

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
const testAccCheckVSphereVirtualMachineConfig_updateAddDisks = `
resource "vsphere_virtual_machine" "foo" {
    name = "terraform-test"
` + testAccTemplateBasicBody + `
    disk {
        size = 1
        iops = 500
	name = "one"
    }
	disk {
        size = 1
        iops = 500
	name = "two"
    }
	disk {
        size = 1
        iops = 500
	name = "three"
    }
}
`

const testAccCheckVSphereVirtualMachineConfig_initType = `
resource "vsphere_virtual_machine" "thin" {
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
const testAccCheckVSphereVirtualMachineConfig_dhcp = `
resource "vsphere_virtual_machine" "bar" {
    name = "terraform-test"
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

const testAccCheckVSphereVirtualMachineConfig_mac_address = `
resource "vsphere_virtual_machine" "mac_address" {
    name = "terraform-mac-address"
%s
    vcpu = 2
    memory = 4096
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

const testAccCheckVSphereVirtualMachineConfig_custom_configs = `
resource "vsphere_virtual_machine" "car" {
    name = "terraform-test-custom"
` + testAccTemplateBasicBody +
	`
    custom_configuration_parameters {
	"foo" = "bar"
	"car" = "ferrari"
	"num" = 42
    }
}
`

const testAccCheckVSphereVirtualMachineConfig_createInFolder = `
resource "vsphere_virtual_machine" "folder" {
    name = "terraform-test-folder"
    folder = "%s"
` + testAccTemplateBasicBodyWithEnd

const testAccCheckVSphereVirtualMachineConfig_createWithFolder = `
resource "vsphere_folder" "with_folder" {
	path = "%s"
%s
}
resource "vsphere_virtual_machine" "with_folder" {
    name = "terraform-test-with-folder"
    folder = "${vsphere_folder.with_folder.path}"
` + testAccTemplateBasicBodyWithEnd

const testAccCheckVsphereVirtualMachineConfig_cdrom = `
resource "vsphere_virtual_machine" "with_cdrom" {
    name = "terraform-test-with-cdrom"
    cdrom {
        datastore = "%s"
        path = "%s"
    }
` + testAccTemplateBasicBodyWithEnd

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
}
`
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

const testAccCheckVSphereVirtualMachineConfig_ipv4Andipv6 = `
resource "vsphere_virtual_machine" "ipv4ipv6" {
    name = "terraform-test-ipv4-ipv6"
%s
    vcpu = 2
    memory = 1024
    network_interface {
        label = "%s"
        ipv4_address = "%s"
        ipv4_prefix_length = 24
        ipv4_gateway = "%s"
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
