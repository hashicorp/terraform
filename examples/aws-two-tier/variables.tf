
# variables have two properties.  description & default.  If the default property is not set, then terraform plan will error.
variable "key_name" {
    description = "Name of the SSH keypair to use in AWS.",
    default = "REPLACE_WITH_AWS_KEYPAIR_NAME"
}

variable "key_path" {
    description = "Path to the private portion of the SSH key specified.",
    default = "REPLACE_WITH_AWS_KEYPAIR_PATH"
}

variable "aws_region" {
    description = "AWS region to launch servers."
    default = "us-west-2"
}

# Ubuntu Precise 12.04 LTS (x64)
variable "aws_amis" {
    default = {
        eu-west-1 = "ami-b1cf19c6"
        us-east-1 = "ami-de7ab6b6"
        us-west-1 = "ami-3f75767a"
        us-west-2 = "ami-21f78e11"
    }
}
