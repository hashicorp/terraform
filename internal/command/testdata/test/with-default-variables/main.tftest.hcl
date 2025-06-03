
variable "input_one" {
  type = string
  default = "hello"
}

variable "input_two" {
  type = string
  default = "world" // we will override this an external value
}

run "test" {
  assert {
    condition = test_resource.resource.value == "hello - universe"
    error_message = "bad concatenation"
  }
}
