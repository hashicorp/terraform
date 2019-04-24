package tfe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationConfigurationList(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	ncTest1, _ := createNotificationConfiguration(t, client, wTest)
	ncTest2, _ := createNotificationConfiguration(t, client, wTest)

	t.Run("with a valid workspace", func(t *testing.T) {
		ncl, err := client.NotificationConfigurations.List(
			ctx,
			wTest.ID,
			NotificationConfigurationListOptions{},
		)
		require.NoError(t, err)
		assert.Contains(t, ncl.Items, ncTest1)
		assert.Contains(t, ncl.Items, ncTest2)

		t.Skip("paging not supported yet in API")
		assert.Equal(t, 1, ncl.CurrentPage)
		assert.Equal(t, 2, ncl.TotalCount)
	})

	t.Run("with list options", func(t *testing.T) {
		t.Skip("paging not supported yet in API")
		// Request a page number which is out of range. The result should
		// be successful, but return no results if the paging options are
		// properly passed along.
		ncl, err := client.NotificationConfigurations.List(
			ctx,
			wTest.ID,
			NotificationConfigurationListOptions{
				ListOptions: ListOptions{
					PageNumber: 999,
					PageSize:   100,
				},
			},
		)
		require.NoError(t, err)
		assert.Empty(t, ncl.Items)
		assert.Equal(t, 999, ncl.CurrentPage)
		assert.Equal(t, 2, ncl.TotalCount)
	})

	t.Run("without a valid workspace", func(t *testing.T) {
		ncl, err := client.NotificationConfigurations.List(
			ctx,
			badIdentifier,
			NotificationConfigurationListOptions{},
		)
		assert.Nil(t, ncl)
		assert.EqualError(t, err, "invalid value for workspace ID")
	})
}

func TestNotificationConfigurationCreate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	t.Run("with all required values", func(t *testing.T) {
		options := NotificationConfigurationCreateOptions{
			DestinationType: NotificationDestination(NotificationDestinationTypeGeneric),
			Enabled:         Bool(false),
			Name:            String(randomString(t)),
			Token:           String(randomString(t)),
			URL:             String("http://example.com"),
			Triggers:        []string{NotificationTriggerCreated},
		}

		_, err := client.NotificationConfigurations.Create(ctx, wTest.ID, options)
		require.NoError(t, err)
	})

	t.Run("without a required value", func(t *testing.T) {
		options := NotificationConfigurationCreateOptions{
			DestinationType: NotificationDestination(NotificationDestinationTypeGeneric),
			Enabled:         Bool(false),
			Token:           String(randomString(t)),
			URL:             String("http://example.com"),
			Triggers:        []string{NotificationTriggerCreated},
		}

		nc, err := client.NotificationConfigurations.Create(ctx, wTest.ID, options)
		assert.Nil(t, nc)
		assert.EqualError(t, err, "name is required")
	})

	t.Run("without a valid workspace", func(t *testing.T) {
		nc, err := client.NotificationConfigurations.Create(ctx, badIdentifier, NotificationConfigurationCreateOptions{})
		assert.Nil(t, nc)
		assert.EqualError(t, err, "invalid value for workspace ID")
	})
}

func TestNotificationConfigurationRead(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	ncTest, ncTestCleanup := createNotificationConfiguration(t, client, nil)
	defer ncTestCleanup()

	t.Run("with a valid ID", func(t *testing.T) {
		nc, err := client.NotificationConfigurations.Read(ctx, ncTest.ID)
		require.NoError(t, err)
		assert.Equal(t, ncTest.ID, nc.ID)
	})

	t.Run("when the notification configuration does not exist", func(t *testing.T) {
		_, err := client.NotificationConfigurations.Read(ctx, "nonexisting")
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the notification configuration ID is invalid", func(t *testing.T) {
		_, err := client.NotificationConfigurations.Read(ctx, badIdentifier)
		assert.EqualError(t, err, "invalid value for notification configuration ID")
	})
}

func TestNotificationConfigurationUpdate(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	ncTest, ncTestCleanup := createNotificationConfiguration(t, client, nil)
	defer ncTestCleanup()

	t.Run("with options", func(t *testing.T) {
		options := NotificationConfigurationUpdateOptions{
			Enabled: Bool(true),
			Name:    String("newName"),
		}

		nc, err := client.NotificationConfigurations.Update(ctx, ncTest.ID, options)
		require.NoError(t, err)
		assert.Equal(t, nc.Enabled, true)
		assert.Equal(t, nc.Name, "newName")
	})

	t.Run("without options", func(t *testing.T) {
		_, err := client.NotificationConfigurations.Update(ctx, ncTest.ID, NotificationConfigurationUpdateOptions{})
		require.NoError(t, err)
	})

	t.Run("when the notification configuration does not exist", func(t *testing.T) {
		_, err := client.NotificationConfigurations.Update(ctx, "nonexisting", NotificationConfigurationUpdateOptions{})
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the notification configuration ID is invalid", func(t *testing.T) {
		_, err := client.NotificationConfigurations.Update(ctx, badIdentifier, NotificationConfigurationUpdateOptions{})
		assert.EqualError(t, err, "invalid value for notification configuration ID")
	})
}

func TestNotificationConfigurationDelete(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	wTest, wTestCleanup := createWorkspace(t, client, nil)
	defer wTestCleanup()

	ncTest, _ := createNotificationConfiguration(t, client, wTest)

	t.Run("with a valid ID", func(t *testing.T) {
		err := client.NotificationConfigurations.Delete(ctx, ncTest.ID)
		require.NoError(t, err)

		_, err = client.NotificationConfigurations.Read(ctx, ncTest.ID)
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the notification configuration does not exist", func(t *testing.T) {
		err := client.NotificationConfigurations.Delete(ctx, "nonexisting")
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the notification configuration ID is invalid", func(t *testing.T) {
		err := client.NotificationConfigurations.Delete(ctx, badIdentifier)
		assert.EqualError(t, err, "invalid value for notification configuration ID")
	})
}

func TestNotificationConfigurationVerify(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()

	ncTest, ncTestCleanup := createNotificationConfiguration(t, client, nil)
	defer ncTestCleanup()

	t.Run("with a valid ID", func(t *testing.T) {
		_, err := client.NotificationConfigurations.Verify(ctx, ncTest.ID)
		require.NoError(t, err)
	})

	t.Run("when the notification configuration does not exists", func(t *testing.T) {
		_, err := client.NotificationConfigurations.Verify(ctx, "nonexisting")
		assert.Equal(t, err, ErrResourceNotFound)
	})

	t.Run("when the notification configuration ID is invalid", func(t *testing.T) {
		_, err := client.NotificationConfigurations.Verify(ctx, badIdentifier)
		assert.EqualError(t, err, "invalid value for notification configuration ID")
	})
}
