package scvmm

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/masterzen/winrm"
)

func resourceSCVMMStartVM() *schema.Resource {
	return &schema.Resource{
		Create: resourceStartVM,
		Read:   resourceStartVMRead,
		Delete: resourceStartVMDelete,
		Schema: map[string]*schema.Schema{
			"vm_name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateVMName,
			},
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
		},
	}
}

func resourceStartVM(d *schema.ResourceData, meta interface{}) error {
	connection := meta.(*winrm.Client)
	vmmServer := d.Get("vmm_server").(string)
	vmName := d.Get("vm_name").(string)
	timeout := d.Get("timeout").(string)
	script := "[CmdletBinding(SupportsShouldProcess=$true)]\nparam(\n    [parameter(Mandatory=$true,HelpMessage=\"Enter Timeout Value\")]\n    [long]$timeval,\n\u0009[parameter(Mandatory=$true,HelpMessage=\"Enter VMMServer\")]\n\u0009[string]$VMMServer,\n\u0009[parameter(Mandatory=$true,HelpMessage=\"Enter vmId\")]\n\u0009[string]$vmName\n\n)\nBegin \n{\n    $code = \n    {\n\n            [CmdletBinding(SupportsShouldProcess=$true)]\n            param(\n            [parameter(Mandatory=$true,HelpMessage=\"Enter VMMServer\")]\n\u0009        [string]$VMMServer,\n\u0009        [parameter(Mandatory=$true,HelpMessage=\"Enter vmId\")]\n\u0009        [string]$vmName\n            )\n            Begin\n            {\n                If (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] \"Administrator\"))\n                {   \n                    $arguments = \"\" + $myinvocation.mycommand.definition + \" \"\n                    $myinvocation.BoundParameters.Values | foreach{\n                        $arguments += \"'$_' \"\n                    }\n                echo $arguments\n                Start-Process powershell -Verb runAs -ArgumentList $arguments\n                Break\n                }\n\u0009            try\n\u0009            {\n\u0009\u0009                Set-SCVMMServer -VMMServer $VMMServer > $null\n\u0009\u0009\u0009            $VM = Get-SCVirtualMachine -Name $vmName\n                        if($VM -eq $null){\n                            Write-Error \"VM does not exist\"\n                            exit\n                        }\n\u0009\u0009\u0009            Start-SCVirtualMachine -VM $VM\n            \u0009}\n            \u0009catch [Exception]\n\u0009            {\n\u0009\u0009            Write-Error $_.Exception.Message\n\u0009            }\n            }\n      }\n\n    $j = Start-Job -ScriptBlock $code -ArgumentList $VMMServer, $vmName\n    if (Wait-Job $j -Timeout $timeval) \n    { \n        Receive-Job $j \n    } \n    else \n    {\n        Remove-Job -force $j\n        Write-Error \"time out\"\n    }\n}\n"
	filename := "startVM"
	arguments := timeout + " \"" + vmmServer + "\" \"" + vmName + "\""
	_, err := execScript(connection, script, filename, arguments)
	if err != "" {
		log.Printf("[Error] Error while starting Virtual Machine: %s", err)
		return fmt.Errorf("[Error] Error while Starting Virtual Machine : %s", err)
	}
	d.SetId("start_" + vmmServer + "_" + vmName)
	return nil
}

func resourceStartVMRead(d *schema.ResourceData, meta interface{}) error {
	connection := meta.(*winrm.Client)
	vmmServer := d.Get("vmm_server").(string)
	vmName := d.Get("vm_name").(string)
	script := "[CmdletBinding(SupportsShouldProcess=$true)]\nparam (\n\n  [Parameter(Mandatory=$true,HelpMessage=\"Enter VM ID\")]\n  [string]$vmId,\n\n  [Parameter(Mandatory=$true,HelpMessage=\"Enter VmmServer\")]\n  [string]$vmmServer\n\n)\n\nBegin\n{\n   \n            If (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] \"Administrator\"))\n          {   \n                $arguments = \"\" + $myinvocation.mycommand.definition + \" \"\n                $myinvocation.BoundParameters.Values | foreach{$arguments += \"'$_' \" }\n            echo $arguments\n            Start-Process powershell -Verb runAs -ArgumentList $arguments\n            Break\n         }\n\u0009    try\n\u0009     {    \n                \n                $VMs = Get-SCVirtualMachine -VMMServer $vmmServer  | where-Object { $_.Name -Match $vmId -And $_.Status -eq \"Running\" }               \n                if($VMs -eq $null)\n                {     \n                  Write-Error \"VM is not running \"        \n                 }  \n            \n         }\n\u0009     catch [Exception]\n\u0009       {\n\u0009\u0009        echo $_.Exception.Message\n\u0009        }\n}"
	arguments := "\"" + vmName + "\" \"" + vmmServer + "\""
	filename := "startvm_test"
	_, err := execScript(connection, script, filename, arguments)

	if err != "" {
		d.SetId("")
		log.Printf("[Error] Error Virtual Machine is not running : %s", err)
	}

	return nil
}

func resourceStartVMDelete(d *schema.ResourceData, meta interface{}) error {
	connection := meta.(*winrm.Client)
	vmmServer := d.Get("vmm_server").(string)
	vmName := d.Get("vm_name").(string)
	timeout := d.Get("timeout").(string)
	script := "[CmdletBinding(SupportsShouldProcess=$true)]\nparam(\n    [parameter(Mandatory=$true,HelpMessage=\"Enter Timeout Value\")]\n    [long]$timeval,\n\u0009[parameter(Mandatory=$true,HelpMessage=\"Enter VMMServer\")]\n\u0009[string]$VMMServer,\n\u0009[parameter(Mandatory=$true,HelpMessage=\"Enter vmId\")]\n\u0009[string]$vmName\n)\nBegin\n{\n    $code = \n    {\n        [CmdletBinding(SupportsShouldProcess=$true)]\n        param(\n        [parameter(Mandatory=$true,HelpMessage=\"Enter VMMServer\")]\n\u0009    [string]$VMMServer,\n\u0009    [parameter(Mandatory=$true,HelpMessage=\"Enter vmId\")]\n\u0009    [string]$vmName\n        )\n        Begin\n        {\n            If (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] \"Administrator\"))\n            {   \n                $arguments = \"\" + $myinvocation.mycommand.definition + \" \"\n                $myinvocation.BoundParameters.Values | foreach{\n                    $arguments += \"'$_' \"\n                }\n                echo $arguments\n                Start-Process powershell -Verb runAs -ArgumentList $arguments\n                Break\n            }\n            \n\u0009        try\n\u0009        {\n\u0009\u0009\u0009    Set-SCVMMServer -VMMServer $VMMServer > $null\n\u0009\u0009\u0009    $VM = Get-SCVirtualMachine -Name $vmName\n\u0009\u0009\u0009    Stop-VM -VM $VM \n\u0009        }\n\u0009        catch [Exception]\n\u0009        {\n    \u0009\u0009    Write-Error $_.Exception.Message\n\u0009        }\n        }\n    }\n    $j = Start-Job -ScriptBlock $code -ArgumentList $VMMServer, $vmName\n    if (Wait-Job $j -Timeout $timeval) \n    { \n        Receive-Job $j \n    } \n    else \n    {\n        Remove-Job -force $j\n        Write-Error \"time out\"\n    }\n}"
	filename := "stopVM"
	arguments := timeout + " \"" + vmmServer + "\" \"" + vmName + "\""
	_, err := execScript(connection, script, filename, arguments)
	if err != "" {
		log.Printf("[Error] Error while stopping Virtual Machine: %s", err)
		return fmt.Errorf("[Error] Error while stopping Virtual Machine : %s", err)
	}
	d.SetId("")
	return nil
}
