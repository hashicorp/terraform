
variable "input_one" {
  type = string
  default = "hello"
}

variable "input_two" {
  type = string
  default = "world" // we will override this an external value
}

variables {
  value = "${var.input_two}_more"
}

run "test" {
  assert {
    condition = test_resource.resource.value == "hello - universe"
    error_message = "bad concatenation"
  }
}

run "nested_ref" {
  variables {
    input_two = var.value
  }
  assert {
    condition = test_resource.resource.value == "hello - universe_more"
    error_message = "bad concatenation"
  }
}
