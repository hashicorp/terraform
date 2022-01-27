# NOTE: The architecture described here is _very loosely_ based on
# the following Google Cloud Platform architecture document:
#    https://cloud.google.com/architecture/prep-kubernetes-engine-for-prod

variable "base_cidr_block" {
  type = string

  validation {
    condition     = can(cidrhost(var.base_cidr_block, 0)) && can(regex("^[\\d\\./]+$"))
    error_message = "Must be an IPv4 CIDR prefix given as ADDRESS/LENGTH."
  }
}

# TODO: Build this out as something equivalent to what aws.tfcomponents.hcl
# defines, based on the Google Cloud Platform architecture doc linked above.

output "http_backend_hosts" {
  value = toset([]) # TODO
}
