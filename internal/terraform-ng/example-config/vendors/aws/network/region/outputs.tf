
output "vpcs" {
  # TODO: Make this a real thing, not hardcoded.
  value = tomap({
    us-east-1 = {
      id         = "vpc-1234"
      cidr_block = "10.1.64.0/18"
      availability_zones = tomap({
        us-east-1a = {
          id         = "subnet-1234"
          cidr_block = "10.1.80.0/20"
        }
      })
    }
  })
}
