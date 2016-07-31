# This will spin up an instance and install cobbler onto it so we can run

# the Terraform acceptance tests against it.

module "ami" {
  source        = "github.com/terraform-community-modules/tf_aws_ubuntu_ami/ebs"
  region        = "us-west-2"
  distribution  = "trusty"
  instance_type = "t2.nano"
}

module "vpc" {
  source = "github.com/terraform-community-modules/tf_aws_vpc"

  name            = "cobbler"
  cidr            = "10.0.0.0/16"
  private_subnets = "10.0.1.0/24"
  public_subnets  = "10.0.101.0/24"
  azs             = "us-west-2a"
}

resource "aws_key_pair" "cobbler" {
  key_name   = "tf-cobbler-acctests"
  public_key = "${file("~/.ssh/id_rsa.pub")}"
}

resource "aws_security_group" "cobbler" {
  vpc_id = "${module.vpc.vpc_id}"

  ingress {
    protocol    = "tcp"
    from_port   = 22
    to_port     = 22
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    protocol    = "-1"
    from_port   = 0
    to_port     = 0
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_instance" "cobbler" {
  instance_type          = "t2.medium"
  ami                    = "${module.ami.ami_id}"
  subnet_id              = "${module.vpc.public_subnets}"
  key_name               = "${aws_key_pair.cobbler.id}"
  vpc_security_group_ids = ["${aws_security_group.cobbler.id}"]

  root_block_device {
    volume_type = "gp2"
    volume_size = 40
  }

  connection {
    user = "ubuntu"
  }

  provisioner "remote-exec" {
    inline = <<-WAIT
while [ ! -f /var/lib/cloud/instance/boot-finished ]; do echo 'Waiting for cloud-init...'; sleep 1; done
    WAIT
  }

  provisioner "remote-exec" {
    script = "${path.module}/deploy.sh"
  }
}

output "ssh" {
  value = "ubuntu@${aws_instance.cobbler.public_ip}"
}
