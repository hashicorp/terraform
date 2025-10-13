
variable "input" {
  type = string
}

module "foobar" {
  source = "./child-module"
  input  = "foobar"
}
