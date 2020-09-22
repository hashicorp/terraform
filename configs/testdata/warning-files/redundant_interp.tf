# It's redundant to write an expression that is just a single template
# interpolation with another expression inside, like "${foo}", but it
# was required before Terraform v0.12 and so there are lots of existing
# examples out there using that style.
#
# We are generating warnings for that situation in order to guide those
# who are following old examples toward the new idiom.

variable "triggers" {
  type = "map" # WARNING: Quoted type constraints are deprecated
}

provider "null" {
  foo = "${var.triggers["foo"]}" # WARNING: Interpolation-only expressions are deprecated
}

resource "null_resource" "a" {
  triggers = "${var.triggers}" # WARNING: Interpolation-only expressions are deprecated

  connection {
    type = "ssh"
    host = "${var.triggers["host"]}" # WARNING: Interpolation-only expressions are deprecated
  }

  provisioner "local-exec" {
    single = "${var.triggers["greeting"]}" # WARNING: Interpolation-only expressions are deprecated

    # No warning for this one, because there's more than just one interpolation
    # in the template.
    template = " ${var.triggers["greeting"]} "

    # No warning for this one, because it's embedded inside a more complex
    # expression and our check is only for direct assignment to attributes.
    wrapped = ["${var.triggers["greeting"]}"]
  }
}

module "foo" {
  source = "./foo"
  foo = "${var.foo}" # WARNING: Interpolation-only expressions are deprecated
}

data "null_data_source" "b" {
  has_computed_default = "${var.foo}" # WARNING: Interpolation-only expressions are deprecated
}

output "output" {
  value = "${var.foo}" # WARNING: Interpolation-only expressions are deprecated
}

locals {
  foo = "${var.foo}" # WARNING: Interpolation-only expressions are deprecated
}
