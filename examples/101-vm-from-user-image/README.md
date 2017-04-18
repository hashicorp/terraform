# Create a Virtual Machine from a User Image
**Prerequisite - The Storage Account with the User Image VHD should already exist**

This template allows you to create a Virtual Machines from a User image. This template also deploys a Virtual Network, Public IP addresses and a Network Interface.

Azure requires that an application is added to Azure Active Directory to generate the client_id, client_secret, and tenant_id needed by Terraform (subscription_id can be recovered from your Azure account details). Please go [here](https://www.terraform.io/docs/providers/azurerm/) for full instructions on how to create this.

