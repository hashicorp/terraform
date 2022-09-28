resource "aws_computed_source" "intermediates" {}

module "test_mod" {
  source = "./mod"

  services = [
    {
      "exists" = "true"
      "elb"    = "${aws_computed_source.intermediates.computed_read_only}"
    },
    {
      "otherexists" = " true"
      "elb"         = "${aws_computed_source.intermediates.computed_read_only}"
    },
  ]
}
