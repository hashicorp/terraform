resource "test_instance" "from_list" {
  list_of_obj = [
    {},
    {},
  ]
}

resource "test_instance" "already_blocks" {
  list_of_obj {}
  list_of_obj {}
}

resource "test_instance" "empty" {
  list_of_obj = []
}
