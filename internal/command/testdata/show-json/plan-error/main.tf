locals {
  ami = "bar"
}

resource "test_instance" "test" {
  ami = local.ami

  lifecycle {
    precondition {
      // failing condition
      condition = local.ami != "bar"
      error_message = "ami is bar"
    }
  }
}