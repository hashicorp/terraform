# Change File Style Guide

This guide governs the content of the `body` field in change files under `.changes/v*/`.
The body is rendered as a bullet point in `CHANGELOG.md`.

**Primary audience:** Terraform practitioners — people writing `.tf` files and running CLI commands day-to-day. Write for them, not for contributors or maintainers who know the internals.

---

## File structure

Each change file is a small YAML file placed in the version directory for the release it targets (e.g. `.changes/v1.16/`).

```yaml
kind: ENHANCEMENTS
body: "workspace: The `workspace list` command can now produce machine-readable output when supplied with the `-json` flag"
time: 2026-04-17T11:06:28.651099+01:00
custom:
  Issue: "38397"
```

### Fields

| Field          | Value                                                                                                            |
| -------------- | ---------------------------------------------------------------------------------------------------------------- |
| `kind`         | One of: `NEW FEATURES`, `ENHANCEMENTS`, `BUG FIXES`, `NOTES`, `UPGRADE NOTES`, `BREAKING CHANGES`.               |
| `body`         | The user-facing description — see the rest of this guide                                                         |
| `time`         | RFC3339 timestamp with timezone offset of when the file was created                                              |
| `custom.Issue` | GitHub PR as a quoted string (e.g. `"38397"`). The field is called "Issue" for legacy reasons and is misleading! |

### Filename format

`<KIND>-<YYYYMMDD-HHmmss>.yaml`

Multi-word kinds include a space: `BUG FIXES-20260401-152120.yaml`, `UPGRADE NOTES-20260330-145227.yaml`.

---

## Core principles for `body`

1. **User-focused, not implementation-focused.** Describe what the user can now do, or what they will now experience. Do not describe what changed internally.
2. **Present tense, active voice.** Write as if describing the world as it is after the release, not what the team did to produce it.
3. **One sentence.** Concise but complete. Only expand to two sentences for `UPGRADE NOTES` (change + call to action).

---

## Area prefix

When the change is scoped to a specific command, feature area, or subsystem, lead with a **lowercase** prefix followed by a colon and a space.

```
init: The `-upgrade` flag now ...
workspace: The `workspace list` command now ...
test: Terraform now raises a warning when ...
stacks: Output values are now included in ...
console: The `terraform console` command now ...
graph: The `terraform graph` command can now ...
state show: The `state show` command can now ...
cloud: ...
policy: ...
```

Omit the prefix only when the change is genuinely cross-cutting (affects all commands, or the core language itself).

**Capitalise the first word after the prefix:**

```
✅  init: Errors due to incompatible flags are now raised earlier
❌  init: errors due to incompatible flags are now raised earlier
```

**When there is no prefix, capitalise the first word of the body:**

```
✅  `import` blocks now correctly respect provider local names
❌  import blocks no longer ignore provider local names
```

---

## Verb tense and voice

Always use **present tense, active voice**. The subject is almost always Terraform, a specific command, or a language construct.

| ❌ Avoid                                                       | ✅ Prefer                                                                                                |
| -------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- |
| `add support for import blocks inside modules`                 | `init: Import blocks inside modules are now supported`                                                   |
| `Fixed crash when configuration has an invalid action_trigger` | `stacks: Terraform no longer panics when a configuration contains an invalid \`action_trigger\` block`   |
| `A -json flag was added to state show`                         | `state show: The \`state show\` command can now produce machine-readable output with the \`-json\` flag` |

---

## Technical identifiers

Always wrap the following in backticks:

- CLI commands and subcommands: `` `terraform plan` ``, `` `workspace list` ``
- Flags and options: `` `-json` ``, `` `-scope=<module address>` ``
- Block types and labels: `` `lifecycle` ``, `` `removed` ``, `` `action_trigger` ``
- Attribute and argument names: `` `skip_cleanup` ``, `` `bastion_host_key` ``, `` `for_each` ``
- Function names: `` `contains()` ``, `` `merge()` ``
- Special values: `` `null` ``

Do **not** use quotes in place of backticks:

```
❌  The "workspace list" command now ...
✅  The `workspace list` command now ...
```

---

## Per-category guidance

### NEW FEATURES

Describe what the user can now do that they couldn't before. Name the HCL syntax, block, flag, or concept they interact with.

```
✅  `import` blocks inside modules are now supported
✅  `terraform_data`: The new `store` block can hold ephemeral and sensitive values across plan and apply
```

### ENHANCEMENTS

Describe the improvement to existing behaviour. Lead with the area prefix where applicable. Focus on the new capability unlocked, not the mechanism added internally.

```
✅  console: The `terraform console` command now accepts an optional `-scope=<module address>` flag,
    which can be used to evaluate expressions within the scope of a specific module instance
✅  graph: The `terraform graph` command can now output graphs in Mermaid format using the `-format=mermaid` flag

❌  graph: add -format flag for Mermaid output
❌  console: implement -scope flag
```

### BUG FIXES

Describe the **correct behaviour now in place**, not the bug that existed. Use "now correctly", "no longer", "now raises" as natural anchors.

```
✅  init: Terraform no longer removes locks from the dependency lock file for providers configured as `dev_override`
✅  workspace: Terraform now raises an error if an invalid workspace name becomes selected due to out-of-band changes
✅  `import` blocks now correctly respect provider local names

❌  Fix a panic when the plan contained a no-op change for a deposed object
❌  Fixed crash when configuration has invalid action_trigger
```

When the bug only manifests under specific conditions, include enough context for users to recognise whether they were affected — but keep it to one sentence:

```
✅  `terraform apply` no longer panics when the plan contains a no-op change for a deposed resource
    that has `lifecycle.precondition` or `lifecycle.postcondition` blocks
```

### NOTES

Use for non-breaking behavioural changes or clarifications that don't fit neatly into `ENHANCEMENTS` or `BUG FIXES`. Same tense and voice rules apply.

### UPGRADE NOTES

Describe the breaking or potentially breaking change, then include an explicit **call to action** — what the user must review, verify, or change before or after upgrading.

Use direct imperative language addressed to the user ("Review...", "Update...", "Verify..."), not passive constructions ("should be verified", "may need to be updated").

```
✅  `bastion_host_key` is now correctly applied by provisioners. Review your provisioner configurations
    to verify the configured key is correct before upgrading.

❌  Provisioner bastion_host_key is now correctly applied. Existing usage of bastion_host_key should
    verify the configured key is correct.
```

---

## Anti-patterns

| Anti-pattern                                                                          | Why                                                             | Fix                                                                               |
| ------------------------------------------------------------------------------------- | --------------------------------------------------------------- | --------------------------------------------------------------------------------- |
| Internal framing: `"add output values to plan component instance change description"` | Describes what the developer did, not what the user experiences | `"stacks: Plan output now includes output values for component instance changes"` |
| Commit-message style: `"Fix crash"`, `"Add support for"`                              | Reads as a commit, not a user-facing note                       | `"Terraform no longer crashes when..."`, `"X is now supported"`                   |
| No area prefix when the change is command-specific                                    | Hard to scan in the rendered CHANGELOG                          | Add the relevant prefix: `"init: ..."`, `"test: ..."`                             |
| Identifiers without backticks: `"The -json flag"`                                     | Inconsistent; harder to parse                                   | ``"The `-json` flag"``                                                            |
| Lowercase sentence start: `"import blocks no longer..."`                              | Reads as a fragment                                             | ``"`import` blocks now correctly..."``                                            |
| Passive voice: `"Errors are now raised earlier"`                                      | Hides the subject                                               | `"Terraform now raises errors earlier"`                                           |
| Over-long body explaining the full feature                                            | The CHANGELOG is a summary; the issue link provides detail      | One sentence max; link the issue                                                  |

---

## Quick checklist

Before committing a change file:

- [ ] Body is written from the user's perspective (impact, not implementation)?
- [ ] Present tense, active voice?
- [ ] Correct lowercase area prefix (with colon and space) if scoped to a command?
- [ ] First word after the prefix (or at the sentence start) is capitalised?
- [ ] All CLI flags, block names, function names, and identifiers are in backticks?
- [ ] `BUG FIXES`: describes the correct behaviour now in place, not the old bug?
- [ ] `UPGRADE NOTES`: includes a direct call to action using imperative language?
- [ ] One sentence (two for `UPGRADE NOTES`)?
