package tfe

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateVersionsList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	orgTest, orgTestCleanup := createOrganization(t, client)
	defer orgTestCleanup()

	wTest, wTestCleanup := createWorkspace(t, client, orgTest)
	defer wTestCleanup()

	svTest1, _ := createStateVersion(t, client, 0, wTest)
	svTest2, _ := createStateVersion(t, client, 1, wTest)

	t.Run("without list options", func(t *testing.T) {
		options := StateVersionListOptions{
			Organization: String(orgTest.Name),
			Workspace:    String(wTest.Name),
		}

		svl, err := client.StateVersions.List(ctx, options)
		require.NoError(t, err)

		// We need to strip the upload URL as that is a dynamic link.
		svTest1.DownloadURL = ""
		svTest2.DownloadURL = ""

		// And for the retrieved configuration versions as well.
		for _, sv := range svl.Items {
			sv.DownloadURL = ""
		}

		assert.Contains(t, svl.Items, svTest1)
		assert.Contains(t, svl.Items, svTest2)
		assert.Equal(t, 1, svl.CurrentPage)
		assert.Equal(t, 2, svl.TotalCount)
	})

	t.Run("with list options", func(t *testing.T) {
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		options := StateVersionListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
			Organization: String(orgTest.Name),
			Workspace:    String(wTest.Name),
		}

		svl, err := client.StateVersions.List(ctx, options)
		require.NoError(t, err)
		assert.Empty(t, svl.Items)
		assert.Equal(t, 999, svl.CurrentPage)
		assert.Equal(t, 2, svl.TotalCount)
	})

	t.Run("without an organization", func(t *testing.T) {
		options := StateVersionListOptions{
			Workspace: String(wTest.Name),
		}

		svl, err := client.StateVersions.List(ctx, options)
		assert.Nil(t, svl)
		assert.EqualError(t, err, "organization is required")
	})

	t.Run("without a workspace", func(t *testing.T) {
		options := StateVersionListOptions{
			Organization: String(orgTest.Name),
		}

		svl, err := client.StateVersions.List(ctx, options)
		assert.Nil(t, svl)
		assert.EqualError(t, err, "workspace is required")
	})
}

func TestStateVersionsCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	state, err := ioutil.ReadFile("test-fixtures/state-version/terraform.tfstate")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("with valid options", func(t *testing.T) {
		ctx := context.Background()
		_, err := client.Workspaces.Lock(ctx, wTest.ID, WorkspaceLockOptions{})
		if err != nil {
			t.Fatal(err)
		}

		sv, err := client.StateVersions.Create(ctx, wTest.ID, StateVersionCreateOptions{
			Lineage: String("741c4949-60b9-5bb1-5bf8-b14f4bb14af3"),
			MD5:     String(fmt.Sprintf("%x", md5.Sum(state))),
			Serial:  Int64(1),
			State:   String(base64.StdEncoding.EncodeToString(state)),
		})
		require.NoError(t, err)

		// Get a refreshed view of the configuration version.
		refreshed, err := client.StateVersions.Read(ctx, sv.ID)
		require.NoError(t, err)

		_, err = client.Workspaces.Unlock(ctx, wTest.ID)
		if err != nil {
			t.Fatal(err)
		}

		for _, item := range []*StateVersion{
			sv,
			refreshed,
		} {
			assert.NotEmpty(t, item.ID)
			assert.Equal(t, int64(1), item.Serial)
			assert.NotEmpty(t, item.CreatedAt)
			assert.NotEmpty(t, item.DownloadURL)
		}
	})

	t.Run("with the force flag set", func(t *testing.T) {
		ctx := context.Background()
		_, err := client.Workspaces.Lock(ctx, wTest.ID, WorkspaceLockOptions{})
		if err != nil {
			t.Fatal(err)
		}

		sv, err := client.StateVersions.Create(ctx, wTest.ID, StateVersionCreateOptions{
			Lineage: String("741c4949-60b9-5bb1-5bf8-b14f4bb14af3"),
			MD5:     String(fmt.Sprintf("%x", md5.Sum(state))),
			Serial:  Int64(1),
			State:   String(base64.StdEncoding.EncodeToString(state)),
		})
		require.NoError(t, err)

		sv, err = client.StateVersions.Create(ctx, wTest.ID, StateVersionCreateOptions{
			Lineage: String("821c4747-a0b9-3bd1-8bf3-c14f4bb14be7"),
			MD5:     String(fmt.Sprintf("%x", md5.Sum(state))),
			Serial:  Int64(2),
			State:   String(base64.StdEncoding.EncodeToString(state)),
			Force:   Bool(true),
		})
		require.NoError(t, err)

		// Get a refreshed view of the configuration version.
		refreshed, err := client.StateVersions.Read(ctx, sv.ID)
		require.NoError(t, err)

		_, err = client.Workspaces.Unlock(ctx, wTest.ID)
		if err != nil {
			t.Fatal(err)
		}

		for _, item := range []*StateVersion{
			sv,
			refreshed,
		} {
			assert.NotEmpty(t, item.ID)
			assert.Equal(t, int64(2), item.Serial)
			assert.NotEmpty(t, item.CreatedAt)
			assert.NotEmpty(t, item.DownloadURL)
		}
	})

	t.Run("with a run to associate with", func(t *testing.T) {
		t.Skip("This can only be tested with the run specific token")

		rTest, _ := createRun(t, client, wTest)

		ctx := context.Background()
		sv, err := client.StateVersions.Create(ctx, wTest.ID, StateVersionCreateOptions{
			MD5:    String(fmt.Sprintf("%x", md5.Sum(state))),
			Serial: Int64(0),
			State:  String(base64.StdEncoding.EncodeToString(state)),
			Run:    rTest,
		})
		require.NoError(t, err)
		require.NotEmpty(t, sv.Run)

		// Get a refreshed view of the configuration version.
		refreshed, err := client.StateVersions.Read(ctx, sv.ID)
		require.NoError(t, err)
		require.NotEmpty(t, refreshed.Run)

		for _, item := range []*StateVersion{
			sv,
			refreshed,
		} {
			assert.NotEmpty(t, item.ID)
			assert.Equal(t, int64(0), item.Serial)
			assert.NotEmpty(t, item.CreatedAt)
			assert.NotEmpty(t, item.DownloadURL)
			assert.Equal(t, rTest.ID, item.Run.ID)
		}
	})

	t.Run("without md5 hash", func(t *testing.T) {
		sv, err := client.StateVersions.Create(ctx, wTest.ID, StateVersionCreateOptions{
			Serial: Int64(0),
			State:  String(base64.StdEncoding.EncodeToString(state)),
		})
		assert.Nil(t, sv)
		assert.EqualError(t, err, "MD5 is required")
	})

	t.Run("withous serial", func(t *testing.T) {
		sv, err := client.StateVersions.Create(ctx, wTest.ID, StateVersionCreateOptions{
			MD5:   String(fmt.Sprintf("%x", md5.Sum(state))),
			State: String(base64.StdEncoding.EncodeToString(state)),
		})
		assert.Nil(t, sv)
		assert.EqualError(t, err, "serial is required")
	})

	t.Run("without state", func(t *testing.T) {
		sv, err := client.StateVersions.Create(ctx, wTest.ID, StateVersionCreateOptions{
			MD5:    String(fmt.Sprintf("%x", md5.Sum(state))),
			Serial: Int64(0),
		})
		assert.Nil(t, sv)
		assert.EqualError(t, err, "state is required")
	})

	t.Run("with invalid workspace id", func(t *testing.T) {
		sv, err := client.StateVersions.Create(ctx, badIdentifier, StateVersionCreateOptions{})
		assert.Nil(t, sv)
		assert.EqualError(t, err, "invalid value for workspace ID")
	})
}

func TestStateVersionsRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	svTest, svTestCleanup := createStateVersion(t, client, 0, nil)
	defer svTestCleanup()

	t.Run("when the state version exists", func(t *testing.T) {
		sv, err := client.StateVersions.Read(ctx, svTest.ID)
		require.NoError(t, err)

		// Don't compare the DownloadURL because it will be generated twice
		// in this test - once at creation of the configuration version, and
		// again during the GET.
		svTest.DownloadURL, sv.DownloadURL = "", ""

		assert.Equal(t, svTest, sv)
	})

	t.Run("when the state version does not exist", func(t *testing.T) {
		sv, err := client.StateVersions.Read(ctx, "nonexisting")
		assert.Nil(t, sv)
		assert.Equal(t, ErrResourceNotFound, err)
	})

	t.Run("with invalid state version id", func(t *testing.T) {
		sv, err := client.StateVersions.Read(ctx, badIdentifier)
		assert.Nil(t, sv)
		assert.EqualError(t, err, "invalid value for state version ID")
	})
}

func TestStateVersionsCurrent(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	wTest1, wTest1Cleanup := createWorkspace(t, client, nil)
	defer wTest1Cleanup()

	wTest2, wTest2Cleanup := createWorkspace(t, client, nil)
	defer wTest2Cleanup()

	svTest, svTestCleanup := createStateVersion(t, client, 0, wTest1)
	defer svTestCleanup()

	t.Run("when a state version exists", func(t *testing.T) {
		sv, err := client.StateVersions.Current(ctx, wTest1.ID)
		require.NoError(t, err)

		// Don't compare the DownloadURL because it will be generated twice
		// in this test - once at creation of the configuration version, and
		// again during the GET.
		svTest.DownloadURL, sv.DownloadURL = "", ""

		assert.Equal(t, svTest, sv)
	})

	t.Run("when a state version does not exist", func(t *testing.T) {
		sv, err := client.StateVersions.Current(ctx, wTest2.ID)
		assert.Nil(t, sv)
		assert.Equal(t, ErrResourceNotFound, err)
	})

	t.Run("with invalid workspace id", func(t *testing.T) {
		sv, err := client.StateVersions.Current(ctx, badIdentifier)
		assert.Nil(t, sv)
		assert.EqualError(t, err, "invalid value for workspace ID")
	})
}

func TestStateVersionsDownload(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	svTest, svTestCleanup := createStateVersion(t, client, 0, nil)
	defer svTestCleanup()

	stateTest, err := ioutil.ReadFile("test-fixtures/state-version/terraform.tfstate")
	require.NoError(t, err)

	t.Run("when the state version exists", func(t *testing.T) {
		state, err := client.StateVersions.Download(ctx, svTest.DownloadURL)
		require.NoError(t, err)
		assert.Equal(t, stateTest, state)
	})

	t.Run("when the state version does not exist", func(t *testing.T) {
		state, err := client.StateVersions.Download(
			ctx,
			svTest.DownloadURL[:len(svTest.DownloadURL)-10]+"nonexisting",
		)
		assert.Nil(t, state)
		assert.Error(t, err)
	})

	t.Run("with an invalid url", func(t *testing.T) {
		state, err := client.StateVersions.Download(ctx, badIdentifier)
		assert.Nil(t, state)
		assert.Equal(t, ErrResourceNotFound, err)
	})
}
