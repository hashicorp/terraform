provider "aws" {region="us-west-2"}
resource "aws_directory_service_directory" "bar" {
  name = "corp.notexample.com"
  password = "SuperSecretPassw0rd"
  type = "MicrosoftAD"

  vpc_settings {
    vpc_id = "${aws_vpc.main.id}"
    subnet_ids = ["${aws_subnet.foo.id}", "${aws_subnet.bar.id}"]
  }
}

resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}

resource "aws_subnet" "foo" {
  vpc_id = "${aws_vpc.main.id}"
  availability_zone = "us-west-2a"
  cidr_block = "10.0.1.0/24"
}
resource "aws_subnet" "bar" {
  vpc_id = "${aws_vpc.main.id}"
  availability_zone = "us-west-2b"
  cidr_block = "10.0.2.0/24"
}

resource "aws_instance" "foo" {
  ami = "ami-4fccb37f"
  availability_zone = "us-west-2a"
  instance_type = "m1.small"
}

resource "aws_ssm_association" "foo" {
  name        = "test_document_association-1",
  instance_id = "${aws_instance.foo.id}"
  parameters {
      directoryId    = "${aws_directory_service_directory.bar.id}"
      directoryName  = "corp.mydomain.com"
  }
}
