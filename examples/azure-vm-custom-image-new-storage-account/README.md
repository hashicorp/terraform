# Create a new VM on a new storage account from a custom image

This Terraform template was based on [this](https://github.com/Azure/azure-quickstart-templates/tree/master/201-vm-custom-image-new-storage-account) Azure Quickstart Template. Changes to the ARM template that may have occurred since the creation of this example may not be reflected here.

This template allows you to create a new Virtual Machine from a custom image on a new storage account deployed together with the storage account, which means the source image VHD must be transferred to the newly created storage account before that Virtual Machine is deployed. This is accomplished by the usage of a transfer virtual machine that is deployed and then uses a script via custom script extension to copy the source VHD to the destination storage account. This process is used to overcome the limitation of the custom VHD that needs to reside at the same storage account where new virtual machines based on it will be spun up, the problem arises when you are also deploying the storage account within your template, since the storage account does not exist yet, how can you add the source VHDs beforehand?

Basically, it creates two VMs, one that is the transfer virtual machine and the second that is the actual virtual machine that is the goal of the deployment. Transfer VM can be removed later.

The process of this template is:

1. A Virtual Network is deployed
2. Virtual NICs for both Virtual Machines
3. Storage Account is created
3. Transfer Virtual Machine gets deployed
4. Transfer Virtual Machine starts the custom script extension to start the VHD copy from source to destination storage acounts
5. The new Virtual Machine based on a custom image VHD gets deployed 

## Requirements

* A preexisting generalized (sysprepped) Windows image. For more information on how to create custom Windows images, please refer to [How to capture a Windows virtual machine in the Resource Manager deployment model](https://azure.microsoft.com/en-us/documentation/articles/virtual-machines-windows-capture-image/) article.
* Source image blob full URL. E.g. https://pmcstorage01.blob.core.windows.net/images/images/Win10MasterImage-osDisk.72451a98-4c26-4375-90c5-0a940dd56bab.vhd. Note that container name always comes after  https://pmcstorage01.blob.core.windows.net, in this example it is images. The actual blob name is **images/Win10MasterImage-osDisk.72451a98-4c26-4375-90c5-0a940dd56bab.vhd**.

## How to deploy this template from Powershell

###### Deploying using existing parameters file (azuredeploy.parameters.json)

1. Modify `azuredeploy.parameters.json` parameters file accordingly, be aware that this method can expose your local admin credential since it is defined in the parameters file.

2. Open Powershell command prompt, change folder to your template folder.

3. Authenticate to this session

  ```powershell
  Add-AzureRmAccount
  ```

4. Create the new Resource Group where your deployment will happen

  ```powershell
  New-AzureRmResourceGroup -Name "myResourceGroupName" -Location "centralus"
  ```

5. Deploy your template

  ```powershell
  New-AzureRmResourceGroupDeployment -Name "myDeploymentName" `
                                     -ResourceGroupName "myResourceGroupName" `
                                     -Mode Incremental `
                                     -TemplateFile .\azuredeploy.json `
                                     -TemplateParameterFile .\azuredeploy.parameters.json `
                                     -Force -Verbose 
  ```                                     

###### Deploying without using existing parameters file (azuredeploy.parameters.json)

1. Open Powershell command prompt, change folder to your template folder.

2. Authenticate to this session

  ```powershell
  Add-AzureRmAccount
  ```

3. Create the new Resource Group where your deployment will happen

  ```powershell
  New-AzureRmResourceGroup -Name "myResourceGroupName" -Location "centralus"
  ```

4. Obtain a credential object that will be used to define your local administrator name and password. Note that you can define those variables in clear text too, but this is not recommended.

  ```powershell
  $credential = Get-Credential 
  ```

5. (Optional if you know the image name already) Getting source storage account authorization Key. Note that this is an automated way to get this key, you can get it directly from the new portal and define the content directly. 
  
  If your source storage account is based on Azure Service Manager management model:
  ```powershell
  $sourceStorageAccountName = "mysourcestorageaccount"
  $storageKey = (Get-AzureStorageKey -StorageAccountName $sourceStorageAccountName).Primary
  $saContext = New-AzureStorageContext -StorageAccountName $sourceStorageAccountName -StorageAccountKey $storageKey
  $sourceVhdContainer = "images"
  ```
  
  If your source storage account is based on Azure Resource Manager management model:
  ```powershell
  $sourceStorageAccountName = "mysourcestorageaccount"
  $sourceSaResourceGroupName = "myRgWhereSaResides"
  $storageKey = (Get-AzureRmStorageAccountKey -StorageAccountName $sourceStorageAccountName -ResourceGroup $sourceSaResourceGroupName).Key1
  $saContext = New-AzureStorageContext -StorageAccountName $sourceStorageAccountName -StorageAccountKey $storagekey
  $sourceVhdContainer = "images"
  ```

6. (Optional if you know the image name already) How to list blobs from Powershell, previous step is mandatory to obtain the storage account context object. In this example the custom images resides in **images** container.
  
  ```powershell
  $vhds = Get-AzureStorageBlob -Container $sourceVhdContainer -Context $saContext -Blob *.vhd
  $vhds | Format-Table -AutoSize
  ```
  
  In this output, Name column is the image name to be used
  ```
  Name                                                                      BlobType Length       ContentType              LastModified               SnapshotTime
  ----                                                                      -------- ------       -----------              ------------               ------------
  images/SqlSccmMasterImage-osDisk.37383203-eeba-414c-a2ea-c7be33f970fa.vhd PageBlob 136367309312 application/octet-stream 3/3/2016 3:13:25 PM +00:00
  images/Win10MasterImage-osDisk.72451a98-4c26-4375-90c5-0a940dd56bab.vhd   PageBlob 136367309312 application/octet-stream 3/3/2016 3:36:09 PM +00:00
  ```

7. Since we already have all necessary information, define the remaining variables required to deploy this template
  
  ```powershell
  $adminUserName = $credential.UserName
  $adminPassword = $credential.GetNetworkCredential().Password
  # Following line is the equivalent of defining "images/Win10MasterImage-osDisk.72451a98-4c26-4375-90c5-0a940dd56bab.vhd", but here we executed optional steps 5 and 6 and have an array of vhds, we are picking the second vhd 
  $customImageName = $vhds[1].Name  
  $sourceImageUri = [string]::Format("http://{0}.blob.core.windows.net/{1}/{2}",$sourceStorageAccountName,$sourceVhdContainer,$customImageName)
  $transferVmName = "myTransferVm"
  $newVmName = "myNewWin10Vm"
  $vmSize = "Standard_D1"
  $sourceStorageAccountResourceGroup = $sourceSaResourceGroupName # if you exected step 5 then you can use this variable, otherwise just add the storage account resource group name as string
  ```
  
8. Define a hashtable with all parameters
  
  ```powershell
  $parameters = @{"AdminUsername"=$adminUserName;"AdminPassword"=$adminPassword;"sourceStorageAccountResourceGroup"=$sourceStorageAccountResourceGroup;"CustomImageName"=$CustomImageName;"sourceImageUri"=$sourceImageUri;"TransferVmName"=$transferVmName;"NewVmName"=$newVmName;"vmSize"=$vmSize}
  ```
  
9. Deploy your template

  ```powershell
  New-AzureRmResourceGroupDeployment -Name "myDeploymentName" `
				                     -ResourceGroupName "myResourceGroupName" `
				                     -Mode Incremental `
			                         -TemplateFile .\azuredeploy.json `
			                         -TemplateParameterObject $parameters `
				                     -Force -Verbose 
  ```                                     