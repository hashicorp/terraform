# ELB with stickiness Example

The example launches a web server, installs nginx, creates an ELB for instance. It also creates security groups for elb/instance 

To run, configure your AWS provider as described in https://www.terraform.io/docs/providers/aws/index.html

Running the example

run `terraform apply -var 'key_name={your_key_name}}'` 

Give couple of mins for userdata to insatll nginx, and then type the ELB DNS Name from outputs in your browser and see the nginx welcome page
