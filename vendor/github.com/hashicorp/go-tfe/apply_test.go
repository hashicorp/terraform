package tfe

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppliesRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	rTest, rTestCleanup := createAppliedRun(t, client, nil)
	defer rTestCleanup()

	t.Run("when the plan exists", func(t *testing.T) {
		a, err := client.Applies.Read(ctx, rTest.Apply.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, a.LogReadURL)
		assert.Equal(t, a.Status, ApplyFinished)
		assert.NotEmpty(t, a.StatusTimestamps)
	})

	t.Run("when the apply does not exist", func(t *testing.T) {
		a, err := client.Applies.Read(ctx, "nonexisting")
		assert.Nil(t, a)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("with invalid apply ID", func(t *testing.T) {
		a, err := client.Applies.Read(ctx, badIdentifier)
		assert.Nil(t, a)
		assert.EqualError(t, err, "invalid value for apply ID")
	})
}

func TestAppliesLogs(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	rTest, rTestCleanup := createAppliedRun(t, client, nil)
	defer rTestCleanup()

	t.Run("when the log exists", func(t *testing.T) {
		a, err := client.Applies.Read(ctx, rTest.Apply.ID)
		require.NoError(t, err)

		logReader, err := client.Applies.Logs(ctx, a.ID)
		require.NoError(t, err)

		logs, err := ioutil.ReadAll(logReader)
		require.NoError(t, err)

		assert.Contains(t, string(logs), "1 added, 0 changed, 0 destroyed")
	})

	t.Run("when the log does not exist", func(t *testing.T) {
		logs, err := client.Applies.Logs(ctx, "nonexisting")
		assert.Nil(t, logs)
		assert.Error(t, err)
	})
}
