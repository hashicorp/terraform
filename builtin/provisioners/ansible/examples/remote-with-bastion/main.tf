variable "region" {}
variable "aws_admin_profile" {}
variable "vpc_cidr_block" {}
variable "infrastructure_name" {}

variable "ami_id" {}

provider "aws" {
  region  = "${var.region}"
  profile = "${var.aws_admin_profile}"
}

resource "aws_vpc" "vpc" {
  cidr_block = "${var.vpc_cidr_block}"
  enable_dns_support = true
  enable_dns_hostnames = true
  tags {
    Name = "${var.infrastructure_name}_vpc"
  }
}

resource "aws_subnet" "public" {
  vpc_id                  = "${aws_vpc.vpc.id}"
  availability_zone       = "${var.region}a"
  cidr_block              = "${cidrsubnet(aws_vpc.vpc.cidr_block, 8, 1)}"
  map_public_ip_on_launch = true
  tags {
    Name = "${var.infrastructure_name}_public_subnet"
  }
}

resource "aws_internet_gateway" "gw" {
  vpc_id = "${aws_vpc.vpc.id}"
  tags {
    Name = "${var.infrastructure_name}_gw"
  }
}

resource "aws_route_table" "rt" {
  vpc_id = "${aws_vpc.vpc.id}"
  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = "${aws_internet_gateway.gw.id}"
  }
}

resource "aws_route_table_association" "rta" {
  subnet_id      = "${aws_subnet.public.id}"
  route_table_id = "${aws_route_table.rt.id}"
}

## -- security groups:

resource "aws_security_group" "ssh_bastion" {
  name        = "ssh_bastion"
  description = "SSH bastion"
  vpc_id      = "${aws_vpc.vpc.id}"

  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
    self        = true
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_security_group" "ssh_box" {
  name        = "ssh_box"
  description = "SSH box"
  vpc_id      = "${aws_vpc.vpc.id}"

  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    security_groups = ["${aws_security_group.ssh_bastion.id}"]
    self        = true
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

## -- machine:

resource "aws_instance" "bastion" {
  ami           = "${var.ami_id}"
  count         = "1"
  instance_type = "t2.medium"

  vpc_security_group_ids = ["${aws_security_group.ssh_bastion.id}"]

  subnet_id = "${aws_subnet.public.id}"

  connection {
    user = "centos"
  }

  root_block_device {
    delete_on_termination = true
    volume_size           = 8
    volume_type           = "gp2"
  }
}

resource "aws_instance" "test_box" {
  ami           = "${var.ami_id}"
  count         = "1"
  instance_type = "m3.medium"

  connection {
    user         = "centos"
    host         = "${self.private_ip}"
    bastion_host = "${aws_instance.bastion.public_ip}"
  }

  vpc_security_group_ids = ["${aws_security_group.ssh_box.id}"]

  subnet_id = "${aws_subnet.public.id}"

  provisioner "ansible" {
    plays {
      playbook = {
        file_path = "${path.module}/../ansible-data/playbooks/install-tree.yml"
        roles_path = [
            "${path.module}/../ansible-data/roles"
        ]
      }
      hosts = ["testBoxToBootstrap"]
    }
    remote {}
  }

  root_block_device {
    delete_on_termination = true
    volume_size           = 8
    volume_type           = "gp2"
  }
}
