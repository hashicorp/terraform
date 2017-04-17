resource "test" "A" {}
resource "test" "B" {
    count = 2
    value = "${test.A.*.value}"
}
