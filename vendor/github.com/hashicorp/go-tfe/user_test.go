package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUsersReadCurrent(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	u, err := client.Users.ReadCurrent(ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, u.ID)
	assert.NotEmpty(t, u.AvatarURL)
	assert.NotEmpty(t, u.Username)

	t.Run("two factor options are decoded", func(t *testing.T) {
		assert.NotNil(t, u.TwoFactor)
	})
}

func TestUsersUpdate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	uTest, err := client.Users.ReadCurrent(ctx)
	require.NoError(t, err)

	// Make sure we reset the current user when were done.
	defer func() {
		client.Users.Update(ctx, UserUpdateOptions{
			Email:    String(uTest.Email),
			Username: String(uTest.Username),
		})
	}()

	t.Run("without any options", func(t *testing.T) {
		_, err := client.Users.Update(ctx, UserUpdateOptions{})
		require.NoError(t, err)

		u, err := client.Users.ReadCurrent(ctx)
		assert.NoError(t, err)
		assert.Equal(t, u, uTest)
	})

	t.Run("with a new username", func(t *testing.T) {
		_, err := client.Users.Update(ctx, UserUpdateOptions{
			Username: String("NewTestUsername"),
		})
		require.NoError(t, err)

		u, err := client.Users.ReadCurrent(ctx)
		assert.NoError(t, err)
		assert.Equal(t, "NewTestUsername", u.Username)
	})

	t.Run("with a new email address", func(t *testing.T) {
		_, err := client.Users.Update(ctx, UserUpdateOptions{
			Email: String("newtestemail@hashicorp.com"),
		})
		require.NoError(t, err)

		u, err := client.Users.ReadCurrent(ctx)
		assert.NoError(t, err)
		assert.Equal(t, "newtestemail@hashicorp.com", u.UnconfirmedEmail)
	})

	t.Run("with invalid email address", func(t *testing.T) {
		u, err := client.Users.Update(ctx, UserUpdateOptions{
			Email: String("notamailaddress"),
		})
		assert.Nil(t, u)
		assert.Error(t, err)
	})
}
