package scvmm

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/masterzen/winrm"
)

func testBasicPreCheckVM(t *testing.T) {

	testAccPreCheck(t)

	if v := os.Getenv("SCVMM_SERVER_IP"); v == "" {
		t.Fatal("SCVMM_SERVER_IP must be set for acceptance tests")
	}

	if v := os.Getenv("SCVMM_SERVER_PORT"); v == "" {
		t.Fatal("SCVMM_SERVER_PORT must be set for acceptance tests")
	}

	if v := os.Getenv("SCVMM_SERVER_USER"); v == "" {
		t.Fatal("SCVMM_SERVER_USER must be set for acceptance tests")
	}

	if v := os.Getenv("SCVMM_SERVER_PASSWORD"); v == "" {
		t.Fatal("SCVMM_SERVER_PASSWORD must be set for acceptance tests")
	}
}

func TestAccVmcreate_Basic(t *testing.T) {
	vmName := "TestVM"
	vmmServer := "WIN-2F929KU8HIU"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVmcreateDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckVMConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVmcreateExists("scvmm_vm.CreateVM", vmName, vmmServer),
					resource.TestCheckResourceAttr(
						"scvmm_vm.CreateVM", "vm_name", "TestVM"),
					resource.TestCheckResourceAttr(
						"scvmm_vm.CreateVM", "vmm_server", "WIN-2F929KU8HIU"),
				),
			},
		},
	})
}

func testAccCheckVmcreateDestroy(s *terraform.State) error {

	for _, rs := range s.RootModule().Resources {

		if rs.Type != "scvmm_vm" {
			continue
		}
		org := testAccProvider.Meta().(*winrm.Client)

		script := "[CmdletBinding(SupportsShouldProcess=$true)]\nparam (\n  \n  [Parameter(Mandatory=$true,HelpMessage=\"Enter VmmServer\")]\n  [string]$vmmServer,\n  [Parameter(Mandatory=$true,HelpMessage=\"Enter VM Name\")]\n  [string]$vmName\n)\n\nBegin\n{\n         If (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] \"Administrator\"))\n          {   \n            $arguments = \"\" + $myinvocation.mycommand.definition + \" \"\n            $myinvocation.BoundParameters.Values | foreach{$arguments += \"'$_' \" }\n            echo $arguments\n            Start-Process powershell -Verb runAs -ArgumentList $arguments\n            Break\n         }\n\u0009     try\n\u0009     {\n               if($vmName -eq $null) \n               {\n                    echo \"VM Name not entered\"\n                    exit\n               } \n               #gets virtual machine objects from the Virtual Machine Manager database\n               Set-SCVMMServer -VMMServer $vmmServer > $null\n\u0009\u0009       $VM = Get-SCVirtualMachine | Where-Object {$_.Name -Eq $vmName }   \n               #check if VM Exists\n               if($VM -eq $null)\n               {     \n                   Write-Error \"VM does not exists\"\n                   exit\n               }\n            \n         }\n\u0009     catch [Exception]\n         {\n               Write-Error $_.Exception.Message\n\u0009     }\n      \n   \n}\n"
		arguments := rs.Primary.Attributes["vmm_server"] + " " + rs.Primary.Attributes["vm_name"]
		filename := "DeleteVM_Test"
		result, err := execScript(org, script, filename, arguments)

		if err == "" {
			return fmt.Errorf("Vm  still exists: %v", result)
		}
	}

	return nil
}

func testAccCheckVmcreateExists(n, vmName string, vmmServer string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Vm ID is set")
		}

		org := testAccProvider.Meta().(*winrm.Client)

		script := "[CmdletBinding(SupportsShouldProcess=$true)]\nparam (\n  \n  [Parameter(Mandatory=$true,HelpMessage=\"Enter VmmServer\")]\n  [string]$vmmServer,\n  [Parameter(Mandatory=$true,HelpMessage=\"Enter VM Name\")]\n  [string]$vmName\n)\n\nBegin\n{\n         If (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] \"Administrator\"))\n          {   \n            $arguments = \"\" + $myinvocation.mycommand.definition + \" \"\n            $myinvocation.BoundParameters.Values | foreach{$arguments += \"'$_' \" }\n            echo $arguments\n            Start-Process powershell -Verb runAs -ArgumentList $arguments\n            Break\n         }\n\u0009     try\n\u0009     {\n               if($vmName -eq $null) \n               {\n                    echo \"VM Name not entered\"\n                    exit\n               } \n               #gets virtual machine objects from the Virtual Machine Manager database\n               Set-SCVMMServer -VMMServer $vmmServer > $null\n\u0009\u0009       $VM = Get-SCVirtualMachine | Where-Object {$_.Name -Eq $vmName }   \n               #check if VM Exists\n               if($VM -eq $null)\n               {     \n                   Write-Error \"VM does not exists\"\n                   exit\n               }\n            \n         }\n\u0009     catch [Exception]\n         {\n               Write-Error $_.Exception.Message\n\u0009     }\n      \n   \n}\n"
		arguments := vmmServer + " " + vmName
		filename := "CreateVM_Test"
		result, err := execScript(org, script, filename, arguments)

		if err != "" {
			return fmt.Errorf("Error while getting the VM %v", result)
		}

		return nil
	}
}

const testAccCheckVMConfigBasic = `
resource "scvmm_vm" "CreateVM"{
     	timeout = "1000"
        vmm_server = "WIN-2F929KU8HIU"
        vm_name = "TestVM"
        template_name = "TestVMTemplate"
        cloud_name = "GSL Cloud"
}`
