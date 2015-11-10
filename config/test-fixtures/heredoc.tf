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
