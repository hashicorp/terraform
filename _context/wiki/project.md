# Project Overview

## What this project is

The **HashiCorp Terraform CLI** — the source-available, locally-executed tool that reads HCL configuration and runs `plan`, `apply`, and related lifecycle operations against infrastructure providers. The Terraform Cloud/Enterprise platform is not a separate concern in the pure sense: it embeds and calls the CLI directly, and there is also a protocol layer between the two (`internal/rpcapi`). Some cloud-integration code lives in this repo (the Stacks feature, the cloud backend driver). The primary focus of this work is the core CLI and execution engine, but changes to core can directly affect the cloud platform's behaviour.

## Goals and current focus

Work covers the full spectrum: bug fixes, new features, performance improvements, and refactoring. There is no single dominant mode — any of these may be the priority at a given time.

## Key stakeholders and users

- **End users**: engineers and operators who write Terraform configuration and run the CLI to manage infrastructure.
- **Product team**: proposes features and changes; their proposals need careful technical review because they may not reflect deep knowledge of Terraform internals.
- **Provider ecosystem**: providers communicate with Terraform via the provider protocol; changes to core must not silently break the protocol contract.
- **Terraform Cloud/Enterprise platform**: embeds and calls the CLI via the RPC API; core changes can directly affect platform behaviour.

---

## Package map

### Core execution engine

| Package | Role |
|---------|------|
| `internal/terraform` | The heart of Terraform. Graph-based execution engine for plan, apply, destroy, validate, import, refresh. Node types represent every object kind (resources, providers, modules, outputs, locals, variables, checks, etc). Most active development area. |
| `internal/plans` | Data model for plan objects: changes, actions, deferred changes, plan files, objchange validation. |
| `internal/states` | State model: module state, resource instance objects, output values. Also contains `statefile` (serialisation) and `statemgr` (locking/persistence). |
| `internal/configs` | Configuration loading and representation. Parses HCL into typed structs consumed by the engine. The config tree is the starting point for almost every operation. |
| `internal/addrs` | Address types for resources, modules, providers, instances, outputs, etc. Used pervasively throughout the codebase — correctness here is foundational. |
| `internal/instances` | Instance key expansion (for_each/count). Tracks which instance keys exist for each module/resource. |
| `internal/namedvals` | Stores resolved values for named objects (variables, locals, outputs) during a walk. |
| `internal/checks` | Tracks the results of custom conditions (`precondition`, `postcondition`, `check` blocks). |
| `internal/refactoring` | Implements `moved` block processing and `removed` block processing. |
| `internal/genconfig` | Generates HCL configuration from imported resource state (used by `terraform import`). |

### Language and expression evaluation

| Package | Role |
|---------|------|
| `internal/lang` | Expression evaluation scope, built-in functions, marks propagation, ephemeral value handling. The `Scope` type is what the engine uses to evaluate expressions in a given context. |
| `internal/lang/marks` | Mark types carried through `go-cty` values: `Sensitive`, `Ephemeral`, `TypeType` (for the console `type()` function), and `DeprecationMark` (carries deprecation message + origin). Any code that unmarks values must handle all of these correctly. |
| `internal/lang/funcs` | Implementations of Terraform's built-in functions (file, cidrhost, toset, etc). |
| `internal/lang/ephemeral` | Handling for ephemeral values and their lifecycle within expression evaluation. |
| `internal/tfdiags` | Diagnostic types used throughout the codebase (wraps HCL diagnostics, adds Terraform-specific context). Every error and warning surface goes through this. |

### Provider protocol

| Package | Role |
|---------|------|
| `internal/providers` | The `Interface` type — the Go interface all provider implementations satisfy. Also contains provider schema types, ephemeral resource interface, functions interface, mock providers. |
| `internal/tfplugin6` | Protocol Buffers generated code for the provider protocol v6 (gRPC). The wire format for provider communication. |
| `internal/tfplugin5` | Protocol Buffers generated code for provider protocol v5 (legacy). |
| `internal/plugin` | gRPC plugin client for provider protocol v5. |
| `internal/plugin6` | gRPC plugin client for provider protocol v6. |
| `internal/grpcwrap` | Wraps a `providers.Interface` Go implementation into a gRPC server for either v5 or v6. Primarily used to build in-process test provider binaries. |

### Stacks (cloud orchestration layer)

| Package | Role |
|---------|------|
| `internal/stacks` | Top-level directory for Terraform Stacks — an orchestration layer over Terraform modules, used by the cloud platform. |
| `internal/stacks/stackruntime` | The Stacks runtime: planning and applying stacks configurations. The dynamic behaviour of the stacks language lives here. |
| `internal/stacks/stackconfig` | Config loading/parsing for the stacks language (analogous to `configs` for modules). |
| `internal/stacks/stackaddrs` | Address types for stacks objects (analogous to `addrs`). |
| `internal/stacks/stackplan`, `stackstate` | Plan and state models for stacks. |
| `internal/stacks/tfstackdata1` | Internal protobuf schema for stacks plan/state persistence. Not a public API. |
| `internal/stacksplugin` | Plugin protocol for the Stacks CLI command. `command/stacks.go` delegates to an external Stacks plugin binary via this package, rather than calling `internal/stacks` or `internal/rpcapi` directly. |

### RPC API (integration protocol with cloud platform)

| Package | Role |
|---------|------|
| `internal/rpcapi` | The Terraform Core RPC API. A gRPC server (over `go-plugin`) that exposes Terraform Core functionality to external callers. Intended as the public interface to the Stacks runtime for external programs; the `rpcapi` sub-command is the entry point. CLI code that directly orchestrates Stacks should use this via `rpcapi.NewInternalClient`, not import `internal/stacks` directly. Currently experimental/subject to change. |

### Backend and state storage

| Package | Role |
|---------|------|
| `internal/backend` | Defines the `Backend` interface (state storage and workspace management). |
| `internal/backend/backendrun` | Defines the `OperationsBackend` interface for backends that can actually *run* plan/apply operations, plus the `Operation` type that carries all operation inputs. Separate from the base `Backend` interface — not all backends implement this. |
| `internal/backend/remote` | The remote backend for Terraform Cloud/Enterprise (older integration path). |
| `internal/cloud` | The cloud backend — the newer Terraform Cloud integration. |
| `internal/backend/remote-state/` | Remote state backends: S3, GCS, Azure, Consul, Kubernetes, OCI, OSS, PostgreSQL. Each is a separate Go module. |

### CLI layer

| Package | Role |
|---------|------|
| `internal/command` | CLI entry points. Each subcommand is implemented here. **This is wiring only — feature logic does not belong here.** |
| `internal/command/arguments` | Typed argument/flag parsing for each command. |
| `internal/command/views` | Output rendering (human, JSON) for each command. |
| `internal/command/jsonplan`, `jsonstate`, `jsonconfig`, etc. | JSON output format implementations. |
| `internal/command/format` | Human-readable diff and state formatting. |

### Testing infrastructure

| Package | Role |
|---------|------|
| `internal/moduletest` | The `terraform test` command's test runner, graph, mocking, and state management. |
| `internal/e2e` | End-to-end test helpers that spawn a real `terraform` binary. |
| `internal/command/e2etest` | End-to-end tests for CLI commands. |

### Infrastructure / utility packages

| Package | Role |
|---------|------|
| `internal/dag` | The directed acyclic graph library underlying the execution engine. Do not modify — the graph primitives are stable and foundational. |
| `internal/experiments` | Opt-in experiment flags for language features that are not yet stable. Activated per-module via `experiments = [...]` in `terraform` block. Features behind experiments may break in any release until stabilised. **Only available in alpha builds or local builds compiled with `-ldflags="-X 'main.experimentsAllowed=yes'"`.** |
| `internal/promising` | A deadlock-free promise/task library used by the Stacks runtime for concurrent evaluation. |
| `internal/getproviders` | Provider installation: registry queries, version constraints, package download and verification. |
| `internal/getmodules` | Module installation from registries, VCS, and local paths. |
| `internal/providercache` | Local provider cache management. |
| `internal/depsfile` | The `.terraform.lock.hcl` dependency lock file model. |
| `internal/initwd` | Working directory initialisation logic (used by `terraform init`). |
| `internal/registry` | Terraform Registry client. |
| `internal/repl` | The interactive console (`terraform console`). |
| `internal/collections` | Generic collection types used across the codebase. |

---

## Key external dependencies

| Library | Role |
|---------|------|
| `github.com/hashicorp/hcl/v2` | The HCL language library. Config parsing, expression evaluation, traversal, syntax errors. Understanding the `hcl.Body`, `hcl.Expression`, `hcl.EvalContext` types is essential for config and lang work. |
| `github.com/zclconf/go-cty` | The type system and value library underlying HCL and the provider protocol. All Terraform values are `cty.Value`; all schemas express types as `cty.Type`. Critical to understand for any engine or provider work. |
| `github.com/zclconf/go-cty-yaml` | YAML serialisation for cty values. |
| `github.com/hashicorp/go-plugin` | Plugin system used to launch and communicate with provider processes over gRPC. |
| `google.golang.org/grpc` + `google.golang.org/protobuf` | gRPC and protobuf runtime — used for both the provider protocol and the RPC API. |
| `github.com/hashicorp/go-tfe` | Go client for the Terraform Cloud/Enterprise API (used by the cloud backend). |
| Provider SDKs | Providers are developed using either `terraform-plugin-framework` or `terraform-plugin-sdk/v2`. When making changes that affect provider interactions, awareness of both SDK patterns is important. These are external repos but core developers need to understand the contract. |

---

## Architecture notes

- The **graph-walk engine** in `internal/terraform` drives both plan and apply. Graph nodes represent every object type; edges encode dependencies. Understanding the node type hierarchy and how transforms build the graph is essential for engine-level work. The DAG itself (`internal/dag`) should not be modified.
- The **provider protocol** is versioned (v5, v6) and must remain stable. Core talks to providers via gRPC over a local socket managed by `go-plugin`. Schema negotiations happen at startup.
- The **RPC API** (`internal/rpcapi`) is the architectural boundary between Terraform Core and external callers including the cloud platform. New functionality that crosses this boundary must go through the RPC API.
- **`go-cty` marks** are how sensitive and ephemeral values flow through evaluation. Any new feature that deals with values must correctly propagate marks — stripping them silently is a bug.
- Terraform's configuration language is **feature-rich**: ephemeral values, ephemeral variables, provider-defined functions, `moved` blocks, `removed` blocks, `import` blocks, `check` blocks, sensitive values, `for_each`/`count` expansion, module calls, workspaces, and experiments. Any new feature must be audited against all of these — even if the feature doesn't support a construct, it must handle encountering it gracefully.

## Things to watch out for

- Product proposals may not account for how the engine actually works. Validate them against the codebase before accepting them as correct.
- New features must be audited against the full set of language features. Missing a case (e.g., forgetting ephemeral variables or provider-defined functions) is a common source of subtle bugs.
- CLI commands are wiring only — feature logic belongs in engine packages.
- The DAG library is foundational and stable. Changes to it have wide blast radius and are almost never the right solution.
- Any code touching the provider protocol must consider both v5 and v6. The `grpcwrap` package wraps Go provider implementations into gRPC servers (used in testing), not a general protocol bridge.
