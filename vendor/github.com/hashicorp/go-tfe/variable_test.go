package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVariablesList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, _ := createWorkspace(t, client, orgTest)

	vTest1, _ := createVariable(t, client, wTest)
	vTest2, _ := createVariable(t, client, wTest)

	t.Run("without list options", func(t *testing.T) {
		vl, err := client.Variables.List(ctx, VariableListOptions{
			Organization: String(orgTest.Name),
			Workspace:    String(wTest.Name),
		})
		require.NoError(t, err)
		assert.Contains(t, vl.Items, vTest1)
		assert.Contains(t, vl.Items, vTest2)

		t.Skip("paging not supported yet in API")
		assert.Equal(t, 1, vl.CurrentPage)
		assert.Equal(t, 2, vl.TotalCount)
	})

	t.Run("with list options", func(t *testing.T) {
		t.Skip("paging not supported yet in API")
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		vl, err := client.Variables.List(ctx, VariableListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
			Organization: String(orgTest.Name),
			Workspace:    String(wTest.Name),
		})
		require.NoError(t, err)
		assert.Empty(t, vl.Items)
		assert.Equal(t, 999, vl.CurrentPage)
		assert.Equal(t, 2, vl.TotalCount)
	})

	t.Run("when options is missing an organization", func(t *testing.T) {
		vl, err := client.Variables.List(ctx, VariableListOptions{
			Workspace: String(wTest.Name),
		})
		assert.Nil(t, vl)
		assert.EqualError(t, err, "organization is required")
	})

	t.Run("when options is missing an workspace", func(t *testing.T) {
		vl, err := client.Variables.List(ctx, VariableListOptions{
			Organization: String(orgTest.Name),
		})
		assert.Nil(t, vl)
		assert.EqualError(t, err, "workspace is required")
	})
}

func TestVariablesCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		options := VariableCreateOptions{
			Key:       String(randomString(t)),
			Value:     String(randomString(t)),
			Category:  Category(CategoryTerraform),
			Workspace: wTest,
		}

		v, err := client.Variables.Create(ctx, options)
		require.NoError(t, err)

		assert.NotEmpty(t, v.ID)
		assert.Equal(t, *options.Key, v.Key)
		assert.Equal(t, *options.Value, v.Value)
		assert.Equal(t, *options.Category, v.Category)
		// The workspace isn't returned correcly by the API.
		// assert.Equal(t, *options.Workspace, v.Workspace)
	})

	t.Run("when options has an empty string value", func(t *testing.T) {
		options := VariableCreateOptions{
			Key:       String(randomString(t)),
			Value:     String(""),
			Category:  Category(CategoryTerraform),
			Workspace: wTest,
		}

		v, err := client.Variables.Create(ctx, options)
		require.NoError(t, err)

		assert.NotEmpty(t, v.ID)
		assert.Equal(t, *options.Key, v.Key)
		assert.Equal(t, *options.Value, v.Value)
		assert.Equal(t, *options.Category, v.Category)
	})

	t.Run("when options is missing value", func(t *testing.T) {
		options := VariableCreateOptions{
			Key:       String(randomString(t)),
			Category:  Category(CategoryTerraform),
			Workspace: wTest,
		}

		v, err := client.Variables.Create(ctx, options)
		require.NoError(t, err)

		assert.NotEmpty(t, v.ID)
		assert.Equal(t, *options.Key, v.Key)
		assert.Equal(t, "", v.Value)
		assert.Equal(t, *options.Category, v.Category)
	})

	t.Run("when options is missing key", func(t *testing.T) {
		options := VariableCreateOptions{
			Value:     String(randomString(t)),
			Category:  Category(CategoryTerraform),
			Workspace: wTest,
		}

		_, err := client.Variables.Create(ctx, options)
		assert.EqualError(t, err, "key is required")
	})

	t.Run("when options has an empty key", func(t *testing.T) {
		options := VariableCreateOptions{
			Key:       String(""),
			Value:     String(randomString(t)),
			Category:  Category(CategoryTerraform),
			Workspace: wTest,
		}

		_, err := client.Variables.Create(ctx, options)
		assert.EqualError(t, err, "key is required")
	})

	t.Run("when options is missing category", func(t *testing.T) {
		options := VariableCreateOptions{
			Key:       String(randomString(t)),
			Value:     String(randomString(t)),
			Workspace: wTest,
		}

		_, err := client.Variables.Create(ctx, options)
		assert.EqualError(t, err, "category is required")
	})

	t.Run("when options is missing workspace", func(t *testing.T) {
		options := VariableCreateOptions{
			Key:      String(randomString(t)),
			Value:    String(randomString(t)),
			Category: Category(CategoryTerraform),
		}

		_, err := client.Variables.Create(ctx, options)
		assert.EqualError(t, err, "workspace is required")
	})
}

func TestVariablesRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	vTest, vTestCleanup := createVariable(t, client, nil)
	defer vTestCleanup()

	t.Run("when the variable exists", func(t *testing.T) {
		v, err := client.Variables.Read(ctx, vTest.ID)
		require.NoError(t, err)
		assert.Equal(t, vTest.ID, v.ID)
		assert.Equal(t, vTest.Category, v.Category)
		assert.Equal(t, vTest.HCL, v.HCL)
		assert.Equal(t, vTest.Key, v.Key)
		assert.Equal(t, vTest.Sensitive, v.Sensitive)
		assert.Equal(t, vTest.Value, v.Value)
	})

	t.Run("when the variable does not exist", func(t *testing.T) {
		v, err := client.Variables.Read(ctx, "nonexisting")
		assert.Nil(t, v)
		assert.Equal(t, ErrResourceNotFound, err)
	})

	t.Run("without a valid variable ID", func(t *testing.T) {
		v, err := client.Variables.Read(ctx, badIdentifier)
		assert.Nil(t, v)
		assert.EqualError(t, err, "invalid value for variable ID")
	})
}

func TestVariablesUpdate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	vTest, vTestCleanup := createVariable(t, client, nil)
	defer vTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		options := VariableUpdateOptions{
			Key:   String("newname"),
			Value: String("newvalue"),
			HCL:   Bool(true),
		}

		v, err := client.Variables.Update(ctx, vTest.ID, options)
		require.NoError(t, err)

		assert.Equal(t, *options.Key, v.Key)
		assert.Equal(t, *options.HCL, v.HCL)
		assert.Equal(t, *options.Value, v.Value)
	})

	t.Run("when updating a subset of values", func(t *testing.T) {
		options := VariableUpdateOptions{
			Key: String("someothername"),
			HCL: Bool(false),
		}

		v, err := client.Variables.Update(ctx, vTest.ID, options)
		require.NoError(t, err)

		assert.Equal(t, *options.Key, v.Key)
		assert.Equal(t, *options.HCL, v.HCL)
	})

	t.Run("with sensitive set", func(t *testing.T) {
		options := VariableUpdateOptions{
			Sensitive: Bool(true),
		}

		v, err := client.Variables.Update(ctx, vTest.ID, options)
		require.NoError(t, err)

		assert.Equal(t, *options.Sensitive, v.Sensitive)
		assert.Empty(t, v.Value) // Because its now sensitive
	})

	t.Run("without any changes", func(t *testing.T) {
		vTest, vTestCleanup := createVariable(t, client, nil)
		defer vTestCleanup()

		v, err := client.Variables.Update(ctx, vTest.ID, VariableUpdateOptions{})
		require.NoError(t, err)

		assert.Equal(t, vTest, v)
	})

	t.Run("with invalid variable ID", func(t *testing.T) {
		_, err := client.Variables.Update(ctx, badIdentifier, VariableUpdateOptions{})
		assert.EqualError(t, err, "invalid value for variable ID")
	})
}

func TestVariablesDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	vTest, _ := createVariable(t, client, wTest)

	t.Run("with valid options", func(t *testing.T) {
		err := client.Variables.Delete(ctx, vTest.ID)
		assert.NoError(t, err)
	})

	t.Run("with non existing variable ID", func(t *testing.T) {
		err := client.Variables.Delete(ctx, "nonexisting")
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("with invalid variable ID", func(t *testing.T) {
		err := client.Variables.Delete(ctx, badIdentifier)
		assert.EqualError(t, err, "invalid value for variable ID")
	})
}
