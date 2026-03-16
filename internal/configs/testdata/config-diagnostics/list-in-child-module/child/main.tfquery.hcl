list "test_resource" "test" {
  provider = azurerm
  count = 1
  
  config {
    tags = {
      Name = "test"
    }
  }
}