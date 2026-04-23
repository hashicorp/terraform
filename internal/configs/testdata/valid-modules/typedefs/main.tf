variable "example_string_1" {
  type    = typedef.custom_string
  default = "hello world!"
}

variable "example_string_2" {
  type    = typedef.custom_string
  default = true
}

variable "example_object" {
  type = typedef.custom_object
  default = {
    a = 100
    b = true
    c = ["hello", "world!"]
    # d = "intentionally omitted value :P"
  }
}
