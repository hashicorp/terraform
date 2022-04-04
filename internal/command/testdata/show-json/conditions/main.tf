variable "ami" {
  type    = string
  default = "ami-test"
}

variable "id_minimum_length" {
  type    = number
  default = 10
}

resource "test_instance" "foo" {
  ami = var.ami

  lifecycle {
    precondition {
      condition     = can(regex("^ami-", var.ami))
      error_message = "Invalid AMI ID: must start with \"ami-\"."
    }
  }
}

resource "test_instance" "bar" {
  ami = "ami-boop"

  lifecycle {
    postcondition {
      condition     = length(self.id) >= var.id_minimum_length
      error_message = "Resource ID is unacceptably short (${length(self.id)} characters)."
    }
  }
}

output "foo_id" {
  value = test_instance.foo.id

  precondition {
    condition     = test_instance.foo.ami != "ami-bad"
    error_message = "Foo has a bad AMI again!"
  }
}
