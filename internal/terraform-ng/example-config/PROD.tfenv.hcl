components = "./platform.tfcomponents.hcl"
variables {
  environment = {
    name            = "PROD"
    domain          = "prod.example.com"
    base_cidr_block = "10.16.0.0/12"
  }
  aws = {
    regions = [
      {
        name               = "us-east-1"
        availability_zones = ["us-east-1a", "us-east-1c"]
      },
      {
        name               = "us-west-2"
        availability_zones = ["us-west-2a", "us-west-2b"]
      },
      {
        name               = "eu-west-1"
        availability_zones = ["eu-west-1a", "us-west-1c"]
      },
    ]
  }
  gcp = {
    # TODO
  }
}
storage "hashicorp/aws/s3@2.3.5" {
  bucket         = "example-bucket"
  path_prefix    = "prod"
  dynamodb_table = "ExampleThing"
}
