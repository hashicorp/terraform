variable "name" {
  type = string
}

variable "auth_jwt" {
  type      = string
  ephemeral = true
  sensitive = true
}
