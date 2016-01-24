resource "aws_db_instance" "default" {
  depends_on = ["aws_security_group.default"]
  identifier = "${var.identifier}"
  allocated_storage = "${var.storage}"
  engine = "${var.engine}"
  engine_version = "${lookup(var.engine_version, var.engine)}"
  instance_class = "${var.instance_class}"
  name = "${var.db_name}"
  username = "${var.username}"
  password = "${var.password}"
  vpc_security_group_ids = ["${aws_security_group.default.id}"]
  db_subnet_group_name = "${aws_db_subnet_group.default.id}"
}

resource "aws_db_subnet_group" "default" {
  name = "main_subnet_group"
  description = "Our main group of subnets"
  subnet_ids = ["${aws_subnet.subnet_1.id}", "${aws_subnet.subnet_2.id}"]
}
