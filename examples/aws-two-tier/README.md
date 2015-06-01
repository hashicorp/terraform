# Basic Two-Tier AWS Architecture

This provides a template for running a simple two-tier architecture on Amazon
Web services. The premise is that you have stateless app servers running behind
an ELB serving traffic.

To simplify the example, this intentionally ignores deploying and
getting your application onto the servers. However, you could do so either via
[provisioners](https://www.terraform.io/docs/provisioners/) and a configuration
management tool, or by pre-baking configured AMIs with
[Packer](http://www.packer.io).

After you run `terraform apply` on this configuration, it will
automatically output the DNS address of the ELB. After your instance
registers, this should respond with the default nginx web page.

To run, set the environment variables as below with correct values

export AWS_ACCESS_KEY_ID="..."
export AWS_SECRET_ACCESS_KEY="..."
export AWS_DEFAULT_REGION="..."

Alternatively, you can configure the provider configuration where you invoke the module.

For example, you can use section similar to below.

# Specify the provider and access details
provider "aws" {
    region = "${var.aws_region}"
    access_key = "${var.access_key}"
    secret_key = "${var.secret_key}" 
}

Running the example

run `terraform apply -var 'key_name={your_aws_Key_name}' -var 'key_path={location_of_your_key_in_your_local_machine}'` 

example

terraform apply -var 'key_name=terraform' -var 'key_path=/Users/jsmith/.ssh/terraform.pem

