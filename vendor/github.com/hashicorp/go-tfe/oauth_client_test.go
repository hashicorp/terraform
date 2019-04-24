package tfe

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOAuthClientsList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	ocTest1, _ := createOAuthClient(t, client, orgTest)
	ocTest2, _ := createOAuthClient(t, client, orgTest)

	t.Run("without list options", func(t *testing.T) {
		options := OAuthClientListOptions{}

		ocl, err := client.OAuthClients.List(ctx, orgTest.Name, options)
		require.NoError(t, err)

		t.Run("the OAuth tokens relationship is decoded correcly", func(t *testing.T) {
			for _, oc := range ocl.Items {
				assert.Equal(t, 1, len(oc.OAuthTokens))
			}
		})

		// We need to strip some fields before the next test.
		for _, oc := range append(ocl.Items, ocTest1, ocTest2) {
			oc.OAuthTokens = nil
			oc.Organization = nil
		}

		assert.Contains(t, ocl.Items, ocTest1)
		assert.Contains(t, ocl.Items, ocTest2)

		t.Skip("paging not supported yet in API")
		assert.Equal(t, 1, ocl.CurrentPage)
		assert.Equal(t, 2, ocl.TotalCount)
	})

	t.Run("with list options", func(t *testing.T) {
		t.Skip("paging not supported yet in API")
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		options := OAuthClientListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		}

		ocl, err := client.OAuthClients.List(ctx, orgTest.Name, options)
		require.NoError(t, err)
		assert.Empty(t, ocl.Items)
		assert.Equal(t, 999, ocl.CurrentPage)
		assert.Equal(t, 2, ocl.TotalCount)
	})

	t.Run("without a valid organization", func(t *testing.T) {
		options := OAuthClientListOptions{}

		ocl, err := client.OAuthClients.List(ctx, badIdentifier, options)
		assert.Nil(t, ocl)
		assert.EqualError(t, err, "invalid value for organization")
	})
}

func TestOAuthClientsCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		t.Fatal("Export a valid GITHUB_TOKEN before running this test!")
	}

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		options := OAuthClientCreateOptions{
			APIURL:          String("https://api.github.com"),
			HTTPURL:         String("https://github.com"),
			OAuthToken:      String(githubToken),
			ServiceProvider: ServiceProvider(ServiceProviderGithub),
		}

		oc, err := client.OAuthClients.Create(ctx, orgTest.Name, options)
		assert.NoError(t, err)
		assert.NotEmpty(t, oc.ID)
		assert.Equal(t, "https://api.github.com", oc.APIURL)
		assert.Equal(t, "https://github.com", oc.HTTPURL)
		assert.Equal(t, 1, len(oc.OAuthTokens))
		assert.Equal(t, ServiceProviderGithub, oc.ServiceProvider)

		t.Run("the organization relationship is decoded correcly", func(t *testing.T) {
			assert.NotEmpty(t, oc.Organization)
		})
	})

	t.Run("without an valid organization", func(t *testing.T) {
		options := OAuthClientCreateOptions{
			APIURL:          String("https://api.github.com"),
			HTTPURL:         String("https://github.com"),
			OAuthToken:      String(githubToken),
			ServiceProvider: ServiceProvider(ServiceProviderGithub),
		}

		_, err := client.OAuthClients.Create(ctx, badIdentifier, options)
		assert.EqualError(t, err, "invalid value for organization")
	})

	t.Run("without an API URL", func(t *testing.T) {
		options := OAuthClientCreateOptions{
			HTTPURL:         String("https://github.com"),
			OAuthToken:      String(githubToken),
			ServiceProvider: ServiceProvider(ServiceProviderGithub),
		}

		_, err := client.OAuthClients.Create(ctx, orgTest.Name, options)
		assert.EqualError(t, err, "API URL is required")
	})

	t.Run("without a HTTP URL", func(t *testing.T) {
		options := OAuthClientCreateOptions{
			APIURL:          String("https://api.github.com"),
			OAuthToken:      String(githubToken),
			ServiceProvider: ServiceProvider(ServiceProviderGithub),
		}

		_, err := client.OAuthClients.Create(ctx, orgTest.Name, options)
		assert.EqualError(t, err, "HTTP URL is required")
	})

	t.Run("without an OAuth token", func(t *testing.T) {
		options := OAuthClientCreateOptions{
			APIURL:          String("https://api.github.com"),
			HTTPURL:         String("https://github.com"),
			ServiceProvider: ServiceProvider(ServiceProviderGithub),
		}

		_, err := client.OAuthClients.Create(ctx, orgTest.Name, options)
		assert.EqualError(t, err, "OAuth token is required")
	})

	t.Run("without a service provider", func(t *testing.T) {
		options := OAuthClientCreateOptions{
			APIURL:     String("https://api.github.com"),
			HTTPURL:    String("https://github.com"),
			OAuthToken: String(githubToken),
		}

		_, err := client.OAuthClients.Create(ctx, orgTest.Name, options)
		assert.EqualError(t, err, "service provider is required")
	})
}

func TestOAuthClientsRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	ocTest, ocTestCleanup := createOAuthClient(t, client, nil)
	defer ocTestCleanup()

	t.Run("when the OAuth client exists", func(t *testing.T) {
		oc, err := client.OAuthClients.Read(ctx, ocTest.ID)
		require.NoError(t, err)
		assert.Equal(t, ocTest.ID, oc.ID)
		assert.Equal(t, ocTest.APIURL, oc.APIURL)
		assert.Equal(t, ocTest.CallbackURL, oc.CallbackURL)
		assert.Equal(t, ocTest.ConnectPath, oc.ConnectPath)
		assert.Equal(t, ocTest.HTTPURL, oc.HTTPURL)
		assert.Equal(t, ocTest.ServiceProvider, oc.ServiceProvider)
		assert.Equal(t, ocTest.ServiceProviderName, oc.ServiceProviderName)
		assert.Equal(t, ocTest.OAuthTokens, oc.OAuthTokens)
	})

	t.Run("when the OAuth client does not exist", func(t *testing.T) {
		oc, err := client.OAuthClients.Read(ctx, "nonexisting")
		assert.Nil(t, oc)
		assert.Equal(t, ErrResourceNotFound, err)
	})

	t.Run("without a valid OAuth client ID", func(t *testing.T) {
		oc, err := client.OAuthClients.Read(ctx, badIdentifier)
		assert.Nil(t, oc)
		assert.EqualError(t, err, "invalid value for OAuth client ID")
	})
}

func TestOAuthClientsDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	ocTest, _ := createOAuthClient(t, client, orgTest)

	t.Run("with valid options", func(t *testing.T) {
		err := client.OAuthClients.Delete(ctx, ocTest.ID)
		require.NoError(t, err)

		// Try loading the OAuth client - it should fail.
		_, err = client.OAuthClients.Read(ctx, ocTest.ID)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the OAuth client does not exist", func(t *testing.T) {
		err := client.OAuthClients.Delete(ctx, ocTest.ID)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the OAuth client ID is invalid", func(t *testing.T) {
		err := client.OAuthClients.Delete(ctx, badIdentifier)
		assert.EqualError(t, err, "invalid value for OAuth client ID")
	})
}
