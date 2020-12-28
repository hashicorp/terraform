# This is testing a module source override use case. The source does not need to
# be a valid module, but it must be set to a local path.

module "example" {
  source = "../"
}
