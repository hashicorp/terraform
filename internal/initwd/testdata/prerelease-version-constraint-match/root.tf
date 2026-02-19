# We expect this test to download the requested version because it is an exact
# match for a prerelease version.

module "acctest_exact" {
  source = "hashicorp/module-installer-acctest/aws"
  version = "=0.0.3-alpha.1"
}
