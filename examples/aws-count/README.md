# Count Example

The count parameter on resources can simplify configurations
and let you scale resources by simply incrementing a number.

Additionally, variables can be used to expand a list of resources
for use elsewhere.

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

run `terraform apply` to see it work.
