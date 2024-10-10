variable "token" {
  ephemeral = true
  default   = "insecure"
}

provider "test" {
  token = var.token
}
