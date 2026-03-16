# This is a very minimal stack configuration just to give us something to
# call as a nested stack in the parent stack configuration.

variable "in" {
  type = string
}

output "out" {
  type  = string
  value = var.in
}
