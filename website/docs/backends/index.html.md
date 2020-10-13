---
layout: "docs"
page_title: "Backends"
sidebar_current: "docs-backends-index"
description: |-
  A "backend" in Terraform determines how state is loaded and how an operation such as `apply` is executed. This abstraction enables non-local file state storage, remote execution, etc.
---

# Backends

A "backend" in Terraform determines how state is loaded and how an operation
such as `apply` is executed. This abstraction enables non-local file state
storage, remote execution, etc.

By default, Terraform uses the "local" backend, which is the normal behavior
of Terraform you're used to. This is the backend that was being invoked
throughout the [introduction](/intro/index.html).

Here are some of the benefits of backends:

  * **Working in a team**: Backends can store their state remotely and
    protect that state with locks to prevent corruption. Some backends
    such as Terraform Cloud even automatically store a history of
    all state revisions.

  * **Keeping sensitive information off disk**: State is retrieved from
    backends on demand and only stored in memory. If you're using a backend
    such as Amazon S3, the only location the state ever is persisted is in
    S3.

  * **Remote operations**: For larger infrastructures or certain changes,
    `terraform apply` can take a long, long time. Some backends support
    remote operations which enable the operation to execute remotely. You can
    then turn off your computer and your operation will still complete. Paired
    with remote state storage and locking above, this also helps in team
    environments.

**Backends are completely optional**. You can successfully use Terraform without
ever having to learn or use backends. However, they do solve pain points that
afflict teams at a certain scale. If you're an individual, you can likely
get away with never using backends.

Even if you only intend to use the "local" backend, it may be useful to
learn about backends since you can also change the behavior of the local
backend.
