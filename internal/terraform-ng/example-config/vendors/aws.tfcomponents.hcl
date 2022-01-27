# NOTE: The architecture described here is are _very loosely_ based on
# the following AWS tutorial blog post:
#    https://aws.amazon.com/blogs/containers/operating-a-multi-regional-stateless-application-using-amazon-eks/

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

# Determines how many CIDR prefix bits to allocate
# to a region number, to allow for future growth of the
# regions list to this number of elements. If not a 
# power of two then it will be rounded up to the next
# one.
variable "max_regions" {
  type = number
}

variable "common_tags" {
  type = map(string)
}

component_group "network" {
  components   = "./aws/network.tfcomponents.hcl"
  display_name = "Network"

  variables = {
    common_tags     = var.common_tags
    base_cidr_block = var.base_cidr_block
    regions         = var.regions
    max_regions     = var.max_regions
  }
}

component_group "kubernetes" {
  components   = "./aws/kubernetes-cluster.tfcomponents.hcl"
  display_name = "Kubernetes Cluster"

  variables = {
    network = component_group.network
  }
}

output "http_backend_hosts" {
  value = toset([]) # TODO
}
