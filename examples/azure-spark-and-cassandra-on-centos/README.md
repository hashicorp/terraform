# Spark & Cassandra on CentOS 7.x

This Terraform template was based on [this](https://github.com/Azure/azure-quickstart-templates/tree/master/spark-and-cassandra-on-centos) Azure Quickstart Template. Changes to the ARM template that may have occurred since the creation of this example may not be reflected here.

This project configures a Spark cluster (1 master and n-slave nodes) and a single node Cassandra on Azure using CentOS 7.x.  The base image starts with CentOS 7.3, and it is updated to the latest version as part of the provisioning steps.

Please note that [Azure Resource Manager][3] is used to provision the environment.

### Software ###

| Category | Software | Version | Notes |
| --- | --- | --- | --- |
| Operating System | CentOS | 7.x | Based on CentOS 7.1 but it will be auto upgraded to the lastest point release |
| Java | OpenJDK | 1.8.0 | Installed on all servers |
| Spark | Spark | 1.6.0 with Hadoop 2.6 | The installation contains libraries needed for Hadoop 2.6 |
| Cassandra | Cassandra | 3.2 | Installed through DataStax's YUM repository |


### Defaults ###

| Component | Setting | Default | Notes |
| --- | --- | --- | --- |
| Spark - Master | VM Size | Standard D1 V2 | |
| Spark - Master | Storage | Standard LRS | |
| Spark - Master | Internal IP | 10.0.0.5 | |
| Spark - Master | Service User Account | spark | Password-less access |
| | | |
| Spark - Slave | VM Size | Standard D3 V2 | |
| Spark - Slave | Storage | Standard LRS | |
| Spark - Slave | Internal IP Range | 10.0.1.5 - 10.0.1.255 | |
| Spark - Slave | # of Nodes | 2 | Maximum of 200 |
| Spark - Slave | Availability | 2 fault domains, 5 update domains | |
| Spark - Slave | Service User Account | spark | Password-less access |
| | | |
| Cassandra | VM Size | Standard D3 V2 | |
| Cassandra | Storage | Standard LRS | |
| Cassandra | Internal IP | 10.2.0.5 | |
| Cassandra | Service User Account | cassandra | Password-less access |

## Prerequisites

1.  Ensure you have an Azure subscription.  
2.  Ensure you have enough available vCPU cores on your subscription.  Otherwise, you will receive an error during the process.  The number of cores can be increased through a support ticket in Azure Portal.

## main.tf
The `main.tf` file contains the actual resources that will be deployed. It also contains the Azure Resource Group definition and any defined variables.

## outputs.tf
This data is outputted when `terraform apply` is called, and can be queried using the `terraform output` command.

## provider.tf
Azure requires that an application is added to Azure Active Directory to generate the `client_id`, `client_secret`, and `tenant_id` needed by Terraform (`subscription_id` can be recovered from your Azure account details). Please go [here](https://www.terraform.io/docs/providers/azurerm/) for full instructions on how to create this to populate your `provider.tf` file.

## terraform.tfvars
If a `terraform.tfvars` or any `.auto.tfvars` files are present in the current directory, Terraform automatically loads them to populate variables. We don't recommend saving usernames and password to version control, but you can create a local secret variables file and use the `-var-file` flag or the `.auto.tfvars` extension to load it.

If you are committing this template to source control, please insure that you add this file to your `.gitignore` file.

## variables.tf
The `variables.tf` file contains all of the input parameters that the user can specify when deploying this Terraform template.

## Post-Deployment

1. All servers will have a public IP and SSH port enabled by default. These can be disabled or modified in the template or by using Azure Portal.
2. All servers are configured with the same username and password. You may SSH into each server and ensure connectivity.
3. Spark WebUI is running on **port 8080**.  Access it using MASTER_WEB_UI_PUBLIC_IP:8080 on your browser.  Public IP is available in the outputs as well as through Azure Portal.
4. Delete the Resource Group that was created to stage the provisioning scripts.
