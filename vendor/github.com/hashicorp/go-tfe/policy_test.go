package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoliciesList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	pTest1, _ := createPolicy(t, client, orgTest)
	pTest2, _ := createPolicy(t, client, orgTest)

	t.Run("without list options", func(t *testing.T) {
		pl, err := client.Policies.List(ctx, orgTest.Name, PolicyListOptions{})
		require.NoError(t, err)
		assert.Contains(t, pl.Items, pTest1)
		assert.Contains(t, pl.Items, pTest2)

		assert.Equal(t, 1, pl.CurrentPage)
		assert.Equal(t, 2, pl.TotalCount)
	})

	t.Run("with pagination", func(t *testing.T) {
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		pl, err := client.Policies.List(ctx, orgTest.Name, PolicyListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)

		assert.Empty(t, pl.Items)
		assert.Equal(t, 999, pl.CurrentPage)
		assert.Equal(t, 2, pl.TotalCount)
	})

	t.Run("with search", func(t *testing.T) {
		// Search by one of the policy's names; we should get only that policy
		// and pagination data should reflect the search as well
		pl, err := client.Policies.List(ctx, orgTest.Name, PolicyListOptions{
			Search: &pTest1.Name,
		})
		require.NoError(t, err)

		assert.Contains(t, pl.Items, pTest1)
		assert.NotContains(t, pl.Items, pTest2)
		assert.Equal(t, 1, pl.CurrentPage)
		assert.Equal(t, 1, pl.TotalCount)
	})

	t.Run("without a valid organization", func(t *testing.T) {
		ps, err := client.Policies.List(ctx, badIdentifier, PolicyListOptions{})
		assert.Nil(t, ps)
		assert.EqualError(t, err, "invalid value for organization")
	})
}

func TestPoliciesCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		name := randomString(t)
		options := PolicyCreateOptions{
			Name:        String(name),
			Description: String("A sample policy"),
			Enforce: []*EnforcementOptions{
				{
					Path: String(name + ".sentinel"),
					Mode: EnforcementMode(EnforcementSoft),
				},
			},
		}

		p, err := client.Policies.Create(ctx, orgTest.Name, options)
		require.NoError(t, err)

		// Get a refreshed view from the API.
		refreshed, err := client.Policies.Read(ctx, p.ID)
		require.NoError(t, err)

		for _, item := range []*Policy{
			p,
			refreshed,
		} {
			assert.NotEmpty(t, item.ID)
			assert.Equal(t, *options.Name, item.Name)
			assert.Equal(t, *options.Description, item.Description)
		}
	})

	t.Run("when options has an invalid name", func(t *testing.T) {
		p, err := client.Policies.Create(ctx, orgTest.Name, PolicyCreateOptions{
			Name: String(badIdentifier),
			Enforce: []*EnforcementOptions{
				{
					Path: String(badIdentifier + ".sentinel"),
					Mode: EnforcementMode(EnforcementSoft),
				},
			},
		})
		assert.Nil(t, p)
		assert.EqualError(t, err, "invalid value for name")
	})

	t.Run("when options is missing name", func(t *testing.T) {
		p, err := client.Policies.Create(ctx, orgTest.Name, PolicyCreateOptions{
			Enforce: []*EnforcementOptions{
				{
					Path: String(randomString(t) + ".sentinel"),
					Mode: EnforcementMode(EnforcementSoft),
				},
			},
		})
		assert.Nil(t, p)
		assert.EqualError(t, err, "name is required")
	})

	t.Run("when options is missing an enforcement", func(t *testing.T) {
		options := PolicyCreateOptions{
			Name: String(randomString(t)),
		}

		p, err := client.Policies.Create(ctx, orgTest.Name, options)
		assert.Nil(t, p)
		assert.EqualError(t, err, "enforce is required")
	})

	t.Run("when options is missing enforcement path", func(t *testing.T) {
		options := PolicyCreateOptions{
			Name: String(randomString(t)),
			Enforce: []*EnforcementOptions{
				{
					Mode: EnforcementMode(EnforcementSoft),
				},
			},
		}

		p, err := client.Policies.Create(ctx, orgTest.Name, options)
		assert.Nil(t, p)
		assert.EqualError(t, err, "enforcement path is required")
	})

	t.Run("when options is missing enforcement path", func(t *testing.T) {
		name := randomString(t)
		options := PolicyCreateOptions{
			Name: String(name),
			Enforce: []*EnforcementOptions{
				{
					Path: String(name + ".sentinel"),
				},
			},
		}

		p, err := client.Policies.Create(ctx, orgTest.Name, options)
		assert.Nil(t, p)
		assert.EqualError(t, err, "enforcement mode is required")
	})

	t.Run("when options has an invalid organization", func(t *testing.T) {
		p, err := client.Policies.Create(ctx, badIdentifier, PolicyCreateOptions{
			Name: String("foo"),
		})
		assert.Nil(t, p)
		assert.EqualError(t, err, "invalid value for organization")
	})
}

func TestPoliciesRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	pTest, pTestCleanup := createPolicy(t, client, orgTest)
	defer pTestCleanup()

	t.Run("when the policy exists without content", func(t *testing.T) {
		p, err := client.Policies.Read(ctx, pTest.ID)
		require.NoError(t, err)

		assert.Equal(t, pTest.ID, p.ID)
		assert.Equal(t, pTest.Name, p.Name)
		assert.Equal(t, pTest.PolicySetCount, p.PolicySetCount)
		assert.Empty(t, p.Enforce)
		assert.Equal(t, pTest.Organization.Name, p.Organization.Name)
	})

	err := client.Policies.Upload(ctx, pTest.ID, []byte(`main = rule { true }`))
	require.NoError(t, err)

	t.Run("when the policy exists with content", func(t *testing.T) {
		p, err := client.Policies.Read(ctx, pTest.ID)
		require.NoError(t, err)

		assert.Equal(t, pTest.ID, p.ID)
		assert.Equal(t, pTest.Name, p.Name)
		assert.Equal(t, pTest.Description, p.Description)
		assert.Equal(t, pTest.PolicySetCount, p.PolicySetCount)
		assert.NotEmpty(t, p.Enforce)
		assert.Equal(t, pTest.Organization.Name, p.Organization.Name)
	})

	t.Run("when the policy does not exist", func(t *testing.T) {
		p, err := client.Policies.Read(ctx, "nonexisting")
		assert.Nil(t, p)
		assert.Equal(t, ErrResourceNotFound, err)
	})

	t.Run("without a valid policy ID", func(t *testing.T) {
		p, err := client.Policies.Read(ctx, badIdentifier)
		assert.Nil(t, p)
		assert.EqualError(t, err, "invalid value for policy ID")
	})
}

func TestPoliciesUpdate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	t.Run("when updating with an existing path", func(t *testing.T) {
		pBefore, pBeforeCleanup := createUploadedPolicy(t, client, true, orgTest)
		defer pBeforeCleanup()

		require.Equal(t, 1, len(pBefore.Enforce))

		pAfter, err := client.Policies.Update(ctx, pBefore.ID, PolicyUpdateOptions{
			Enforce: []*EnforcementOptions{
				{
					Path: String(pBefore.Enforce[0].Path),
					Mode: EnforcementMode(EnforcementAdvisory),
				},
			},
		})
		require.NoError(t, err)
		require.Equal(t, 1, len(pAfter.Enforce))

		assert.Equal(t, pBefore.ID, pAfter.ID)
		assert.Equal(t, pBefore.Name, pAfter.Name)
		assert.Equal(t, pBefore.Description, pAfter.Description)
		assert.Equal(t, pBefore.Enforce[0].Path, pAfter.Enforce[0].Path)
		assert.Equal(t, EnforcementAdvisory, pAfter.Enforce[0].Mode)
	})

	t.Run("when updating with a nonexisting path", func(t *testing.T) {
		pBefore, pBeforeCleanup := createUploadedPolicy(t, client, true, orgTest)
		defer pBeforeCleanup()

		require.Equal(t, 1, len(pBefore.Enforce))

		pAfter, err := client.Policies.Update(ctx, pBefore.ID, PolicyUpdateOptions{
			Enforce: []*EnforcementOptions{
				{
					Path: String("nonexisting"),
					Mode: EnforcementMode(EnforcementAdvisory),
				},
			},
		})
		require.NoError(t, err)

		// Weirdly enough this is not equal as updating a nonexisting path
		// causes the enforce mode to reset to the default hard-mandatory
		t.Skip("see comment...")
		assert.Equal(t, pBefore, pAfter)
	})

	t.Run("with a new description", func(t *testing.T) {
		pBefore, pBeforeCleanup := createUploadedPolicy(t, client, true, orgTest)
		defer pBeforeCleanup()

		pAfter, err := client.Policies.Update(ctx, pBefore.ID, PolicyUpdateOptions{
			Description: String("A brand new description"),
		})
		require.NoError(t, err)

		assert.Equal(t, pBefore.Name, pAfter.Name)
		assert.Equal(t, pBefore.Enforce, pAfter.Enforce)
		assert.NotEqual(t, pBefore.Description, pAfter.Description)
		assert.Equal(t, "A brand new description", pAfter.Description)
	})

	t.Run("without a valid policy ID", func(t *testing.T) {
		p, err := client.Policies.Update(ctx, badIdentifier, PolicyUpdateOptions{})
		assert.Nil(t, p)
		assert.EqualError(t, err, "invalid value for policy ID")
	})
}

func TestPoliciesDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	pTest, _ := createPolicy(t, client, orgTest)

	t.Run("with valid options", func(t *testing.T) {
		err := client.Policies.Delete(ctx, pTest.ID)
		require.NoError(t, err)

		// Try loading the policy - it should fail.
		_, err = client.Policies.Read(ctx, pTest.ID)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the policy does not exist", func(t *testing.T) {
		err := client.Policies.Delete(ctx, pTest.ID)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the policy ID is invalid", func(t *testing.T) {
		err := client.Policies.Delete(ctx, badIdentifier)
		assert.EqualError(t, err, "invalid value for policy ID")
	})
}

func TestPoliciesUpload(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	pTest, pTestCleanup := createPolicy(t, client, nil)
	defer pTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		err := client.Policies.Upload(ctx, pTest.ID, []byte(`main = rule { true }`))
		assert.NoError(t, err)
	})

	t.Run("with empty content", func(t *testing.T) {
		err := client.Policies.Upload(ctx, pTest.ID, []byte{})
		assert.NoError(t, err)
	})

	t.Run("without any content", func(t *testing.T) {
		err := client.Policies.Upload(ctx, pTest.ID, nil)
		assert.NoError(t, err)
	})

	t.Run("without a valid policy ID", func(t *testing.T) {
		err := client.Policies.Upload(ctx, badIdentifier, []byte(`main = rule { true }`))
		assert.EqualError(t, err, "invalid value for policy ID")
	})
}

func TestPoliciesDownload(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	pTest, pTestCleanup := createPolicy(t, client, nil)
	defer pTestCleanup()

	testContent := []byte(`main = rule { true }`)

	t.Run("without existing content", func(t *testing.T) {
		content, err := client.Policies.Download(ctx, pTest.ID)
		assert.Equal(t, ErrResourceNotFound, err)
		assert.Nil(t, content)
	})

	t.Run("with valid options", func(t *testing.T) {
		err := client.Policies.Upload(ctx, pTest.ID, testContent)
		require.NoError(t, err)

		content, err := client.Policies.Download(ctx, pTest.ID)
		assert.NoError(t, err)
		assert.Equal(t, testContent, content)
	})

	t.Run("without a valid policy ID", func(t *testing.T) {
		content, err := client.Policies.Download(ctx, badIdentifier)
		assert.EqualError(t, err, "invalid value for policy ID")
		assert.Nil(t, content)
	})
}
