package tfe

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlansRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	rTest, rTestCleanup := createPlannedRun(t, client, nil)
	defer rTestCleanup()

	t.Run("when the plan exists", func(t *testing.T) {
		p, err := client.Plans.Read(ctx, rTest.Plan.ID)
		require.NoError(t, err)
		assert.True(t, p.HasChanges)
		assert.NotEmpty(t, p.LogReadURL)
		assert.Equal(t, p.Status, PlanFinished)
		assert.NotEmpty(t, p.StatusTimestamps)
	})

	t.Run("when the plan does not exist", func(t *testing.T) {
		p, err := client.Plans.Read(ctx, "nonexisting")
		assert.Nil(t, p)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("with invalid plan ID", func(t *testing.T) {
		p, err := client.Plans.Read(ctx, badIdentifier)
		assert.Nil(t, p)
		assert.EqualError(t, err, "invalid value for plan ID")
	})
}

func TestPlansLogs(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	rTest, rTestCleanup := createPlannedRun(t, client, nil)
	defer rTestCleanup()

	t.Run("when the log exists", func(t *testing.T) {
		p, err := client.Plans.Read(ctx, rTest.Plan.ID)
		require.NoError(t, err)

		logReader, err := client.Plans.Logs(ctx, p.ID)
		require.NoError(t, err)

		logs, err := ioutil.ReadAll(logReader)
		require.NoError(t, err)

		assert.Contains(t, string(logs), "1 to add, 0 to change, 0 to destroy")
	})

	t.Run("when the log does not exist", func(t *testing.T) {
		logs, err := client.Plans.Logs(ctx, "nonexisting")
		assert.Nil(t, logs)
		assert.Error(t, err)
	})
}
