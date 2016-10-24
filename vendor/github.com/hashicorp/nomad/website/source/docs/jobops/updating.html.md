---
layout: "docs"
page_title: "Operating a Job: Updating Jobs"
sidebar_current: "docs-jobops-updating"
description: |-
  Learn how to do safely update Nomad Jobs.
---

# Updating a Job

When operating a service, updating the version of the job will be a common task.
Under a cluster scheduler the same best practices apply for reliably deploying
new versions including: rolling updates, blue-green deploys and canaries which
are special cased blue-green deploys. This section will explore how to do each
of these safely with Nomad.

## Rolling Updates

In order to update a service without introducing down-time, Nomad has build in
support for rolling updates. When a job specifies a rolling update, with the
below syntax, Nomad will only update `max-parallel` number of task groups at a
time and will wait `stagger` duration before updating the next set.

```
job "rolling" {
    ...
    update {
        stagger = "30s"
        max_parallel = 1
    }
    ...
}
```

We can use the [`nomad plan` command](/docs/commands/plan.html) while updating
jobs to ensure the scheduler will do as we expect. In this example, we have 3
web server instances that we want to update their version. After the job file
was modified we can run `plan`:

```
$ nomad plan my-web.nomad
+/- Job: "my-web"
+/- Task Group: "web" (3 create/destroy update)
  +/- Task: "web" (forces create/destroy update)
    +/- Config {
      +/- image:             "nginx:1.10" => "nginx:1.11"
          port_map[0][http]: "80"
    }

Scheduler dry-run:
- All tasks successfully allocated.
- Rolling update, next evaluation will be in 10s.

Job Modify Index: 7
To submit the job with version verification run:

nomad run -check-index 7 my-web.nomad

When running the job with the check-index flag, the job will only be run if the
server side version matches the the job modify index returned. If the index has
changed, another user has modified the job and the plan's results are
potentially invalid.
```

Here we can see that Nomad will destroy the 3 existing tasks and create 3
replacements but it will occur with a rolling update with a stagger of `10s`.
For more details on the update block, see
the [Jobspec documentation](/docs/jobspec/index.html#update).

## Blue-green and Canaries

Blue-green deploys have several names, Red/Black, A/B, Blue/Green, but the
concept is the same. The idea is to have two sets of applications with only one
of them being live at a given time, except while transitioning from one set to
another.  What the term "live" means is that the live set of applications are
the set receiving traffic.

So imagine we have an API server that has 10 instances deployed to production
at version 1 and we want to upgrade to version 2. Hopefully the new version has
been tested in a QA environment and is now ready to start accepting production
traffic.

In this case we would consider version 1 to be the live set and we want to
transition to version 2. We can model this workflow with the below job:

```
job "my-api" {
    ...

    group "api-green" {
        count = 10

        task "api-server" {
            driver = "docker"
            
            config {
                image = "api-server:v1"
            }
        }
    }

    group "api-blue" {
        count = 0

        task "api-server" {
            driver = "docker"
            
            config {
                image = "api-server:v2"
            }
        }
    }
}
```

Here we can see the live group is "api-green" since it has a non-zero count. To
transition to v2, we up the count of "api-blue" and down the count of
"api-green". We can now see how the canary process is a special case of
blue-green. If we set "api-blue" to `count = 1` and "api-green" to `count = 9`,
there will still be the original 10 instances but we will be testing only one
instance of the new version, essentially canarying it.

If at any time we notice that the new version is behaving incorrectly and we
want to roll back, all that we have to do is drop the count of the new group to
0 and restore the original version back to 10. This fine control lets job
operators be confident that deployments will not cause down time. If the deploy
is successful and we fully transition from v1 to v2 the job file will look like
this:

```
job "my-api" {
    ...

    group "api-green" {
        count = 0

        task "api-server" {
            driver = "docker"
            
            config {
                image = "api-server:v1"
            }
        }
    }

    group "api-blue" {
        count = 10

        task "api-server" {
            driver = "docker"
            
            config {
                image = "api-server:v2"
            }
        }
    }
}
```

Now "api-blue" is the live group and when we are ready to update the api to v3,
we would modify "api-green" and repeat this process. The rate at which the count
of groups are incremented and decremented is totally up to the user. It is
usually good practice to start by transition one at a time until a certain
confidence threshold is met based on application specific logs and metrics.

## Handling Drain Signals

On operating systems that support signals, Nomad will signal the application
before killing it. This gives the application time to gracefully drain
connections and conduct any other cleanup that is necessary. Certain
applications take longer to drain than others and as such Nomad lets the job
file specify how long to wait in-between signaling the application to exit and
forcefully killing it. This is configurable via the `kill_timeout`. More details
can be seen in the [Jobspec documentation](/docs/jobspec/index.html#kill_timeout).
