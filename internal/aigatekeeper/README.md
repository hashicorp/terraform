# AI Gatekeeper for Terraform

The `aigatekeeper` package implements the AI-mediated zero-trust handshake for Terraform.

## Feature Flag

To enable the AI-mediated handshake, set the following environment variable:

```sh
export TF_ENABLE_AI_MEDIATED_RPC=1
```

When enabled, Terraform will negotiate an `AITripartiteHandshakeRequest` with an AI Gatekeeper service to obtain a signed JWT Authorization Token. The token encapsulates permissions for resource types, providers, and capabilities.

## Interceptors

The `rpcapi.AIGatekeeperInterceptor` validates incoming gRPC calls against the JWT authorization token.
If the authorization is missing or invalid, the interceptor denies the request and returns `PERMISSION_DENIED`.

## Minimal Design

This design eliminates the need for full RPC proxying by relying on gRPC interceptors and cryptographically signed JWTs, ensuring minimal latency overhead and adhering to standard capability-based security.
