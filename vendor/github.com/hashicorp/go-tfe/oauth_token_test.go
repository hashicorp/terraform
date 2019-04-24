package tfe

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOAuthTokensList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	otTest1, _ := createOAuthToken(t, client, orgTest)
	otTest2, _ := createOAuthToken(t, client, orgTest)

	t.Run("without list options", func(t *testing.T) {
		options := OAuthTokenListOptions{}

		otl, err := client.OAuthTokens.List(ctx, orgTest.Name, options)
		require.NoError(t, err)

		t.Run("the OAuth client relationship is decoded correcly", func(t *testing.T) {
			for _, ot := range otl.Items {
				assert.NotEmpty(t, ot.OAuthClient)
			}
		})

		// We need to strip some fields before the next test.
		for _, ot := range otl.Items {
			ot.CreatedAt = time.Time{}
			ot.ServiceProviderUser = ""
			ot.OAuthClient = nil
		}

		assert.Contains(t, otl.Items, otTest1)
		assert.Contains(t, otl.Items, otTest2)

		t.Skip("paging not supported yet in API")
		assert.Equal(t, 1, otl.CurrentPage)
		assert.Equal(t, 2, otl.TotalCount)
	})

	t.Run("with list options", func(t *testing.T) {
		t.Skip("paging not supported yet in API")
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		options := OAuthTokenListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		}

		otl, err := client.OAuthTokens.List(ctx, orgTest.Name, options)
		require.NoError(t, err)
		assert.Empty(t, otl.Items)
		assert.Equal(t, 999, otl.CurrentPage)
		assert.Equal(t, 2, otl.TotalCount)
	})

	t.Run("without a valid organization", func(t *testing.T) {
		options := OAuthTokenListOptions{}

		otl, err := client.OAuthTokens.List(ctx, badIdentifier, options)
		assert.Nil(t, otl)
		assert.EqualError(t, err, "invalid value for organization")
	})
}

func TestOAuthTokensRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	otTest, otTestCleanup := createOAuthToken(t, client, nil)
	defer otTestCleanup()

	t.Run("when the OAuth token exists", func(t *testing.T) {
		ot, err := client.OAuthTokens.Read(ctx, otTest.ID)
		require.NoError(t, err)
		assert.Equal(t, otTest.ID, ot.ID)
		assert.NotEmpty(t, ot.OAuthClient)
	})

	t.Run("when the OAuth token does not exist", func(t *testing.T) {
		ot, err := client.OAuthTokens.Read(ctx, "nonexisting")
		assert.Nil(t, ot)
		assert.Equal(t, ErrResourceNotFound, err)
	})

	t.Run("without a valid OAuth token ID", func(t *testing.T) {
		ot, err := client.OAuthTokens.Read(ctx, badIdentifier)
		assert.Nil(t, ot)
		assert.EqualError(t, err, "invalid value for OAuth token ID")
	})
}

func TestOAuthTokensUpdate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	otTest, otTestCleanup := createOAuthToken(t, client, nil)
	defer otTestCleanup()

	t.Run("before updating with an SSH key", func(t *testing.T) {
		assert.False(t, otTest.HasSSHKey)
	})

	t.Run("without options", func(t *testing.T) {
		ot, err := client.OAuthTokens.Update(ctx, otTest.ID, OAuthTokenUpdateOptions{})
		require.NoError(t, err)
		assert.False(t, ot.HasSSHKey)
	})

	t.Run("when updating with a valid SSH key", func(t *testing.T) {
		dummyPrivateSSHKey := `-----BEGIN RSA PRIVATE KEY-----
MIICXgIBAAKBgQDIF0s2yX7dSQQL1grdTbai1Mb7sEco6RIOz8iqrHTGqmESpu5n
d8imMkV5KadgVBJ/UvHsWpg446O3DAMYn0Y6f8dDlK7pmCEtiGVKTR1PaVRMpF8R
5Guvrmlru8Kex5ozh0pPMB15aGsIzSezCKgSs74Od9YL4smdgKyYwqsu3wIDAQAB
AoGBAKCs6+4j4icqYgBrMjBCHp4lRWCJTqtQdfrE6jv73o5F9Uu4FwupScwD5HwG
cezNtkjeP3zvxvsv+aCdGcNk60vSz4n9Nt6gEJveWFSpePYXKZ9cz/IjFLI7nSzc
1msLyE3DfUqB91s/A/aT5p0LiVDc8i4mCGDOga2OINIwqDGZAkEA/Vz8dkcqsAVW
CL1F000hWTrM6tu0V+x8Nm8CRx7wM/Gy/19PbV0t26wCVG0GXyLWsV2//huY7w5b
3AcSl5pfJQJBAMosYQXk5L4S+qivz2zmZdtyz+Ik6IZ3PwZoED32PxGSdW5rG8iP
V+iSJek5ESkS1zeXwDMnF4LeoBY9H07DiLMCQQCrHm1o2SIMpl34IxWQ4+wdHuid
yuuf4pn2Db2lGVE0VA8ICXBUtfUuA5vDN6tw/8+vFVmBn1QISVNjZOd6uwl9AkA+
jIRoAm0SsWSDlAEkvBN/VYIjgS+/il0haki8ItdYZGuYgeLSpiaYeb7o7RL2FjIn
rPd12/5WKvJ0buykvbIpAkEA5Uy3T8xQJkDGbp0+xA0yThoOYiB09lAok8I7Sv/5
dpIe8YOINN27XaojJvVpT5uBVCcZLF+G7kaMjSwCTlDx3Q==
-----END RSA PRIVATE KEY-----`

		ot, err := client.OAuthTokens.Update(ctx, otTest.ID, OAuthTokenUpdateOptions{
			PrivateSSHKey: String(dummyPrivateSSHKey),
		})
		require.NoError(t, err)

		assert.Equal(t, otTest.ID, ot.ID)
		assert.True(t, ot.HasSSHKey)
	})

	t.Run("when updating with an invalid SSH key", func(t *testing.T) {
		ot, err := client.OAuthTokens.Update(ctx, otTest.ID, OAuthTokenUpdateOptions{
			PrivateSSHKey: String(randomString(t)),
		})
		assert.Nil(t, ot)
		assert.Contains(t, err.Error(), "Ssh key is invalid")
	})

	t.Run("without a valid policy ID", func(t *testing.T) {
		ot, err := client.OAuthTokens.Update(ctx, badIdentifier, OAuthTokenUpdateOptions{})
		assert.Nil(t, ot)
		assert.EqualError(t, err, "invalid value for OAuth token ID")
	})
}

func TestOAuthTokensDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	otTest, _ := createOAuthToken(t, client, orgTest)

	t.Run("with valid options", func(t *testing.T) {
		err := client.OAuthTokens.Delete(ctx, otTest.ID)
		require.NoError(t, err)

		// Try loading the OAuth token - it should fail.
		_, err = client.OAuthTokens.Read(ctx, otTest.ID)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the OAuth token does not exist", func(t *testing.T) {
		err := client.OAuthTokens.Delete(ctx, otTest.ID)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the OAuth token ID is invalid", func(t *testing.T) {
		err := client.OAuthTokens.Delete(ctx, badIdentifier)
		assert.EqualError(t, err, "invalid value for OAuth token ID")
	})
}
