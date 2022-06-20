locals {
  a = "hello world"
  b = 2
  single = test_thing.single.id
}

resource "test_thing" "single" {
  string = local.a
  number = local.b

}

resource "test_thing" "for_each" {
  for_each = {"q": local.a}

  string = local.a

  single {
    z = test_thing.single.string
  }
}

resource "test_thing" "count" {
  for_each = length(local.a)

  string = local.a
}

module "single" {
  source = "./child"

  a = test_thing.single
}

module "for_each" {
  source   = "./child"
  for_each = {"q": test_thing.single}

  a = test_thing.single
}

module "count" {
  source = "./child"
  count  = length(test_thing.single.string)

  a = test_thing.single
}
