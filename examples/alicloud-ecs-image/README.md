### ECS Example

The example gains image info and use it to launche ECS instance, disk, and attached the disk on ECS. the count parameter in variables.tf can let you gain specify image and use it to create specify number ECS instances.

### Get up and running

* Planning phase

		terraform plan 
    		var.availability_zones
  				Enter a value: {var.availability_zones}  /*cn-beijing-b*/
	    	var.datacenter
	    		Enter a value: {datacenter}
	    	....

* Apply phase

		terraform apply 
		    var.availability_zones
  				Enter a value: {var.availability_zones}  /*cn-beijing-b*/
	    	var.datacenter
	    		Enter a value: {datacenter}
	    	....

* Destroy 

		terraform destroy