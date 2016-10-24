# Dist

The `dist` folder contains sample configs for various platforms.

## Conventions

On unixes we will place agent configs under `/etc/nomad` and store data under `/var/lib/nomad/`. You will need to create both of these directories. We assume that `nomad` is installed to `/usr/bin/nomad`.

## Agent Configs

The following example configuration files are provided:

- `server.hcl`
- `client.hcl`

Place one of these under `/etc/nomad` depending on the node's role. You should use `server.hcl` to configure a node as a server (which is responsible for scheduling) or `client.hcl` to configure a node as a client (which is responsible for running workloads).

Read <https://nomadproject.io/docs/agent/config.html> to learn which options are available and how to configure them.

## Upstart

On systems using upstart the basic upstart file under `upstart/nomad.conf` starts and stops the nomad agent. Place it under `/etc/init/nomad.conf`.

You can control Nomad with `start|stop|restart nomad`.

## Systemd

On systems using systemd the basic systemd unit file under `systemd/nomad.service` starts and stops the nomad agent. Place it under `/etc/systemd/system/nomad.service`.

You can control Nomad with `systemctl start|stop|restart nomad`.