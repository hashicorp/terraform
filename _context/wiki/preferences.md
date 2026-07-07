# Working Preferences

## Code style and standards

- **Follow existing patterns.** Terraform's codebase has established conventions for a reason. When implementing something new, find the closest analogous existing code and mirror its structure. Do not introduce new abstractions or patterns without a compelling reason.
- **No feature implementation in `command/`.** The `command/` package is for CLI wiring only. Business logic and new feature behaviour belongs in engine packages (primarily `internal/terraform` and related packages).
- **Account for all language features.** Terraform's HCL language is extensive. Before considering an implementation complete, verify it handles (or explicitly rejects with a good error): ephemeral values, ephemeral variables, provider-defined functions, `moved` blocks, `import` blocks, `check` blocks, sensitive values, `for_each`/`count` expansion, module calls, and any other relevant language features.
- **Validate product proposals.** The product team may not have deep knowledge of Terraform internals. Always fact-check proposals against the actual codebase before treating them as technically sound.

## Testing preferences

- **No giant table-driven tests.** Large monolithic table tests are hard to read and debug. It is okay to write multiple smaller, focused test functions with some repetition. DRY is not the primary goal in tests — clarity and debuggability are.
- **Tests should match existing test style.** Look at existing tests in the package being modified and mirror their structure and naming conventions.
- **Quality over quantity.** A few well-written focused tests are preferred over many auto-generated tests that lack precision.

## AI collaboration preferences

- **Investigate before answering.** Always read the relevant code before making claims about how something works. Do not speculate.
- **Fact-check proposals.** A key use of AI on this project is validating whether product or design proposals are feasible and correct given Terraform's actual internals. Approach this critically.
- **Issue checking.** AI is useful for spotting problems, edge cases, and missed language-feature coverage in proposed or existing code.
- **Concise by default, detailed when needed.** Prefer focused, actionable responses. Add depth when the complexity of the problem warrants it.
- **Minimal change.** Implement exactly what is asked. Do not refactor surrounding code, add unrequested features, or over-engineer solutions.
- **No speculative error handling.** Do not add error handling for scenarios that cannot occur given the actual code paths.

## Communication preferences

- Be direct and technical.
- Call out concerns or problems clearly rather than hedging.
- When something in a proposal conflicts with how Terraform actually works, say so explicitly.
