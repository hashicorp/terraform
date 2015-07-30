# ASG example

This example shows how to launch instances using Auto Scaling Groups.

This creates a security group, launch configuration, auto scaling group and an ELB. The user data for launch configuration installs nginx and it listens on port 80.

The example uses latest Ubuntu AMIs. 

Make sure you change the list of availability zones that is applicable to your account and region.

To run, configure your AWS provider as described in https://www.terraform.io/docs/providers/aws/index.html

Running the example

For planning phase 

terraform plan -var 'key_name={your_key_name}}'

For apply phase 

terraform apply -var 'key_name={your_key_name}}'

Once the stack is created, wait for few minutes and test the stack by launching a browser with ELB url. 

To remove the stack

 terraform destroy -var 'key_name={your_key_name}}'

