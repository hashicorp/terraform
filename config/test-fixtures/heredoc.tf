provider "aws" {
  access_key = "foo"
  secret_key = "bar"
}

resource "aws_iam_policy" "policy" {
    name = "test_policy"
    path = "/"
    description = "My test policy"
    policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": [
        "ec2:Describe*"
      ],
      "Effect": "Allow",
      "Resource": "*"
    }
  ]
}
EOF
}

resource "aws_instance" "test" {
  ami = "foo"

  provisioner "remote-exec" {
    inline = [
<<EOT
sudo \
A=val \
B=val2 \
sh script.sh
EOT
    ]
  }
}

resource "aws_instance" "heredocwithnumbers" {
  ami = "foo"

  provisioner "local-exec" {
    command = <<FOO123
echo several
      lines
    of output
FOO123
  }
}
