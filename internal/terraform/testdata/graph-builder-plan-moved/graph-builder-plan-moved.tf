terraform {
  experiments = [config_driven_move]
}

resource "test_object" "b" {
}

resource "test_object" "d" {
}

moved {
  from = test_object.b["foo"]
  to   = test_object.b
}

moved {
  from = test_object.c
  to   = test_object.d
}

moved {
  from = test_object.e
  to   = test_object.f # Note: this resource intentionally not declared
}
