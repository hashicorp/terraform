### VPC Cluster Example

The example launches VPC cluster, include VPC, VSwitch, Nategateway, ECS, SecurityGroups. the example used the "module" to create instances. The variables.tf can let you create specify parameter instances, such as image_id, ecs_type, count etc.

### Get up and running

* Planning phase

		terraform plan 
    		

* Apply phase

		terraform apply 
		   

* Destroy 

		terraform destroy