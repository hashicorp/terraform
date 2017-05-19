package scvmm

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/masterzen/winrm"
)

func resourceSCVMMVM() *schema.Resource {
	return &schema.Resource{
		Create: resourceVMCreate,
		Read:   resourceVMRead,
		Delete: resourceVMDelete,
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
			"template_name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateTemplateName,
			},
			"cloud_name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateCloudName,
			},
		},
	}
}

func resourceVMCreate(d *schema.ResourceData, metadata interface{}) error {

	connection := metadata.(*winrm.Client)
	vmmServer := d.Get("vmm_server").(string)
	templateName := d.Get("template_name").(string)
	cloudName := d.Get("cloud_name").(string)
	vmName := d.Get("vm_name").(string)

	//Validation for VM Config compatibilty for given vm server and template
	log.Println("[INFO] Performing check if requested Specs are feasible in given VMM Server")
	validationScript := "[CmdletBinding(SupportsShouldProcess=$true)]\nparam(\n    [parameter(Mandatory=$true,HelpMessage=\"Enter VMMServer\")]\n\u0009[string]$VMMServer,\n\n    [parameter(Mandatory=$true,HelpMessage=\"Enter VMMServer\")]\n\u0009[string]$vmName,\n\n    [parameter(Mandatory=$true,HelpMessage=\"Enter VMMServer\")]\n\u0009[string]$VMTemplateId\n)\n\n\nBegin\n{\n    If (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] \"Administrator\"))\n            {   \n                $arguments = \"\" + $myinvocation.mycommand.definition + \" \"\n                $myinvocation.BoundParameters.Values | foreach{\n                $arguments += \"'$_' \"\n            }\n            echo $arguments\n            Start-Process powershell -Verb runAs -ArgumentList $arguments\n            Break\n        }\n\u0009    try\n\u0009    {\n    $vm= Get-SCVirtualMachine -VMMServer $VMMServer -Name $vmName\n    if($vm -ne $null) {\n        $errorMsg = [string]::Format(\"VM with Name: {0} already Exists!\",$vmName)\n        Write-Error $errorMsg\n    }\n    $template = Get-SCVMTemplate -VMMServer $VMMServer -Name $VMTemplateId\n    if($template -eq $null) {\n        $errorMsg = [string]::Format(\"Cannot get template with ID: {0}!\",$VMTemplateId)\n        Write-Error $errorMsg\n    }\n    $VMHost=Get-SCVMHost -ComputerName $VMMServer\n    if($VMHost -eq $null){\n        $errorMsg = [string]::Format(\"VMHost {0} is not available\",$VMMServer)\n        Write-Error $errorMsg\n    }\n    $SCVMHost = Read-SCVMHost -VMHost $VMHost\n    if($SCVMHost -eq $null){\n        Write-Error \"Cannot Read SCVMHost\"\n    }\n    $LS = Get-SCLibraryServer -VMMServer $VMMServer\n    \n    if($LS -eq $null){\n        Write-Host \"Cannot get Library Server\"\n    }\n    \n    $rating  = Get-SCLibraryRating -LibraryServer $LS\n    if($rating -eq $null){\n    Write-Host \"Cannot get rating\"\n    }\n    Write-Host \"Rating is \" $rating.Rating\n    \n    if(-not $SCVMHost.AvailableForPlacement){\n        $errorMsg = [string]::Format(\"VMMServer {0} Not Available for placement\",$VMMServer)\n        Write-Error $errorMsg\n    }\n    if($SCVMHost.AvailableMemory -lt ($template.Memory + 64)){\n        $errorMsg = [string]::Format(\"Memory is not sufficient. Available Memory : {0} Requested Memory : {1}\",$SCVMHost.AvailableMemory,($template.Memory + 64))\n        Write-Error $errorMsg\n    }\n    \n    if(($SCVMHost.TotalStorageCapacity * 0.8) -le $SCVMHost.UsedStorageCapacity){\n        $errorMsg = [string]::Format(\"Some Space is resrved for Dynamic disks to Grow Available Disk Space : {0} Total Disk Space Used : {1}\",($SCVMHost.AvailableStorageCapacity/(1024*1024)), ($SCVMHost.UsedStorageCapacity/(1024*1024)))\n        Write-Error $errorMsg       \n    }\n    \n    $maxSize = 0\n    $template.VirtualDiskDrives | foreach {$maxSize = $maxSize + $_.VirtualHardDisk.MaximumSize;}\n    if(($SCVMHost.AvailableStorageCapacity/(1024*1024) - 200) -lt ($maxSize/(1024*1024))){\n        $errorMsg = [string]::Format(\"Disk Storage Not Sufficient Available Disk Space : {0} Requested Disk Space : {1}\",($SCVMHost.AvailableStorageCapacity/(1024*1024) - 200), ($maxSize/(1024*1024)))\n        Write-Error $errorMsg\n    }\n    \n    if($SCVMHost.CoresPerCPU -lt $template.CPUCount){\n        $errorMsg = [string]::Format(\"No of cpu are less than cores required Cores Available: {0} Cores Requested : {1}\",$SCVMHost.CoresPerCPU, $template.CPUCount)\n        Write-Error $errorMsg\n    }\n    }\n    catch [Exception]\n\u0009{\n\u0009\u0009        Write-Error $_.Exception.Message\n\u0009}\n}\n\n"
	validationFileName := "validateCreateVM"
	validationArguments := "\"" + vmmServer + "\" \"" + vmName + "\" \"" + templateName + "\""
	_, validationError := execScript(connection, validationScript, validationFileName, validationArguments)
	if validationError != "" {
		log.Printf("[Error] Error in executing  command : %s", validationError)
		return fmt.Errorf("[Error] Error in executing  command : %s", validationError)
	}
	log.Println("[INFO] Validation successful")

	script := "[CmdletBinding(SupportsShouldProcess=$true)]\nparam(\n\u0009[parameter(Mandatory=$true,HelpMessage=\"Enter Template Name\")]\n\u0009[string]$tempName,\n\u0009[parameter(Mandatory=$true,HelpMessage=\"Enter VMMServer\")]\n\u0009[string]$VMMServer,\n\u0009[parameter(Mandatory=$true,HelpMessage=\"Enter Cloud Name\")]\n\u0009[string]$cloudName,\n\u0009[parameter(Mandatory=$true,HelpMessage=\"Enter Name\")]\n\u0009[string]$Name\n)\nBegin {\n\n    If (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] \"Administrator\"))\n    {   $arguments = \"\" + $myinvocation.mycommand.definition + \" \"\n        $myinvocation.BoundParameters.Values | foreach{\n            $arguments += \"'$_' \"\n        }\n        echo $arguments\n        Start-Process powershell -Verb runAs -ArgumentList $arguments\n        Break\n    }\n\u0009try\n\u0009{\n\u0009\u0009\u0009$setServer = Set-SCVMMServer -VMMServer $VMMServer\n\n\u0009\u0009\u0009$VMTemp = Get-SCVMTemplate -Name  $tempName\n\n\u0009\u0009\u0009$virtualMachineConfiguration = New-SCVMConfiguration -VMTemplate $VMTemp -Name $Name\n\n\u0009\u0009\u0009$cloud = Get-SCCloud -Name $cloudName\n\n\u0009\u0009\u0009$createVM = New-SCVirtualMachine -Name $Name -VMConfiguration $virtualMachineConfiguration -Cloud $cloud -Description \"\"  -UseDiffDiskOptimization -StartAction \"NeverAutoTurnOnVM\" -StopAction \"SaveVM\"\n\n\u0009\u0009\u0009echo $createVM | Select ID\n\u0009}\n\u0009catch [Exception]\n\u0009{\n\u0009\u0009Write-Error $_.Exception.Message\n\u0009}\n\n}"
	filename := "createVM"
	arguments := " \"" + templateName + "\" \"" + vmmServer + "\" \"" + cloudName + "\" \"" + vmName + "\""
	_, err := execScript(connection, script, filename, arguments)
	if err != "" {
		log.Printf("[Error] Error in creating Virtual Machine : %s ", err)
		return fmt.Errorf("Error in creating Virtual Machine : %s", err)
	}

	if err == "" {
		terraformID := vmmServer + "_" + cloudName + "_" + vmName
		d.SetId(terraformID)
	}

	return nil
}

func resourceVMRead(d *schema.ResourceData, metadata interface{}) error {

	connection := metadata.(*winrm.Client)
	vmmServer := d.Get("vmm_server").(string)
	vmName := d.Get("vm_name").(string)
	timeout := d.Get("timeout").(string)
	script := "[CmdletBinding(SupportsShouldProcess=$true)]\nparam (\n  [parameter(Mandatory=$true,HelpMessage=\"Enter Timeout Value\")]\n  [string]$timeval,\n  [Parameter(Mandatory=$true,HelpMessage=\"Enter VmmServer\")]\n  [string]$vmmServer,\n  [Parameter(Mandatory=$true,HelpMessage=\"Enter VM Name\")]\n  [string]$vmName\n)\n\nBegin\n{\n    $code = \n    {\n        [CmdletBinding(SupportsShouldProcess=$true)]\n        param (\n\n            [Parameter(Mandatory=$true,HelpMessage=\"Enter VmmServer\")]\n            [string]$vmmServer,\n\n            [Parameter(Mandatory=$true,HelpMessage=\"Enter VM Name\")]\n            [string]$vmName\n\n        )\n        Begin\n        {\n            If (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] \"Administrator\"))\n          {   \n            $arguments = \"\" + $myinvocation.mycommand.definition + \" \"\n            $myinvocation.BoundParameters.Values | foreach{$arguments += \"'$_' \" }\n            echo $arguments\n            Start-Process powershell -Verb runAs -ArgumentList $arguments\n            Break\n         }\n\u0009     try\n\u0009     {\n               if($vmName -eq $null) \n               {\n                    echo \"VM Name not entered\"\n                    exit\n               } \n               #gets virtual machine objects from the Virtual Machine Manager database\n               Set-SCVMMServer -VMMServer $vmmServer > $null\n\u0009\u0009       $VM = Get-SCVirtualMachine | Where-Object {$_.Name -Eq $vmName }   \n               #check if VM Exists\n               if($VM -eq $null)\n               {     \n                   Write-Error \"VM does not exists\"\n               }\n            \n         }\n\u0009     catch [Exception]\n         {\n               Write-Error $_.Exception.Message\n\u0009     }\n      }\n    }\n    $j = Start-Job -ScriptBlock $code -ArgumentList $vmmServer, $vmName\n    if (Wait-Job $j -Timeout $timeval) \n    { \n        Receive-Job $j \n    } \n    else \n    {\n        Remove-Job -force $j\n        Write-Error \"Timeout\"\n        exit 1\n    }\n}"
	filename := "readVM"
	arguments := timeout + " \"" + vmmServer + "\" \"" + vmName + "\""
	_, err := execScript(connection, script, filename, arguments)

	if err != "" {
		log.Printf("[Error] Error in reading Virtual Machine : %s ", err)
		d.SetId("")
	}

	return nil
}

func resourceVMDelete(d *schema.ResourceData, metadata interface{}) error {
	log.Println("[INFO] Checking if VM Exists")
	resourceVMRead(d, metadata)
	if d.Id() == "" {
		log.Println("[Error] VM is not available")
		return fmt.Errorf("[Error] VM does not Exist")
	}
	log.Println("[INFO] VM Found")
	connection := metadata.(*winrm.Client)
	vmmServer := d.Get("vmm_server").(string)
	vmName := d.Get("vm_name").(string)
	timeout := d.Get("timeout").(string)
	script := "[CmdletBinding(SupportsShouldProcess=$true)]\nparam(\n    [parameter(Mandatory=$true,HelpMessage=\"Enter Timeout Value\")]\n    [string]$timeval,\n\u0009[parameter(Mandatory=$true,HelpMessage=\"Enter vmName\")]\n\u0009[string]$vmname,\n\u0009[parameter(Mandatory=$true,HelpMessage=\"Enter vmmServer\")]\n\u0009[string]$vmmServer\n)\nBegin\n{\n    If (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] \"Administrator\"))\n    {   $arguments = \"\" + $myinvocation.mycommand.definition + \" \"\n        $myinvocation.BoundParameters.Values | foreach{\n            $arguments += \"'$_' \"\n        }\n        echo $arguments\n        Start-Process powershell -Verb runAs -ArgumentList $arguments\n        Break\n    }\n\u0009try\n\u0009{\n\u0009\u0009$timeout = new-timespan -Seconds $timeval\n\u0009\u0009$sw = [diagnostics.stopwatch]::StartNew()\n\u0009\u0009while ($sw.elapsed -lt $timeout)\n\u0009\u0009{\n\u0009\u0009\u0009Set-SCvmmServer -vmmServer $vmmServer\n\u0009\u0009\u0009$VM = Get-SCVirtualMachine -Name $vmname\n\u0009\u0009\u0009if($VM.Status -ne \"PowerOff\")\n\u0009\u0009\u0009{\n\u0009\u0009\u0009\u0009Stop-VM -VM $VM \n\u0009\u0009\u0009}\n\u0009\u0009\u0009Remove-SCVirtualMachine -VM $VM\n\u0009\u0009\u0009if($sw.elapsed -lt $timeout)\n\u0009\u0009\u0009{break}\n\u0009\u0009\u0009return\n\u0009\u0009}\n\u0009\u0009write-host \"Timed out\"\n\u0009}\n\u0009catch [Exception]\n\u0009{\n\u0009\u0009echo $_.Exception.Message\n\u0009}\n}"
	filename := "deleteVM"
	arguments := timeout + " \"" + vmName + "\" \"" + vmmServer + "\""
	_, err := execScript(connection, script, filename, arguments)

	if err != "" {
		log.Printf("[Error] Error in deleting Virtual Machine: %s", err)
		return fmt.Errorf("[Error] Error in  deleting Virtual Machine: %s", err)
	}

	return nil
}
