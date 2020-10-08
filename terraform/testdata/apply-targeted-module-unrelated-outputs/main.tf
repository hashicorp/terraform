resource "aws_instance" "foo" {}

module "child1" {
  source = "./child1"
  instance_id = "${aws_instance.foo.id}"
}

module "child2" {
  source = "./child2"
}

output "child1_id" {
  value = "${module.child1.instance_id}"
}

output "child1_given_id" {
  value = "${module.child1.given_instance_id}"
}

output "child2_id" {
  # This should get updated even though we're targeting specifically
  # module.child2, because outputs are implicitly targeted when their
  # dependencies are.
  value = "${module.child2.instance_id}"
}

output "all_ids" {
  # Here we are intentionally referencing values covering three different scenarios:
  # - not targeted and not already in state
  # - not targeted and already in state
  # - targeted
  # This is important because this output must appear in the graph after
  # target filtering in case the targeted node changes its value, but we must
  # therefore silently ignore the failure that results from trying to
  # interpolate the un-targeted, not-in-state node.
  value = "${aws_instance.foo.id} ${module.child1.instance_id} ${module.child2.instance_id}"
}
