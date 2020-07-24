# Terraform v0.13 Beta: Module `depends_on`

In previous versions of Terraform, module instances have served only as separate
namespaces and have not been nodes in Terraform's dependency graph themselves.
Terraform has always tracked dependencies via the input variables and output
values of a module, but users have frequently requested a concise way to
declare that _all_ objects inside a module share a particular dependency in
the calling module.

Terraform v0.13 introduces this capability by allowing `depends_on` as a
meta-argument inside `module` blocks:

```hcl
resource "aws_iam_policy_attachment" "example" {
  name       = "example"
  roles      = [aws_iam_role.example.name]
  policy_arn = aws_iam_policy.example.arn
}

module "uses-role" {
  # ...

  depends_on = [aws_iam_policy_attachment.example]
}
```

This is a far more coarse declaration of dependency than Terraform previously
encouraged, so we do still recommend considering this a last resort and using
data flow to imply dependencies wherever possible. Allowing Terraform to infer
dependencies automatically will tend to make your configuration easier to
maintain and will allow Terraform to maximize concurrency when making many
changes in a single operation.

## Usage Notes

As with all new Terraform language features, its new behaviors interact with
other existing language features in various ways. We've included some examples
we know about below.

### Module `for_each` and `count` create module-level dependencies too

Also included in the v0.13 release are
[`for_each` and `count` arguments for `module` blocks](../module-repetition/).
Because Terraform must evaluate the `for_each` or `count` expression in order
to determine how many instances of a module to create, these two new arguments
can also effectively create some indirect dependencies for the contents of the
module.

We do expect the interactions between these features to be intuitive in most
cases, but as always we must be careful when using explicit or implicit
dependencies not to create any dependency cycles. We mention this here only
because these new features together can result in many new dependency
relationships in Terraform's graph, and so we must always consider how those
interact with other dependencies that are implied or explicitly declared
elsewhere in the configuration.

