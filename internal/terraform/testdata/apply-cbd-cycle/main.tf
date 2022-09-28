resource "test_instance" "a" {
  foo = test_instance.b.id
  require_new = "changed"

  lifecycle {
    create_before_destroy = true
  }
}

resource "test_instance" "b" {
  foo = test_instance.c.id
  require_new = "changed"
}


resource "test_instance" "c" {
  require_new = "changed"
}

