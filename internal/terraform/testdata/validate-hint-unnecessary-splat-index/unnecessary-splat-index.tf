
resource "test_thing" "a" {
  count = 1
}

resource "test_thing" "b" {
  name = test_thing.a[*].id[0]
}

resource "test_thing" "c" {
  count = length(test_thing.a)

  name = test_thing.a[*].id[count.index]
}

resource "test_thing" "d" {
  name = test_thing.a[*].id["not a number"]
}
