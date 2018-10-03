
# This configuration is just here to allow the tests in session_test to
# evaluate expressions without getting errors about things not being declared.
# Therefore it's intended to just be the minimum config to make those
# expressions work against the equally-minimal mock provider.
resource "test_instance" "foo" {
}

module "module" {
  source = "./child"
}
