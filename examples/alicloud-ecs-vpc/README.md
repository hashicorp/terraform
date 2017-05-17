### ECS In VPC Example

The example launches ECS in VPC, vswitch_id parameter is the vswitch id from VPC. It also create disk, and attached the disk on ECS. The variables.tf can let you create specify parameter instances, such as image_id, ecs_type, count etc.

### Get up and running

* Planning phase

		terraform plan 
    		var.availability_zones
  				Enter a value: {var.availability_zones}  /*cn-beijing-b*/
	    	var.datacenter
	    		Enter a value: {datacenter}
	    	var.vswitch_id
	    		Enter a value: {vswitch_id}
	    	....

* Apply phase

		terraform apply 
		    var.availability_zones
  				Enter a value: {var.availability_zones}  /*cn-beijing-b*/
	    	var.datacenter
	    		Enter a value: {datacenter}
	    	var.vswitch_id
	    		Enter a value: {vswitch_id}
	    	....

* Destroy 

		terraform destroy