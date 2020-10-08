# This fixture indirectly depends on a github repo at:
#     https://github.com/hashicorp/terraform-aws-module-installer-acctest
# ...and expects its v0.0.1 tag to be pointing at the following commit:
#     d676ab2559d4e0621d59e3c3c4cbb33958ac4608
#
# This repository is accessed indirectly via:
#     https://registry.terraform.io/modules/hashicorp/module-installer-acctest/aws/0.0.1
#
# Since the tag's id is included in a downloaded archive, it is expected to
# have the following id:
#     853d03855b3290a3ca491d4c3a7684572dd42237
# (this particular assumption is encoded in the tests that use this fixture)


variable "v" {
  description = "in local caller for registry-modules"
  default     = ""
}

module "acctest_root" {
  source  = "hashicorp/module-installer-acctest/aws"
  version = "0.0.1"
}

module "acctest_child_a" {
  source  = "hashicorp/module-installer-acctest/aws//modules/child_a"
  version = "0.0.1"
}

module "acctest_child_b" {
  source  = "hashicorp/module-installer-acctest/aws//modules/child_b"
  version = "0.0.1"
}
