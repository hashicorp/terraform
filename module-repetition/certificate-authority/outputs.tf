output "cert_pem" {
  value = tls_self_signed_cert.ca.cert_pem
}

output "private_key_pem" {
  value = tls_private_key.ca.private_key_pem
}

output "key_algorithm" {
  value = tls_private_key.ca.algorithm
}
