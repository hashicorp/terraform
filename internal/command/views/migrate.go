// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"bufio"
	encJson "encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/migrate"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ---------------------------------------------------------------------------
// MigrateList
// ---------------------------------------------------------------------------

// MigrateList is the view interface for the "terraform migrate list" command.
type MigrateList interface {
	List(migrations []migrate.Migration, results map[string][]migrate.SubMigrationResult, detail bool) int
	Diagnostics(diags tfdiags.Diagnostics)
}

// NewMigrateList creates a MigrateList view appropriate for the given ViewType.
func NewMigrateList(vt arguments.ViewType, view *View) MigrateList {
	switch vt {
	case arguments.ViewJSON:
		return &MigrateListJSON{view: view}
	case arguments.ViewHuman:
		return &MigrateListHuman{view: view}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

// MigrateListHuman renders migrate list output for human consumption.
type MigrateListHuman struct {
	view *View
}

var _ MigrateList = (*MigrateListHuman)(nil)

func (v *MigrateListHuman) List(migrations []migrate.Migration, results map[string][]migrate.SubMigrationResult, detail bool) int {
	// Filter to only migrations that have results (i.e. matched something).
	type entry struct {
		migration migrate.Migration
		results   []migrate.SubMigrationResult
	}

	// Group by namespace/provider.
	type group struct {
		key     string // "namespace/provider"
		entries []entry
	}

	groupMap := make(map[string]*group)
	var groupOrder []string

	for _, m := range migrations {
		res, ok := results[m.ID()]
		if !ok || len(res) == 0 {
			continue
		}

		key := m.Namespace + "/" + m.Provider
		g, exists := groupMap[key]
		if !exists {
			g = &group{key: key}
			groupMap[key] = g
			groupOrder = append(groupOrder, key)
		}
		g.entries = append(g.entries, entry{migration: m, results: res})
	}

	if len(groupOrder) == 0 {
		v.view.streams.Println("No applicable migrations found.")
		return 0
	}

	for _, key := range groupOrder {
		g := groupMap[key]

		// Count total migrations for this group.
		migrationCount := len(g.entries)
		noun := "migration"
		if migrationCount != 1 {
			noun = "migrations"
		}
		v.view.streams.Println(fmt.Sprintf("%s (%d %s available):", key, migrationCount, noun))

		for _, e := range g.entries {
			// Count files and changes from results.
			fileSet := make(map[string]struct{})
			changes := 0
			for _, r := range e.results {
				changes += len(r.Files)
				for _, f := range r.Files {
					fileSet[f.Filename] = struct{}{}
				}
			}
			files := len(fileSet)

			v.view.streams.Println(fmt.Sprintf("  %-18s %-40s %d files, %d changes",
				e.migration.Name, e.migration.Description, files, changes))

			// Show sub-migrations.
			subs := e.results
			limit := 3
			if detail || len(subs) <= limit {
				for _, s := range subs {
					v.view.streams.Println(fmt.Sprintf("    - %-24s %s", s.SubMigration.Name, s.SubMigration.Description))
				}
				if !detail && len(subs) > 0 {
					v.view.streams.Println(fmt.Sprintf("    (%d sub-migrations total)", len(subs)))
				}
			} else {
				for _, s := range subs[:limit] {
					v.view.streams.Println(fmt.Sprintf("    - %-24s %s", s.SubMigration.Name, s.SubMigration.Description))
				}
				remaining := len(subs) - limit
				v.view.streams.Println(fmt.Sprintf("    (+%d more, use -detail to list all)", remaining))
			}
		}
	}

	return 0
}

func (v *MigrateListHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

// MigrateListJSON renders migrate list output as JSON.
type MigrateListJSON struct {
	view *View
}

var _ MigrateList = (*MigrateListJSON)(nil)

func (v *MigrateListJSON) List(migrations []migrate.Migration, results map[string][]migrate.SubMigrationResult, detail bool) int {
	type jsonSubMigration struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		FileCount   int    `json:"file_count"`
	}

	type jsonMigration struct {
		ID            string             `json:"id"`
		Namespace     string             `json:"namespace"`
		Provider      string             `json:"provider"`
		Name          string             `json:"name"`
		Description   string             `json:"description"`
		FileCount     int                `json:"file_count"`
		ChangeCount   int                `json:"change_count"`
		SubMigrations []jsonSubMigration `json:"sub_migrations"`
	}

	var out []jsonMigration
	for _, m := range migrations {
		res, ok := results[m.ID()]
		if !ok || len(res) == 0 {
			continue
		}

		fileSet := make(map[string]struct{})
		changes := 0
		var subs []jsonSubMigration
		for _, r := range res {
			subFiles := len(r.Files)
			changes += subFiles
			for _, f := range r.Files {
				fileSet[f.Filename] = struct{}{}
			}
			subs = append(subs, jsonSubMigration{
				Name:        r.SubMigration.Name,
				Description: r.SubMigration.Description,
				FileCount:   subFiles,
			})
		}

		out = append(out, jsonMigration{
			ID:            m.ID(),
			Namespace:     m.Namespace,
			Provider:      m.Provider,
			Name:          m.Name,
			Description:   m.Description,
			FileCount:     len(fileSet),
			ChangeCount:   changes,
			SubMigrations: subs,
		})
	}

	if out == nil {
		out = []jsonMigration{}
	}

	bytes, err := encJson.Marshal(out)
	if err != nil {
		v.view.streams.Eprintf("error marshalling migration list: %v", err)
		return 1
	}
	v.view.streams.Println(string(bytes))
	return 0
}

func (v *MigrateListJSON) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

// ---------------------------------------------------------------------------
// MigrateApply
// ---------------------------------------------------------------------------

// MigrateApply is the view interface for the "terraform migrate apply" command.
type MigrateApply interface {
	Applying(id string)
	Progress(sm migrate.SubMigration, filenames []string)
	Summary(changes int, files int)
	DryRunHeader(id string)
	Diff(filename string, before, after []byte)
	DryRunSummary(changes int, files int)
	StepHeader(index, total int, sm migrate.SubMigration)
	StepPrompt(streams *terminal.Streams) byte
	Diagnostics(diags tfdiags.Diagnostics)
}

// NewMigrateApply creates a MigrateApply view appropriate for the given ViewType.
func NewMigrateApply(vt arguments.ViewType, view *View) MigrateApply {
	switch vt {
	case arguments.ViewJSON:
		return &MigrateApplyJSON{view: view}
	case arguments.ViewHuman:
		return &MigrateApplyHuman{view: view}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

// MigrateApplyHuman renders migrate apply output for human consumption.
type MigrateApplyHuman struct {
	view    *View
	scanner *bufio.Scanner // lazily initialized for StepPrompt
}

var _ MigrateApply = (*MigrateApplyHuman)(nil)

func (v *MigrateApplyHuman) Applying(id string) {
	v.view.streams.Println(fmt.Sprintf("Applying %s...", id))
}

func (v *MigrateApplyHuman) Progress(sm migrate.SubMigration, filenames []string) {
	check := v.view.colorize.Color("[green]\u2713[reset]")
	fileList := strings.Join(filenames, ", ")
	v.view.streams.Println(fmt.Sprintf("  %s %-18s (%s)", check, sm.Name, fileList))
}

func (v *MigrateApplyHuman) Summary(changes int, files int) {
	v.view.streams.Println(fmt.Sprintf("\nApplied %d changes across %d files.", changes, files))
}

func (v *MigrateApplyHuman) DryRunHeader(id string) {
	v.view.streams.Println(fmt.Sprintf("Planning %s...", id))
}

func (v *MigrateApplyHuman) Diff(filename string, before, after []byte) {
	v.view.streams.Println(fmt.Sprintf("--- %s", filename))
	v.view.streams.Println(fmt.Sprintf("+++ %s", filename))

	beforeLines := splitLines(before)
	afterLines := splitLines(after)

	// Build a list of diff operations using LCS.
	ops := computeDiffOps(beforeLines, afterLines)

	// Render with context: show up to 3 lines of context around changes,
	// and print "..." separators when skipping more than 6 unchanged lines.
	const contextSize = 3

	// First, mark which operations should be visible (changed lines + context).
	visible := make([]bool, len(ops))
	for i, op := range ops {
		if op.kind != diffKeep {
			// Mark this line and up to contextSize lines before/after.
			for j := max(0, i-contextSize); j <= min(len(ops)-1, i+contextSize); j++ {
				visible[j] = true
			}
		}
	}

	// Render visible operations, inserting "..." for gaps.
	lastPrinted := -1
	for i, op := range ops {
		if !visible[i] {
			continue
		}
		// If there's a gap since the last printed line, show a separator.
		if lastPrinted >= 0 && i > lastPrinted+1 {
			v.view.streams.Println("...")
		}
		lastPrinted = i

		switch op.kind {
		case diffKeep:
			v.view.streams.Println(fmt.Sprintf(" %s", op.text))
		case diffRemove:
			line := v.view.colorize.Color(fmt.Sprintf("[red]-%s[reset]", op.text))
			v.view.streams.Println(line)
		case diffAdd:
			line := v.view.colorize.Color(fmt.Sprintf("[green]+%s[reset]", op.text))
			v.view.streams.Println(line)
		}
	}
}

func (v *MigrateApplyHuman) DryRunSummary(changes int, files int) {
	v.view.streams.Println(fmt.Sprintf("\n%d changes would be applied across %d files.", changes, files))
}

func (v *MigrateApplyHuman) StepHeader(index, total int, sm migrate.SubMigration) {
	v.view.streams.Println(fmt.Sprintf("[%d/%d] %s: %s", index, total, sm.Name, sm.Description))
}

func (v *MigrateApplyHuman) StepPrompt(streams *terminal.Streams) byte {
	fmt.Fprint(streams.Stdout.File, "Apply this change? [y]es / [n]o / [q]uit: ")

	if v.scanner == nil {
		v.scanner = bufio.NewScanner(streams.Stdin.File)
	}
	if !v.scanner.Scan() {
		return 'n'
	}
	line := strings.TrimSpace(v.scanner.Text())
	if len(line) == 0 {
		return 'n'
	}

	switch line[0] {
	case 'y', 'Y':
		return 'y'
	case 'n', 'N':
		return 'n'
	case 'q', 'Q':
		return 'q'
	default:
		return 'n'
	}
}

func (v *MigrateApplyHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

// MigrateApplyJSON renders migrate apply output as JSON events.
type MigrateApplyJSON struct {
	view *View
}

var _ MigrateApply = (*MigrateApplyJSON)(nil)

func (v *MigrateApplyJSON) output(eventType string, data any) {
	payload := map[string]any{
		"type": eventType,
		"data": data,
	}
	bytes, _ := encJson.Marshal(payload)
	v.view.streams.Println(string(bytes))
}

func (v *MigrateApplyJSON) Applying(id string) {
	v.output("applying", map[string]string{"id": id})
}

func (v *MigrateApplyJSON) Progress(sm migrate.SubMigration, filenames []string) {
	v.output("progress", map[string]any{
		"name":  sm.Name,
		"files": filenames,
	})
}

func (v *MigrateApplyJSON) Summary(changes int, files int) {
	v.output("summary", map[string]int{
		"changes": changes,
		"files":   files,
	})
}

func (v *MigrateApplyJSON) DryRunHeader(id string) {
	v.output("dry_run_header", map[string]string{"id": id})
}

func (v *MigrateApplyJSON) Diff(filename string, before, after []byte) {
	v.output("diff", map[string]string{
		"filename": filename,
		"before":   string(before),
		"after":    string(after),
	})
}

func (v *MigrateApplyJSON) DryRunSummary(changes int, files int) {
	v.output("dry_run_summary", map[string]int{
		"changes": changes,
		"files":   files,
	})
}

func (v *MigrateApplyJSON) StepHeader(index, total int, sm migrate.SubMigration) {
	v.output("step_header", map[string]any{
		"index":       index,
		"total":       total,
		"name":        sm.Name,
		"description": sm.Description,
	})
}

func (v *MigrateApplyJSON) StepPrompt(_ *terminal.Streams) byte {
	// JSON mode doesn't support interactive prompts; default to 'n'.
	return 'n'
}

func (v *MigrateApplyJSON) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

// ---------------------------------------------------------------------------
// Diff helpers
// ---------------------------------------------------------------------------

// splitLines splits content into lines, handling the trailing newline gracefully.
func splitLines(data []byte) []string {
	if len(data) == 0 {
		return nil
	}
	s := string(data)
	lines := strings.Split(s, "\n")
	// Remove trailing empty element caused by a final newline.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// diffOpKind represents the type of a diff operation.
type diffOpKind int

const (
	diffKeep   diffOpKind = iota // Unchanged line
	diffRemove                   // Line removed from before
	diffAdd                      // Line added in after
)

// diffOp is a single line-level diff operation.
type diffOp struct {
	kind diffOpKind
	text string
}

// computeDiffOps produces a sequence of keep/remove/add operations by walking
// the before and after lines against their LCS.
func computeDiffOps(before, after []string) []diffOp {
	lcs := computeLCS(before, after)
	var ops []diffOp

	bi, ai, li := 0, 0, 0
	for bi < len(before) || ai < len(after) {
		if li < len(lcs) && bi < len(before) && before[bi] == lcs[li] &&
			ai < len(after) && after[ai] == lcs[li] {
			ops = append(ops, diffOp{kind: diffKeep, text: before[bi]})
			bi++
			ai++
			li++
		} else if bi < len(before) && (li >= len(lcs) || before[bi] != lcs[li]) {
			ops = append(ops, diffOp{kind: diffRemove, text: before[bi]})
			bi++
		} else if ai < len(after) && (li >= len(lcs) || after[ai] != lcs[li]) {
			ops = append(ops, diffOp{kind: diffAdd, text: after[ai]})
			ai++
		}
	}
	return ops
}

// computeLCS returns the longest common subsequence of two string slices.
func computeLCS(a, b []string) []string {
	m, n := len(a), len(b)
	// Build the LCS table.
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] >= dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}

	// Backtrack to find the LCS.
	lcs := make([]string, 0, dp[m][n])
	i, j := m, n
	for i > 0 && j > 0 {
		if a[i-1] == b[j-1] {
			lcs = append(lcs, a[i-1])
			i--
			j--
		} else if dp[i-1][j] >= dp[i][j-1] {
			i--
		} else {
			j--
		}
	}

	// Reverse the LCS (built backwards).
	for left, right := 0, len(lcs)-1; left < right; left, right = left+1, right-1 {
		lcs[left], lcs[right] = lcs[right], lcs[left]
	}

	return lcs
}
