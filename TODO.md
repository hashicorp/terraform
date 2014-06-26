This is just to keep track of what we need to do before 0.1:

  * `terraform destroy`
  * `terraform apply` should take a plan file
  * `terraform plan` should output a plan file 
  * `terraform refresh` to update state to latest
  * `terraform apply/plan/refresh/destroy` need to be able to take variables as input
  * UI output
  * Provisioners on top of static resource creation: Shell, Chef, Puppet, etc.
  * `ValidateResource` ResourceProvider API for checking the structure of a resource config
  * A module system for better Terraform file organization. More details on this later.
  * Commands to inspect and manipulate plans and states.
