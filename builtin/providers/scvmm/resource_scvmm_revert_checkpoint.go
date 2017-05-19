package scvmm

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/masterzen/winrm"
)

func resourceSCVMMRevertCheckpoint() *schema.Resource {
	return &schema.Resource{
		Create: resourceRevertCheckpointAdd,
		Read:   resourceRevertCheckpointRead,
		Delete: resourceRevertCheckpointDelete,
		Schema: map[string]*schema.Schema{

			"timeout": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateTimeout,
			},
			"vmm_server": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateVMMServer,
			},

			"vm_name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateVMName,
			},
			"checkpoint_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}
func resourceRevertCheckpointAdd(d *schema.ResourceData, meta interface{}) error {

	d.SetId("test")
	resourceCheckpointRead(d, meta)
	if d.Id() == "" {
		log.Printf("[Error] Checkpoint does not Exist")
		return fmt.Errorf("[Error] Checkpoint does not Exist")
	}
	d.SetId("")

	connection := meta.(*winrm.Client)
	vmName := d.Get("vm_name").(string)
	timeout := d.Get("timeout").(string)
	vmmServer := d.Get("vmm_server").(string)
	checkpointName := d.Get("checkpoint_name").(string)
	script := "[CmdletBinding(SupportsShouldProcess=$true)]\nparam (\n  [parameter(Mandatory=$true,HelpMessage=\"Enter Timeout Value\")]\n  [string]$timeval,\n\n  [Parameter(Mandatory=$true,HelpMessage=\"Enter VM Name\")]\n  [string]$vmName,\n\n  [Parameter(Mandatory=$true,HelpMessage=\"Enter VmmServer\")]\n  [string]$vmmServer,\n\n  [Parameter(Mandatory=$true,HelpMessage=\"Enter Checkpoint Name\")]\n  [string]$checkpointName\n)\n\nBegin\n{\n    \n\n    $code = \n    {\n        [CmdletBinding(SupportsShouldProcess=$true)]\n        param (\n            [Parameter(Mandatory=$true,HelpMessage=\"Enter VM Name\")]\n            [string]$vmName,\n\n            [Parameter(Mandatory=$true,HelpMessage=\"Enter Checkpoint Name\")]\n            [string]$checkpointName,\n\n            [Parameter(Mandatory=$true,HelpMessage=\"Enter VmmServer\")]\n            [string]$vmmServer\n        )\n        Begin\n        {\n            If (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] \"Administrator\"))\n          {   \n                $arguments = \"\" + $myinvocation.mycommand.definition + \" \"\n                $myinvocation.BoundParameters.Values | foreach{$arguments += \"'$_' \" }\n            echo $arguments\n            Start-Process powershell -Verb runAs -ArgumentList $arguments\n            Break\n         }\n\u0009    try\n\u0009     {       \n\n                 if($vmName -eq $null) \n                 {\n                    echo \"VM Name not entered\"\n                    exit\n                 } \n\n                #gets virtual machine objects from the Virtual Machine Manager database\n                Set-SCVMMServer -VMMServer $vmmServer\n\u0009\u0009$VM = Get-SCVirtualMachine | Where-Object {$_.Name -Eq $vmName }   \n                #check if VM Exists\n                if($VM -ne $null)\n                {      if($checkpointName -ne $null)\n                       {\n                          $checkpoint = Get-SCVMCheckpoint | Where-Object {$_.VM.Name -Match $vmName -And  $_.Name -Eq $checkpointName}\n\n                      if($checkpoint -ne $null)\n                      {\n                        Restore-SCVMCheckpoint -VMCheckpoint $checkpoint\n                      }\n                      else{\n                      echo \"Checkpoint is not present\"\n                      }\n                        }\n                     else {echo \"Name of the checkpoint is not entered\"}\n\u0009                \n                }\n            \n         }\n\u0009     catch [Exception]\n\u0009       {\n\u0009\u0009        echo $_.Exception.Message\n\u0009        }\n       \n        }\n    }\n    $j = Start-Job -ScriptBlock $code -ArgumentList $vmName, $checkpointName, $vmmServer\n    if (Wait-Job $j -Timeout $timeval) \n    { \n        Receive-Job $j \n    } \n    else \n    {\n        Remove-Job -force $j\n        echo \"time out\"\n        exit 1\n    }\n\n}"
	arguments := " " + timeout + " \"" + vmName + "\" \"" + vmmServer + "\" \"" + checkpointName + "\""
	filename := "revertCheckpoint"
	_, err := execScript(connection, script, filename, arguments)

	if err != "" {
		log.Printf("[Error] Error while reverting Checkpoint : %s ", err)
		return fmt.Errorf("[Error] Error while reverting Checkpoint : %s", err)
	}
	if err == "" {
		terraformID := "revert_" + vmmServer + "_" + vmName + "_" + checkpointName
		d.SetId(terraformID)
	}
	return nil
}
func resourceRevertCheckpointRead(d *schema.ResourceData, meta interface{}) error {
	connection := meta.(*winrm.Client)
	vmName := d.Get("vm_name").(string)
	vmmServer := d.Get("vmm_server").(string)
	checkpointName := d.Get("checkpoint_name").(string)
	script := "[CmdletBinding(SupportsShouldProcess=$true)]\nparam(\n    [parameter(Mandatory=$true,HelpMessage=\"Enter VMMServer\")]\n    [string]$vmmServer,\n\n    [parameter(Mandatory=$true,HelpMessage=\"Enter Virtual Machine Name\")]\n    [string]$vmName,\n\n    [parameter(Mandatory=$true,HelpMessage=\"Enter Checkpoint Name\")]\n    [string]$checkpointName\n)\n\n\nBegin\n{  \n       If (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] \"Administrator\"))\n    {   $arguments = \"\" + $myinvocation.mycommand.definition + \" \"\n        $myinvocation.BoundParameters.Values | foreach{\n            $arguments += \"'$_' \"\n        }\n        echo $arguments\n        Start-Process powershell -Verb runAs -ArgumentList $arguments\n        Break\n    }                 \n    \n        try\n\u0009     {\n\u0009\u0009 Set-SCVMMServer -VMMServer $vmmServer > $null\n                 $VM = Get-SCVirtualMachine -Name  $vmName\n\n                 $checkpoint = $VM.LastRestoredVMCheckpoint.Name\n\n                 if($checkpoint -ne $checkpointName)\n                  {\n                    Write-Error \" Checkpoint is not restored\"\n                  }\n                \n             \n             }catch [Exception]\n\u0009        {\n\u0009\u0009        echo $_.Exception.Message\n                 }    \n}"
	arguments := "\"" + vmmServer + "\" \"" + vmName + "\" \"" + checkpointName + "\""
	filename := "revertCPRead"
	_, err := execScript(connection, script, filename, arguments)
	if err != "" {
		log.Printf("[Error] Error VM is not at specified Checkpoint : %s ", err)
		d.SetId("")
	}
	return nil
}

func resourceRevertCheckpointDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
