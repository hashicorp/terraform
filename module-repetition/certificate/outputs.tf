output "certificate_pem" {
  value = tls_locally_signed_cert.cert.cert_pem
}
