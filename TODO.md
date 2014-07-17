This is just to keep track of what we need to do in the short term.

  * `terraform apply/plan/refresh/destroy` need to be able to take variables as input
  * Mappings
  * CLI: Improve output with # of resources changed, updated, etc.
  * CLI: Improve apply output when a pure-destroy was done. Currently it
      shows "resources added/changed" but it should have a different message
      saying infra was destroyed.
  * Configuration schemas for providers and provisioners
  * Helper improvements around complex types perhaps
  * Providers/AWS: `aws_security_group` needs an update func
  * Providers/AWS: `aws_eip` needs an update func (re-assocation, I think this
      is possible in the AWS API). `aws_eip` also needs improved tests

