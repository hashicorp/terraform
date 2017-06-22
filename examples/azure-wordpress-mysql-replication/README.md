# Deploys a WordPress web site backed by MySQL master-slave replication

This Terraform template was based on [this](https://github.com/Azure/azure-quickstart-templates/tree/master/wordpress-mysql-replication) Azure Quickstart Template. Changes to the ARM template that may have occurred since the creation of this example may not be reflected here.

This template deploys a WordPress site in Azure backed by MySQL replication with one master and one slave server.  It has the following capabilities:

- Installs and configures GTID based MySQL replication on CentOS 6
- Deploys a load balancer in front of the 2 MySQL VMs 
- MySQL, SSH, and MySQL probe ports are exposed through the load balancer using Network Security Group rules.  
- WordPress accesses MySQL through the load balancer.
- Configures an http based health probe for each MySQL instance that can be used to monitor MySQL health.
- WordPress deployment starts immediately after MySQL deployment finishes.
- Details about MySQL management, including failover, can be found [here](https://github.com/Azure/azure-quickstart-templates/tree/master/mysql-replication).

If you would like to leverage an existing VNET, then please see the [documentation here](https://www.terraform.io/docs/import/index.html) to learn about importing existing resources into Terraform and bringing them under state management by this template. To import your existing VNET, you may use this command.

```
terraform import azurerm_virtual_network.testNetwork /subscriptions/<YOUR-SUB-ID-HERE>/resourceGroups/<existing-resource-group-name>/providers/Microsoft.Network/virtualNetworks/<existing-vnet-name>
```

## main.tf
The `main.tf` file contains the resources necessary for the MySql replication deployment that will be created. It also contains the Azure Resource Group definition and any defined variables.

## website.tf
The `website.tf` contains an `azurerm_template_deployment` that will deploy the Wordpress website.

## outputs.tf
This data is outputted when `terraform apply` is called, and can be queried using the `terraform output` command.

## provider.tf
You may leave the provider block in the `main.tf`, as it is in this template, or you can create a file called `provider.tf` and add it to your `.gitignore` file.

Azure requires that an application is added to Azure Active Directory to generate the `client_id`, `client_secret`, and `tenant_id` needed by Terraform (`subscription_id` can be recovered from your Azure account details). Please go [here](https://www.terraform.io/docs/providers/azurerm/) for full instructions on how to create this to populate your `provider.tf` file.

## terraform.tfvars
If a `terraform.tfvars` or any `.auto.tfvars` files are present in the current directory, Terraform automatically loads them to populate variables. We don't recommend saving usernames and password to version control, but you can create a local secret variables file and use the `-var-file` flag or the `.auto.tfvars` extension to load it.

If you are committing this template to source control, please insure that you add this file to your `.gitignore` file.

## variables.tf
The `variables.tf` file contains all of the input parameters that the user can specify when deploying this Terraform template.
