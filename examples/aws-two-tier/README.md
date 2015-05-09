# Basic Two-Tier AWS Architecture

This provides a template for running a simple two-tier architecture on Amazon
Web services. The premise is that you have stateless app servers running behind
an ELB serving traffic.

To simplify the example, this intentionally ignores deploying and
getting your application onto the servers. However, you could do so either via
[provisioners](/docs/provisioners/index.html) and a configuration
management tool, or by pre-baking configured AMIs with
[Packer](http://www.packer.io).

A typical execution will be as below

Replace the following with appropriate values, refer to variables.tf for description of each variable

{your_region e.g. us-east-1}
{your_key_name e.g. terraform}
{your_key_path_in_local_workstation e.g. /Users/jxyz/terraform.pem}
{your_access_key}
{your_secret_key}   

Run the plan first and inspect the output.  

terraform plan -var 'aws_region={your_region}' -var 'key_name={your_key_name}}' -var 'key_path={your_key_path_in_local_workstation}}' -var 'access_key={your_access_key}}' -var 'secret_key={your_secret_key}}'

Once you are satisfied with plan, run the apply

terraform apply -var 'aws_region={your_region}' -var 'key_name={your_key_name}}' -var 'key_path={your_key_path_in_local_workstation}}' -var 'access_key={your_access_key}}' -var 'secret_key={your_secret_key}}'

After you run `terraform apply` on this configuration, it will
automatically output the DNS address of the ELB. After your instance
registers, this should respond with the default nginx web page.

To destroy the stack run

terraform destroy -var 'access_key={your_access_key}}' -var 'secret_key={your_secret_key}}'
