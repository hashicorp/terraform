package nomad

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	r "github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/hashicorp/nomad/api"
)

func TestResourceJob_basic(t *testing.T) {
	r.Test(t, r.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []r.TestStep{
			{
				Config: testResourceJob_initialConfig,
				Check:  testResourceJob_initialCheck,
			},
		},

		CheckDestroy: testResourceJob_checkDestroy("foo"),
	})
}

func TestResourceJob_refresh(t *testing.T) {
	r.Test(t, r.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []r.TestStep{
			{
				Config: testResourceJob_initialConfig,
				Check:  testResourceJob_initialCheck,
			},

			// This should successfully cause the job to be recreated,
			// testing the Exists function.
			{
				PreConfig: testResourceJob_deregister(t, "foo"),
				Config:    testResourceJob_initialConfig,
			},
		},
	})
}

func TestResourceJob_disableDestroyDeregister(t *testing.T) {
	r.Test(t, r.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []r.TestStep{
			{
				Config: testResourceJob_noDestroy,
				Check:  testResourceJob_initialCheck,
			},

			// Destroy with our setting set
			{
				Destroy: true,
				Config:  testResourceJob_noDestroy,
				Check:   testResourceJob_checkExists,
			},

			// Re-apply without the setting set
			{
				Config: testResourceJob_initialConfig,
				Check:  testResourceJob_checkExists,
			},
		},
	})
}

func TestResourceJob_idChange(t *testing.T) {
	r.Test(t, r.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []r.TestStep{
			{
				Config: testResourceJob_initialConfig,
				Check:  testResourceJob_initialCheck,
			},

			// Change our ID
			{
				Config: testResourceJob_updateConfig,
				Check:  testResourceJob_updateCheck,
			},
		},
	})
}

func TestResourceJob_parameterizedJob(t *testing.T) {
	r.Test(t, r.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []r.TestStep{
			{
				Config: testResourceJob_parameterizedJob,
				Check:  testResourceJob_initialCheck,
			},
		},
	})
}

var testResourceJob_initialConfig = `
resource "nomad_job" "test" {
    jobspec = <<EOT
job "foo" {
    datacenters = ["dc1"]
    type = "service"
    group "foo" {
        task "foo" {
            driver = "raw_exec"
            config {
                command = "/bin/sleep"
                args = ["1"]
            }

            resources {
                cpu = 20
                memory = 10
            }

            logs {
                max_files = 3
                max_file_size = 10
            }
        }
    }
}
EOT
}
`

var testResourceJob_noDestroy = `
resource "nomad_job" "test" {
    deregister_on_destroy = false
    jobspec = <<EOT
job "foo" {
    datacenters = ["dc1"]
    type = "service"
    group "foo" {
        task "foo" {
            driver = "raw_exec"
            config {
                command = "/bin/sleep"
                args = ["1"]
            }

            resources {
                cpu = 20
                memory = 10
            }

            logs {
                max_files = 3
                max_file_size = 10
            }
        }
    }
}
EOT
}
`

func testResourceJob_initialCheck(s *terraform.State) error {
	resourceState := s.Modules[0].Resources["nomad_job.test"]
	if resourceState == nil {
		return errors.New("resource not found in state")
	}

	instanceState := resourceState.Primary
	if instanceState == nil {
		return errors.New("resource has no primary instance")
	}

	jobID := instanceState.ID

	client := testProvider.Meta().(*api.Client)
	job, _, err := client.Jobs().Info(jobID, nil)
	if err != nil {
		return fmt.Errorf("error reading back job: %s", err)
	}

	if got, want := job.ID, jobID; got != want {
		return fmt.Errorf("jobID is %q; want %q", got, want)
	}

	return nil
}

func testResourceJob_checkExists(s *terraform.State) error {
	jobID := "foo"

	client := testProvider.Meta().(*api.Client)
	_, _, err := client.Jobs().Info(jobID, nil)
	if err != nil {
		return fmt.Errorf("error reading back job: %s", err)
	}

	return nil
}

func testResourceJob_checkDestroy(jobID string) r.TestCheckFunc {
	return func(*terraform.State) error {
		client := testProvider.Meta().(*api.Client)
		job, _, err := client.Jobs().Info(jobID, nil)
		// This should likely never happen, due to how nomad caches jobs
		if err != nil && strings.Contains(err.Error(), "404") || job == nil {
			return nil
		}

		if job.Status != "dead" {
			return fmt.Errorf("Job %q has not been stopped. Status: %s", jobID, job.Status)
		}

		return nil
	}
}

func testResourceJob_deregister(t *testing.T, jobID string) func() {
	return func() {
		client := testProvider.Meta().(*api.Client)
		_, _, err := client.Jobs().Deregister(jobID, nil)
		if err != nil {
			t.Fatalf("error deregistering job: %s", err)
		}
	}
}

var testResourceJob_updateConfig = `
resource "nomad_job" "test" {
    jobspec = <<EOT
job "bar" {
    datacenters = ["dc1"]
    type = "service"
    group "foo" {
        task "foo" {
            driver = "raw_exec"
            config {
                command = "/bin/sleep"
                args = ["1"]
            }

            resources {
                cpu = 20
                memory = 10
            }

            logs {
                max_files = 3
                max_file_size = 10
            }
        }
    }
}
EOT
}
`

func testResourceJob_updateCheck(s *terraform.State) error {
	resourceState := s.Modules[0].Resources["nomad_job.test"]
	if resourceState == nil {
		return errors.New("resource not found in state")
	}

	instanceState := resourceState.Primary
	if instanceState == nil {
		return errors.New("resource has no primary instance")
	}

	jobID := instanceState.ID

	client := testProvider.Meta().(*api.Client)
	job, _, err := client.Jobs().Info(jobID, nil)
	if err != nil {
		return fmt.Errorf("error reading back job: %s", err)
	}

	if got, want := job.ID, jobID; got != want {
		return fmt.Errorf("jobID is %q; want %q", got, want)
	}

	{
		// Verify foo doesn't exist
		job, _, err := client.Jobs().Info("foo", nil)
		if err != nil {
			// Job could have already been purged from nomad server
			if !strings.Contains(err.Error(), "(job not found)") {
				return fmt.Errorf("error reading %q job: %s", "foo", err)
			}
			return nil
		}
		if job.Status != "dead" {
			return fmt.Errorf("%q job is not dead. Status: %q", "foo", job.Status)
		}
	}

	return nil
}

var testResourceJob_parameterizedJob = `
resource "nomad_job" "test" {
    jobspec = <<EOT
job "bar" {
    datacenters = ["dc1"]
    type = "batch"
    parameterized {
      payload = "required"
    }
    group "foo" {
        task "foo" {
            driver = "raw_exec"
            config {
                command = "/bin/sleep"
                args = ["1"]
            }
            resources {
                cpu = 20
                memory = 10
            }

            logs {
                max_files = 3
                max_file_size = 10
            }
        }
    }
}
EOT
}
`
