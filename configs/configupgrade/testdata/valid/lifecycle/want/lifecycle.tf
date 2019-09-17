resource "test_instance" "foo" {
  lifecycle {
    create_before_destroy = true
    prevent_destroy       = true
  }
}

resource "test_instance" "bar" {
  lifecycle {
    ignore_changes = all
  }
}

resource "test_instance" "baz" {
  lifecycle {
    ignore_changes = [
      image,
      tags.name,
    ]
  }
}

resource "test_instance" "boop" {
  lifecycle {
    ignore_changes = [image]
  }
}
