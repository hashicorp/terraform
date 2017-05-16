output "CDN Endpoint ID" {
  value = "${azurerm_cdn_endpoint.cdnendpt.name}.azureedge.net"
}
