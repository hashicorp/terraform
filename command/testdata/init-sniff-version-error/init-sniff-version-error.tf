# The following is invalid because we don't permit multiple nested blocks
# all one one line. Instead, we require the backend block to be on a line
# of its own.
# The purpose of this test case is to see that HCL still produces a valid-enough
# AST that we can try to sniff in this block for a terraform_version argument
# without crashing, since we do that during init to try to give a better
# error message if we detect that the configuration is for a newer Terraform
# version.
terraform { backend "local" {} }
