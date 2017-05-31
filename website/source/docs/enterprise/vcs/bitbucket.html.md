---
layout: "enterprise"
page_title: "Bitbucket Cloud - VCS Integrations - Terraform Enterprise"
sidebar_current: "docs-enterprise-vcs-bitbucket-cloud"
description: |-
  Bitbucket Cloud repositories can be integrated with Terraform Enterprise by using push command.
---
# Bitbucket Cloud

Bitbucket Cloud can be used to import Terraform configuration, automatically
queuing runs when changes are merged into a repository's default branch.
Additionally, plans are run when a pull request is created or updated. Terraform
Enterprise will update the pull request with the result of the Terraform plan
providing quick feedback on proposed changes.

## Registering an OAuth Application & Client

### Creating and Updating a Bitbucket Cloud OAuth Application

You will need to register Terraform Enterprise as an OAuth Application within your Bitbucket Cloud account. Proceed to https://bitbucket.org/account/user/your-username/oauth-consumers/new. Fill out the required information and set the Redirect URI to a placeholder (ie: http://example.com), as you will need to register the Bitbucket Client with Terraform Enterprise prior to receiving this value. Check all of the permission fields that apply to you, and click Save

Upon saving the application, you will be redirected to https://bitbucket.org/account/user/your-username/api. Scroll down to OAuth Consumers and click on the application you just created. Copy the Key and Secret. Leave this tab open in your browser as you will need to return to it in a moment.

### Creating a Terraform Enterprise OAuth Client

In a new tab, navigate to https://atlas.hashicorp.com/settings and, in the left-side panel, select the organization that you’d like to administer your Bitbucket connection, then click on “configuration” in the left-side panel.

Within the “Add OAuthClient” pane, select Bitbucket Cloud and fill in your application key and secret. In the base url field, enter the root url of your Bitbucket instance (i.e. https://bitbucket.org). In the API url field, enter the base api url (i.e. https://api.bitbucket.org/2.0). Create the OAuth client.

Once you have created your client, you will be redirected back to the configurations page for your chosen organization. On that page, find the “OAuth Clients” pane and copy the Callback URL for your Bitbucket Cloud OAuth Client. In the open Bitbucket tab, select the Terraform Enterprise OAuth Application and click edit. Enter the Callback URL you just copied in the field labeled Redirect URI. Save the application.

Your OAuth Client should now be enabled for your Organization to use within Terraform Enterprise.

## Using Terraform Enterprise with Bitbucket Cloud

There are two ways to connect your preferred VCS Host to Terraform Enterprise.
You can generate an OAuth token both at the user and organization level.

### Linking your Terraform Enterprise Organization

Return to the settings page for the organization in which you created the OAuth Client (https://atlas.hashicorp.com/settings/organizations/your-organization/configuration). Find the section entitled Organization Connections to OAuth Client and click connect beneath your Bitbucket Cloud integration. You will be briefly redirected to Bitbucket in order to authenticate the client.

Once you are redirected back to Terraform Enterprise, you should see that the token was created with a unique identifier. If you don’t, check the values in your OAuth Client and make sure they match exactly with the values associated with your Bitbucket OAuth Application. There is also an option to destroy the token and disconnect the organization from your Bitbucket installation.

You are now ready to use your organization's token to manage builds and configurations within Terraform Enterprise.

### Linking your Terraform Enterprise User Account

Navigate to https://atlas.hashicorp.com/settings/connections and click on “Connect Bitbucket Cloud to Atlas”. You will briefly be redirected to Bitbucket in order to authenticate your OAuth Client. Once redirected back to Terraform Enterprise, You should see a green flash banner with the message: "Successfully Linked to Bitbucket".

You are now ready to use your personal token to manage builds and configurations within Terraform Enterprise.

## Connecting Configurations

Once you have linked a Bitbucket installation to your account or organization,
you are ready to begin creating Packer Builds and Terraform Environments linked
to your desired Bitbucket Cloud repository.

Terraform Enterprise environments are linked to individual GitHub repositories.
However, a single GitHub repository can be linked to multiple environments
allowing a single set of Terraform configuration to be used across multiple
environments.

Environments can be linked when they're initially created using the New
Environment process. Existing environments can be linked by setting GitHub
details in their **Integrations**.

To link a Terraform Enterprise environment to a Bitbucket Cloud repository, you need
three pieces of information:

- **Bitbucket Cloud repository** - The location of the repository being imported in the
format _username/repository_.

- **Bitbucket Cloud branch** - The branch from which to ingress new versions. This
defaults to the value GitHub provides as the default branch for this repository.

- **Path to directory of Terraform files** - The repository's subdirectory that
contains its terraform files. This defaults to the root of the repository.

### Connecting a Bitbucket Cloud Repository to a Terraform Environment

Navigate to https://atlas.hashicorp.com/configurations/import and select Link to Bitbucket Cloud. A menu will appear asking you to name the environment. Then use the autocomplete field for repository and select the repository for which you'd like to create a webhook & environment. If necessary, fill out information about the VCS branch to pull from as well as the directory where the Terraform files live within the repository. Click Create and Continue.

Upon success, you will be redirected to the environment's runs page (https://atlas.hashicorp.com/terraform/your-organization/environments/your-environment/changes/runs). A message will display letting you know that the repository is ingressing from Bitbucket and once finished you will be able to Queue, Run, & Apply a Terraform Plan. Depending on your webhook settings, changes will be triggered through git events on the specified branch.

The events currently supported are repository and branch push, pull request, and merge.

### Connecting a Bitbucket Cloud Repository to a Packer Build Configuration

Navigate to https://atlas.hashicorp.com/builds/new and select the organization for which you'd like to create a build configuration. Name your build & select Connect build configuration to a Git Repository. A form will appear asking you to select your Git Host. Select Bitbucket Cloud.

Choose the repository for which you'd like to create a webhook. Fill out any other information in the form such as preferred branch to build from (your default branch will be selected should this field be left blank), Packer directory, and Packer Template.

Upon clicking Create you will be redirected to the build configuration (https://atlas.hashicorp.com/packer/your-organization/build-configurations/your-build-configuration). On this page, you will have the opportunity to make any changes to your packer template, push changes via the CLI, or manually queue a Packer build.

Depending on your webhook settings, changes will be triggered through git events on the specified branch. The events currently supported are repository and branch push, pull request, and merge.
