# Prototyping

This is a living document that contains guidelines and best practices for prototyping new features in Terraform.
If we want to change our processes for prototyping, we should update this document.

Prototyping is a learnable skill, separate from normal software development. The more practice we get with prototyping, the more natural it will feel and the more effective we'll be. This document aims to give us a shared foundation to build that skill together.

## When to Prototype

When a new feature or a significant addition to an existing feature is planned we should prototype it first.

The motivation for prototyping is reducing risk — both implementation risk (can we build this?) and design risk (should we build this, and in what form?). We should prototype as early as possible, and likely earlier than feels comfortable. Prototyping once we have already committed to solving a specific problem is too late; what would we do if the prototype reveals that the problem cannot reasonably be solved, or that a more important problem should be tackled? Prototypes are most effective when used to explore the solution space for a project before commitments are made.

## Before Starting

### Identifying Risks

Before starting a prototype it can be helpful to take a look at the existing implementation or adjacent implementations to build a mental model of the workflow. This can help us to identify potential issues or challenges early on and gives us more context to probe into potential problem areas in the prototype. It can also be helpful since one might understand the overlap area and potential issues with existing features the project will need to integrate / cooperate with.

When examining the existing system, focus particularly on the interfaces between components — APIs, file formats, and protocols exposed or consumed by services and tools. Project risk often concentrates at these interface points. Determine where the existing system is missing necessary functionality and map out the gaps that the prototype will need to address.

Aside from existing features it is also important to think about possible overlap with upcoming features that are in implementation or prototyping.

We should also make sure that if the project might affect other teams (providers, cloud, etc.) so that we communicate with them early and check in what they might need from the prototype to be able to do their work. Aside from ensuring our prototype is extra helpful this also helps put potential work on their radar.

### Formulate Hypotheses and Success Criteria

Overall it makes sense to think about specific success criteria ahead of time to ensure we stay on track and cover everything we intended to cover. Specifically when the project and/or prototype scope is limited it makes sense to think about how one can ensure the next steps that are omitted from the project/prototype are still feasible with the current approach.

Beyond general success criteria, we should formulate specific, testable hypotheses about what we're trying to learn. This applies to both technical feasibility ("can Terraform produce an execution plan with unknown inputs?") and user needs ("users would rather be able to do X than Y").

When formulating hypotheses about user needs, aim for hypotheses that can be falsified, not just confirmed. For example:

- Weak: "The lack of X is making users turn to a competitor product" — this is hard to test because it's much easier to verify than falsify.
- Strong: "Users would rather be able to do X than Y" or "When Z happens, users know why" — these can be clearly confirmed or rejected by putting alternatives in front of users.

It's also worth considering the null hypothesis: that users prefer to use existing tools or workarounds over a given new feature. Sometimes user interviews will confirm the null hypothesis — then the best code to write is no code.

## Types of Prototypes

Depending on the phase and the ambiguity of the project, we can make use of different types of prototypes.
Not all projects require all types of prototypes, and some projects may require multiple iterations of the same type of prototype.
Keep in mind that not only the finished implementation can be a result of a prototype, but also a deeper understanding of constraints, user needs, and even that we can't / shouldn't implement this feature.

An important distinction is between prototypes that test **feasibility** (can we build this?) and prototypes that test **design** (should we build this, and in what form?). In the past, projects have sometimes been ambiguous about which goal a prototype serves. Being explicit about this up front helps us choose the right type, the right fidelity, and the right audience.

### Exploratory Prototypes

When the project is in the early stages and the specific API design or user experience is not yet clear, we can create exploratory prototypes. The goal of these prototypes is to have something tangible to discuss and iterate on; ideally something we can show to users for feedback.
We will for sure throw these prototypes away after we have learned what we need to learn.

These prototypes can be quick and dirty, focusing on the core functionality or user experience. It's okay to put them in a separate (private) repository and even to mock the implementation completely.

The goal of these prototypes is to learn about the problem space, gather feedback, and check the general feasibility.

A specific sub-category of exploratory prototypes are **design prototypes**, which focus specifically on testing hypotheses about user needs rather than technical feasibility. These can be extremely minimal — even an example CLI command string with a new flag, example syntax for a new language construct, or example JSON output from a new API endpoint. The goal is to put something concrete in front of users to help them imagine the experience of using the software, and to test whether our assumptions about what they want are correct. If you have several ideas for alternative solutions, prototyping them side-by-side and showing them to users often reveals preferences the team did not anticipate.


### Tracer Bullet / Scaffold Prototypes

This term comes from [the pragmatic programmer](https://pragprog.com/titles/tpp20/the-pragmatic-programmer-20th-anniversary-edition/) and describes a prototype that implements the entire feature end-to-end, but in a very simplified way. 

The goal is to test the overall architecture and design of the feature, and to identify any major issues or challenges. This type of prototype is more polished than an exploratory prototype and might be used as a basis for the final implementation. 

This type of prototype is appropriate when the goal is clearly defined and the solution space does not cover any major changes to the internals (e.g. to the Graph or to evaluation). It can also be a helpful onboarding tool for others coming into the project since it can provide a good overview of the feature and its implementation.

### Full Prototypes

This kind of prototype is a somewhat complete implementation of the feature. It includes proper tests and aims to be production-ready in the core workflows. What differentiates this prototype from the final implementation is

- that we don't need to worry about topics outside the core workflows (e.g. adjacent commands / features where the implementation is clear). We need to worry about edge cases and cooperation with other features since sometimes they can reveal (language) design issues.
- that we can still throw it away if we learn that the design is not good enough. We need to leave this door open since we might learn something new during the implementation that makes us want to change the design. We should not be afraid to throw away a full prototype if we learn that the design is not good enough.
- that we can be more lenient with code quality and documentation than in the final implementation since we might throw it away. We should focus on what makes us keep the velocity of the prototype and implementation high. (See the note on documentation in "During Prototyping" — rationale and intention should still be documented even when code quality is relaxed.)

This prototype is very close to the final implementation. Once done we can ideally just polish it and merge it into the main branch.

We should aim for this prototype if the feature is high in complexity or touches complex parts of the codebase (e.g. the graph or evaluation). If the feature involves language design it's a good hint that we need to use this kind of prototype; it's a "high cost, high security that we are doing the right thing" kind of situation.


### Summary

| Type of Prototype | Goal | When to Use |
| --- | --- | --- |
| Exploratory Prototype | Learn about the problem space, gather feedback, check general feasibility | Early stages of the project when the specific API design or user experience is not yet clear |
| Tracer Bullet Prototype | Test the overall architecture and design of the feature, identify major issues or challenges | When the goal is clearly defined and the solution space does not cover any major changes to the internals |
| Full Prototype | Implement a somewhat complete version of the feature, including proper tests, and aim for production-readiness in core workflows | When the feature is high in complexity or touches complex parts of the codebase, especially when it involves language design |


## During Prototyping

All of these are ideas / suggestions, it's not one size fits all. Use the ones that are helpful to you and the problem you are trying to solve.

### Sharing is Caring

Make sure to showcase the prototype scope / design decisions / technical challenges to members of the team on a regular basis. Being heads down on a prototype can lead to tunnel vision and thus to surprising feedback once the project goes into implementation phase. One of the reasons to do prototyping is to incorporate feedback early on, the way to get feedback is to talk to different people on a regular basis. Make sure to take enough time so the other person can build a deep enough understanding to give proper feedback.

### Start in the Middle

A counter-intuitive but effective prototyping strategy is to start in the middle of the solution. The edges of the software are often boring and well-understood, and as a result give little return on investment of time when our goal is to build new understanding.

In practice this means identifying the epicenter of the idea: the place from which all other decisions flow. Describe the idea in bullet points or prose and identify the piece which seems to have the most dependencies, or is the least understood, or has the highest risk. Focus on this part of the solution first, scaffolding or fudging anything necessary to do so.

It doesn't really matter if you choose the correct place to start. As you start prototyping, it will become increasingly clear what the middle of the problem is, and you can iterate towards it.

### Prioritize by Unknowns

Once you've identified the gaps in the system which need to be addressed, prioritize them by how unknown they are. This allows you to sequence work to rapidly gain understanding and reduce wasted time. This is different from production task breakdown where you might prioritize by dependency order or business value — here the goal is to learn the most important things first.

### Consider Manual / No-Code Prototyping

The best software is no software: it takes no time to write, there are no bugs, and it's very easy to make changes. If you can prototype your ideas without writing code, try doing so.

For example, you can sketch out an example configuration in an arbitrary, unimplemented format, then take on the role of the system yourself and manually work through the steps, documenting as you go. This no-code approach can take only minutes and successfully demonstrate whether the basic concepts are likely to work when implemented in software.

When manual prototyping, be rigorous: don't do things that you can't replicate in software. Instead, be the computer — imagine you're executing unwritten software and dutifully follow the flowchart in your head. If you notice that you have to do something creative or magical, that might be a sign that your solution is more complex than you think. Document the steps you take in enough detail that someone else could reproduce your work.

### Leave a Breadcrumb Trail

The goal of prototyping is understanding, not useful software. Keeping notes as you work is more important during prototyping than in normal development.

- Be liberal with code comments, recording the **intention** of your code. This will be useful to anyone who refers to the prototype when designing production software.
- Use TODO and FIXME comments when skipping error handling, edge cases, or making other trade-offs. Notes about your rationale and guesses about future work will be valuable to your future counterpart.
- Keep a logbook or work journal during prototyping. These notes will be useful when you need to backtrack or hand over work, and can be referred back to once the prototyping phase is complete.

This is compatible with being lenient on code quality — the point is to document *why* you made decisions, not to polish the code itself.

### Breadth builds Confidence

Starting a prototype several times with different solution strategies can be helpful to ensure the problem area is discovered in depth. When testing design hypotheses, also consider building alternatives side-by-side and presenting them to users; The comparison often reveals preferences the team did not anticipate.
It can be tempting to continue with the original plan for the prototype even if road-blocks are uncovered, but it is often wise to take a step back and rethink the plan. This will deepen the understanding on the problem domain and inform a better design.

## After Prototyping

Especially for prototypes of a non-trivial size we should start the implementation from scratch. During prototyping the code evolves and we might miss better solutions to sub-problems or include unnecessary additions. Starting from scratch is cleaner.
If your prototype was in `terraform-private` it makes sense to leave a comment that this work won't be finished as not to confuse people stumbling upon it.


## Co-Prototyping Strategies

More often than not we want multiple people to work together on a project and therefore collaborate on the prototype(s). Collaborating in the early phases of a project helps share context and build agency and ownership across the team. It also helps us learn from each other and come up with better solutions.

**Only when prototyping teams have a shared context can we build effectively together.**

Since all projects and prototypes are very different, we should be flexible in how we collaborate on prototypes. Here are some strategies we have seen work well in the past:

### Pair Prototyping

Especially in the early phases of a project / prototype and if all collaborators have significant (>3 hours) timezone overlap, pair programming is a great way to build a shared project context and understanding. Having a clearly scoped goal for a pairing session is important so that everyone can engage equally. One should also aim to share the driving and navigating roles equally to ensure that everyone is engaged and has a chance to contribute. We found that especially the navigating role is important during prototyping since it's easy to get lost in the details or future issues / plans instead of focusing on the current problem. Switching between these roles sharpens our focus and helps us stay on track. Pairing can be exhausting, so we should be mindful of that and take breaks when needed.

Because communication during the prototyping process is so important, pairing is also a great opportunity to talk through ideas as you're working. Having a pairing partner to keep you on the narrow path helps a lot.

### Async Single Branch Prototyping

This is a variation of pair programming that works well when there is less timezone overlap between collaborators. The idea is to have an ideally synchronous handover between collaborators at the end of their workday (in most cases one side needs to do an async handover). 
Together all collaborators work on the same branch and try to move the prototype forward. To make this efficient we need to ensure to take the last 30-60 minutes of the day to sync or write a good handover message. 
In this working mode we should make sure to build a shared roadmap since this can facilitate the handover and ensure that everyone is working towards the same goal.

### Fan-Out Prototyping

This strategy works well when there is already some ground work done. We can divide tasks up and in parallel explore various sub-problems.
To do this we already need to have a lot of shared context and we should make sure to sync regularly to share learnings and ensure that we are still on the same page. To do this with less shared context, the issues at hand need to be self-reliant, well defined and adjacent to the core problem. 
Thinking about strong and clear contracts between different components and dividing the components between one another can also be a helpful strategy.


## How to Deal with Team Members Joining a Prototype Late

The main goal we should have when someone joins a prototype or project in general when it's already in progress is to share as much context as possible to give the other person the best chance to contribute effectively and feel ownership over the project. It took you effort and time to accumulate the context you have, so ensure you make the same room for the new person. The best way of building context is highly personal, here are some options:

- Let the new person pair and drive as much as possible to help them build their own context and understanding of the project.
- If you have a PR (e.g. for a partial feature) ready for review already, try giving a walkthrough of the PR to the new person to share the context and the design decisions you made. This can also be a good opportunity to reflect on your design decisions and see if they still make sense.
- Writing down the data flow and linking to the relevant code parts can help the other person more understand the implementation and the design decisions you made. This can be especially helpful if the implementation is complex or if there are a lot of moving parts.


By having to transfer the project / prototype context to a new person one has to re-explain the problem, which often helps solidifying the thinking and uncover blind spots.


## Terraform-specific Prototyping Hints

### Provider Enhancements

When working on a feature that includes changes to the provider protocol, we more often than not need to work ahead of the teams working on the SDK/Framework and the providers. During prototyping it can be useful to use the builtin provider to test out new features against the go provider interface. In a later stage if we want to test the feature against an external binary, building a simple provider with [terraform-plugin-go](https://github.com/hashicorp/terraform-plugin-go) can be a good option since we don't need to forward our new feature through layers of abstraction. If this is needed depends on the timing of the other teams and the uncertainty involved with the protocol changes.


### TFC Testing

In general it makes sense to at least smoke test most prototypes against TFC to understand if any extra work will be needed on `tfc-agent` or `atlas`.
You don't need to wait for an alpha release to do that. You can run [`tfc-agent` against a locally built Terraform binary](https://github.com/hashicorp/tfc-agent/?tab=readme-ov-file#execute-with-local-terraform-binary) and [configure the agent to run against the production TFC](https://developer.hashicorp.com/terraform/tutorials/cloud/cloud-agents). Make sure to change your workspace setting to use the agent pool with your agent.

### Terraform Experiments / CLI flags and other feature toggles

If you want to bring a prototype in front of customers for a prolonged period of time versus in field testing / customer interviews you will need to use a feature toggle. We have different mechanisms:

For experimental language constructs we use [experiments](https://github.com/hashicorp/terraform/blob/73c225dff4f1dc2a2464ffffe6f590e1d5692c05/internal/experiments/experiment.go) which the user needs to enable through `terraform { experiments = [...] }` in their config. They have also been used for language level (additive) behavior changes.

For functionality we use CLI flags enabling the new behavior. We only want to enable this functionality in alpha releases usually so we always need to check the `AllowExperimentalFeatures` on `Meta` during flag evaluation. Below is an example for deferred actions.


```go
if c.AllowExperimentalFeatures {
	opReq.DeferralAllowed = args.DeferralAllowed
} else if args.DeferralAllowed {
	// Belated flag parse error, since we don't know about experiments
	// support at actual parse time.
	diags = diags.Append(tfdiags.Sourceless(
		tfdiags.Error,
		"Failed to parse command-line flags",
		"The -allow-deferral flag is only valid in experimental builds of Terraform.",
	))
	return nil, diags
}
```
