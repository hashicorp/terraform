# The "etcd" backend has been removed from Terraform versions 1.3+
run "test_removed_backend" {
  variables {
    input = "foobar"
  }

  backend "etcd" {
  }
}
