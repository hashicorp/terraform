run "setup" {
  # We have a dedicated repo for this test module.
  # See ../registry-modules/root.tf for more info.
  module {
    source = "hashicorp/module-installer-acctest/aws"
    version = "0.0.1"
  }
}
