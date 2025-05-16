list "test_resource" "test" {
  provider = azurerm
  count = 1
  tags = {
    Name = "test"
  }
}