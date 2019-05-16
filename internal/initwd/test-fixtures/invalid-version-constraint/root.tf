# This fixture depends on a github repo at:
#     https://github.com/hashicorp/terraform-aws-module-installer-acctest

module "acctest_root" {
  source  = "github.com/hashicorp/terraform-aws-module-installer-acctest"
  version = "0.0.1"
}