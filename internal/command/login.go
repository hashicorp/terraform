// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	tfe "github.com/hashicorp/go-tfe"
	svchost "github.com/hashicorp/terraform-svchost"
	svcauth "github.com/hashicorp/terraform-svchost/auth"
	"github.com/hashicorp/terraform-svchost/disco"
	"github.com/hashicorp/terraform/internal/command/cliconfig"
	"github.com/hashicorp/terraform/internal/httpclient"
	"github.com/hashicorp/terraform/internal/logging"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"

	uuid "github.com/hashicorp/go-uuid"
	"golang.org/x/oauth2"
)

// LoginCommand is a Command implementation that runs an interactive login
// flow for a remote service host. It then stashes credentials in a tfrc
// file in the user's home directory.
type LoginCommand struct {
	Meta
}

// Run implements cli.Command.
func (c *LoginCommand) Run(args []string) int {
	args = c.Meta.process(args)
	cmdFlags := c.Meta.extendedFlagSet("login")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) > 1 {
		c.Ui.Error(
			"The login command expects at most one argument: the host to log in to.")
		cmdFlags.Usage()
		return 1
	}

	var diags tfdiags.Diagnostics

	if !c.input {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Login is an interactive command",
			"The \"terraform login\" command uses interactive prompts to obtain and record credentials, so it can't be run with input disabled.\n\nTo configure credentials in a non-interactive context, write existing credentials directly to a CLI configuration file.",
		))
		c.showDiagnostics(diags)
		return 1
	}

	givenHostname := "app.terraform.io"
	if len(args) != 0 {
		givenHostname = args[0]
	}

	hostname, err := svchost.ForComparison(givenHostname)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid hostname",
			fmt.Sprintf("The given hostname %q is not valid: %s.", givenHostname, err.Error()),
		))
		c.showDiagnostics(diags)
		return 1
	}

	// From now on, since we've validated the given hostname, we should use
	// dispHostname in the UI to ensure we're presenting it in the canonical
	// form, in case that helpers users with debugging when things aren't
	// working as expected. (Perhaps the normalization is part of the cause.)
	dispHostname := hostname.ForDisplay()

	host, err := c.Services.Discover(hostname)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Service discovery failed for "+dispHostname,

			// Contrary to usual Go idiom, the Discover function returns
			// full sentences with initial capitalization in its error messages,
			// and they are written with the end-user as the audience. We
			// only need to add the trailing period to make them consistent
			// with our usual error reporting standards.
			err.Error()+".",
		))
		c.showDiagnostics(diags)
		return 1
	}

	creds := c.Services.CredentialsSource().(*cliconfig.CredentialsSource)
	filename, _ := creds.CredentialsFilePath()
	credsCtx := &loginCredentialsContext{
		Location:      creds.HostCredentialsLocation(hostname),
		LocalFilename: filename, // empty in the very unlikely event that we can't select a config directory for this user
		HelperType:    creds.CredentialsHelperType(),
	}

	clientConfig, err := host.ServiceOAuthClient("login.v1")
	switch err.(type) {
	case nil:
		// Great! No problem, then.
	case *disco.ErrServiceNotProvided:
		// This is also fine! We'll try the manual token creation process.
	case *disco.ErrVersionNotSupported:
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Warning,
			"Host does not support Terraform login",
			fmt.Sprintf("The given hostname %q allows creating Terraform authorization tokens, but requires a newer version of Terraform CLI to do so.", dispHostname),
		))
	default:
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Warning,
			"Host does not support Terraform login",
			fmt.Sprintf("The given hostname %q cannot support \"terraform login\": %s.", dispHostname, err),
		))
	}

	// If login service is unavailable, check for a TFE v2 API as fallback
	var tfeservice *url.URL
	if clientConfig == nil {
		tfeservice, err = host.ServiceURL("tfe.v2")
		switch err.(type) {
		case nil:
			// Success!
		case *disco.ErrServiceNotProvided:
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Host does not support Terraform tokens API",
				fmt.Sprintf("The given hostname %q does not support creating Terraform authorization tokens.", dispHostname),
			))
		case *disco.ErrVersionNotSupported:
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Host does not support Terraform tokens API",
				fmt.Sprintf("The given hostname %q allows creating Terraform authorization tokens, but requires a newer version of Terraform CLI to do so.", dispHostname),
			))
		default:
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Host does not support Terraform tokens API",
				fmt.Sprintf("The given hostname %q cannot support \"terraform login\": %s.", dispHostname, err),
			))
		}
	}

	if credsCtx.Location == cliconfig.CredentialsInOtherFile {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			fmt.Sprintf("Credentials for %s are manually configured", dispHostname),
			"The \"terraform login\" command cannot log in because credentials for this host are already configured in a CLI configuration file.\n\nTo log in, first revoke the existing credentials and remove that block from the CLI configuration.",
		))
	}

	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	var token svcauth.HostCredentialsToken
	var tokenDiags tfdiags.Diagnostics

	// Prefer Terraform login if available
	if clientConfig != nil {
		var oauthToken *oauth2.Token

		switch {
		case clientConfig.SupportedGrantTypes.Has(disco.OAuthAuthzCodeGrant):
			// We prefer an OAuth code grant if the server supports it.
			oauthToken, tokenDiags = c.interactiveGetTokenByCode(hostname, credsCtx, clientConfig)
		case clientConfig.SupportedGrantTypes.Has(disco.OAuthOwnerPasswordGrant) && hostname == svchost.Hostname("app.terraform.io"):
			// The password grant type is allowed only for HCP Terraform SaaS.
			// Note this case is purely theoretical at this point, as HCP Terraform currently uses
			// its own bespoke login protocol (tfe)
			oauthToken, tokenDiags = c.interactiveGetTokenByPassword(hostname, credsCtx, clientConfig)
		default:
			tokenDiags = tokenDiags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Host does not support Terraform login",
				fmt.Sprintf("The given hostname %q does not allow any OAuth grant types that are supported by this version of Terraform.", dispHostname),
			))
		}
		if oauthToken != nil {
			token = svcauth.HostCredentialsToken(oauthToken.AccessToken)
		}
	} else if tfeservice != nil {
		token, tokenDiags = c.interactiveGetTokenByUI(hostname, credsCtx, tfeservice)
	}

	diags = diags.Append(tokenDiags)
	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	err = creds.StoreForHost(hostname, token)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to save API token",
			fmt.Sprintf("The given host returned an API token, but Terraform failed to save it: %s.", err),
		))
	}

	c.showDiagnostics(diags)
	if diags.HasErrors() {
		return 1
	}

	c.Ui.Output("\n---------------------------------------------------------------------------------\n")
	if hostname == "app.terraform.io" { // HCP Terraform
		var motd struct {
			Message string        `json:"msg"`
			Errors  []interface{} `json:"errors"`
		}

		// Throughout the entire process of fetching a MOTD from TFC, use a default
		// message if the platform-provided message is unavailable for any reason -
		// be it the service isn't provided, the request failed, or any sort of
		// platform error returned.

		motdServiceURL, err := host.ServiceURL("motd.v1")
		if err != nil {
			c.logMOTDError(err)
			c.outputDefaultTFCLoginSuccess()
			return 0
		}

		req, err := http.NewRequest("GET", motdServiceURL.String(), nil)
		if err != nil {
			c.logMOTDError(err)
			c.outputDefaultTFCLoginSuccess()
			return 0
		}

		req.Header.Set("Authorization", "Bearer "+token.Token())

		resp, err := httpclient.New().Do(req)
		if err != nil {
			c.logMOTDError(err)
			c.outputDefaultTFCLoginSuccess()
			return 0
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			c.logMOTDError(err)
			c.outputDefaultTFCLoginSuccess()
			return 0
		}

		defer resp.Body.Close()
		json.Unmarshal(body, &motd)

		if motd.Errors == nil && motd.Message != "" {
			c.Ui.Output(
				c.Colorize().Color(motd.Message),
			)
			return 0
		} else {
			c.logMOTDError(fmt.Errorf("platform responded with errors or an empty message"))
			c.outputDefaultTFCLoginSuccess()
			return 0
		}
	}

	if tfeservice != nil { // Terraform Enterprise
		c.outputDefaultTFELoginSuccess(dispHostname)
	} else {
		c.Ui.Output(
			fmt.Sprintf(
				c.Colorize().Color(strings.TrimSpace(`
[green][bold]Success![reset] [bold]Terraform has obtained and saved an API token.[reset]

The new API token will be used for any future Terraform command that must make
authenticated requests to %s.
`)),
				dispHostname,
			) + "\n",
		)
	}

	return 0
}

func (c *LoginCommand) outputDefaultTFELoginSuccess(dispHostname string) {
	c.Ui.Output(
		fmt.Sprintf(
			c.Colorize().Color(strings.TrimSpace(`
[green][bold]Success![reset] [bold]Logged in to Terraform Enterprise (%s)[reset]
`)),
			dispHostname,
		) + "\n",
	)
}

func (c *LoginCommand) outputDefaultTFCLoginSuccess() {
	c.Ui.Output(c.Colorize().Color(strings.TrimSpace(`
[green][bold]Success![reset] [bold]Logged in to HCP Terraform[reset]
` + "\n")))
}

func (c *LoginCommand) logMOTDError(err error) {
	log.Printf("[TRACE] login: An error occurred attempting to fetch a message of the day for HCP Terraform: %s", err)
}

// Help implements cli.Command.
func (c *LoginCommand) Help() string {
	defaultFile := c.defaultOutputFile()
	if defaultFile == "" {
		// Because this is just for the help message and it's very unlikely
		// that a user wouldn't have a functioning home directory anyway,
		// we'll just use a placeholder here. The real command has some
		// more complex behavior for this case. This result is not correct
		// on all platforms, but given how unlikely we are to hit this case
		// that seems okay.
		defaultFile = "~/.terraform/credentials.tfrc.json"
	}

	helpText := fmt.Sprintf(`
Usage: terraform [global options] login [hostname]

  Retrieves an authentication token for the given hostname, if it supports
  automatic login, and saves it in a credentials file in your home directory.

  If no hostname is provided, the default hostname is app.terraform.io, to
  log in to HCP Terraform.

  If not overridden by credentials helper settings in the CLI configuration,
  the credentials will be written to the following local file:
      %s
`, defaultFile)
	return strings.TrimSpace(helpText)
}

// Synopsis implements cli.Command.
func (c *LoginCommand) Synopsis() string {
	return "Obtain and save credentials for a remote host"
}

func (c *LoginCommand) defaultOutputFile() string {
	if c.CLIConfigDir == "" {
		return "" // no default available
	}
	return filepath.Join(c.CLIConfigDir, "credentials.tfrc.json")
}

func (c *LoginCommand) interactiveGetTokenByCode(hostname svchost.Hostname, credsCtx *loginCredentialsContext, clientConfig *disco.OAuthClient) (*oauth2.Token, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	confirm, confirmDiags := c.interactiveContextConsent(hostname, disco.OAuthAuthzCodeGrant, credsCtx)
	diags = diags.Append(confirmDiags)
	if !confirm {
		diags = diags.Append(errors.New("Login cancelled"))
		return nil, diags
	}

	// We'll use an entirely pseudo-random UUID for our temporary request
	// state. The OAuth server must echo this back to us in the callback
	// request to make it difficult for some other running process to
	// interfere by sending its own request to our temporary server.
	reqState, err := uuid.GenerateUUID()
	if err != nil {
		// This should be very unlikely, but could potentially occur if e.g.
		// there's not enough pseudo-random entropy available.
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Can't generate login request state",
			fmt.Sprintf("Cannot generate random request identifier for login request: %s.", err),
		))
		return nil, diags
	}

	proofKey, proofKeyChallenge, err := c.proofKey()
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Can't generate login request state",
			fmt.Sprintf("Cannot generate random prrof key for login request: %s.", err),
		))
		return nil, diags
	}

	listener, callbackURL, err := c.listenerForCallback(clientConfig.MinPort, clientConfig.MaxPort)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Can't start temporary login server",
			fmt.Sprintf(
				"The login process uses OAuth, which requires starting a temporary HTTP server on localhost. However, no TCP port numbers between %d and %d are available to create such a server.",
				clientConfig.MinPort, clientConfig.MaxPort,
			),
		))
		return nil, diags
	}

	// codeCh will allow our temporary HTTP server to transmit the OAuth code
	// to the main execution path that follows.
	codeCh := make(chan string)
	server := &http.Server{
		Handler: http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			log.Printf("[TRACE] login: request to callback server")
			err := req.ParseForm()
			if err != nil {
				log.Printf("[ERROR] login: cannot ParseForm on callback request: %s", err)
				resp.WriteHeader(400)
				return
			}
			gotState := req.Form.Get("state")
			if gotState != reqState {
				log.Printf("[ERROR] login: incorrect \"state\" value in callback request")
				resp.WriteHeader(400)
				return
			}
			gotCode := req.Form.Get("code")
			if gotCode == "" {
				log.Printf("[ERROR] login: no \"code\" argument in callback request")
				resp.WriteHeader(400)
				return
			}

			log.Printf("[TRACE] login: request contains an authorization code")

			// Send the code to our blocking wait below, so that the token
			// fetching process can continue.
			codeCh <- gotCode
			close(codeCh)

			log.Printf("[TRACE] login: returning response from callback server")

			resp.Header().Add("Content-Type", "text/html")
			resp.WriteHeader(200)
			resp.Write([]byte(callbackSuccessMessage))
		}),
	}
	go func() {
		defer logging.PanicHandler()
		err := server.Serve(listener)
		if err != nil && err != http.ErrServerClosed {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Can't start temporary login server",
				fmt.Sprintf(
					"The login process uses OAuth, which requires starting a temporary HTTP server on localhost. However, no TCP port numbers between %d and %d are available to create such a server.",
					clientConfig.MinPort, clientConfig.MaxPort,
				),
			))
			close(codeCh)
		}
	}()

	oauthConfig := &oauth2.Config{
		ClientID:    clientConfig.ID,
		Endpoint:    clientConfig.Endpoint(),
		RedirectURL: callbackURL,
		Scopes:      clientConfig.Scopes,
	}

	authCodeURL := oauthConfig.AuthCodeURL(
		reqState,
		oauth2.SetAuthURLParam("code_challenge", proofKeyChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)

	launchBrowserManually := false
	if c.BrowserLauncher != nil {
		err = c.BrowserLauncher.OpenURL(authCodeURL)
		if err == nil {
			c.Ui.Output(fmt.Sprintf("Terraform must now open a web browser to the login page for %s.\n", hostname.ForDisplay()))
			c.Ui.Output(fmt.Sprintf("If a browser does not open this automatically, open the following URL to proceed:\n    %s\n", authCodeURL))
		} else {
			// Assume we're on a platform where opening a browser isn't possible.
			launchBrowserManually = true
		}
	} else {
		launchBrowserManually = true
	}

	if launchBrowserManually {
		c.Ui.Output(fmt.Sprintf("Open the following URL to access the login page for %s:\n    %s\n", hostname.ForDisplay(), authCodeURL))
	}

	c.Ui.Output("Terraform will now wait for the host to signal that login was successful.\n")

	code, ok := <-codeCh
	if !ok {
		// If we got no code at all then the server wasn't able to start
		// up, so we'll just give up.
		return nil, diags
	}

	if err := server.Close(); err != nil {
		// The server will close soon enough when our process exits anyway,
		// so we won't fuss about it for right now.
		log.Printf("[WARN] login: callback server can't shut down: %s", err)
	}

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, httpclient.New())
	token, err := oauthConfig.Exchange(
		ctx, code,
		oauth2.SetAuthURLParam("code_verifier", proofKey),
	)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to obtain auth token",
			fmt.Sprintf("The remote server did not assign an auth token: %s.", err),
		))
		return nil, diags
	}

	return token, diags
}

func (c *LoginCommand) interactiveGetTokenByPassword(hostname svchost.Hostname, credsCtx *loginCredentialsContext, clientConfig *disco.OAuthClient) (*oauth2.Token, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	confirm, confirmDiags := c.interactiveContextConsent(hostname, disco.OAuthOwnerPasswordGrant, credsCtx)
	diags = diags.Append(confirmDiags)
	if !confirm {
		diags = diags.Append(errors.New("Login cancelled"))
		return nil, diags
	}

	c.Ui.Output("\n---------------------------------------------------------------------------------\n")
	c.Ui.Output("Terraform must temporarily use your password to request an API token.\nThis password will NOT be saved locally.\n")

	username, err := c.UIInput().Input(context.Background(), &terraform.InputOpts{
		Id:    "username",
		Query: fmt.Sprintf("Username for %s:", hostname.ForDisplay()),
	})
	if err != nil {
		diags = diags.Append(fmt.Errorf("Failed to request username: %s", err))
		return nil, diags
	}
	password, err := c.UIInput().Input(context.Background(), &terraform.InputOpts{
		Id:     "password",
		Query:  fmt.Sprintf("Password for %s:", hostname.ForDisplay()),
		Secret: true,
	})
	if err != nil {
		diags = diags.Append(fmt.Errorf("Failed to request password: %s", err))
		return nil, diags
	}

	oauthConfig := &oauth2.Config{
		ClientID: clientConfig.ID,
		Endpoint: clientConfig.Endpoint(),
		Scopes:   clientConfig.Scopes,
	}
	token, err := oauthConfig.PasswordCredentialsToken(context.Background(), username, password)
	if err != nil {
		// FIXME: The OAuth2 library generates errors that are not appropriate
		// for a Terraform end-user audience, so once we have more experience
		// with which errors are most common we should try to recognize them
		// here and produce better error messages for them.
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to retrieve API token",
			fmt.Sprintf("The remote host did not issue an API token: %s.", err),
		))
	}

	return token, diags
}

func (c *LoginCommand) interactiveGetTokenByUI(hostname svchost.Hostname, credsCtx *loginCredentialsContext, service *url.URL) (svcauth.HostCredentialsToken, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	confirm, confirmDiags := c.interactiveContextConsent(hostname, disco.OAuthGrantType(""), credsCtx)
	diags = diags.Append(confirmDiags)
	if !confirm {
		diags = diags.Append(errors.New("Login cancelled"))
		return "", diags
	}

	c.Ui.Output("\n---------------------------------------------------------------------------------\n")

	tokensURL := url.URL{
		Scheme:   "https",
		Host:     service.Hostname(),
		Path:     "/app/settings/tokens",
		RawQuery: "source=terraform-login",
	}

	launchBrowserManually := false
	if c.BrowserLauncher != nil {
		err := c.BrowserLauncher.OpenURL(tokensURL.String())
		if err == nil {
			c.Ui.Output(fmt.Sprintf("Terraform must now open a web browser to the tokens page for %s.\n", hostname.ForDisplay()))
			c.Ui.Output(fmt.Sprintf("If a browser does not open this automatically, open the following URL to proceed:\n    %s\n", tokensURL.String()))
		} else {
			log.Printf("[DEBUG] error opening web browser: %s", err)
			// Assume we're on a platform where opening a browser isn't possible.
			launchBrowserManually = true
		}
	} else {
		launchBrowserManually = true
	}

	if launchBrowserManually {
		c.Ui.Output(fmt.Sprintf("Open the following URL to access the tokens page for %s:\n    %s\n", hostname.ForDisplay(), tokensURL.String()))
	}

	c.Ui.Output("\n---------------------------------------------------------------------------------\n")
	c.Ui.Output("Generate a token using your browser, and copy-paste it into this prompt.\n")

	// credsCtx might not be set if we're using a mock credentials source
	// in a test, but it should always be set in normal use.
	if credsCtx != nil {
		switch credsCtx.Location {
		case cliconfig.CredentialsViaHelper:
			c.Ui.Output(fmt.Sprintf("Terraform will store the token in the configured %q credentials helper\nfor use by subsequent commands.\n", credsCtx.HelperType))
		case cliconfig.CredentialsInPrimaryFile, cliconfig.CredentialsNotAvailable:
			c.Ui.Output(fmt.Sprintf("Terraform will store the token in plain text in the following file\nfor use by subsequent commands:\n    %s\n", credsCtx.LocalFilename))
		}
	}

	token, err := c.UIInput().Input(context.Background(), &terraform.InputOpts{
		Id:     "token",
		Query:  fmt.Sprintf("Token for %s:", hostname.ForDisplay()),
		Secret: true,
	})
	if err != nil {
		diags := diags.Append(fmt.Errorf("Failed to retrieve token: %s", err))
		return "", diags
	}

	token = strings.TrimSpace(token)
	cfg := &tfe.Config{
		Address:  service.String(),
		BasePath: service.Path,
		Token:    token,
		Headers:  make(http.Header),
	}
	client, err := tfe.NewClient(cfg)
	if err != nil {
		diags = diags.Append(fmt.Errorf("Failed to create API client: %s", err))
		return "", diags
	}
	user, err := client.Users.ReadCurrent(context.Background())
	if err == tfe.ErrUnauthorized {
		diags = diags.Append(fmt.Errorf("Token is invalid: %s", err))
		return "", diags
	} else if err != nil {
		diags = diags.Append(fmt.Errorf("Failed to retrieve user account details: %s", err))
		return "", diags
	}
	c.Ui.Output(fmt.Sprintf(c.Colorize().Color("\nRetrieved token for user [bold]%s[reset]\n"), user.Username))

	return svcauth.HostCredentialsToken(token), nil
}

func (c *LoginCommand) interactiveContextConsent(hostname svchost.Hostname, grantType disco.OAuthGrantType, credsCtx *loginCredentialsContext) (bool, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	mechanism := "OAuth"
	if grantType == "" {
		mechanism = "your browser"
	}

	c.Ui.Output(fmt.Sprintf("Terraform will request an API token for %s using %s.\n", hostname.ForDisplay(), mechanism))

	if grantType.UsesAuthorizationEndpoint() {
		c.Ui.Output(
			"This will work only if you are able to use a web browser on this computer to\ncomplete a login process. If not, you must obtain an API token by another\nmeans and configure it in the CLI configuration manually.\n",
		)
	}

	// credsCtx might not be set if we're using a mock credentials source
	// in a test, but it should always be set in normal use.
	if credsCtx != nil {
		switch credsCtx.Location {
		case cliconfig.CredentialsViaHelper:
			c.Ui.Output(fmt.Sprintf("If login is successful, Terraform will store the token in the configured\n%q credentials helper for use by subsequent commands.\n", credsCtx.HelperType))
		case cliconfig.CredentialsInPrimaryFile, cliconfig.CredentialsNotAvailable:
			c.Ui.Output(fmt.Sprintf("If login is successful, Terraform will store the token in plain text in\nthe following file for use by subsequent commands:\n    %s\n", credsCtx.LocalFilename))
		}
	}

	v, err := c.UIInput().Input(context.Background(), &terraform.InputOpts{
		Id:          "approve",
		Query:       "Do you want to proceed?",
		Description: `Only 'yes' will be accepted to confirm.`,
	})
	if err != nil {
		// Should not happen because this command checks that input is enabled
		// before we get to this point.
		diags = diags.Append(err)
		return false, diags
	}

	return strings.ToLower(v) == "yes", diags
}

func (c *LoginCommand) listenerForCallback(minPort, maxPort uint16) (net.Listener, string, error) {
	if minPort < 1024 || maxPort < 1024 {
		// This should never happen because it should've been checked by
		// the svchost/disco package when reading the service description,
		// but we'll prefer to fail hard rather than inadvertently trying
		// to open an unprivileged port if there are bugs at that layer.
		panic("listenerForCallback called with privileged port number")
	}

	availCount := int(maxPort) - int(minPort)

	// We're going to try port numbers within the range at random, so we need
	// to terminate eventually in case _none_ of the ports are available.
	// We'll make that 150% of the number of ports just to give us some room
	// for the random number generator to generate the same port more than
	// once.
	// Note that we don't really care about true randomness here... we're just
	// trying to hop around in the available port space rather than always
	// working up from the lowest, because we have no information to predict
	// that any particular number will be more likely to be available than
	// another.
	maxTries := availCount + (availCount / 2)

	for tries := 0; tries < maxTries; tries++ {
		port := rand.Intn(availCount) + int(minPort)
		addr := fmt.Sprintf("127.0.0.1:%d", port)
		log.Printf("[TRACE] login: trying %s as a listen address for temporary OAuth callback server", addr)
		l, err := net.Listen("tcp4", addr)
		if err == nil {
			// We use a path that doesn't end in a slash here because some
			// OAuth server implementations don't allow callback URLs to
			// end with slashes.
			callbackURL := fmt.Sprintf("http://localhost:%d/login", port)
			log.Printf("[TRACE] login: callback URL will be %s", callbackURL)
			return l, callbackURL, nil
		}
	}

	return nil, "", fmt.Errorf("no suitable TCP ports (between %d and %d) are available for the temporary OAuth callback server", minPort, maxPort)
}

func (c *LoginCommand) proofKey() (key, challenge string, err error) {
	// Wel use a UUID-like string as the "proof key for code exchange" (PKCE)
	// that will eventually authenticate our request to the token endpoint.
	// Standard UUIDs are explicitly not suitable as secrets according to the
	// UUID spec, but our go-uuid just generates totally random number sequences
	// formatted in the conventional UUID syntax, so that concern does not
	// apply here: this is just a 128-bit crypto-random number.
	uu, err := uuid.GenerateUUID()
	if err != nil {
		return "", "", err
	}

	key = fmt.Sprintf("%s.%09d", uu, rand.Intn(999999999))

	h := sha256.New()
	h.Write([]byte(key))
	challenge = base64.RawURLEncoding.EncodeToString(h.Sum(nil))

	return key, challenge, nil
}

type loginCredentialsContext struct {
	Location      cliconfig.CredentialsLocation
	LocalFilename string
	HelperType    string
}

const callbackSuccessMessage = `
<html>
<head>
<title>Terraform Login</title>
<style type="text/css">
body {
	font-family: monospace;
	color: #fff;
	background-color: #000;
}
</style>
</head>
<body>

<p>The login server has returned an authentication code to Terraform.</p>
<p>Now close this page and return to the terminal where <tt>terraform login</tt>
is running to see the result of the login process.</p>

</body>
</html>
`
