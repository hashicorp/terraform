output "security_group" {
  value = "${aws_security_group.default.id}"
}

output "launch_configuration" {
  value = "${aws_launch_configuration.web-lc.id}"
}

output "asg_name" {
  value = "${aws_autoscaling_group.web-asg.id}"
}

output "elb_name" {
  value = "${aws_elb.web-elb.dns_name}"
}
