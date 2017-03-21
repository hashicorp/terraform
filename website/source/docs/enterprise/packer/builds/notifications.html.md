---
title: "About Packer Build Notifications"
---

# About Packer Build Notifications

Atlas can send build notifications to your organization via one of our
[supported notification methods](/help/consul/alerts/notification-methods). The
following events are configurable:

- **Starting** - The build has begun.
- **Finished** - All build jobs have finished successfully.
- **Errored** - An error has occurred during one of the build jobs.
- **Canceled** - A user in Atlas has canceled the build.

> Emails will include logs for the **Finished** and **Errored** events.

You can toggle notifications for each of these events on the "Integrations" tab
of a build configuration.
