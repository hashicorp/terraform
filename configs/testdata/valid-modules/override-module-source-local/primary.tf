# This fixture depends on a registry module. However, the test that uses it
# is testing the source and version override functionality. The registry does
# not need to be accessed, and the source can be any registry URL.

module "example" {
  source  = "hashicorp/subnets/cidr"
  version = "1.0.0"
}
