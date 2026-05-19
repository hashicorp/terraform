// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/cli"
)

func newStateSearchCommand(t *testing.T) (*StateSearchCommand, *cli.MockUi) {
	t.Helper()
	ui := cli.NewMockUi()
	return &StateSearchCommand{Meta: Meta{Ui: ui}}, ui
}

func TestStateSearch_Help(t *testing.T) {
	c, _ := newStateSearchCommand(t)
	if !strings.Contains(c.Help(), "terraform state find") {
		t.Fatal("help missing usage")
	}
	if !strings.Contains(c.Help(), "keyword") {
		t.Fatal("help missing keyword documentation")
	}
}

func TestStateSearch_Synopsis(t *testing.T) {
	c, _ := newStateSearchCommand(t)
	if c.Synopsis() == "" {
		t.Fatal("synopsis empty")
	}
}

func TestStateSearch_MissingKeyword(t *testing.T) {
	c, ui := newStateSearchCommand(t)
	if c.Run([]string{}) != 1 {
		t.Fatal("expected error when no keyword provided")
	}
	if !strings.Contains(ui.ErrorWriter.String(), "keyword") {
		t.Fatal("expected error message about missing keyword")
	}
}

func TestStateSearch_NotFound(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "terraform.tfstate")
	stateContent := `{
		"version": 4,
		"terraform_version": "1.0.0",
		"serial": 1,
		"lineage": "test",
		"outputs": {},
		"resources": [
			{
				"type": "aws_instance",
				"name": "web",
				"instances": [
					{
						"attributes": {
							"id": "i-1234567890abcdef0",
							"instance_type": "t2.micro"
						}
					}
				]
			}
		]
	}`
	if err := os.WriteFile(statePath, []byte(stateContent), 0644); err != nil {
		t.Fatal(err)
	}

	c, ui := newStateSearchCommand(t)
	if c.Run([]string{"-state=" + statePath, "nonexistent"}) != 0 {
		t.Fatal("expected 0 exit code for no matches")
	}
	if !strings.Contains(ui.OutputWriter.String(), "No resources found") {
		t.Fatal("expected no matches message")
	}
}

func TestStateSearch_MatchByType(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "terraform.tfstate")
	stateContent := `{
		"version": 4,
		"terraform_version": "1.0.0",
		"serial": 1,
		"lineage": "test",
		"outputs": {},
		"resources": [
			{
				"type": "aws_instance",
				"name": "web",
				"instances": [
					{
						"attributes": {
							"id": "i-1234567890abcdef0",
							"instance_type": "t2.micro"
						}
					}
				]
			}
		]
	}`
	if err := os.WriteFile(statePath, []byte(stateContent), 0644); err != nil {
		t.Fatal(err)
	}

	c, ui := newStateSearchCommand(t)
	if c.Run([]string{"-state=" + statePath, "aws_instance"}) != 0 {
		t.Fatalf("expected success: %s", ui.ErrorWriter.String())
	}
	output := ui.OutputWriter.String()
	if !strings.Contains(output, "aws_instance") && !strings.Contains(output, "Found") {
		t.Fatalf("expected match result in output: %s", output)
	}
}

func TestStateSearch_MatchByName(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "terraform.tfstate")
	stateContent := `{
		"version": 4,
		"terraform_version": "1.0.0",
		"serial": 1,
		"lineage": "test",
		"resources": [
			{
				"type": "aws_instance",
				"name": "webserver",
				"instances": [
					{
						"attributes": {
							"id": "i-1234567890abcdef0"
						}
					}
				]
			}
		]
	}`
	if err := os.WriteFile(statePath, []byte(stateContent), 0644); err != nil {
		t.Fatal(err)
	}

	c, ui := newStateSearchCommand(t)
	if c.Run([]string{"-state=" + statePath, "webserver"}) != 0 {
		t.Fatalf("expected success: %s", ui.ErrorWriter.String())
	}
	if !strings.Contains(ui.OutputWriter.String(), "Found") {
		t.Fatal("expected to find resource by name")
	}
}

func TestStateSearch_JSONFormat(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "terraform.tfstate")
	stateContent := `{
		"version": 4,
		"terraform_version": "1.0.0",
		"serial": 1,
		"lineage": "test",
		"resources": [
			{
				"type": "aws_instance",
				"name": "web",
				"instances": [
					{
						"attributes": {
							"id": "i-1234567890abcdef0"
						}
					}
				]
			}
		]
	}`
	if err := os.WriteFile(statePath, []byte(stateContent), 0644); err != nil {
		t.Fatal(err)
	}

	c, ui := newStateSearchCommand(t)
	if c.Run([]string{"-state=" + statePath, "-format=json", "aws"}) != 0 {
		t.Fatalf("expected success: %s", ui.ErrorWriter.String())
	}

	output := ui.OutputWriter.String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("expected valid JSON output: %v", err)
	}
	if _, ok := result["results"]; !ok {
		t.Fatal("expected results field in JSON")
	}
	if _, ok := result["count"]; !ok {
		t.Fatal("expected count field in JSON")
	}
}

func TestStateSearch_ExactMatch(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "terraform.tfstate")
	stateContent := `{
		"version": 4,
		"terraform_version": "1.0.0",
		"serial": 1,
		"lineage": "test",
		"resources": [
			{
				"type": "aws_instance",
				"name": "web",
				"instances": [
					{
						"attributes": {
							"id": "i-1234567890abcdef0"
						}
					}
				]
			}
		]
	}`
	if err := os.WriteFile(statePath, []byte(stateContent), 0644); err != nil {
		t.Fatal(err)
	}

	c, ui := newStateSearchCommand(t)
	if c.Run([]string{"-state=" + statePath, "-exact", "aws_instance"}) != 0 {
		t.Fatalf("expected success: %s", ui.ErrorWriter.String())
	}
	if !strings.Contains(ui.OutputWriter.String(), "Found") {
		t.Fatal("expected exact match to find resource")
	}
}

func TestStateSearch_CaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "terraform.tfstate")
	stateContent := `{
		"version": 4,
		"terraform_version": "1.0.0",
		"serial": 1,
		"lineage": "test",
		"resources": [
			{
				"type": "aws_instance",
				"name": "WebServer",
				"instances": [
					{
						"attributes": {
							"id": "i-1234567890abcdef0"
						}
					}
				]
			}
		]
	}`
	if err := os.WriteFile(statePath, []byte(stateContent), 0644); err != nil {
		t.Fatal(err)
	}

	c, ui := newStateSearchCommand(t)
	if c.Run([]string{"-state=" + statePath, "webserver"}) != 0 {
		t.Fatalf("expected success: %s", ui.ErrorWriter.String())
	}
	if !strings.Contains(ui.OutputWriter.String(), "Found") {
		t.Fatal("expected case-insensitive search to work")
	}
}
