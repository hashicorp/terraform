
resource "test_thing" "a" {
  count = 1
}

resource "test_thing" "b" {
}

resource "test_thing" "c" {
}

resource "test_thing" "d" {
}

resource "test_thing" "e" {
}

resource "test_thing" "f" {
  count = 1
}

resource "test_thing" "depender" {
  name = test_thing.a[0].name

  single_block {
    id = test_thing.c.name
  }

  map_block "boop" {
    doodad = test_thing.d.name
  }

  depends_on = [
    test_thing.a,
    test_thing.b,
    test_thing.b,
    test_thing.c,
    test_thing.d,
    test_thing.f[0],
    test_thing.f,

    # This one is okay because this is the only reference to it.
    test_thing.e,
  ]
}
