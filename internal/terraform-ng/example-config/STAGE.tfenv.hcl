components = "./platform.tfcomponents.hcl"
variables {
  environment = {
    name            = "STAGE"
    domain          = "stage.example.com"
    base_cidr_block = "10.32.0.0/12"
  }
  aws = {
    regions = [
      {
        name               = "us-east-1"
        availability_zones = ["us-east-1e"]
      },
      {
        name               = "us-west-2"
        availability_zones = ["us-west-2c"]
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
