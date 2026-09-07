The test using this test fixture asserts that a plan generated from this configuration includes both:
1. Details of the backend defined in the config when the plan was created
2. Details of the workspace that was selected when the plan was generated

The `inmem` backend is used because it supports the use of CE workspaces.
We set a non-default workspace in `internal/command/testdata/plan-out-backend-workspace/.terraform/environment`.