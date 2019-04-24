package tfe

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunsList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	rTest1, _ := createRun(t, client, wTest)
	rTest2, _ := createRun(t, client, wTest)

	t.Run("without list options", func(t *testing.T) {
		rl, err := client.Runs.List(ctx, wTest.ID, RunListOptions{})
		require.NoError(t, err)

		found := []string{}
		for _, r := range rl.Items {
			found = append(found, r.ID)
		}

		assert.Contains(t, found, rTest1.ID)
		assert.Contains(t, found, rTest2.ID)
		assert.Equal(t, 1, rl.CurrentPage)
		assert.Equal(t, 2, rl.TotalCount)
	})

	t.Run("with list options", func(t *testing.T) {
		t.Skip("paging not supported yet in API")

		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		rl, err := client.Runs.List(ctx, wTest.ID, RunListOptions{
			ListOptions: ListOptions{
				PageNumber: 999,
				PageSize:   100,
			},
		})
		require.NoError(t, err)
		assert.Empty(t, rl.Items)
		assert.Equal(t, 999, rl.CurrentPage)
		assert.Equal(t, 2, rl.TotalCount)
	})

	t.Run("without a valid workspace ID", func(t *testing.T) {
		rl, err := client.Runs.List(ctx, badIdentifier, RunListOptions{})
		assert.Nil(t, rl)
		assert.EqualError(t, err, "invalid value for workspace ID")
	})
}

func TestRunsCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	cvTest, _ := createUploadedConfigurationVersion(t, client, wTest)

	t.Run("without a configuration version", func(t *testing.T) {
		options := RunCreateOptions{
			Workspace: wTest,
		}

		_, err := client.Runs.Create(ctx, options)
		assert.NoError(t, err)
	})

	t.Run("with a configuration version", func(t *testing.T) {
		options := RunCreateOptions{
			ConfigurationVersion: cvTest,
			Workspace:            wTest,
		}

		r, err := client.Runs.Create(ctx, options)
		require.NoError(t, err)
		assert.Equal(t, cvTest.ID, r.ConfigurationVersion.ID)
	})

	t.Run("without a workspace", func(t *testing.T) {
		r, err := client.Runs.Create(ctx, RunCreateOptions{})
		assert.Nil(t, r)
		assert.EqualError(t, err, "workspace is required")
	})

	t.Run("with additional attributes", func(t *testing.T) {
		options := RunCreateOptions{
			Message:   String("yo"),
			Workspace: wTest,
		}

		r, err := client.Runs.Create(ctx, options)
		require.NoError(t, err)
		assert.Equal(t, *options.Message, r.Message)
	})
}

func TestRunsRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	rTest, rTestCleanup := createPlannedRun(t, client, nil)
	defer rTestCleanup()

	t.Run("when the run exists", func(t *testing.T) {
		r, err := client.Runs.Read(ctx, rTest.ID)
		assert.NoError(t, err)
		assert.Equal(t, rTest, r)
	})

	t.Run("when the run does not exist", func(t *testing.T) {
		r, err := client.Runs.Read(ctx, "nonexisting")
		assert.Nil(t, r)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("with invalid run ID", func(t *testing.T) {
		r, err := client.Runs.Read(ctx, badIdentifier)
		assert.Nil(t, r)
		assert.EqualError(t, err, "invalid value for run ID")
	})
}

func TestRunsApply(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	rTest, rTestCleanup := createPlannedRun(t, client, nil)
	defer rTestCleanup()

	t.Run("when the run exists", func(t *testing.T) {
		err := client.Runs.Apply(ctx, rTest.ID, RunApplyOptions{})
		assert.NoError(t, err)
	})

	t.Run("when the run does not exist", func(t *testing.T) {
		err := client.Runs.Apply(ctx, "nonexisting", RunApplyOptions{})
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("with invalid run ID", func(t *testing.T) {
		err := client.Runs.Apply(ctx, badIdentifier, RunApplyOptions{})
		assert.EqualError(t, err, "invalid value for run ID")
	})
}

func TestRunsCancel(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	// We need to create 2 runs here. The first run will automatically
	// be planned so that one cannot be cancelled. The second one will
	// be pending until the first one is confirmed or discarded, so we
	// can cancel that one.
	_, _ = createRun(t, client, wTest)
	rTest2, _ := createRun(t, client, wTest)

	t.Run("when the run exists", func(t *testing.T) {
		err := client.Runs.Cancel(ctx, rTest2.ID, RunCancelOptions{})
		assert.NoError(t, err)
	})

	t.Run("when the run does not exist", func(t *testing.T) {
		err := client.Runs.Cancel(ctx, "nonexisting", RunCancelOptions{})
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("with invalid run ID", func(t *testing.T) {
		err := client.Runs.Cancel(ctx, badIdentifier, RunCancelOptions{})
		assert.EqualError(t, err, "invalid value for run ID")
	})
}

func TestRunsForceCancel(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	// We need to create 2 runs here. The first run will automatically
	// be planned so that one cannot be cancelled. The second one will
	// be pending until the first one is confirmed or discarded, so we
	// can cancel that one.
	_, _ = createRun(t, client, wTest)
	rTest, _ := createRun(t, client, wTest)

	t.Run("run is not force-cancelable", func(t *testing.T) {
		assert.False(t, rTest.Actions.IsForceCancelable)
	})

	t.Run("user is allowed to force-cancel", func(t *testing.T) {
		assert.True(t, rTest.Permissions.CanForceCancel)
	})

	t.Run("after a normal cancel", func(t *testing.T) {
		// Request the normal cancel
		err := client.Runs.Cancel(ctx, rTest.ID, RunCancelOptions{})
		require.NoError(t, err)

		for i := 1; ; i++ {
			// Refresh the view of the run
			rTest, err = client.Runs.Read(ctx, rTest.ID)
			require.NoError(t, err)

			// Check if the timestamp is present.
			if !rTest.ForceCancelAvailableAt.IsZero() {
				break
			}

			if i > 30 {
				t.Fatal("Timeout waiting for run to be canceled")
			}

			time.Sleep(time.Second)
		}

		t.Run("force-cancel-available-at timestamp is present", func(t *testing.T) {
			assert.True(t, rTest.ForceCancelAvailableAt.After(time.Now()))
		})

		// This test case is minimal because a force-cancel is not needed in
		// any normal circumstance. Only if Terraform encounters unexpected
		// errors or behaves abnormally should this functionality be required.
		// Force-cancel only becomes available if a normal cancel is performed
		// first, and the desired canceled state is not reached within a pre-
		// determined amount of time (see
		// https://www.terraform.io/docs/enterprise/api/run.html#forcefully-cancel-a-run).
	})

	t.Run("when the run does not exist", func(t *testing.T) {
		err := client.Runs.ForceCancel(ctx, "nonexisting", RunForceCancelOptions{})
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("with invalid run ID", func(t *testing.T) {
		err := client.Runs.ForceCancel(ctx, badIdentifier, RunForceCancelOptions{})
		assert.EqualError(t, err, "invalid value for run ID")
	})
}

func TestRunsDiscard(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	rTest, rTestCleanup := createPlannedRun(t, client, nil)
	defer rTestCleanup()

	t.Run("when the run exists", func(t *testing.T) {
		err := client.Runs.Discard(ctx, rTest.ID, RunDiscardOptions{})
		assert.NoError(t, err)
	})

	t.Run("when the run does not exist", func(t *testing.T) {
		err := client.Runs.Discard(ctx, "nonexisting", RunDiscardOptions{})
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("with invalid run ID", func(t *testing.T) {
		err := client.Runs.Discard(ctx, badIdentifier, RunDiscardOptions{})
		assert.EqualError(t, err, "invalid value for run ID")
	})
}
