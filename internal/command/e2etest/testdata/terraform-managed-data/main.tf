resource "mnptu_data" "a" {
}

resource "mnptu_data" "b" {
  input = mnptu_data.a.id
}

resource "mnptu_data" "c" {
  triggers_replace = mnptu_data.b
}

resource "mnptu_data" "d" {
  input = [ mnptu_data.b, mnptu_data.c ]
}

output "d" {
  value = mnptu_data.d
}
