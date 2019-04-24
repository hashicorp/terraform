package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTeamMembersList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	tmTest, tmTestCleanup := createTeam(t, client, nil)
	defer tmTestCleanup()

	options := TeamMemberAddOptions{
		Usernames: []string{"admin"},
	}
	err := client.TeamMembers.Add(ctx, tmTest.ID, options)
	require.NoError(t, err)

	t.Run("with valid options", func(t *testing.T) {
		users, err := client.TeamMembers.List(ctx, tmTest.ID)
		require.NoError(t, err)
		require.Equal(t, 1, len(users))

		found := false
		for _, user := range users {
			if user.Username == "admin" {
				found = true
				break
			}
		}

		assert.True(t, found)
	})

	t.Run("when the team ID is invalid", func(t *testing.T) {
		users, err := client.TeamMembers.List(ctx, badIdentifier)
		assert.EqualError(t, err, "invalid value for team ID")
		assert.Nil(t, users)
	})
}

func TestTeamMembersAdd(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	tmTest, tmTestCleanup := createTeam(t, client, nil)
	defer tmTestCleanup()

	t.Run("with valid options", func(t *testing.T) {
		options := TeamMemberAddOptions{
			Usernames: []string{"admin"},
		}

		err := client.TeamMembers.Add(ctx, tmTest.ID, options)
		require.NoError(t, err)

		users, err := client.TeamMembers.List(ctx, tmTest.ID)
		require.NoError(t, err)

		found := false
		for _, user := range users {
			if user.Username == "admin" {
				found = true
				break
			}
		}

		assert.True(t, found)
	})

	t.Run("when options is missing usernames", func(t *testing.T) {
		err := client.TeamMembers.Add(ctx, tmTest.ID, TeamMemberAddOptions{})
		assert.EqualError(t, err, "usernames is required")
	})

	t.Run("when usernames is empty", func(t *testing.T) {
		err := client.TeamMembers.Add(ctx, tmTest.ID, TeamMemberAddOptions{
			Usernames: []string{},
		})
		assert.EqualError(t, err, "invalid value for usernames")
	})

	t.Run("when the team ID is invalid", func(t *testing.T) {
		err := client.TeamMembers.Add(ctx, badIdentifier, TeamMemberAddOptions{
			Usernames: []string{"user1"},
		})
		assert.EqualError(t, err, "invalid value for team ID")
	})
}

func TestTeamMembersRemove(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	tmTest, tmTestCleanup := createTeam(t, client, nil)
	defer tmTestCleanup()

	options := TeamMemberAddOptions{
		Usernames: []string{"admin"},
	}
	err := client.TeamMembers.Add(ctx, tmTest.ID, options)
	require.NoError(t, err)

	t.Run("with valid options", func(t *testing.T) {
		options := TeamMemberRemoveOptions{
			Usernames: []string{"admin"},
		}

		err := client.TeamMembers.Remove(ctx, tmTest.ID, options)
		assert.NoError(t, err)
	})

	t.Run("when options is missing usernames", func(t *testing.T) {
		err := client.TeamMembers.Remove(ctx, tmTest.ID, TeamMemberRemoveOptions{})
		assert.EqualError(t, err, "usernames is required")
	})

	t.Run("when usernames is empty", func(t *testing.T) {
		err := client.TeamMembers.Remove(ctx, tmTest.ID, TeamMemberRemoveOptions{
			Usernames: []string{},
		})
		assert.EqualError(t, err, "invalid value for usernames")
	})

	t.Run("when the team ID is invalid", func(t *testing.T) {
		err := client.TeamMembers.Remove(ctx, badIdentifier, TeamMemberRemoveOptions{
			Usernames: []string{"user1"},
		})
		assert.EqualError(t, err, "invalid value for team ID")
	})
}
