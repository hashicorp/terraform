variable "example" { # ERROR: Const variable cannot be sensitive
  type      = string
  default   = "hello"
  const     = true
  sensitive = true
}
