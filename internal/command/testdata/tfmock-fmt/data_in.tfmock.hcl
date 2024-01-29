mock_data "aws_availability_zones" {
  defaults = {
    names = [
"us-east-1a",
      "us-east-1b",
      "us-east-1c",
      "us-east-1d",
      "us-east-1e",
      "us-east-1f"
    ]
  }
}

override_data {
target = data.aws_subnets.private_subnets
  values = {
    ids = ["subnet-a",
      "subnet-b",
      "subnet-c"
    ]
  }
}
