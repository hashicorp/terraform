locals {
  cert_options = <<EOF
--cert-file=/etc/ssl/etcd/server.crt \
  --peer-trusted-ca-file=/etc/ssl/etcd/ca.crt \
  --peer-client-cert-auth=trueEOF

}

output "local" {
value = local.cert_options
}
