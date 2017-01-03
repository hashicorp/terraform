### ECS With special SLB and SecurityGroup Example

The example launches 6 ECS and create it on special SLB and securityGroup.
Also additional first and second instance to the SLB backend server.
The variables.tf can let you create specify parameter instances, such as image_id, ecs_type etc.

### Get up and running

* Planning phase

		terraform plan 

* Apply phase

		terraform apply 


* Destroy 

		terraform destroy