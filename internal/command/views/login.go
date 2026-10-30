// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"

	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// OAuthTokenGetter is used to log user-facing messages while obtaining a token via login using OAuth
type OAuthTokenGetter interface {
	// Log that the page will be opened in the web browser automatically.
	OpeningLoginPage(hostname svchost.Hostname, url string)

	// Log that the user needs to open the page in their web browser manually.
	PromptToOpenLoginPage(hostname svchost.Hostname, url string)

	// Log that the program is waiting for a signal that the user has successfully logged in.
	WaitingForLoginSuccess()
}

// Methods used to log user-facing messages during the `login` command
type Login interface {
	Diagnostics(diags tfdiags.Diagnostics)

	OAuthTokenGetter
}

func NewLogin(viewType arguments.ViewType, view *View) Login {
	switch viewType {
	case arguments.ViewHuman:
		return &LoginHuman{
			view: view,
		}
	default:
		panic("unsupported view type: " + string(viewType))
	}
}

type LoginHuman struct {
	view *View
}

var (
	_ Login            = (*LoginHuman)(nil)
	_ OAuthTokenGetter = (*LoginHuman)(nil)
)

func (v *LoginHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

// Implements OAuthTokenGetter
func (v *LoginHuman) OpeningLoginPage(hostname svchost.Hostname, url string) {
	// Note, newlines are load-bearing in the message string below
	message := `Terraform must now open a web browser to the login page for %s.

If a browser does not open this automatically, open the following URL to proceed:
    %s

`
	v.view.streams.Print(fmt.Sprintf(message, hostname.ForDisplay(), url))
}

// Implements OAuthTokenGetter
func (v *LoginHuman) PromptToOpenLoginPage(hostname svchost.Hostname, url string) {
	v.view.streams.Print(fmt.Sprintf("Open the following URL to access the login page for %s:\n    %s\n\n", hostname.ForDisplay(), url))
}

// Implements OAuthTokenGetter
func (v *LoginHuman) WaitingForLoginSuccess() {
	v.view.streams.Print("Terraform will now wait for the host to signal that login was successful.\n\n")
}
