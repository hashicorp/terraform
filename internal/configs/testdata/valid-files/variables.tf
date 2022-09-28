
variable "foo" {
}

variable "bar" {
  default = "hello"
}

variable "baz" {
  type = list
}

variable "bar-baz" {
  default = []
  type    = list(string)
}

variable "cheeze_pizza" {
  description = "Nothing special"
}

variable "π" {
  default = 3.14159265359
}

variable "sensitive_value" {
  default = {
    "a" = 1,
    "b" = 2
  }
  sensitive = true
}

variable "nullable" {
  type = string
  nullable = true
  default = "ok"
}

variable "nullable_default_null" {
  type = map(string)
  nullable = true
  default = null
}
