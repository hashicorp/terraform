package scvmm

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/masterzen/winrm"
)

func testBasicPreCheckVMStart(t *testing.T) {

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

func TestAccVmstart_Basic(t *testing.T) {
	vmName := "TestSujay"
	vmmServer := "WIN-2F929KU8HIU"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVmstartDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckVMStartConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVMStartExists("scvmm_start_vm.StartVM", vmName, vmmServer),
					resource.TestCheckResourceAttr(
						"scvmm_start_vm.StartVM", "vm_name", "TestSujay"),
				),
			},
		},
	})
}

func testAccCheckVmstartDestroy(s *terraform.State) error {
	return nil
}

func testAccCheckVMStartExists(n, vmName string, vmmServer string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Vm ID is set")
		}

		org := testAccProvider.Meta().(*winrm.Client)

		script := "[CmdletBinding(SupportsShouldProcess=$true)]\nparam (\n\n  [Parameter(Mandatory=$true,HelpMessage=\"Enter VM ID\")]\n  [string]$vmId,\n\n  [Parameter(Mandatory=$true,HelpMessage=\"Enter VmmServer\")]\n  [string]$vmmServer\n\n)\n\nBegin\n{\n   \n            If (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] \"Administrator\"))\n          {   \n                $arguments = \"\" + $myinvocation.mycommand.definition + \" \"\n                $myinvocation.BoundParameters.Values | foreach{$arguments += \"'$_' \" }\n            echo $arguments\n            Start-Process powershell -Verb runAs -ArgumentList $arguments\n            Break\n         }\n\u0009    try\n\u0009     {    \n                \n                $VMs = Get-SCVirtualMachine -VMMServer $vmmServer  | where-Object { $_.Name -Match $vmId -And $_.Status -eq \"Running\" }               \n                if($VMs -eq $null)\n                {     \n                  Write-Error \"VM is not running \"        \n                 }  \n            \n         }\n\u0009     catch [Exception]\n\u0009       {\n\u0009\u0009        echo $_.Exception.Message\n\u0009        }\n}"
		arguments := vmName + " " + vmmServer
		filename := "startvm_test"
		result, err := execScript(org, script, filename, arguments)

		if err != "" {
			return fmt.Errorf("Error , Still VM is not started %v", result)
		}

		return nil
	}
}

const testAccCheckVMStartConfigBasic = `
resource "scvmm_start_vm" "StartVM"{
	    vm_name = "TestSujay"
		timeout= "1000"
        vmm_server = "WIN-2F929KU8HIU"
	}`
