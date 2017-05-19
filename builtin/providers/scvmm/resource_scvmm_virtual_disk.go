package scvmm

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/masterzen/winrm"
)

func resourceSCVMMVirtualDisk() *schema.Resource {
	return &schema.Resource{
		Create: resourceCreateVirtualDisk,
		Read:   resourceReadVirtualDisk,
		Delete: resourceDeleteVirtualDisk,
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
			"virtual_disk_name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateVMName,
			},
			"virtual_disk_size": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateTimeout,
			},
		},
	}
}

func resourceCreateVirtualDisk(d *schema.ResourceData, meta interface{}) error {
	connection := meta.(*winrm.Client)
	vmName := d.Get("vm_name").(string)
	vmmServer := d.Get("vmm_server").(string)
	volumeName := d.Get("virtual_disk_name").(string)
	timeout := d.Get("timeout").(string)
	volumeSize := d.Get("virtual_disk_size").(string)

	log.Println("[INFO] Performing check if requested Specs are feasible in given VMM Server")
	validationScript := "[CmdletBinding(SupportsShouldProcess=$true)]\nparam(\n    [parameter(Mandatory=$true,HelpMessage=\"Enter VMMServer\")]\n\u0009[string]$VMMServer,\n\n    [parameter(Mandatory=$true,HelpMessage=\"Enter VMMServer\")]\n\u0009[long]$DiskSize\n)\n\n\nBegin\n{\n    If (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] \"Administrator\"))\n            {   \n                $arguments = \"\" + $myinvocation.mycommand.definition + \" \"\n                $myinvocation.BoundParameters.Values | foreach{\n                $arguments += \"'$_' \"\n            }\n            echo $arguments\n            Start-Process powershell -Verb runAs -ArgumentList $arguments\n            Break\n        }\n\u0009    try\n\u0009    {\n\n    $VMHost=Get-SCVMHost -ComputerName $VMMServer\n    if($VMHost -eq $null){\n        Write-Error [string]::Format(\"VMHost {0} is not available\",$VMMServer)\n    }\n    $SCVMHost = Read-SCVMHost -VMHost $VMHost\n    if($SCVMHost -eq $null){\n        Write-Error \"Cannot Read SCVMHost\"\n    }\n    \n    if(($SCVMHost.TotalStorageCapacity * 0.8) -le $SCVMHost.UsedStorageCapacity){\n        $errorMsg = [string]::Format(\"Some Space is resrved for Dynamic disks to Grow Available Disk Space : {0} Total Disk Space Used : {1}\",($SCVMHost.AvailableStorageCapacity/(1024*1024)), ($SCVMHost.UsedStorageCapacity/(1024*1024)))\n        Write-Error $errorMsg       \n    }\n    \n    if(($SCVMHost.AvailableStorageCapacity/(1024*1024) - 200) -lt ($DiskSize)){\n        $errorMsg = [string]::Format(\"Disk Storage Not Sufficient Available Disk Space : {0} Requested Disk Space : {1}\",($SCVMHost.AvailableStorageCapacity/(1024*1024) - 200), ($DiskSize))\n        Write-Error $errorMsg\n    }\n    }\n    catch [Exception]\n\u0009{\n\u0009\u0009        Write-Error $_.Exception.Message\n\u0009}\n}"
	validationFileName := "validateCreateVirtualDisk"
	validationArguments := "\"" + vmmServer + "\" \"" + volumeSize + "\""
	_, validationError := execScript(connection, validationScript, validationFileName, validationArguments)
	if validationError != "" {
		log.Printf("[Error] Enough space is not available on selected Server : %s", validationError)
		return fmt.Errorf("[Error] Enough space is not available on selected Server : %s", validationError)
	}

	script := "[CmdletBinding(SupportsShouldProcess=$true)]\nparam (\n  [parameter(Mandatory=$true,HelpMessage=\"Enter Timeout Value\")]\n  [string]$timeval,\n\n  [Parameter(Mandatory=$true,HelpMessage=\"Enter VM Name\")]\n  [string]$vmmServer,\n\n  [Parameter(Mandatory=$true,HelpMessage=\"Enter VM Name\")]\n  [string]$vmName,\n\n  [Parameter(Mandatory=$true,HelpMessage=\"Enter Volume Name\")]\n  [string]$virtualDiskName,\n\n  [Parameter(Mandatory=$true,HelpMessage=\"Enter Volume Size in MegaBytes\")]\n  [long]$virtualDiskSize\n)\n\nBegin\n{\n    \n\n    $code = \n    {\n        [CmdletBinding(SupportsShouldProcess=$true)]\n        param (\n            [Parameter(Mandatory=$true,HelpMessage=\"Enter VM Name\")]\n            [string]$vmmServerw,\n\n            [Parameter(Mandatory=$true,HelpMessage=\"Enter VM Name\")]\n            [string]$vmNamew,\n\n            [Parameter(Mandatory=$true,HelpMessage=\"Enter Volume Name\")]\n            [string]$virtualDiskNamew,\n\n            [Parameter(Mandatory=$true,HelpMessage=\"Enter Volume Size in MegaBytes\")]\n            [long]$virtualDiskSizew\n        )\n        Begin\n        {\n            If (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] \"Administrator\"))\n            {   \n                $arguments = \"\" + $myinvocation.mycommand.definition + \" \"\n                $myinvocation.BoundParameters.Values | foreach{\n                $arguments += \"'$_' \"\n            }\n            echo $arguments\n            Start-Process powershell -Verb runAs -ArgumentList $arguments\n            Break\n        }\n\u0009    try\n\u0009    {\n                if($vmNamew -eq $null) \n                {\n                    echo \"VM Name not entered\"\n                    exit 1\n                }\n   \u0009\u0009\n\u0009\u0009        Set-SCVMMServer -VMMServer $VMMServerw\u0009> $null\n                \n\u0009\u0009        $busNumber = 0\n                $lunNumber = 0\n                $VM = Get-SCVirtualMachine -Name $vmNamew\n\n                #check if VM Exists\n                if($VM -ne $null)\n                {\n                    #Calculate lun Number\n                    $lunNumber = 0\n                    do {\n                        $vDrive = Get-SCVirtualDiskDrive -VM $VM | Where-Object { $_.BusType -Match \"SCSI\" -and $_.Lun -eq $lunNumber }\n                        $lunNumber++\n                    }while ($vDrive.Length -gt 0)\n                    $lunNumber = $lunNumber-1\n                    #Create Virtual Disk Drive\n                    $vmVolume = New-SCVirtualDiskDrive -VM $VM -Dynamic -FileName $virtualDiskNamew -SCSI -Size $virtualDiskSizew -Bus 0 -LUN $lunNumber\n                    if($sw.elapsed -lt $timeout)\n\u0009\u0009\u0009        {\n                        break\n                    }\n\u0009\u0009            #Result Handling\n                    if($? -eq $false)\n                    {\n                        Write-Error \"Command was not execute successfully you may have to use Repair-SCVirtualMachine\"\n                        return -1\n                    }\n                    else \n                    {\n                        return $vmVolume.VirtualHardDiskId.ToString()\n                    }\n                } else {\n                    Write-Error \"VM does not exist\"\n                }\n            \n            }\n\u0009        catch [Exception]\n\u0009        {\n\u0009\u0009        echo $_.Exception.Message\n\u0009        }\n       \n        }\n    }\n    $j = Start-Job -ScriptBlock $code -ArgumentList $vmmServer, $vmName, $virtualDiskName, $virtualDiskSize\n    if (Wait-Job $j -Timeout $timeval) \n    { \n        Receive-Job $j \n    } \n    else \n    {\n        Remove-Job -force $j\n        Write-Error \"Timeout\"\n        exit 1\n    }\n\n}"
	filename := "createVD"
	arguments := timeout + " \"" + vmmServer + "\" \"" + vmName + "\" \"" + volumeName + "\" " + volumeSize
	_, err := execScript(connection, script, filename, arguments)

	if err != "" {
		log.Printf("[Error] Error in creating Virtual Disk : %s ", err)
		return fmt.Errorf("[Error] Error in creating Virtual Disk : %s", err)
	}
	if err == "" {
		terraformID := vmmServer + "_" + vmName + "_" + volumeName
		d.SetId(terraformID)
	}
	return nil
}

func resourceReadVirtualDisk(d *schema.ResourceData, meta interface{}) error {
	connection := meta.(*winrm.Client)
	vmName := d.Get("vm_name").(string)
	vmmServer := d.Get("vmm_server").(string)
	volumeName := d.Get("virtual_disk_name").(string)
	timeout := d.Get("timeout").(string)
	script := "\n[CmdletBinding(SupportsShouldProcess=$true)]\nparam (\n  [parameter(Mandatory=$true,HelpMessage=\"Enter Timeout Value\")]\n  [string]$timeval,\n\n  \n  [Parameter(Mandatory=$true,HelpMessage=\"Enter VmmServer\")]\n  [string]$vmmServer,\n\n  [Parameter(Mandatory=$true,HelpMessage=\"Enter VM Name\")]\n  [string]$vmName,\n\n   [Parameter(Mandatory=$true,HelpMessage=\"Enter Volume Name\")]\n  [string]$diskdriveName\n)\n\nBegin\n{\n    \n\n    $code = \n    {\n        [CmdletBinding(SupportsShouldProcess=$true)]\n        param (\n            \n            [Parameter(Mandatory=$true,HelpMessage=\"Enter VmmServer\")]\n            [string]$vmmServer,\n\n            [Parameter(Mandatory=$true,HelpMessage=\"Enter VM Name\")]\n            [string]$vmName,\n\n            [Parameter(Mandatory=$true,HelpMessage=\"Enter Volume Name\")]\n            [string]$diskdriveName\n        )\n        Begin\n        {\n            If (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] \"Administrator\"))\n          {   \n                $arguments = \"\" + $myinvocation.mycommand.definition + \" \"\n                $myinvocation.BoundParameters.Values | foreach{$arguments += \"'$_' \" }\n            echo $arguments\n            Start-Process powershell -Verb runAs -ArgumentList $arguments\n            Break\n         }\n\u0009    try\n\u0009     {     if($vmName -eq $null) \n               {\n                    echo \"VM Name not entered\"\n                    exit\n                } \n                #gets virtual machine objects from the Virtual Machine Manager database\n                Set-SCVMMServer -VMMServer $vmmServer > $null\n\u0009\u0009        $VM = Get-SCVirtualMachine | Where-Object {$_.Name -Eq $vmName }   \n                #check if VM Exists\n                if($VM -ne $null)\n                {     \n                          if($diskdriveName -ne $null)\n                          {\n                             #gets the specified volume object\n                             $diskdrive=Get-SCVirtualDiskDrive -VM $VM | Where-Object { $_.VirtualHardDisk.Name -Eq $diskdriveName} \n                             if($diskdrive -eq $null)\n                             {\n                               $diskdrive=Get-SCVirtualDiskDrive -VM $VM | Where-Object { $_.VirtualHardDisk.ParentDisk.Name -Eq $diskdriveName}\n                               if($diskdrive -eq $null) {\n                                    Write-Error \"Virtual Disk Not Found\"\n                               }\n                             }\n\n                          }  \n                          else\n                           {\n                                echo \"Name of the disk drive  is not entered\"\n                           }                  \n\u0009\u0009\u0009\u0009\u0009   \n                }\n                else\n                {\n                    Write-Error \"VM is not exists\"\n                }\n                \n            \n         }\n\u0009     catch [Exception]\n\u0009       {\n\u0009\u0009        echo $_.Exception.Message\n\u0009        }\n       \n        }\n    }\n    $j = Start-Job -ScriptBlock $code -ArgumentList $vmmServer, $vmName, $diskdriveName\n    if (Wait-Job $j -Timeout $timeval) \n    { \n        Receive-Job $j \n    } \n    else \n    {\n        Remove-Job -force $j\n        echo \"time out\"\n        exit 1\n    }\n\n}\n\n"
	filename := "readVD"
	arguments := timeout + " \"" + vmmServer + "\" \"" + vmName + "\" \"" + volumeName + "\""
	_, err := execScript(connection, script, filename, arguments)
	if err != "" {
		log.Printf("[Error] Error in reading Virtual Disk : %s ", err)
		d.SetId("")
	}

	return nil
}

func resourceDeleteVirtualDisk(d *schema.ResourceData, meta interface{}) error {
	log.Println("[INFO] Checking if Virtual Disk Exists")
	resourceReadVirtualDisk(d, meta)
	if d.Id() == "" {
		log.Println("[Error] Virtual Disk is not available")
		return fmt.Errorf("[Error] Virtual Disk does not Exist")
	}
	log.Println("[INFO] Virtual Disk Found")
	connection := meta.(*winrm.Client)
	vmName := d.Get("vm_name").(string)
	vmmServer := d.Get("vmm_server").(string)
	volumeName := d.Get("virtual_disk_name").(string)
	timeout := d.Get("timeout").(string)
	script := "\n[CmdletBinding(SupportsShouldProcess=$true)]\nparam (\n  [parameter(Mandatory=$true,HelpMessage=\"Enter Timeout Value\")]\n  [string]$timeval,\n\n  \n  [Parameter(Mandatory=$true,HelpMessage=\"Enter VmmServer\")]\n  [string]$vmmServer,\n\n  [Parameter(Mandatory=$true,HelpMessage=\"Enter VM Name\")]\n  [string]$vmName,\n\n   [Parameter(Mandatory=$true,HelpMessage=\"Enter Volume Name\")]\n  [string]$diskdriveName\n)\n\nBegin\n{\n    \n\n    $code = \n    {\n        [CmdletBinding(SupportsShouldProcess=$true)]\n        param (\n            \n            [Parameter(Mandatory=$true,HelpMessage=\"Enter VmmServer\")]\n            [string]$vmmServer,\n\n            [Parameter(Mandatory=$true,HelpMessage=\"Enter VM Name\")]\n            [string]$vmName,\n\n            [Parameter(Mandatory=$true,HelpMessage=\"Enter Volume Name\")]\n            [string]$diskdriveName\n        )\n        Begin\n        {\n            If (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] \"Administrator\"))\n          {   \n                $arguments = \"\" + $myinvocation.mycommand.definition + \" \"\n                $myinvocation.BoundParameters.Values | foreach{$arguments += \"'$_' \" }\n            echo $arguments\n            Start-Process powershell -Verb runAs -ArgumentList $arguments\n            Break\n         }\n\u0009    try\n\u0009     {     if($vmName -eq $null) \n               {\n                    echo \"VM Name not entered\"\n                    exit\n                } \n                #gets virtual machine objects from the Virtual Machine Manager database\n                Set-SCVMMServer -VMMServer $vmmServer\n\u0009\u0009$VM = Get-SCVirtualMachine | Where-Object {$_.Name -Eq $vmName }   \n                #check if VM Exists\n                if($VM -ne $null)\n                {     \n                          if($diskdriveName -ne $null)\n                          {\n                             #gets the specified volume object\n                             $diskdrive=Get-SCVirtualDiskDrive -VM $VM | Where-Object { $_.VirtualHardDisk.Name -Eq $diskdriveName} \n                             if($diskdrive -eq $null)\n                             {\n                               $diskdrive=Get-SCVirtualDiskDrive -VM $VM | Where-Object { $_.VirtualHardDisk.ParentDisk.Name -Eq $diskdriveName}\n                               if($diskdrive -eq $null) {\n                                    Write-Error \"Virtual Disk Not Found\"\n                               } \n                               else\n                               {\n                                    Remove-SCVirtualDiskDrive -VirtualDiskDrive $diskdrive\n                               }\n                             } \n                             else \n                             {\n                                Remove-SCVirtualDiskDrive -VirtualDiskDrive $diskdrive\n                             }\n\n                          }  \n                          else\n                           {\n                                echo \"Name of the disk drive  is not entered\"\n                           }                  \n\u0009\u0009\u0009\u0009\u0009   \n                }\n                else\n                {\n                    Write-Error \"VM is not exists\"\n                }\n                \n            \n         }\n\u0009     catch [Exception]\n\u0009       {\n\u0009\u0009        echo $_.Exception.Message\n\u0009        }\n       \n        }\n    }\n    $j = Start-Job -ScriptBlock $code -ArgumentList $vmmServer, $vmName, $diskdriveName\n    if (Wait-Job $j -Timeout $timeval) \n    { \n        Receive-Job $j \n    } \n    else \n    {\n        Remove-Job -force $j\n        echo \"time out\"\n        exit 1\n    }\n\n}\n\n"
	filename := "deleteVD"
	arguments := " " + timeout + " \"" + vmmServer + "\" \"" + vmName + "\" \"" + volumeName + "\""
	_, err := execScript(connection, script, filename, arguments)
	if err != "" {
		log.Printf("[Error] Error deleting Virtual Disk : %s ", err)
		return fmt.Errorf("[Error] Error deleting Virtual Disk : %s", err)
	}
	if err == "" {
		d.SetId("")
	}
	return nil
}
