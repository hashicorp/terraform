
# NOTE: The architecture described here is _very loosely_ based on
# the following AWS tutorial blog post:
#    https://aws.amazon.com/blogs/containers/operating-a-multi-regional-stateless-application-using-amazon-eks/
# ...and the following Google Cloud Platform architecture document:
#    https://cloud.google.com/architecture/prep-kubernetes-engine-for-prod

variable "environment" {
  type = object({
    name            = string
    domain          = string
    base_cidr_block = string
  })

  validation {
    condition     = can(cidrhost(var.environment.base_cidr_block, 0)) && substr(var.environment.base_cidr_block, -3, -1) == "/12"
    error_message = "Must be an IPv4 CIDR prefix of length 12."
  }
}

variable "aws" {
  type = object({
    regions = list(object({
      # The name of the region, as the AWS provider would
      # expect it.
      name = string

      # Currently-used availability zones. The indices in
      # this list decide the IP address numbering, so
      # this list must not be reordered.
      availability_zones = list(string)
    }))
  })

  validation {
    condition     = len(var.aws.regions) <= 4
    error_message = "Regions list may have no more than four regions."
  }

  validation {
    condition = alltrue([
      for r in var.aws.regions : len(r.availability_zones) <= 4
    ])
    error_message = "Cannot specify more than four regions in each availability zone."
  }
}

variable "gcp" {
  type = object({
    # TODO
  })
}

component_group "aws" {
  components   = "./vendors/aws.tfcomponents.hcl"
  display_name = "Amazon Web Services"
  variables = {
    base_cidr_block = cidrsubnet(var.environment.base_cidr_block, 4, 1)
    regions         = {
      for r in var.aws.regions : {
        name                   = r.name
        availability_zones     = r.availability_zones
        max_availability_zones = 4
      }
    }
    max_regions     = 4
    common_tags = {
      Environment = var.environment.name
    }
  }
}

component_group "gcp" {
  components   = "./vendors/gcp.tfcomponents.hcl"
  display_name = "Google Cloud Platform"
  variables = {
    base_cidr_block = cidrsubnet(var.environment.base_cidr_block, 4, 2)
    # TODO: More of this
  }
}

output "kubeconfigs" {
  kubeconfigs = setunion(
    values(component_group.aws.kubeconfigs),
    values(component_group.gcp.kubeconfigs),
  )
  sensitive = true
}

output "http_backend_hosts" {
  value = setunion(
    component_group.aws.http_backend_hosts,
    component_group.gcp.http_backend_hosts,
  )
}
