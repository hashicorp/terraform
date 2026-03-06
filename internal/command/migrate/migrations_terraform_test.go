// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package migrate

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestTerraformRequiredProvidersMap(t *testing.T) {
	sub := findSubMigration(t, terraformMigrations(), "terraform/terraform/v1.x-to-v2.x", "required-providers-map")

	tests := map[string]struct {
		input    string
		expected string
	}{
		"converts old-style to new-style": {
			input: `terraform {
  required_providers {
    aws = "~> 3.0"
  }
}
`,
			expected: `terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 3.0"
    }
  }
}
`,
		},
		"converts multiple providers": {
			input: `terraform {
  required_providers {
    aws   = "~> 3.0"
    azurerm = "~> 2.0"
  }
}
`,
			expected: `terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 3.0"
    }
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 2.0"
    }
  }
}
`,
		},
		"no match leaves input unchanged": {
			input: `terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 3.0"
    }
  }
}
`,
			expected: `terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 3.0"
    }
  }
}
`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := sub.Apply("main.tf", []byte(tc.input))
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if diff := cmp.Diff(tc.expected, string(got)); diff != "" {
				t.Fatalf("unexpected result (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTerraformBackendToCloud(t *testing.T) {
	sub := findSubMigration(t, terraformMigrations(), "terraform/terraform/v1.x-to-v2.x", "backend-to-cloud")

	tests := map[string]struct {
		input    string
		expected string
	}{
		"converts backend remote to cloud block": {
			input: `terraform {
  backend "remote" {
    organization = "my-org"

    workspaces {
      name = "my-workspace"
    }
  }
}
`,
			expected: `terraform {
  cloud {
    organization = "my-org"

    workspaces {
      name = "my-workspace"
    }
  }
}
`,
		},
		"no match leaves input unchanged": {
			input: `terraform {
  backend "s3" {
    bucket = "my-bucket"
  }
}
`,
			expected: `terraform {
  backend "s3" {
    bucket = "my-bucket"
  }
}
`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := sub.Apply("main.tf", []byte(tc.input))
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if diff := cmp.Diff(tc.expected, string(got)); diff != "" {
				t.Fatalf("unexpected result (-want +got):\n%s", diff)
			}
		})
	}
}
