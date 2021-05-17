# This fixture depends on a registry module, which indirectly refers to the
# following github repository:
#
# However, the test that uses it is testing for an error, so in practice the
# registry does not need to be accessed when this test is successful.

module "child" {
  source  = "hashicorp/module-installer-acctest/aws"
  version = "0.0.1"
}
