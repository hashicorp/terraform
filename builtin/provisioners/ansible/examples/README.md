# Ansible provisioner for Terraform - examples

This repository contains [Ansible provisioner for Terraform](https://github.com/radekg/terraform-provisioner-ansible) examples.

Provided examples use AWS `eu-central-1`. The operating system is CentOS 7.

## Getting started

1. [Install the provisioner](https://github.com/radekg/terraform-provisioner-ansible#installation)
2. Create a key pair:
    
    ```
    $ ssh-keygen -t rsa -b 4096 -f ~/.ssh/terraform-provisioner-ansible
    Generating public/private rsa key pair.
    Enter passphrase (empty for no passphrase):
    Enter same passphrase again:
    Your identification has been saved in /Users/rad/.ssh/terraform-provisioner-ansible.
    Your public key has been saved in /Users/rad/.ssh/terraform-provisioner-ansible.pub.
    The key fingerprint is:
    SHA256:f3J9ZxkXUewW8MyhqnQagVoB4Kr68LoLQDDEVMXYxrk rad@noan.fritz.box
    The key's randomart image is:
    +---[RSA 4096]----+
    |*o.oBoo.     ..+o|
    |.o.. *  o     =.+|
    | . .. .o .   . *.|
    |. .  Eo   . .   +|
    |..   .  So o   o.|
    |o       ..=  .  +|
    |+        oo o ..+|
    |oo         +   o.|
    |==o              |
    +----[SHA256]-----+
    ```
    
3. Add the key to `ssh-agent`:
    
    ```
    $ ssh-add ~/.ssh/terraform-provisioner-ansible
    Enter passphrase for /Users/rad/.ssh/terraform-provisioner-ansible:
    Identity added: /Users/rad/.ssh/terraform-provisioner-ansible (/Users/rad/.ssh/terraform-provisioner-ansible)
    ```

4. Create an AWS profile in your `~/.aws/credentials` file:

    ```
    [terraform-provisioner-ansible]
    aws_access_key_id = AKIA...
    aws_secret_access_key = ...
    region = eu-central-1
    ```

5. Create base AMI:

    ```
    cd packer
    AWS_PROFILE=terraform-provisioner-ansible packer build base.json
    ```

6. Write down the AMI ID returned by packer. In my case, it was:

    ```
    ==> Builds finished. The artifacts of successful builds are:
    --> amazon-ebs: AMIs were created:
    eu-central-1: ami-07ceb11ca54d9d04a
    ```

## Examples

All examples execute a great task of installing `tree` on the bootstrapped host. Use AMI ID created with packer:

    export TERRAFORM_PROVISIONER_ANSIBLE_AMI_ID=ami...

After testing each of the examples, you will need to destroy the infrastructure. Examples share names but they don't share state.

1. `local-no-bastion`: run local provisioning for a host without a bastion
    
    ```
    cd local-no-bastion
    terraform apply -var "ami_id=${TERRAFORM_PROVISIONER_ANSIBLE_AMI_ID}"
    # ...
    terraform destroy -var "ami_id=${TERRAFORM_PROVISIONER_ANSIBLE_AMI_ID}"
    ```

2. `remote-no-bastion`: run remote provisioning for a host without a bastion

    ```
    cd remote-no-bastion
    terraform apply -var "ami_id=${TERRAFORM_PROVISIONER_ANSIBLE_AMI_ID}"
    # ...
    terraform destroy -var "ami_id=${TERRAFORM_PROVISIONER_ANSIBLE_AMI_ID}"
    ```

3. `local-with-bastion`: VPC setup, bastion, provision local over bastion
    
    ```
    cd local-with-bastion
    export R_NAME=terraform-provisioner-ansible
    export R_REGION=eu-central-1
    export R_VPC_CIDR_BLOCK=10.0.0.0/16
    terraform plan -var "ami_id=${TERRAFORM_PROVISIONER_ANSIBLE_AMI_ID}" \
        -var "region=${R_REGION}" \
        -var "aws_admin_profile=${R_NAME}" \
        -var "vpc_cidr_block=${R_VPC_CIDR_BLOCK}" \
        -var "infrastructure_name=${R_NAME}-local"
    # ...
    terraform apply -var "ami_id=${TERRAFORM_PROVISIONER_ANSIBLE_AMI_ID}" \
        -var "region=${R_REGION}" \
        -var "aws_admin_profile=${R_NAME}" \
        -var "vpc_cidr_block=${R_VPC_CIDR_BLOCK}" \
        -var "infrastructure_name=${R_NAME}-local"
    # ...
    terraform destroy -var "ami_id=${TERRAFORM_PROVISIONER_ANSIBLE_AMI_ID}" \
        -var "region=${R_REGION}" \
        -var "aws_admin_profile=${R_NAME}-ansible" \
        -var "vpc_cidr_block=${R_VPC_CIDR_BLOCK}" \
        -var "infrastructure_name=${R_NAME}-local"
    ```

4. `remote-with-bastion`: VPC setup, bastion, provision remote over bastion
    
    ```
    cd remote-with-bastion
    export R_NAME=terraform-provisioner-ansible
    export R_REGION=eu-central-1
    export R_VPC_CIDR_BLOCK=10.0.0.0/16
    terraform plan -var "ami_id=${TERRAFORM_PROVISIONER_ANSIBLE_AMI_ID}" \
        -var "region=${R_REGION}" \
        -var "aws_admin_profile=${R_NAME}" \
        -var "vpc_cidr_block=${R_VPC_CIDR_BLOCK}" \
        -var "infrastructure_name=${R_NAME}-remote"
    # ...
    terraform apply -var "ami_id=${TERRAFORM_PROVISIONER_ANSIBLE_AMI_ID}" \
        -var "region=${R_REGION}" \
        -var "aws_admin_profile=${R_NAME}" \
        -var "vpc_cidr_block=${R_VPC_CIDR_BLOCK}" \
        -var "infrastructure_name=${R_NAME}-remote"
    # ...
    terraform destroy -var "ami_id=${TERRAFORM_PROVISIONER_ANSIBLE_AMI_ID}" \
        -var "region=${R_REGION}" \
        -var "aws_admin_profile=${R_NAME}-ansible" \
        -var "vpc_cidr_block=${R_VPC_CIDR_BLOCK}" \
        -var "infrastructure_name=${R_NAME}-remote"
    ```