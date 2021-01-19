package main

import (
	"log"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform/internal/stresstest/internal/gitlog"
	"github.com/hashicorp/terraform/internal/stresstest/internal/stressaddr"
	"github.com/hashicorp/terraform/internal/stresstest/internal/stressrun"
	"github.com/mitchellh/cli"
)

// graphExportSeriesCommand implements the "stresstest graph export-series"
// command, which exercises a particular given configuration series and
// documents the series of steps and their results in a generated git
// repository.
type graphExportSeriesCommand struct {
}

var _ cli.Command = (*graphExportSeriesCommand)(nil)

func (c *graphExportSeriesCommand) Run(args []string) int {
	if len(args) != 2 {
		return cli.RunResultHelp
	}
	seriesAddr, err := stressaddr.ParseConfigSeries(args[0])
	if err != nil {
		log.Fatalf("invalid series address %q: %s", args[0], err)
	}
	targetDir, err := filepath.Abs(args[1])
	if err != nil {
		log.Fatalf("invalid target directory %q: %s", args[1], err)
	}

	log.Printf("Will run series %s and put the results in %s", seriesAddr, targetDir)

	repo, err := gitlog.InitRepository(targetDir)
	if err != nil {
		log.Fatalf("failed to initialize target directory: %s", err)
	}

	runLog := stressrun.RunSeries(seriesAddr)
	for _, step := range runLog {
		log.Printf("%s %s", step.Time, step.Message)
	}

	latestCommit, targetCommit, err := gitlog.BuildCommitsForLog(runLog, repo)
	if err != nil {
		log.Fatalf("failed to format the log as a git history: %s", err)
	}
	err = repo.SetRef("refs/heads/target", targetCommit)
	if err != nil {
		log.Fatalf("failed to update the 'target' branch: %s", err)
	}
	err = repo.SetRef("refs/heads/main", latestCommit)
	if err != nil {
		log.Fatalf("failed to update the 'main' branch: %s", err)
	}

	err = gitlog.CreateRepositoryWorkTree(targetDir)
	if err != nil {
		log.Fatalf("failed to prepare the git work tree: %s", err)
	}

	return 0
}

func (c *graphExportSeriesCommand) Synopsis() string {
	return "Run a single series and export a log"
}

func (c *graphExportSeriesCommand) Help() string {
	return strings.TrimSpace(`
Usage: stresstest graph export-series <series-id> <target-dir>

Runs the given configuration series (specified by id) and exports a log of
the results as a git repository in the target directory.

The generated git repository will contain a branch "current" which refers to
a commit describing the final action taken before a possible error, a branch
"end" which points to a commit describing the final commit in the series even
if there was an error earlier, and various other commits describing all of the
plan and apply actions taken as part of the series.

The generated filesystem layout in the git repository is intended to allow
you to navigate back and forth in the git history and then run Terraform CLI
commands as you normally would in order to exercise the configuration in the
same way the stresstest harness would've. It includes the configuration files,
the latest state snapshot, and the input variables at each step. However, the
generated configuration will refer to the special "stressful" provider and so
unfortunately it will take some special configuration in order to actually
exercise it.
`)
}
