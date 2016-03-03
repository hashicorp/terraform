## Creating an RDS instance in AWS

This example provides sample configuration for creating a mysql or postgres instance. For Oracle/SQL Servers, replace default values with appropriate values, they are not included in sample since the number of options are high.

The example creates db subnet groups and a VPC security group as inputs to the instance creation

For AWS provider, set up your AWS environment as outlined in https://www.terraform.io/docs/providers/aws/index.html

If you need to use existing security groups and subnets, remove the `sg.tf` and `subnets.tf` files and replace the corresponding sections in `main.tf` under `aws_db_instance`

Pass the password variable through your ENV variable.

Several parameters are externalized, review the different variables.tf files and change them to fit your needs. Carefully review the CIDR blocks, egress/ingress rules, availability zones that are very specific to your account.

Once ready run `terraform plan` to review. 
At the minimum, provide the vpc_id as input variable.

Once satisfied with plan, run `terraform apply`  
