provider "aws" {
  region = "us-west-2"
}

resource "aws_launch_configuration" "foobar" {
    name_prefix = "test-"
    image_id = "ami-21f78e11"
    instance_type = "t1.micro"
}

resource "aws_autoscaling_group" "foobar" {
    availability_zones = ["us-west-2a"]
    name = "test"
    max_size = 1
    min_size = 1
    health_check_grace_period = 300
    health_check_type = "ELB"
    force_delete = true
    termination_policies = ["OldestInstance"]
    launch_configuration = "${aws_launch_configuration.foobar.name}"
    tag {
        key = "Foo"
        value = "foo-bar"
        propagate_at_launch = true
    }
}

resource "aws_autoscaling_schedule" "foobar" {
    scheduled_action_name = "foobar"
    min_size = 0
    max_size = 1
    desired_capacity = 0
    start_time             = "2017-12-11T18:00:00Z"
  end_time               = "2017-12-12T06:00:00Z"
    autoscaling_group_name = "${aws_autoscaling_group.foobar.name}"
}
