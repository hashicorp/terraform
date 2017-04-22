# [Create a Virtual Machine from a User Image](https://docs.microsoft.com/en-us/azure/virtual-machines/linux/cli-deploy-templates#create-a-custom-vm-image)

<a href="https://portal.azure.com/#create/Microsoft.Template/uri/https%3A%2F%2Fraw.githubusercontent.com%2FAzure%2Fazure-quickstart-templates%2Fmaster%2F101-vm-from-user-image%2Fazuredeploy.json" target="_blank">
    <img src="http://azuredeploy.net/deploybutton.png"/>
</a>
<a href="http://armviz.io/#/?load=https%3A%2F%2Fraw.githubusercontent.com%2FAzure%2Fazure-quickstart-templates%2Fmaster%2F101-vm-from-user-image%2Fazuredeploy.json" target="_blank">
    <img src="http://armviz.io/visualizebutton.png"/>
</a>

> Prerequisite - The generalized image VHD should exist, as well as a Storage Account for boot diagnostics

This template allows you to create a Virtual Machine from an unmanaged User image vhd. This template also deploys a Virtual Network, Public IP addresses and a Network Interface.

If you are looking to accomplish the above scenario through PowerShell instead of a template, you can use a PowerShell script like below

##### Variables
    ## Global
    $rgName = "testrg"
    $location = "westus"

    ## Storage
    $storageName = "teststore"
    $storageType = "Standard_GRS"

    ## Network
    $nicname = "testnic"
    $subnet1Name = "subnet1"
    $vnetName = "testnet"
    $vnetAddressPrefix = "10.0.0.0/16"
    $vnetSubnetAddressPrefix = "10.0.0.0/24"

    ## Compute
    $vmName = "testvm"
    $computerName = "testcomputer"
    $vmSize = "Standard_A2"
    $osDiskName = $vmName + "osDisk"

##### Resource Group
    New-AzureRmResourceGroup -Name $rgName -Location $location

##### Storage
    $storageacc = New-AzureRmStorageAccount -ResourceGroupName $rgName -Name $storageName -Type $storageType -Location $location

##### Network
    $pip = New-AzureRmPublicIpAddress -Name $nicname -ResourceGroupName $rgName -Location $location -AllocationMethod Dynamic
    $subnetconfig = New-AzureRmVirtualNetworkSubnetConfig -Name $subnet1Name -AddressPrefix $vnetSubnetAddressPrefix
    $vnet = New-AzureRmVirtualNetwork -Name $vnetName -ResourceGroupName $rgName -Location $location -AddressPrefix $vnetAddressPrefix -Subnet $subnetconfig
    $nic = New-AzureRmNetworkInterface -Name $nicname -ResourceGroupName $rgName -Location $location -SubnetId $vnet.Subnets[0].Id -PublicIpAddressId $pip.Id

##### Compute
    ## Setup local VM object
    $cred = Get-Credential
    $vm = New-AzureRmVMConfig -VMName $vmName -VMSize $vmSize
    $vm = Set-AzureRmVMOperatingSystem -VM $vm -Windows -ComputerName $computerName -Credential $cred -ProvisionVMAgent -EnableAutoUpdate

    $vm = Add-AzureRmVMNetworkInterface -VM $vm -Id $nic.Id

    $osDiskUri = "http://test.blob.core.windows.net/vmcontainer10798c80-131-1231-a94a-f9d2a712251f/osDisk.10798c80-2919-4100-a94a-f9d2a712251f.vhd"
    $imageUri = "http://test.blob.core.windows.net/system/Microsoft.Compute/Images/captured/image-osDisk.8b021d87-913c-4f94-a01a-944ad92d7388.vhd"
    $vm = Set-AzureRmVMOSDisk -VM $vm -Name $osDiskName -VhdUri $osDiskUri -CreateOption fromImage -SourceImageUri $imageUri -Windows

    $dataImageUri = "http://test.blob.core.windows.net/system/Microsoft.Compute/Images/captured/image-dataDisk-0.8b021d87-913c-4f94-a01a-944ad92d7388.vhd"
    $dataDiskUri = "http://test.blob.core.windows.net/vmcontainer10798c80-sa11-41sa-dsad-f9d2a712251f/dataDisk-0.10798c80-2919-4100-a94a-f9d2a712251f.vhd"
    $vm = Add-AzureRmVMDataDisk -VM $vm -Name "dd1" -VhdUri $dataDiskUri -SourceImageUri $dataImageUri -Lun 0 -CreateOption fromImage

    ## Create the VM in Azure
    New-AzureRmVM -ResourceGroupName $rgName -Location $location -VM $vm -Verbose


## azuredeploy.tf
The `azuredeploy.tf` file contains the actual resources that will be deployed. It also contains the Azure Resource Group definition and any defined variables. 

## outputs.tf
This data is outputted when `terraform apply` is called, and can be queried using the `terraform output` command.

## provider.tf
Azure requires that an application is added to Azure Active Directory to generate the `client_id`, `client_secret`, and `tenant_id` needed by Terraform (`subscription_id` can be recovered from your Azure account details). Please go [here](https://www.terraform.io/docs/providers/azurerm/) for full instructions on how to create this to populate your `provider.tf` file.

## terraform.tfvars
If a `terraform.tfvars` file is present in the current directory, Terraform automatically loads it to populate variables. We don't recommend saving usernames and password to version control, but you can create a local secret variables file and use `-var-file` to load it.

## variables.tf
The `variables.tf` file contains all of the input parameters that the user can specify when deploying this Terraform template.

## .gitignore
If you are committing this template to source control, please insure that the following files are added to your `.gitignore` file.

```
terraform.tfstate*
terraform.tfvars*
provider.tf*
```
