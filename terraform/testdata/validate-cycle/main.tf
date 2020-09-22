provider "aws" { }

/*
 * When a CBD resource depends on a non-CBD resource,
 * a cycle is formed that only shows up when Destroy
 * nodes are included in the graph.
 */
resource "aws_security_group" "firewall" {
}

resource "aws_instance" "web" {
    security_groups = [
        "foo",
        "${aws_security_group.firewall.foo}"
    ]
    lifecycle {
      create_before_destroy = true
    }
}
