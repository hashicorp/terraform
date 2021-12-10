# This fixture depends on a github repo at:
#     https://github.com/hashicorp/terraform-aws-module-installer-acctest

variable "v" {
  description = "in local caller for go-getter-modules"
  default     = ""
}

# The fbad92afe22792b939ceb233acd86ebd57af8fc7 in the following source addresses
# is, at the time of authoring this test, the commit associated with the
# tag v0.0.2, but given directly specifically so we can make sure we support
# arbitrary commits and not just named refs.

module "acctest_root" {
  source = "github.com/hashicorp/terraform-aws-module-installer-acctest?ref=fbad92afe22792b939ceb233acd86ebd57af8fc7"
}

module "acctest_child_a" {
  source = "github.com/hashicorp/terraform-aws-module-installer-acctest//modules/child_a?ref=fbad92afe22792b939ceb233acd86ebd57af8fc7"
}

module "acctest_child_b" {
  source = "github.com/hashicorp/terraform-aws-module-installer-acctest//modules/child_b?ref=fbad92afe22792b939ceb233acd86ebd57af8fc7"
}
