# This fixture depends on a github repo at:
#     https://github.com/hashicorp/terraform-aws-module-installer-acctest
# ...and expects its v0.0.1 tag to be pointing at the following commit:
#     d676ab2559d4e0621d59e3c3c4cbb33958ac4608

module "acctest_root" {
  source = "github.com/hashicorp/terraform-aws-module-installer-acctest?ref=v0.0.1"
}

module "acctest_child_a" {
  source = "github.com/hashicorp/terraform-aws-module-installer-acctest//modules/child_a?ref=v0.0.1"
}

module "acctest_child_b" {
  source = "github.com/hashicorp/terraform-aws-module-installer-acctest//modules/child_b?ref=v0.0.1"
}
