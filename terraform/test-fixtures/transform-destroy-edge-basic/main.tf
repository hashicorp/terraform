resource "test" "A" {}
resource "test" "B" { value = "${test.A.value}" }
