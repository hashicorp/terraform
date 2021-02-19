variable "inner_in" {
  type = map(string)
  default = {
    us-west-1 = "ami-12345"
    us-west-2 = "ami-67890"
  }
}

resource "null_resource" "inner_noop" {}

output "inner_out" {
  value = lookup(var.inner_in, "us-west-1")
}
