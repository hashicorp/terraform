
variable "input" {
    type = string
    default = "Hello, world!"
}

resource "tfcoremock_simple_resource" "resource" {
    string = var.input
}
