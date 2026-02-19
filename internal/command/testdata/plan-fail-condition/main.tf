locals {
  ami = "bar"
}

resource "test_instance" "foo" {
  ami = local.ami

  lifecycle {
    precondition {
      // failing condition
      condition = local.ami != "bar"
      error_message = "ami is bar"
    }
  }
}
