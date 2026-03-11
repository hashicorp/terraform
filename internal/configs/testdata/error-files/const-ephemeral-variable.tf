variable "example" { # ERROR: Const variable cannot be ephemeral
  type      = string
  default   = "hello"
  const     = true
  ephemeral = true
}
