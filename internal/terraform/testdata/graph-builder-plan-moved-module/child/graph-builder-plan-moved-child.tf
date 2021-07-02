terraform {
  experiments = [config_driven_move]
}

resource "test_object" "a" {
}

resource "test_object" "b" {
}

resource "test_object" "c" {
}

resource "test_object" "d" {
}

moved {
  from = test_object.c
  to   = test_object.d
}
