# Running Consul for Terraform Acceptance Tests

## TLS

Some of the acceptance tests for the `consul` provider use
TLS. To service these tests, a Consul server must be started
with HTTPS enabled with TLS certificates.

### Test fixtures

File | Description
--- | ---
`agent.json.example` | Configures the Consul agent to respond to HTTPS requests, and verifies the authenticity of HTTPS requests
`agentcert.pem` | A PEM-encoded certificate used by the Consul agent, valid only for 127.0.0.1 signed by `cacert.pem`, expires 2026
`agentkey.pem` | A PEM-encoded private key used by the Consul agent
`cacert.pem` | A PEM-encoded Certificate Authority, expires 2036
`usercert.pem` | A PEM-encoded certificate used by the Terraform acceptance tests, signed by `cacert.pem`, expires 2026
`userkey.pem` | A PEM-encoded private key used by the Terraform acceptance tests

### Start

Start a Consul server configured to serve HTTP traffic, and validate incoming
HTTPS requests.

    ~/.go/src/github.com/hashicorp/terraform> consul agent \
        -bind 127.0.0.1 \
        -data-dir=/tmp \
        -dev \
        -config-file=builtin/providers/consul/text-fixtures/agent.json.example \
        -server

### Test

With TLS, `CONSUL_HTTP_ADDR` must match the Common Name of the agent certificate.

    ~/.go/src/github.com/hashicorp/terraform> CONSUL_CERT_FILE=test-fixtures/usercert.pem \
        CONSUL_KEY_FILE=test-fixtures/userkey.pem \
        CONSUL_CA_FILE=test-fixtures/cacert.pem \
        CONSUL_SCHEME=https \
        CONSUL_HTTP_ADDR=127.0.0.1:8943  \
        make testacc TEST=./builtin/providers/consul/
