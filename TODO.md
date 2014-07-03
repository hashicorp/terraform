This is just to keep track of what we need to do before 0.1:

  * `terraform apply/plan/refresh/destroy` need to be able to take variables as input
  * Provisioners on top of static resource creation: Shell, Chef, Puppet, etc.
  * `ValidateResource` ResourceProvider API for checking the structure of a resource config
  * A module system for better Terraform file organization. More details on this later.
  * Commands to inspect and manipulate plans and states.
  * Support for "outputs" in configuration
  * "count" meta-parameter for instantiating multiple identical resources
