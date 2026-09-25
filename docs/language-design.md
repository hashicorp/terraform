# Hitchhiker's guide to language design

Terraform maintains a growing family of related languages, including Terraform configuration, Search/Query, Test, Stacks, and Policy. This document captures several hard-earned principles for designing them:

- [preserve compatibility](#1-once-released-the-language-must-behave-exactly-like-this-forever)
- [assume every capability will reach mission-critical use](#2-whatever-your-language-allows-a-user-to-expressed-will-be-expressed-in-a-mission-critical-workflow-of-at-least-one-very-important-customer)
- [handle language-version differences](#3-not-all-code-to-be-consumed-was-written-for-a-version-of-the-language-that-is-currently-running)
- [make authorship delightful](#5-make-the-authorship-experience-delightful)
- [design features to cooperate](#6-your-feature-needs-cooperate-with-every-other-feature)
- [balance openness with constraints](#7-a-balancing-act-between-openness-and-closedness)
- [anticipate the complexity DevOps practitioners will create](#8-never-underestimate-the-complexity-that-can-be-invented-by-devops).

Language features are differently shaped from conventional product features. Once released, their syntax, semantics, diagnostics, APIs, and artifacts become durable contracts. Syntax Designers (Engineering / Product) must consider where and how a feature will be used, how it behaves with unknown, ephemeral, or sensitive values, and how it interacts with resource lifecycles, graph operations, providers, automation, and other language features.

Many of these dependencies only become visible through design and prototyping. A feature may work during create and update but fail during destroy; omitting an integration to reduce scope may block customer adoption; and independently developed features may create circular dependencies when combined.

As a result, visible feature size is a poor predictor of delivery time. Language work is less predictable early on, but confidence improves as contracts, invalid states, interactions, and real-world uses are discovered. This additional design time reduces migration costs, prevents adoption-blocking gaps, and protects the compatibility trust Terraform has built over time.

## Contents

- [Background](#background)
- [Guidelines](#guidelines)
  - [1. Once released the language must behave exactly like this (forever)](#1-once-released-the-language-must-behave-exactly-like-this-forever)
  - [2. Whatever your language allows a user to expressed will be expressed in a mission critical workflow of at least one very important customer](#2-whatever-your-language-allows-a-user-to-expressed-will-be-expressed-in-a-mission-critical-workflow-of-at-least-one-very-important-customer)
  - [3. Not all code to be consumed was written for a version of the language that is currently running](#3-not-all-code-to-be-consumed-was-written-for-a-version-of-the-language-that-is-currently-running)
  - [4. Great Error Messaging brings your language to the next level](#4-great-error-messaging-brings-your-language-to-the-next-level)
  - [5. Make the authorship experience delightful](#5-make-the-authorship-experience-delightful)
  - [6. Your feature needs cooperate with every other feature](#6-your-feature-needs-cooperate-with-every-other-feature)
  - [7. A balancing act between openness and closeness](#7-a-balancing-act-between-openness-and-closedness)
  - [8. Never underestimate the complexity that can be invented by DevOps](#8-never-underestimate-the-complexity-that-can-be-invented-by-devops)
- [Building a language interfacing with Terraform](#building-a-language-interfacing-with-terraform)
- [Further Reading List](#further-reading-list)

## Background

This document tries to convey the basic foundational principles we apply to language design. It is probably nowhere near complete, but it should be treated as the baseline standard for things we need to consider when designing a new language or adding features to a new language.

For a long time the Terraform Core team has been primarily responsible for building and maintaining programming languages in our side of the org. This is slowly changing with new capabilities like Stacks, Policy and Search coming with their own HCL-based language.

This means a more diverse group of people (yay!) are designing and building languages. We have done little so far to support folks structurally, primarily there are a few hard-earned truths leading to guidelines we try to follow that are not shared out but only passed down on occasion.

This document tries to at least provide some guidance and structure to people eager to break into language design. Some ideas might overlap with traditional software engineering best practices, I'm highlighting them because of more severe implications in the context of language design.

## Guidelines

Instead of giving you the exact guidelines right away I want to build an understanding on why they are in place to begin with. Therefore we will start from hard-earned truths, sprinkle in some examples before moving on to the concrete guidelines.

### 1. Once released the language must behave exactly like this (forever)

People expect a very high bar of quality and stability from HashiCorp products, especially from the languages we roll out. For them a language feature must behave exactly the same in a newer version of the language than it did on an older version. This includes the syntax and the semantics. The same configuration must be equally valid in the next version as it is in the current version and behave exactly the same.

If it was not clearly a bug that we fix neither patch nor minor versions can break any contracts, this includes

- Syntax and Semantics
- Diagnostics: the level must be the same or more permissive as before, but a previously valid syntax may not error. Also the messaging should stay similar, some CI pipelines (and HCPT) regex on certain error messages
  - The only exception to the rule would be deprecation warnings ahead of a breaking change
- APIs: Any (e.g. GRPC) API needs to work with older versions of the consumer and any CLI flags or commands need to remain unchanged
- Artifacts: Anything produced that can be consumed by automations will be depended on in a critical workflow. This ranges from state files over log outputs to details like address strings being a certain format
- CLI exit codes: Not only must non-zero exit code stay non-zero exit codes, it also needs to stay the same non-zero value

The bottom line is we don't know which parts of our product the users will rely on therefore to not break our users workflows we must be conscious of the implicit contracts we have.

This means when designing a new feature or enhancing an existing one we need to consider if this change might break any of the existing contracts. At the same time we need to think about the new contracts this feature will form and if the cost of honoring these contracts is worth the upside they bring.

HashiCorp in general and Terraform specifically has build a lot of trust with our customers and the community in large by taking our compatibility promises really serious. It takes a lot of time to build trust but only little time to erode it.

#### Examples

- Actions: As part of the actions work we reworded the log message indicating what has been planned / applied to include actions. In the classic output mode (not the structured logging one) HCPT relied on the specific shape of this message for a RegEx, leading to errors in the integration.

  ```diff
   Terraform will perform the following actions:

     # aws_instance.web will be created
     + resource "aws_instance" "web" {

  -Plan: 1 to add, 0 to change, 0 to destroy.
  +Plan: 1 to add, 0 to change, 0 to destroy, 2 actions to invoke.
  ```

- SDKv2: We are trying for a lot of years to get providers to move from SDKv2 to the framework. There is a related syntax change that a large customer estimated to cause potential 7 figure costs in engineering effort with the size of their configuration. A small change for us can have a huge cost in adoption for larger organizations.

#### Guidelines

- Always check if your change might impact an existing behavior
- Make sure your feature is scoped so it does not open up unnecessary contracts
  - In stacks instead of using addresses to corelate components with one another we obfuscated the ids so that users would not be able to parse the ids and build their own workflows on top of a semantic we did not want to take on as a contract.

### 2. Whatever your language allows a user to expressed will be expressed in a mission critical workflow of at least one very important customer

When building a feature we often have a preconceived notion of how the feature is going to be used. Once we release it we are often surprised by how it actually is used and what problems we haven't thought about can actually be solved with this feature. We might even think it is a bad idea to use this feature in this way, but as soon as we release the software everything one can express is fair game and has to work the same in upcoming releases.

New features are often designed by their capabilities, but defining what they can't do / what is illegal and how to catch these invalid configurations as early as possible is just as important. Everything we allow needs to be valid and make sense in the context of the feature. Leaving a feature to open increases the complexity since all enhancements to the feature will need to interface with the open design as well.

This often means when the design is too open to begin with once we add more capabilities in future iterations we have a fractured user experience where in some cases things are allowed and in others they aren't. If one had designed the feature with these enhancements from the get go the trade-off between the added value and the compromises in what is allowed in which configuration might have resulted in a more closed feature. But since we started open there is no way to go back.

#### Examples

- Actions: We allowed ephemeral values in action configurations. This was no problem since we were running the lifecycle actions only on create and update. Now that we added destroy actions we found that during destroy we can't support ephemeral values which means we need to add an extra validation to ensure destroy–triggered actions can't contain ephemeral values in their config.
- Policy: We allow users to use `getresources` as a function call to get resources from state. The function call is just part of HCL expressions and operates on values. Which means the user has no restrictions on what to call it with and what to do with the return value. This form is very flexible, which is great for expressiveness. But it is too flexible to do any optimizations with or change the dynamic on how things are executed.

#### Guidelines

- If there is a syntax option that narrows what the user can do down to the things we want them to do with the feature we should choose the narrower design

  ```hcl
  # Open: arbitrary boolean expression over a free-form global.
  # Expressive, but the engine can't statically know which ops apply,
  # and typos like "cretae" fail silently instead of erroring.
  policy "require_tags" {
    trigger = operation == "create" || operation == "update"
  }

  # Closed: fixed enum list, validated at parse time. Engine can skip
  # evaluating the policy entirely for non-matching operations.
  policy "require_tags" {
    operations = ["create", "update"]
  }
  ```

- If we can't find a way to make the syntax stricter we should, if possible, add validations to hedge against misuse
- If we can not find a different syntax and we can't make it stricter we should take a step back and see if we can redefine the problem in a way that allows us to do these changes. Since nothing we release can be changed (at least not in a timeframe that one can consider practical) we should not be shipping things that will be hard to support to keep our future velocity up

### 3. Not all code to be consumed was written for a version of the language that is currently running

This means your language might encounter features from newer versions as well as older versions. Within minor versions (and we don't have a track record of ever really updating major versions, so read as always) there must be backward and forward compatibility. This means your code needs to be defensive (e.g. we have three valid options, if the first two don't match it must be the third one is wrong, we could add a fourth option a year down the road and your code would misbehave).

Ideally we want to be able to tell users that they need to update their language version. For this we need a mechanism to express compatible language versions as part of a language construct, e.g. Terraform uses `required_version` for this.

#### Guidelines

- There needs to be a system for versioning so that we can alert the user to update their system if the language is newer than the software we are running. There are generally two kinds of versioning systems
  - Language Versions / [Editions](https://doc.rust-lang.org/edition-guide/editions/): Some languages like Rust use language editions that allow the language designers to add backwards-incompatible changes to later versions of the language
  - Tool Versions: Terraform is an example of a tool-versioned language, the language is defined per version of Terraform
- Terraform is released in versions and the user is in control of the version they use, this means we can show helpful error messages asking them to update
- If the language chooses a different model where the user is not in control of the versions they use we still need to document the minimum version required to give helpful feedback, although the feedback won't be actionable if they can't update the version used
  - We can generally say that it is advisable to allow users to pick their respective version to use. Each bug being immediately shipped to the entire customer base without a way for them to roll back puts a lot of pressure on providing a quick fix.

### 4. Great Error Messaging brings your language to the next level

The first impression a user gets of your language often comes from the errors they run into when learning the language. This means we need to give users the most information about what is going wrong that we can, ideally enough that they know the next steps to take to fix the situation just by reading the error.

One of the languages that does (compiler) errors the best is Rust. They have to communicate complex circumstances to their users and their error messages provide a ton of additional context that make understanding the issue at hand easier, see [this blog post](https://kobzol.github.io/rust/rustc/2025/05/16/evolution-of-rustc-errors.html).

Terraform has the [tfdiags](https://github.com/hashicorp/terraform/tree/main/internal/tfdiags) package for this, it also provides a way to add extra information to errors that make the issue at hand clearer.

```text
# Bad: no location, no context, no next step
Error: invalid value

# Good: exact source range, the value that was wrong, and why
Error: Invalid value for "count"

  on main.tf line 12, in resource "aws_instance" "web":
  12:   count = "three"

The given value is not suitable for "count": a whole number is
required, but "three" is a string. Did you mean count = 3?
```

#### Guidelines

- We need to make users aware of faulty configuration as early in the process as possible. This means not just throwing the first error but evaluating further until all errors findable at this time are found. It also means we need to make the extra effort when possible to find issues at the earliest possible time. The language server needs to catch as many errors as possible as well, we should consider it as the fastest way to give feedback to the user
- Every error needs to be as actionable as possible, users should directly understand what to do to fix the issue. We should also consider why the user might try to do the thing that caused the error, maybe there is a feature we can inform them about that might be of help
- Errors should have the meta-information (source ranges, adjacent configuration, just context in general) attached to them that is necessary for a user to understand the problem at hand

### 5. Make the authorship experience delightful

Our users spend way more time with the language and features we build than we do. For some of our users interacting with this language is going to be a primary part of their job, for others it is a chore they need to do every once in a while when they almost forgot how it worked.

Both user groups are very allergic to friction, every extra step, iteration, precaution they have to take, every minute detail they have to keep in mind on how to work with this product leads to frustration. We need to deliberately set them up for success.

If we have multiple ways to make something expressible we should think about what requires the least knowledge of our product to work with the feature. We should think about how can we make impossible states unable to describe, how can we warn the user of as many issues as possible as early as possible.

This is a balancing act, we want users to be able to express their needs through our products, but ideally we don't want to allow them to express things the product can not deliver on.

Thinking about potential footguns and designs that would not expose them can be helpful. To come up with footguns we should not only consider direct functionality but also think about performance (allowing the user to build something that takes way longer when there is a simpler, faster alternative) and about the integration with other features.

#### Examples

- For policy during the alpha we had no way of validating policies against provider schemas. This meant that typos and other issues would only be noticed when the policy was executed against a resource of the given type. In the real world this can be spanning multiple teams and weeks / months of rollout time. The person hitting the error is usually not the person who wrote the policy, so the feedback does not even reach the author directly, and it only ever surfaces for the subset of resource types and attributes that a given configuration happens to exercise. A policy can look healthy for months and still be silently broken for the one resource type it was written for. We made this a validation concern since, so the same class of mistake is caught at authoring time, with a source range, in the editor and in CI.

#### Guidelines

- We should strive to make invalid desired states / config impossible to express. For every error you write, stop and think if there is a way to change the structure of the configuration to make reaching this point impossible.
- We need to maintain strong boundaries between third party systems, when we integrate with e.g. a provider we need to ensure the expectations that we have with responses from the providers are encoded and verified so that we can point out potential bugs in the third party system

### 6. Your feature needs cooperate with every other feature

If your feature can be used in conjunction with another feature (or a set of multiple features combined) it will be. We need to think through the impact our feature has on all other features in every combination. For some the answer is going to be none, but others might be greatly impacted.

We need to implement our feature so it makes sense in the context of other features. Where this is not possible we can and should forbid the use. Where it is possible but extra work it can be tempting to also forbid the behavior and ship it at a later point. Sometimes that is valid, sometimes later never comes and we need to consider the ongoing bad user experience. And sometimes the feature not working in conjunction with another feature will block certain users / use-cases from adopting your feature.

#### Examples

- For Terraform Actions we decide to not do an integration with Terraform Test to cut scope and deliver on time. This now means that people trying to use Actions in modules that are tested through Terraform Test run into issues and can't use Terraform Actions properly. The gap is not evenly distributed either: it hits exactly the users we want to reach most, module authors who already invest in a test suite. For them adopting Actions means giving up test coverage for the part of the module that has side effects outside of Terraform, which is the part they would most want covered. What looked like a bounded scope cut at delivery time turned into an open ended adoption blocker, and closing it after the fact is more expensive than building it alongside the feature would have been, because the surrounding surface (test runtime, assertions, mocking) has since moved on and now has to be retrofitted rather than designed for.
- For Dynamic Module Sources we build a graph to evaluate constant variables during `terraform init`, for this we need the backend. Normally the backends ship with Terraform, but pluggable state storage now allows users bring their own backend through a provider. These features were developed in conjunction, during the integration period we noticed this circular dependency and were able to resolve it

#### Guidelines

- When you are doing a prototype for your feature look out for other features it might interact with and note them down. You will want to discuss these topics in your RFC.
  - Please see [the prototyping guide](./prototyping.md) for a guide on how to build solid prototypes
- When other teams are developing features that might overlap with your feature at the same time it makes sense to talk to each other on an ongoing basis to make sure the connection points of your features are well defined
- Please take a look at the bottom of this document for a likely incomplete list of features to check against

### 7. A balancing act between openness and closedness

There are two contradicting perspectives on the language feature you are designing that need balancing: Openness (what is allowed to be expressed) and Closeness (what can not be expressed).

<!-- figure: openness vs closeness diagram from the original document -->

To make the terms clear, let's take the task of filtering resources a policy applies to.

A very open design would be putting a `filter = <expr>` attribute in the language construct for the policy and allowing access to everything Terraform knows about through globals.

A very closed one would be taking one of the use cases for this feature and having a `filter_by_resource_type = [<string>]` attribute that takes a set of type names.

```hcl
# Open: `filter` is a full HCL expression with globals in scope.
policy "no_public_buckets" {
  filter = resource.type == "aws_s3_bucket" && resource.acl != "private"
  enforce { condition = !resource.public_access_block }
}

# Closed: `filter_by_resource_type` only takes known type strings --
# validatable against provider schemas before the policy ever runs.
# `filter_by_attribute` also takes the schema into account
policy "no_public_buckets" {
  # More attributes to know about, more verbose, but easier to validate and reason about.
  filter_by_resource_type = ["aws_s3_bucket"]
  filter_by_attribute     = { acl : "private" }
  enforce { condition = !resource.public_access_block }
}
```

The closeness vs openness of the language feature is a spectrum, there are a lot of different solutions in between that are viable.

<!-- figure: openness/closeness spectrum from the original document -->

Each side of the spectrum has different trade-offs. Open designs are very expressive, allowing the user maximum freedom. That freedom comes at a cost: more contracts to be honored on the language side, fewer options for catching footguns and doing optimizations at the language level. The more a design is closed the more the language is in control of the constraints and can manage complexity for the user that with open solutions the user needs to account for.

Take this filtering example.

In the open solution, we would need to do extra work in parsing the expression, finding all relevant access, dealing with all kinds of comparisons (e.g. includes, equal sign, for expressions) to ensure if people are filtering by resource type that this resource type exists. In the closed one we get this feature very cheaply.

If the filtering happens on the side where policy evaluation happens, the closed solution can be communicated to Terraform ahead of time and the policy plugin never sees resources of unwanted types. In the open solution we are forced to evaluate the expression on the plugin side after we already sent over the data for the resource in question.

There are a lot more things one can express with open solutions, but often there is an attached cost in the guardrails we can put up for our users. Some are no longer feasible, some are exponentially more expensive.

At the same time we want to satisfy customer demands without bloating the language through repetitive features for each special case. This is the balance we need to strike.

An additional perspective to the closed designs is: How would we extend this with features users might want in the future? In the case of this example for the closed design we would need to add a new attribute if we want to filter by a certain attribute being set, with more use-cases this can come repetitive.

#### Guidelines

- A design should be as closed as possible while being as open as necessary. Closed designs shift the complexity to us on the language side, leading to a better user experience generally speaking.
- Open designs that allow the user a lot of flexibility often proof to be a liability later on

### 8. Never underestimate the complexity that can be invented by DevOps

When we build new features we think about them in isolation and at a small scale. This is natural, we first need to understand what we are building before being able to mentally scale it up.

But before we commit to the feature and ship it we need to think about the last times we abused a simple language feature / CLI tool in a weird way to achieve something the tool clearly was not designed for to begin with. Be honest, we all do it at times. Now with that mindset, think about how someone can use your feature in unforeseen ways to achieve something it was not designed for.

#### Examples

- TF Policy: When trying to design a relationship mapping language feature similar to a SQL Join we found out that real-world policies also take multiple relationships into account in a union way, they derive the s3 bucket names used for relationships through the buckets URL, they care about each resource of type a being matched to exactly one variant of type b, etc.

  ```hcl
  # Naive model: one clean reference per relationship.
  relationship "vpc_to_subnet" {
    match = each.vpc.id == each.subnet.vpc_id
  }

  # Real world: relationships are unions of several shapes, and matching
  # is derived by parsing an identifier out of a nested string field
  # instead of following a direct reference -- exactly the kind of
  # complexity DevOps invents that a simple 1:1 join model can't express.
  relationship "bucket_to_replication_target" {
    match = each.bucket.replication_configuration[0].rules[0].destination.bucket
            == "arn:aws:s3:::${each.target.id}"

    # each bucket must resolve to exactly one target, even though targets
    # can be matched via replication rules OR via lifecycle policies
    require_unique_match = true
  }
  ```

#### Guidelines

1. Think about your feature being used extensively by power users, in giant configurations, to achieve automation goals it was not designed for.
2. Look at complex real-world use cases and try modeling them using your feature
3. Think about different team architectures (1 platform team empowering 100 application teams, small startup, indie developer, consultant with different customers) using the feature
4. Think about third party tools your feature might get combined with

## Building a language interfacing with Terraform

When building a language that integrates or interfaces with Terraform directly (like Stacks, Policy, Query), there are a set of features in Terraform that the respective language constructs need to think about. This means either the language needs to explicitly allow the integration with the given phenomenon or disallow it. It needs to deal with all language constructs in all possibly expressible expressions in a reliable way; there can not be undefined behavior.

The following list is meant as an overview and is non-exhaustive.

- Runtimes: How does this feature work in the context of
  - Stacks
  - Policy
  - Query/Search
  - Module
  - Test
- Environments
  - Locally through the CLI
    - One-off by the user
    - Through a CI pipeline
  - On TFC
  - On TFE
  - On TFC/TFE with agents
  - In an airgapped environment
- Values
  - Simple / Complex values
  - Unknown values
  - Ephemeral values
  - Sensitive Values
- Output Artifacts
  - Statefile
  - Planfile
- Resource lifecycle
  - CRUD
  - Replacements (Create then destroy, destroy then create)
- Graph
  - Expansion (count / for_each)
  - Order of operations reversing during destroy
    - During destroy operations the entire graph is reversed
    - During normal plan / apply with deleted code / removed instances the to be removed part of the graph is reversed
- Providers
  - New concepts not being there all of the time, e.g. old providers don't support newer features
  - Deprecated SDKv2 providers needing to support the feature
- Languages
  - HCL
  - JSON

## Further Reading List

- [Marin Atkins Blog Post: Evolving the Terraform Language](https://log.martinatkins.me/2019/03/01/terraform-language/)
- [Martin Atkins Blog Post: Terraform is a Data Flow Language](https://log.martinatkins.me/2019/08/19/terraform-data-flow-language/)
