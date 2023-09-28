
variable "interrupts" {
  type = number
}

resource "test_resource" "primary" {
  value = "primary"
}

resource "test_resource" "secondary" {
  value = "secondary"
  interrupt_count = var.interrupts

  depends_on = [
    test_resource.primary
  ]
}

resource "test_resource" "tertiary" {
  value = "tertiary"

  depends_on = [
    test_resource.secondary
  ]
}
