# Vagrant Nomad Demo

This Vagrantfile and associated Nomad configuration files are meant
to be used along with the
[getting started guide](https://nomadproject.io/intro/getting-started/install.html).

Follow along with the guide, or just start the Vagrant box with:

    $ vagrant up

Once it is finished, you should be able to SSH in and interact with Nomad:

    $ vagrant ssh
    ...
    $ nomad
    usage: nomad [--version] [--help] <command> [<args>]

    Available commands are:
        agent                 Runs a Nomad agent
        agent-info            Display status information about the local agent
    ...

To learn more about starting Nomad see the [official site](https://nomadproject.io).

