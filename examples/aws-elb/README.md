# ELB with stickiness Example

The example launches a web server, installs nginx, creates an ELB for instance. It also creates security groups for the ELB and EC2 instance. 

To run, configure your AWS provider as described in https://www.terraform.io/docs/providers/aws/index.html

Run this example using:

    terraform apply -var 'key_name=YOUR_KEY_NAME'

Wait a couple of minutes for the EC2 userdata to install nginx, and then type the ELB DNS Name from outputs in your browser and see the nginx welcome page
