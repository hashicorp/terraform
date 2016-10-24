---
layout: "docs"
page_title: "Operating a Job: Task Configuration"
sidebar_current: "docs-jobops-task-config"
description: |-
  Learn how to ship task configuration in a Nomad Job.
---

# Task Configurations

Most tasks need to be parameterized in some way. The simplest is via
command-line arguments but often times tasks consume complex configurations via
config files.  Here we explore how to configure Nomad jobs to support many
common configuration use cases.

## Command-line Arguments

The simplest type of configuration to support is tasks which take their
configuration via command-line arguments that will not change.

Nomad has many [drivers](/docs/drivers/index.html) and most support passing
arguments to their tasks via the `args` parameter. To configure these simply
provide the appropriate arguments. Below is an example using the [`docker`
driver](/docs/drivers/docker.html) to launch `memcached(8)` and set its thread count
to 4, increase log verbosity, as well as assign the correct port and address
bindings using interpolation:

```
task "memcached" {
    driver = "docker"
    
	config {
		image = "memcached:1.4.27"
		args = [
			# Set thread count
			"-t", "4",

			# Enable the highest verbosity logging mode
			"-vvv", 

			# Use interpolations to limit memory usage and bind
			# to the proper address
			"-m", "${NOMAD_MEMORY_LIMIT}",
			"-p", "${NOMAD_PORT_db}",
			"-l", "${NOMAD_ADDR_db}"
		]

		network_mode = "host"
	}

	resources {
		cpu = 500 # 500 MHz
		memory = 256 # 256MB
		network {
			mbits = 10
			port "db" {
			}
		}
	}
}
```

In the above example, we see how easy it is to pass configuration options using
the `args` section and even see how
[interpolation](docs/jobspec/interpreted.html) allows us to pass arguments
based on the dynamic port and address Nomad chose for this task.

## Config Files

Often times applications accept their configurations using configuration files
or have so many arguments to be set it would be unwieldy to pass them via
arguments. Nomad supports downloading
[`artifacts`](/docs/jobspec/index.html#artifact_doc) prior to launching tasks.
This allows shipping of configuration files and other assets that the task
needs to run properly.

An example can be seen below, where we download two artifacts, one being the
binary to run and the other beings its configuration:

```
task "example" {
    driver = "exec"
    
	config {
		command = "my-app"
		args = ["-config", "local/config.cfg"]
	}

    # Download the binary to run
	artifact {
		source = "http://example.com/example/my-app"
    }

	# Download the config file
	artifact {
		source = "http://example.com/example/config.cfg"
    }
}
```

Here we can see a basic example of downloading static configuration files. By
default, an `artifact` is downloaded to the task's `local/` directory but is
[configurable](/docs/jobspec/index.html#artifact_doc).
