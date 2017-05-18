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
* Source image blob full URL. e.g. https://pmcstorage01.blob.core.windows.net/images/images/Win10MasterImage-osDisk.72451a98-4c26-4375-90c5-0a940dd56bab.vhd. Note that container name always comes after  https://pmcstorage01.blob.core.windows.net, in this example it is images. The actual blob name is **images/Win10MasterImage-osDisk.72451a98-4c26-4375-90c5-0a940dd56bab.vhd**.

![graph](/examples/azure-vm-custom-image-new-storage-account/graph.png)