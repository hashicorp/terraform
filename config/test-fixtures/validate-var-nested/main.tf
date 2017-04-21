variable "foo" {
  default = [["foo", "bar"]]
}
variable "bar" {
  default = [{foo = "bar"}]
}
