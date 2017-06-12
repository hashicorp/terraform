package scvmm

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/masterzen/winrm"
)

func testBasicPreCheckCP(t *testing.T) {

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

func TestAccspr_Basic(t *testing.T) {
	vmName := "TestSujay"
	checkpointName := "Testsp"
	vmmServer := "WIN-2F929KU8HIU"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCheckpointRDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckRevertCheckpointConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCheckpointRExists("scvmm_revert_checkpoint.RevertCheckpoint", vmName, vmmServer, checkpointName),
					resource.TestCheckResourceAttr(
						"scvmm_revert_checkpoint.RevertCheckpoint", "vm_name", "TestSujay"),
					resource.TestCheckResourceAttr(
						"scvmm_revert_checkpoint.RevertCheckpoint", "vmm_server", "WIN-2F929KU8HIU"),
					resource.TestCheckResourceAttr(
						"scvmm_revert_checkpoint.RevertCheckpoint", "checkpoint_name", "Testsp"),
				),
			},
		},
	})
}

func testAccCheckCheckpointRDestroy(s *terraform.State) error {
	return nil
}

func testAccCheckCheckpointRExists(n, vmName string, vmmServer string, checkpointName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Vm ID is set")
		}

		org := testAccProvider.Meta().(*winrm.Client)

		script := "[CmdletBinding(SupportsShouldProcess=$true)]\nparam(\n    [parameter(Mandatory=$true,HelpMessage=\"Enter VMMServer\")]\n    [string]$vmmServer,\n\n    [parameter(Mandatory=$true,HelpMessage=\"Enter Virtual Machine Name\")]\n    [string]$vmName,\n\n    [parameter(Mandatory=$true,HelpMessage=\"Enter Checkpoint Name\")]\n    [string]$checkpointName\n)\n\n\nBegin\n{  \n       If (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] \"Administrator\"))\n    {   $arguments = \"\" + $myinvocation.mycommand.definition + \" \"\n        $myinvocation.BoundParameters.Values | foreach{\n            $arguments += \"'$_' \"\n        }\n        echo $arguments\n        Start-Process powershell -Verb runAs -ArgumentList $arguments\n        Break\n    }                 \n    \n        try\n\u0009     {\n\u0009\u0009 Set-SCVMMServer -VMMServer $vmmServer > $null\n                 $VM = Get-SCVirtualMachine -Name  $vmName\n\n                 $checkpoint = $VM.LastRestoredVMCheckpoint.Name\n\n                 if($checkpoint -ne $checkpointName)\n                  {\n                    Write-Error \" Checkpoint is not restored\"\n                  }\n                \n             \n             }catch [Exception]\n\u0009        {\n\u0009\u0009        echo $_.Exception.Message\n                 }    \n}"
		arguments := vmmServer + " " + vmName + " " + checkpointName
		filename := "revertcp"
		result, err := execScript(org, script, filename, arguments)

		if err != "" {
			return fmt.Errorf("Error, checkpoint is not restored %v", result)
		}

		return nil
	}
}

const testAccCheckRevertCheckpointConfigBasic = `
resource "scvmm_revert_checkpoint" "RevertCheckpoint"{
     	timeout = "1000"
        vmm_server = "WIN-2F929KU8HIU"
        vm_name = "TestSujay"
        checkpoint_name= "Testsp"
}`
