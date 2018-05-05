resource "test" "A" {}

resource "test" "B" {
  value = "${test.A.value}"
}

resource "test" "C" {
  value = "${test.B.value}"
}
