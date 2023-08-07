// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package genconfig

import (
	"fmt"
	"io"
	"os"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

func ShouldWriteConfig(out string) bool {
	// No specified out file, so don't write anything.
	return len(out) != 0
}

func ValidateTargetFile(out string) (diags tfdiags.Diagnostics) {
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Target generated file already exists",
			"Terraform can only write generated config into a new file. Either choose a different target location or move all existing configuration out of the target file, delete it and try again."))

	}
	return diags
}

type Change struct {
	Addr            string
	ImportID        string
	GeneratedConfig string
}

func (c *Change) MaybeWriteConfig(writer io.Writer, out string) (io.Writer, bool, tfdiags.Diagnostics) {
	var wroteConfig bool
	var diags tfdiags.Diagnostics
	if len(c.GeneratedConfig) > 0 {
		if writer == nil {
			// Lazily create the generated file, in case we have no
			// generated config to create.
			if w, err := os.Create(out); err != nil {
				if os.IsPermission(err) {
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Failed to create target generated file",
						fmt.Sprintf("Terraform did not have permission to create the generated file (%s) in the target directory. Please modify permissions over the target directory, and try again.", out)))
					return nil, false, diags
				}

				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to create target generated file",
					fmt.Sprintf("Terraform could not create the generated file (%s) in the target directory: %v. Depending on the error message, this may be a bug in Terraform itself. If so, please report it!", out, err)))
				return nil, false, diags
			} else {
				writer = w
			}

			header := "# __generated__ by Terraform\n# Please review these resources and move them into your main configuration files.\n"
			// Missing the header from the file, isn't the end of the world
			// so if this did return an error, then we will just ignore it.
			_, _ = writer.Write([]byte(header))
		}

		header := "\n# __generated__ by Terraform"
		if len(c.ImportID) > 0 {
			header += fmt.Sprintf(" from %q", c.ImportID)
		}
		header += "\n"
		if _, err := writer.Write([]byte(fmt.Sprintf("%s%s\n", header, c.GeneratedConfig))); err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Warning,
				"Failed to save generated config",
				fmt.Sprintf("Terraform encountered an error while writing generated config: %v. The config for %s must be created manually before applying. Depending on the error message, this may be a bug in Terraform itself. If so, please report it!", err, c.Addr)))
		}
		wroteConfig = true
	}

	return writer, wroteConfig, diags
}
