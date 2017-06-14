# Azure traffic manager with load balanced scale sets

This example shows how to create a load balanced scale set in multiple locations and then geographically load balance these using traffic manager. This example the scale set uses a market place Ubuntu image, this could be customised using an extension or a generalized image created using packer. 

This script demonstrates how variable can be passed in and out of reusable modules. You will need to run `terraform get` for terrafrom to get so that modules are pre-processed.

## Keys and variables

To use this you will need to populate the `terraform.tfvars.example` file with your Azure credentials and key. Rename this to `terraform.tfvars` and copy this somewhere private. If you need to generate credentials follow the instructions on the Azure provider documented [here](https://www.terraform.io/docs/providers/azurerm)

You may also want to modify some of the settings in `variables.tf`, DNS names must be unique within an Azure location and globally for traffic management 

## To start the script

### Planning 

`terraform get`

`terraform plan -var-file="C:\Users\eltimmo\.terraform\keys.tfvars"`

### Apply phase

`terraform apply -var-file="C:\Users\eltimmo\.terraform\keys.tfvars"`

### Destroy

`terraform destroy -var-file="C:\Users\eltimmo\.terraform\keys.tfvars"`
