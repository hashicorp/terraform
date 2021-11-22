---
layout: "language"
page_title: "The depends_on Meta-Argument - Configuration Language"
description: "The depends_on meta-argument allows you to handle hidden resource or module dependencies."
---

# The `depends_on` Meta-Argument

-> **Version note:** Module support for `depends_on` was added in Terraform 0.13, and
previous versions can only use it with resources.

Use the `depends_on` meta-argument to handle hidden resource or module dependencies that
Terraform can't automatically infer.

Explicitly specifying a dependency is only necessary when a resource or module relies on
some other resource's behavior but _doesn't_ access any of that resource's data
in its arguments.

This argument is available in `module` blocks and in all `resource` blocks,
regardless of resource type. For example:

```hcl
resource "aws_iam_role" "example" {
  name = "example"

  # assume_role_policy is omitted for brevity in this example. See the
  # documentation for aws_iam_role for a complete example.
  assume_role_policy = "..."
}

resource "aws_iam_instance_profile" "example" {
  # Because this expression refers to the role, Terraform can infer
  # automatically that the role must be created first.
  role = aws_iam_role.example.name
}

resource "aws_iam_role_policy" "example" {
  name   = "example"
  role   = aws_iam_role.example.name
  policy = jsonencode({
    "Statement" = [{
      # This policy allows software running on the EC2 instance to
      # access the S3 API.
      "Action" = "s3:*",
      "Effect" = "Allow",
    }],
  })
}

resource "aws_instance" "example" {
  ami           = "ami-a1b2c3d4"
  instance_type = "t2.micro"

  # Terraform can infer from this that the instance profile must
  # be created before the EC2 instance.
  iam_instance_profile = aws_iam_instance_profile.example

  # However, if software running in this EC2 instance needs access
  # to the S3 API in order to boot properly, there is also a "hidden"
  # dependency on the aws_iam_role_policy that Terraform cannot
  # automatically infer, so it must be declared explicitly:
  depends_on = [
    aws_iam_role_policy.example,
  ]
}
```

The `depends_on` meta-argument, if present, must be a list of references
to other resources or child modules in the same calling module.
Arbitrary expressions are not allowed in the `depends_on` argument value,
because its value must be known before Terraform knows resource relationships
and thus before it can safely evaluate expressions.

The `depends_on` argument should be used only as a last resort. When using it,
always include a comment explaining why it is being used, to help future
maintainers understand the purpose of the additional dependency.

