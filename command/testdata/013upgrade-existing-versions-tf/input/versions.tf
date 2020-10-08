# This is a file called versions.tf which does not originally have a
# required_providers block. 
resource foo_resource a {}

terraform {
  required_version = ">= 0.12"
}
