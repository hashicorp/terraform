
variable "base_cidr_block" {
  type = string

  validation {
    condition     = can(cidrhost(var.base_cidr_block, 0)) && can(regex("^[\\d\\./]+$"))
    error_message = "Must be an IPv4 CIDR prefix given as ADDRESS/LENGTH."
  }
}

variable "regions" {
  type = list(object({
    # The name of the region, as the AWS provider would
    # expect it.
    name = string

    # Currently-used availability zones. The indices in
    # this list decide the IP address numbering, so
    # this list must not be reordered.
    availability_zones = list(string)

    # Determines how many CIDR prefix bits to allocate
    # to an availability zone number, to allow for
    # future growth of the availability_zones list to
    # this number of elements. If not a power of two then
    # it will be rounded up to the next one.
    max_availability_zones = number
  }))
}

variable "common_tags" {
  type = map(string)
}

locals {
  region_address_newbits = ceil(log(var.max_regions, 2))
}

component "regions" {
  module = "./network/region"
  for_each = {
    for i, r in var.aws.regions : r.name => merge(r, {
      network_number = i
    })
  }
  display_name = "VPC and Subnets for ${each.key}"

  variables = {
    base_cidr_block        = cidrsubnet(var.base_cidr_block, local.region_address_newbits, each.value.network_number)
    region_name            = each.value.name
    availability_zones     = each.value.availability_zones
    max_availability_zones = each.value.max_availability_zones
    common_tags            = var.common_tags
  }
}

component "peering" {
  module       = "./network/peering"
  display_name = "Regional Peering Connections"

  variables = {
    regions     = component.regions
    common_tags = var.common_tags
  }
}
