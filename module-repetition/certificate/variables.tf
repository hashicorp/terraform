variable "ca" {
  type = object({
    cert_pem        = string
    private_key_pem = string
    key_algorithm   = string
  })
}

variable "dns_names" {
  type = list(string)

  validation {
    condition     = length(var.dns_names) > 0
    error_message = "The dns_names list must contain at least one hostname."
  }
}

variable "key" {
  type = object({
    algorithm       = string
    private_key_pem = string
  })
}

variable "organization_name" {
  type = string
}
