variable "map_in" {
  type = map(string)

  default = {
    us-west-1 = "ami-12345"
    us-west-2 = "ami-67890"
  }
}

// We have to reference it so it isn't pruned
output "output" {
  value = var.map_in
}
