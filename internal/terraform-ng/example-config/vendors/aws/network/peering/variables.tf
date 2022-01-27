variable "regions" {
  type = object({
    vpcs = map(object({
      id              = string
      base_cidr_block = string
      subnets = map(object({
        id              = string
        base_cidr_block = string
      })
    })
  })
}

variable "common_tags" {
  type = map(string)
}
