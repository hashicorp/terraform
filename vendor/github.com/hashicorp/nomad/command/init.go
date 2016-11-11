package command

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

const (
	// DefaultInitName is the default name we use when
	// initializing the example file
	DefaultInitName = "example.nomad"
)

// InitCommand generates a new job template that you can customize to your
// liking, like vagrant init
type InitCommand struct {
	Meta
}

func (c *InitCommand) Help() string {
	helpText := `
Usage: nomad init

  Creates an example job file that can be used as a starting
  point to customize further.
`
	return strings.TrimSpace(helpText)
}

func (c *InitCommand) Synopsis() string {
	return "Create an example job file"
}

func (c *InitCommand) Run(args []string) int {
	// Check for misuse
	if len(args) != 0 {
		c.Ui.Error(c.Help())
		return 1
	}

	// Check if the file already exists
	_, err := os.Stat(DefaultInitName)
	if err != nil && !os.IsNotExist(err) {
		c.Ui.Error(fmt.Sprintf("Failed to stat '%s': %v", DefaultInitName, err))
		return 1
	}
	if !os.IsNotExist(err) {
		c.Ui.Error(fmt.Sprintf("Job '%s' already exists", DefaultInitName))
		return 1
	}

	// Write out the example
	err = ioutil.WriteFile(DefaultInitName, []byte(defaultJob), 0660)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to write '%s': %v", DefaultInitName, err))
		return 1
	}

	// Success
	c.Ui.Output(fmt.Sprintf("Example job file written to %s", DefaultInitName))
	return 0
}

var defaultJob = strings.TrimSpace(`
# There can only be a single job definition per file.
# Create a job with ID and Name 'example'
job "example" {
	# Run the job in the global region, which is the default.
	# region = "global"

	# Specify the datacenters within the region this job can run in.
	datacenters = ["dc1"]

	# Service type jobs optimize for long-lived services. This is
	# the default but we can change to batch for short-lived tasks.
	# type = "service"

	# Priority controls our access to resources and scheduling priority.
	# This can be 1 to 100, inclusively, and defaults to 50.
	# priority = 50

	# Restrict our job to only linux. We can specify multiple
	# constraints as needed.
	constraint {
		attribute = "${attr.kernel.name}"
		value = "linux"
	}

	# Configure the job to do rolling updates
	update {
		# Stagger updates every 10 seconds
		stagger = "10s"

		# Update a single task at a time
		max_parallel = 1
	}

	# Create a 'cache' group. Each task in the group will be
	# scheduled onto the same machine.
	group "cache" {
		# Control the number of instances of this group.
		# Defaults to 1
		# count = 1

		# Configure the restart policy for the task group. If not provided, a
		# default is used based on the job type.
		restart {
			# The number of attempts to run the job within the specified interval.
			attempts = 10
			interval = "5m"
			
			# A delay between a task failing and a restart occurring.
			delay = "25s"

			# Mode controls what happens when a task has restarted "attempts"
			# times within the interval. "delay" mode delays the next restart
			# till the next interval. "fail" mode does not restart the task if
			# "attempts" has been hit within the interval.
			mode = "delay"
		}

		# Define a task to run
		task "redis" {
			# Use Docker to run the task.
			driver = "docker"

			# Configure Docker driver with the image
			config {
				image = "redis:latest"
				port_map {
					db = 6379
				}
			}

			service {
				name = "${TASKGROUP}-redis"
				tags = ["global", "cache"]
				port = "db"
				check {
					name = "alive"
					type = "tcp"
					interval = "10s"
					timeout = "2s"
				}
			}

			# We must specify the resources required for
			# this task to ensure it runs on a machine with
			# enough capacity.
			resources {
				cpu = 500 # 500 MHz
				memory = 256 # 256MB
				network {
					mbits = 10
					port "db" {
					}
				}
			}

			# The artifact block can be specified one or more times to download
			# artifacts prior to the task being started. This is convenient for
			# shipping configs or data needed by the task.
			# artifact {
			#	  source = "http://foo.com/artifact.tar.gz"
			#	  options {
			#	      checksum = "md5:c4aa853ad2215426eb7d70a21922e794"
			#     }
			# }
			
			# Specify configuration related to log rotation
			# logs {
			#     max_files = 10
			#     max_file_size = 15
			# }
			 
			# Controls the timeout between signalling a task it will be killed
			# and killing the task. If not set a default is used.
			# kill_timeout = "20s"
		}
	}
}
`)
