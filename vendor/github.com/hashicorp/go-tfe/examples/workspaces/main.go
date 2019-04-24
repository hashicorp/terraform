package main

import (
	"context"
	"log"

	tfe "github.com/hashicorp/go-tfe"
)

func main() {
	config := &tfe.Config{
		Token: "insert-your-token-here",
	}

	client, err := tfe.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}

	// Create a context
	ctx := context.Background()

	// Create a new workspace
	w, err := client.Workspaces.Create(ctx, "org-name", tfe.WorkspaceCreateOptions{
		Name: tfe.String("my-app-tst"),
	})
	if err != nil {
		log.Fatal(err)
	}

	// Update the workspace
	w, err = client.Workspaces.Update(ctx, "org-name", w.Name, tfe.WorkspaceUpdateOptions{
		AutoApply:        tfe.Bool(false),
		TerraformVersion: tfe.String("0.11.1"),
		WorkingDirectory: tfe.String("my-app/infra"),
	})
	if err != nil {
		log.Fatal(err)
	}
}
