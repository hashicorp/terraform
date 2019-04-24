package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTeamTokensGenerate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	tmTest, tmTestCleanup := createTeam(t, client, nil)
	defer tmTestCleanup()

	var tmToken string
	t.Run("with valid options", func(t *testing.T) {
		tt, err := client.TeamTokens.Generate(ctx, tmTest.ID)
		require.NoError(t, err)
		require.NotEmpty(t, tt.Token)
		tmToken = tt.Token
	})

	t.Run("when a token already exists", func(t *testing.T) {
		tt, err := client.TeamTokens.Generate(ctx, tmTest.ID)
		require.NoError(t, err)
		require.NotEmpty(t, tt.Token)
		assert.NotEqual(t, tmToken, tt.Token)
	})

	t.Run("without valid team ID", func(t *testing.T) {
		tt, err := client.TeamTokens.Generate(ctx, badIdentifier)
		assert.Nil(t, tt)
		assert.EqualError(t, err, "invalid value for team ID")
	})
}
func TestTeamTokensRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	tmTest, tmTestCleanup := createTeam(t, client, nil)
	defer tmTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		_, ttTestCleanup := createTeamToken(t, client, tmTest)

		tt, err := client.TeamTokens.Read(ctx, tmTest.ID)
		assert.NoError(t, err)
		assert.NotEmpty(t, tt)

		ttTestCleanup()
	})

	t.Run("when a token doesn't exists", func(t *testing.T) {
		tt, err := client.TeamTokens.Read(ctx, tmTest.ID)
		assert.Equal(t, ErrResourceNotFound, err)
		assert.Nil(t, tt)
	})

	t.Run("without valid organization", func(t *testing.T) {
		tt, err := client.OrganizationTokens.Read(ctx, badIdentifier)
		assert.Nil(t, tt)
		assert.EqualError(t, err, "invalid value for organization")
	})
}

func TestTeamTokensDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	tmTest, tmTestCleanup := createTeam(t, client, nil)
	defer tmTestCleanup()

	createTeamToken(t, client, tmTest)

	t.Run("with valid options", func(t *testing.T) {
		err := client.TeamTokens.Delete(ctx, tmTest.ID)
		assert.NoError(t, err)
	})

	t.Run("when a token does not exist", func(t *testing.T) {
		err := client.TeamTokens.Delete(ctx, tmTest.ID)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("without valid team ID", func(t *testing.T) {
		err := client.TeamTokens.Delete(ctx, badIdentifier)
		assert.EqualError(t, err, "invalid value for team ID")
	})
}
