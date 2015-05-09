# Count Example

The count parameter on resources can simplify configurations
and let you scale resources by simply incrementing a number.

Additionally, variables can be used to expand a list of resources
for use elsewhere.

A typical execution will be as below

Replace the following with appropriate values, refer to variables.tf for description of each variable

{your_region e.g. us-east-1}
{your_access_key}
{your_secret_key}   

Run the plan first and inspect the output.  

terraform plan -var 'aws_region={your_region}' -var 'access_key={your_access_key}}' -var 'secret_key={your_secret_key}}'

Once you are satisfied with plan, run the apply

terraform apply -var 'aws_region={your_region}' -var 'access_key={your_access_key}}' -var 'secret_key={your_secret_key}}'

To destroy the stack run

terraform destroy -var 'aws_region={your_region}' -var 'access_key={your_access_key}}' -var 'secret_key={your_secret_key}}'