# ALLOW-LANGUAGE-EXPERIMENTS

# If the ephemeral_values features get stabilized, this test input will fail
# due to the experiment being concluded, in which case it might make sense to
# move this file to valid-files and remove the experiment opt-in
#
# If this experiment is removed without stabilizing it then this will fail
# and should be removed altogether.

terraform {
  experiments = [ephemeral_values] # WARNING: Experimental feature "ephemeral_values" is active
}

variable "in" {
  ephemeral = true
}

output "out" {
  ephemeral = true
  value     = var.in
}
