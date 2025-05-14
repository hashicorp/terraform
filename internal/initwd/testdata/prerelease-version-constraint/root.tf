# We expect this test to download the version 0.0.2, the one before the
# specified version even with the equality because the specified version is a
# prerelease.

module "acctest_partial" {
  source = "hashicorp/module-installer-acctest/aws"
  version = "<=0.0.3-alpha.1"
}
