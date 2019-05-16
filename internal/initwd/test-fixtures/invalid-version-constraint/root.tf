# This fixture references the github repo at:
#     https://github.com/hashicorp/terraform-aws-module-installer-acctest
# However, due to the nature of this test (verifying early error), the URL will not be contacted,
# and the test is safe to execute as part of the normal test suite.

module "acctest_root" {
  source  = "github.com/hashicorp/terraform-aws-module-installer-acctest"
  version = "0.0.1"
}