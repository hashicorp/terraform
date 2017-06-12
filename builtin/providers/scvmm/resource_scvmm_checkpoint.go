package scvmm

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/masterzen/winrm"
)

func resourceSCVMMCheckpoint() *schema.Resource {
	return &schema.Resource{
		Create: resourceCheckpointAdd,
		Read:   resourceCheckpointRead,
		Delete: resourceCheckpointDelete,
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
func resourceCheckpointAdd(d *schema.ResourceData, meta interface{}) error {
	connection := meta.(*winrm.Client)
	vmName := d.Get("vm_name").(string)
	timeout := d.Get("timeout").(string)
	vmmServer := d.Get("vmm_server").(string)
	checkpointName := d.Get("checkpoint_name").(string)

	validationScript := "[CmdletBinding(SupportsShouldProcess=$true)]\nparam(\n    [parameter(Mandatory=$true,HelpMessage=\"Enter Timeout Value\")]\n    [long]$timeval,\n [parameter(Mandatory=$true,HelpMessage=\"Enter VMMServer\")]\n       [string]$VMMServer,\n    [parameter(Mandatory=$true,HelpMessage=\"Enter Virtual Machine Name\")]\n    [string]$VMName,\n    [parameter(Mandatory=$true,HelpMessage=\"Enter Checkpoint Name\")]\n    [string]$CheckpointName\n)\n\n\nBegin\n{\n    $code = \n    {\n        [CmdletBinding(SupportsShouldProcess=$true)]\n        param (\n          [parameter(Mandatory=$true,HelpMessage=\"Enter VMMServer\")]\n               [string]$VMMServerw,\n\n            [Parameter(Mandatory=$true,HelpMessage=\"Enter VM Name\")]\n            [string]$vmNamew,\n\n            [Parameter(Mandatory=$true,HelpMessage=\"Enter checkpoint Name\")]\n            [string]$checkpointNamew\n        )\n        Begin\n        {\n            \n            \n            try\n          {\n                   Set-SCVMMServer -VMMServer $VMMServerw > $null\n                $vm = Get-SCVirtualMachine -Name $vmNamew\n                if($vm -eq $null){\n                    Write-Error \"VM Does not Exist\"\n                }\n                $checkpoint = Get-SCVMCheckpoint -VM $vmNamew | Where-Object {$_.Name -eq $checkpointNamew}\n                if($checkpoint -ne $null){\n                    Write-Error \"Checkpoint already Exists\"\n                }\n\n            }\n                catch [Exception]\n                {\n                   Write-Error $_.Exception.Message\n           }\n       \n            \n        }\n    }\n   \n    If (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] \"Administrator\"))\n    {   $arguments = \"\" + $myinvocation.mycommand.definition + \" \"\n        $myinvocation.BoundParameters.Values | foreach{\n            $arguments += \"'$_' \"\n        }\n        echo $arguments\n        Start-Process powershell -Verb runAs -ArgumentList $arguments\n        Break\n    }\n    $j = Start-Job -ScriptBlock $code -ArgumentList $VMMServer, $vmName, $checkpointName\n    if (Wait-Job $j -Timeout $timeval) \n    { \n        Receive-Job $j \n    } \n    else \n    {\n        Remove-Job -force $j\n        Write-Error \"time out\"\n        exit 1\n    }\n}\n\n"
	validationFileName := "validateCheckpoint"
	validationArguments := " " + timeout + " \"" + vmmServer + "\" \"" + vmName + "\" \"" + checkpointName + "\""
	_, validationError := execScript(connection, validationScript, validationFileName, validationArguments)
	if validationError != "" {
		log.Printf("[Error] Checkpoint already Exists: %s ", validationError)
		return fmt.Errorf("[Error] Error in Creating Checkpoint: %s", validationError)
	}

	script := "[CmdletBinding(SupportsShouldProcess=$true)]\nparam (\n  [parameter(Mandatory=$true,HelpMessage=\"Enter Timeout Value\")]\n  [string]$timeval,\n\n  [Parameter(Mandatory=$true,HelpMessage=\"Enter VM Name\")]\n  [string]$vmName,\n\n  [Parameter(Mandatory=$true,HelpMessage=\"Enter VmmServer\")]\n  [string]$vmmServer,\n\n  [Parameter(Mandatory=$true,HelpMessage=\"Enter Snapshot Name\")]\n  [string]$checkpointName\n)\n\nBegin\n{\n    \n\n    $code = \n    {\n        [CmdletBinding(SupportsShouldProcess=$true)]\n        param (\n            [Parameter(Mandatory=$true,HelpMessage=\"Enter VM Name\")]\n            [string]$vmName,\n\n            [Parameter(Mandatory=$true,HelpMessage=\"Enter Snapshot Name\")]\n            [string]$checkpointName,\n\n            [Parameter(Mandatory=$true,HelpMessage=\"Enter VmmServer\")]\n            [string]$vmmServer\n        )\n        Begin\n        {\n            If (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] \"Administrator\"))\n          {   \n                $arguments = \"\" + $myinvocation.mycommand.definition + \" \"\n                $myinvocation.BoundParameters.Values | foreach{$arguments += \"'$_' \" }\n            echo $arguments\n            Start-Process powershell -Verb runAs -ArgumentList $arguments\n            Break\n         }\n\u0009    try\n\u0009     {        \n                if($vmName -eq $null) \n                 {\n                    echo \"VM Name not entered\"\n                    exit\n                 } \n                \n                #gets virtual machine objects from the Virtual Machine Manager database\n                Set-SCVMMServer -VMMServer $vmmServer\n                 if($checkpointName -ne $null)\n                 {$Checkpoints = Get-SCVirtualMachine -Name $vmName | New-SCVMCheckpoint -name $checkpointName}\n                 else\n                  {echo \"Checkpoint Name is not entered\"}\n                          \n              }\n\u0009     catch [Exception]\n\u0009       {\n\u0009\u0009        echo $_.Exception.Message\n\u0009        }\n       \n        }\n    }\n    $j = Start-Job -ScriptBlock $code -ArgumentList $vmName, $checkpointName, $vmmServer\n    if (Wait-Job $j -Timeout $timeval) \n    { \n        Receive-Job $j \n    } \n    else \n    {\n        Remove-Job -force $j\n        echo \"time out\"\n        exit 1\n    }\n\n}"
	filename := "createCheckpoint"
	arguments := " " + timeout + " \"" + vmName + "\" \"" + vmmServer + "\" \"" + checkpointName + "\""
	_, err := execScript(connection, script, filename, arguments)
	if err != "" {
		log.Printf("[Error] Error in Creating Checkpoint : %s ", err)
		return fmt.Errorf("[Error] Error in Creating Checkpoint : %s", err)
	}
	if err == "" {
		terraformID := vmmServer + "_" + vmName + "_" + checkpointName
		d.SetId(terraformID)
	}
	return nil
}

func resourceCheckpointRead(d *schema.ResourceData, meta interface{}) error {
	connection := meta.(*winrm.Client)
	vmName := d.Get("vm_name").(string)
	timeout := d.Get("timeout").(string)
	vmmServer := d.Get("vmm_server").(string)
	checkpointName := d.Get("checkpoint_name").(string)
	script := "[CmdletBinding(SupportsShouldProcess=$true)]\nparam(\n    [parameter(Mandatory=$true,HelpMessage=\"Enter Timeout Value\")]\n    [long]$timeval,\n\u0009[parameter(Mandatory=$true,HelpMessage=\"Enter VMMServer\")]\n\u0009[string]$VMMServer,\n    [parameter(Mandatory=$true,HelpMessage=\"Enter Virtual Machine Name\")]\n    [string]$VMName,\n    [parameter(Mandatory=$true,HelpMessage=\"Enter Checkpoint Name\")]\n    [string]$CheckpointName\n)\n\n\nBegin\n{\n    $code = \n    {\n        [CmdletBinding(SupportsShouldProcess=$true)]\n        param (\n\u0009        [parameter(Mandatory=$true,HelpMessage=\"Enter VMMServer\")]\n\u0009        [string]$VMMServerw,\n\n            [Parameter(Mandatory=$true,HelpMessage=\"Enter VM Name\")]\n            [string]$vmNamew,\n\n            [Parameter(Mandatory=$true,HelpMessage=\"Enter checkpoint Name\")]\n            [string]$checkpointNamew\n        )\n        Begin\n        {\n            \n            \n            try\n\u0009        {\n\u0009\u0009        Set-SCVMMServer -VMMServer $VMMServerw > $null\n                $checkpoint = Get-SCVMCheckpoint -VM $vmNamew | Where-Object {$_.Name -eq $checkpointNamew}\n                if($checkpoint-eq$null){\n                    Write-Error \"No Checkpoint found\"\n                }\n\n            }\n\u0009        catch [Exception]\n\u0009        {\n\u0009\u0009        Write-Error $_.Exception.Message\n\u0009        }\n       \n            \n        }\n    }\n   \n    If (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] \"Administrator\"))\n    {   $arguments = \"\" + $myinvocation.mycommand.definition + \" \"\n        $myinvocation.BoundParameters.Values | foreach{\n            $arguments += \"'$_' \"\n        }\n        echo $arguments\n        Start-Process powershell -Verb runAs -ArgumentList $arguments\n        Break\n    }\n    $j = Start-Job -ScriptBlock $code -ArgumentList $VMMServer, $vmName, $checkpointName\n    if (Wait-Job $j -Timeout $timeval) \n    { \n        Receive-Job $j \n    } \n    else \n    {\n        Remove-Job -force $j\n        Write-Error \"time out\"\n        exit 1\n    }\n}\n"
	filename := "readCheckpoint"
	arguments := " " + timeout + " \"" + vmmServer + "\" \"" + vmName + "\" \"" + checkpointName + "\""
	_, err := execScript(connection, script, filename, arguments)
	if err != "" {
		log.Printf("[Error] Checkpoint Not Found: %s ", err)
		d.SetId("")
	}
	return nil
}

func resourceCheckpointDelete(d *schema.ResourceData, meta interface{}) error {

	log.Println("[INFO] Checking if Checkpoint Exists")
	resourceCheckpointRead(d, meta)
	if d.Id() == "" {
		log.Println("[Error] Checkpoint is not available")
		return fmt.Errorf("[Error] Checkpoint does not Exist")
	}
	log.Println("[INFO] Checkpoint Found")

	connection := meta.(*winrm.Client)
	vmName := d.Get("vm_name").(string)
	timeout := d.Get("timeout").(string)
	vmmServer := d.Get("vmm_server").(string)
	checkpointName := d.Get("checkpoint_name").(string)
	script := "[CmdletBinding(SupportsShouldProcess=$true)]\nparam(\n    [parameter(Mandatory=$true,HelpMessage=\"Enter Timeout Value\")]\n    [long]$timeval,\n\u0009[parameter(Mandatory=$true,HelpMessage=\"Enter VMMServer\")]\n\u0009[string]$VMMServer,\n    [parameter(Mandatory=$true,HelpMessage=\"Enter Virtual Machine Name\")]\n    [string]$VMName,\n    [parameter(Mandatory=$true,HelpMessage=\"Enter Checkpoint Name\")]\n    [string]$CheckpointName\n)\n\n\nBegin\n{\n    $code = \n    {\n        [CmdletBinding(SupportsShouldProcess=$true)]\n        param (\n\u0009        [parameter(Mandatory=$true,HelpMessage=\"Enter VMMServer\")]\n\u0009        [string]$VMMServerw,\n\n            [Parameter(Mandatory=$true,HelpMessage=\"Enter VM Name\")]\n            [string]$vmNamew,\n\n            [Parameter(Mandatory=$true,HelpMessage=\"Enter checkpoint Name\")]\n            [string]$checkpointNamew\n        )\n        Begin\n        {\n            \n            \n            try\n\u0009        {\n\u0009\u0009        Set-SCVMMServer -VMMServer $VMMServerw > $null\n                $checkpoint = Get-SCVMCheckpoint -VM $vmNamew | Where-Object {$_.Name -eq $checkpointNamew}\n                Remove-SCVMCheckpoint -VMCheckpoint $checkpoint\n            }\n\u0009        catch [Exception]\n\u0009        {\n\u0009\u0009        echo $_.Exception.Message\n\u0009        }\n       \n            \n        }\n    }\n   \n    If (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] \"Administrator\"))\n    {   $arguments = \"\" + $myinvocation.mycommand.definition + \" \"\n        $myinvocation.BoundParameters.Values | foreach{\n            $arguments += \"'$_' \"\n        }\n        echo $arguments\n        Start-Process powershell -Verb runAs -ArgumentList $arguments\n        Break\n    }\n    $j = Start-Job -ScriptBlock $code -ArgumentList $VMMServer, $vmName, $checkpointName\n    if (Wait-Job $j -Timeout $timeval) \n    { \n        Receive-Job $j \n    } \n    else \n    {\n        Remove-Job -force $j\n        echo \"time out\"\n        exit 1\n    }\n}"

	filename := "deleteCheckpoint"
	arguments := " " + timeout + " \"" + vmmServer + "\" \"" + vmName + "\" \"" + checkpointName + "\""
	_, err := execScript(connection, script, filename, arguments)

	if err != "" {
		log.Printf("[Error] Error while deleting Checkpoint : %s ", err)
		return fmt.Errorf("[Error] Error while deleting Checkpoint : %s", err)
	}
	if err == "" {
		d.SetId("")
	}
	return nil
}
