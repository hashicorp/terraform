// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"

	"github.com/hashicorp/go-tfe"
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

	TokenRequester
}

// PasswordTokenGetter is used to log user-facing messages while obtaining a token via login with username and password
type PasswordTokenGetter interface {
	// Divide UI output with a horizontal line
	HorizontalRule()

	// Contextualize the prompt for the user to enter their username and password.
	PromptForPassword()

	TokenRequester
}

// WebUiTokenGetter is used to log user-facing messages while obtaining a token via using the web UI
type WebUiTokenGetter interface {
	// Divide UI output with a horizontal line
	HorizontalRule()

	// Log that the page will be opened in the web browser automatically.
	OpeningTokensPage(hostname svchost.Hostname, url string)

	// Log that the user needs to open the page in their web browser manually.
	PromptToOpenTokensPage(hostname svchost.Hostname, url string)

	// Prompt the user to enter the token obtained from the web UI.
	PromptForToken()

	// Log that the token has been successfully retrieved.
	TokenRetrieved(*tfe.User)

	TokenRequester
}

// TokenRequester is used to log user-facing messages about obtaining
// an API token that are common to all means of obtaining it.
type TokenRequester interface {
	// Contextualize next steps for Terraform obtaining an API token.
	RequestingToken(hostname svchost.Hostname, mechanism string)

	// Advise the user that access to a web browser is required.
	NotifyNeedWebBrowser()

	// Advise the user that credentials will be stored using the named credentials helper.
	NotifyTokenStoredInCredentialsHelper(credsHelperType string)

	// Advise the user that credentials will be stored in a file.
	NotifyTokenStoredInFile(file string)
}

// Methods used to log user-facing messages during the `login` command
type Login interface {
	Diagnostics(diags tfdiags.Diagnostics)

	OAuthTokenGetter
	PasswordTokenGetter
	WebUiTokenGetter
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
	_ Login               = (*LoginHuman)(nil)
	_ OAuthTokenGetter    = (*LoginHuman)(nil)
	_ PasswordTokenGetter = (*LoginHuman)(nil)
	_ WebUiTokenGetter    = (*LoginHuman)(nil)
	_ TokenRequester      = (*LoginHuman)(nil)
)

// Implements Login, PasswordTokenGetter, WebUiTokenGetter
func (v *LoginHuman) HorizontalRule() {
	v.view.streams.Print("\n---------------------------------------------------------------------------------\n\n")
}

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

// Implements PasswordTokenGetter
func (v *LoginHuman) PromptForPassword() {
	v.view.streams.Print("Terraform must temporarily use your password to request an API token.\nThis password will NOT be saved locally.\n\n")
}

// Implements WebUiTokenGetter
func (v *LoginHuman) OpeningTokensPage(hostname svchost.Hostname, url string) {
	message := `Terraform must now open a web browser to the tokens page for %s.
If a browser does not open this automatically, open the following URL to proceed:
    %s

`
	v.view.streams.Print(fmt.Sprintf(message, hostname.ForDisplay(), url))
}

// Implements WebUiTokenGetter
func (v *LoginHuman) PromptToOpenTokensPage(hostname svchost.Hostname, url string) {
	v.view.streams.Print(fmt.Sprintf("Open the following URL to access the tokens page for %s:\n    %s\n\n", hostname.ForDisplay(), url))
}

// Implements WebUiTokenGetter
func (v *LoginHuman) PromptForToken() {
	v.view.streams.Print("Generate a token using your browser, and copy-paste it into this prompt.\n\n")
}

// Implements WebUiTokenGetter
func (v *LoginHuman) TokenRetrieved(user *tfe.User) {
	v.view.streams.Print(fmt.Sprintf("Retrieved token for user %s\n\n", user.Username))
}

// Implements TokenRequester
func (v *LoginHuman) RequestingToken(hostname svchost.Hostname, mechanism string) {
	v.view.streams.Print(fmt.Sprintf("Terraform will request an API token for %s using %s.\n\n", hostname.ForDisplay(), mechanism))
}

// Implements TokenRequester
func (v *LoginHuman) NotifyNeedWebBrowser() {
	v.view.streams.Print(`This will work only if you are able to use a web browser on this computer to
complete a login process. If not, you must obtain an API token by another
means and configure it in the CLI configuration manually.

`)
}

// Implements TokenRequester
func (v *LoginHuman) NotifyTokenStoredInCredentialsHelper(credsHelperType string) {
	v.view.streams.Print(
		fmt.Sprintf("If login is successful, Terraform will store the token in the configured\n%q credentials helper for use by subsequent commands.\n\n", credsHelperType),
	)
}

// Implements TokenRequester
func (v *LoginHuman) NotifyTokenStoredInFile(file string) {
	v.view.streams.Print(
		fmt.Sprintf("If login is successful, Terraform will store the token in plain text in\nthe following file for use by subsequent commands:\n    %s\n\n", file),
	)
}
