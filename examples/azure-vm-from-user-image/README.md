# [Create a Virtual Machine from a User Image](https://docs.microsoft.com/en-us/azure/virtual-machines/linux/cli-deploy-templates#create-a-custom-vm-image)
**Prerequisite - The Storage Account with the User Image VHD should already exist**

This template allows you to create a Virtual Machines from a User image. This template also deploys a Virtual Network, Public IP addresses and a Network Interface.

Azure requires that an application is added to Azure Active Directory to generate the client_id, client_secret, and tenant_id needed by Terraform (subscription_id can be recovered from your Azure account details). Please go [here](https://www.terraform.io/docs/providers/azurerm/) for full instructions on how to create this.

`image_uri` - Specifies the `image_uri` in the form publisherName:offer:skus:version. `image_uri` can also specify the VHD uri of a custom VM image to clone.
`os_type` -  When cloning a custom disk image the `os_type` documented below becomes required. Specifies the operating system Type, valid values are windows, linux. 
