package tfe

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigurationVersionsList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	cvTest1, cvTest1Cleanup := createConfigurationVersion(t, client, wTest)
	defer cvTest1Cleanup()
	cvTest2, cvTest2Cleanup := createConfigurationVersion(t, client, wTest)
	defer cvTest2Cleanup()

	t.Run("without list options", func(t *testing.T) {
		options := ConfigurationVersionListOptions{}

		cvl, err := client.ConfigurationVersions.List(ctx, wTest.ID, options)
		require.NoError(t, err)

		// We need to strip the upload URL as that is a dynamic link.
		cvTest1.UploadURL = ""
		cvTest2.UploadURL = ""

		// And for the retrieved configuration versions as well.
		for _, cv := range cvl.Items {
			cv.UploadURL = ""
		}

		assert.Contains(t, cvl.Items, cvTest1)
		assert.Contains(t, cvl.Items, cvTest2)
		assert.Equal(t, 1, cvl.CurrentPage)
		assert.Equal(t, 2, cvl.TotalCount)
	})

	t.Run("with list options", func(t *testing.T) {
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		options := ConfigurationVersionListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		}

		cvl, err := client.ConfigurationVersions.List(ctx, wTest.ID, options)
		require.NoError(t, err)
		assert.Empty(t, cvl.Items)
		assert.Equal(t, 999, cvl.CurrentPage)
		assert.Equal(t, 2, cvl.TotalCount)
	})

	t.Run("without a valid organization", func(t *testing.T) {
		options := ConfigurationVersionListOptions{}

		cvl, err := client.ConfigurationVersions.List(ctx, badIdentifier, options)
		assert.Nil(t, cvl)
		assert.EqualError(t, err, "invalid value for workspace ID")
	})
}

func TestConfigurationVersionsCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		cv, err := client.ConfigurationVersions.Create(ctx,
			wTest.ID,
			ConfigurationVersionCreateOptions{},
		)
		require.NoError(t, err)

		// Get a refreshed view of the configuration version.
		refreshed, err := client.ConfigurationVersions.Read(ctx, cv.ID)
		require.NoError(t, err)

		for _, item := range []*ConfigurationVersion{
			cv,
			refreshed,
		} {
			assert.NotEmpty(t, item.ID)
			assert.Empty(t, item.Error)
			assert.Equal(t, item.Source, ConfigurationSourceAPI)
			assert.Equal(t, item.Status, ConfigurationPending)
			assert.NotEmpty(t, item.UploadURL)
		}
	})

	t.Run("with invalid workspace id", func(t *testing.T) {
		cv, err := client.ConfigurationVersions.Create(
			ctx,
			badIdentifier,
			ConfigurationVersionCreateOptions{},
		)
		assert.Nil(t, cv)
		assert.EqualError(t, err, "invalid value for workspace ID")
	})
}

func TestConfigurationVersionsRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	cvTest, cvTestCleanup := createConfigurationVersion(t, client, nil)
	defer cvTestCleanup()

	t.Run("when the configuration version exists", func(t *testing.T) {
		cv, err := client.ConfigurationVersions.Read(ctx, cvTest.ID)
		require.NoError(t, err)

		// Don't compare the UploadURL because it will be generated twice in
		// this test - once at creation of the configuration version, and
		// again during the GET.
		cvTest.UploadURL, cv.UploadURL = "", ""

		assert.Equal(t, cvTest, cv)
	})

	t.Run("when the configuration version does not exist", func(t *testing.T) {
		cv, err := client.ConfigurationVersions.Read(ctx, "nonexisting")
		assert.Nil(t, cv)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("with invalid configuration version id", func(t *testing.T) {
		cv, err := client.ConfigurationVersions.Read(ctx, badIdentifier)
		assert.Nil(t, cv)
		assert.EqualError(t, err, "invalid value for configuration version ID")
	})
}

func TestConfigurationVersionsUpload(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	cv, cvCleanup := createConfigurationVersion(t, client, nil)
	defer cvCleanup()

	t.Run("with valid options", func(t *testing.T) {
		err := client.ConfigurationVersions.Upload(
			ctx,
			cv.UploadURL,
			"test-fixtures/config-version",
		)
		require.NoError(t, err)

		// We do this is a small loop, because it can take a second
		// before the upload is finished.
		for i := 0; ; i++ {
			refreshed, err := client.ConfigurationVersions.Read(ctx, cv.ID)
			require.NoError(t, err)

			if refreshed.Status == ConfigurationUploaded {
				break
			}

			if i > 10 {
				t.Fatal("Timeout waiting for the configuration version to be uploaded")
			}

			time.Sleep(1 * time.Second)
		}
	})

	t.Run("without a valid upload URL", func(t *testing.T) {
		err := client.ConfigurationVersions.Upload(
			ctx,
			cv.UploadURL[:len(cv.UploadURL)-10]+"nonexisting",
			"test-fixtures/config-version",
		)
		assert.Error(t, err)
	})

	t.Run("without a valid path", func(t *testing.T) {
		err := client.ConfigurationVersions.Upload(
			ctx,
			cv.UploadURL,
			"nonexisting",
		)
		assert.Error(t, err)
	})
}
