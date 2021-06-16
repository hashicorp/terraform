terraform {
  required_version = ">= 1.0, != 2.0"
  required_providers {
    # AWS Provider
    aws = {
      source = "hashicorp/aws"
      version = "~> 3.44"
    }
  }
}

# Configuration of aws provider
provider "aws" {
  profile = "ops-lab"
  region  = "us-east-1"
  default_tags {
    tags = {
      Environment = "Lab"
    }
  }
}