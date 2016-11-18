---
layout: "intro"
page_title: "Install Nomad"
sidebar_current: "getting-started-install"
description: |-
  The first step to using Nomad is to get it installed.
---

# Install Nomad

The task drivers that are available to Nomad vary by operating system,
for example Docker is only available on Linux machines. To simplify the
getting started experience, we will be working in a Vagrant environment.
Create a new directory, and download [this `Vagrantfile`](https://raw.githubusercontent.com/hashicorp/nomad/master/demo/vagrant/Vagrantfile).

## Vagrant Setup

Note: To use the Vagrant Setup first install Vagrant following these instructions: https://www.vagrantup.com/docs/installation/

Once you have created a new directory and downloaded the `Vagrantfile`
you must create the virtual machine:

    $ vagrant up

This will take a few minutes as the base Ubuntu box must be downloaded
and provisioned with both Docker and Nomad. Once this completes, you should
see output similar to:

    Bringing machine 'default' up with 'vmware_fusion' provider...
    ==> default: Checking if box 'puphpet/ubuntu1404-x64' is up to date...
    ==> default: Machine is already running.

At this point the Vagrant box is running and ready to go.

## Verifying the Installation

After starting the Vagrant box, verify the installation worked by connecting
to the box using SSH and checking that `nomad` is available. By executing
`nomad`, you should see help output similar to the following:

```
$ vagrant ssh
...

vagrant@nomad:~$ nomad

usage: nomad [--version] [--help] <command> [<args>]

Available commands are:
    agent                 Runs a Nomad agent
    agent-info            Display status information about the local agent
    alloc-status          Display allocation status information and metadata
    client-config         View or modify client configuration details
    eval-status           Display evaluation status and placement failure reasons
    fs                    Inspect the contents of an allocation directory
    init                  Create an example job file
    inspect               Inspect a submitted job
    logs                  Streams the logs of a task.
    node-drain            Toggle drain mode on a given node
    node-status           Display status information about nodes
    plan                  Dry-run a job update to determine its effects
    run                   Run a new job or update an existing job
    server-force-leave    Force a server into the 'left' state
    server-join           Join server nodes together
    server-members        Display a list of known servers and their status
    status                Display status information about jobs
    stop                  Stop a running job
    validate              Checks if a given job specification is valid
    version               Prints the Nomad version
```

If you get an error that Nomad could not be found, then your Vagrant box
may not have provisioned correctly. Check for any error messages that may have
been emitted during `vagrant up`. You can always destroy the box and
re-create it.

## Next Steps

Vagrant is running and Nomad is installed. Let's [start Nomad](/intro/getting-started/running.html)!


