package command

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/hashicorp/cli"
)

var cmdLogger = log.New(os.Stdout, "CommandDocsLog: ", 0)

// CommandDocs is a Command implementation that provides access to provider
// documentation by fetching and organizing provider docs from their repositories.
type CommandDocs struct {
	Meta
}

type ProviderDetails struct {
	Source      string
	Version     string
	RepoOwner   string
	RepoName    string
	DocsVersion string
}

func (c *CommandDocs) Run(args []string) int {
	// Process the command-line arguments
	args = c.Meta.process(args)
	if c.Ui == nil {
		c.Ui = &cli.BasicUi{
			Reader:      os.Stdin,
			Writer:      os.Stdout,
			ErrorWriter: os.Stderr,
		}
	}
	// Set up our custom command flags
	cmdFlags := c.Meta.extendedFlagSet("docs")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }

	if err := cmdFlags.Parse(args); err != nil {
		return c.showError(fmt.Errorf("Error parsing command line flags: %s", err))
	}

	args = cmdFlags.Args()
	if len(args) < 1 {
		return c.showError(fmt.Errorf("The docs command expects a provider name as an argument"))
	}

	providerName := args[0]
	cmdLogger.Printf("Fetching documentation for provider: %s", providerName)

	// Get provider details from lock file
	providerDetails, err := c.getProviderFromLockFile(providerName)
	if err != nil {
		return c.showError(err)
	}
	cmdLogger.Printf("Provider details found: owner=%s, repo=%s",
		providerDetails.RepoOwner, providerDetails.RepoName)

	docsDir := filepath.Join(".terraform", "docs", providerName)
	cmdLogger.Printf("Using documentation directory: %s", docsDir)

	if err := c.ensureDirectory(docsDir); err != nil {
		return c.showError(err)
	}

	// Check if docs need updating
	needsUpdate := true
	if c.isDocumentationCached(docsDir) {
		if c.checkDocsVersion(docsDir, providerDetails) {
			needsUpdate = false
		} else {
			cmdLogger.Printf("Documentation version mismatch, updating...")
			if err := c.cleanupOldDocs(docsDir); err != nil {
				return c.showError(err)
			}
		}
	}

	if needsUpdate {
		cmdLogger.Printf("Documentation not cached, cloning repository...")
		if err := c.cloneAndOrganizeDocs(providerDetails, docsDir); err != nil {
			return c.showError(err)
		}
		if err := c.saveDocsVersion(docsDir, providerDetails); err != nil {
			cmdLogger.Printf("Warning: Failed to save docs version: %s", err)
		}
	}

	// Validate documentation structure
	if err := c.validateDocumentation(docsDir); err != nil {
		return c.showError(err)
	}

	// Handle command options
	if len(args) > 1 {
		if args[1] == "-l" {
			return c.handleListCommand(docsDir)
		}
		return c.handleResourceCommand(docsDir, args)
	}

	c.Ui.Error("Please specify either -l to list resources or provide a resource name with -d/-r flag")
	return 1
}

func (c *CommandDocs) Synopsis() string {
	return "Shows provider documentation for resources and data sources"
}

func (c *CommandDocs) handleListCommand(docsDir string) int {
	cmdLogger.Printf("Listing resources from: %s", docsDir)
	isModern, isLegacy := c.analyzeDocStructure(docsDir)

	resources := make([]string, 0)
	dataSources := make([]string, 0)

	if isModern {
		modernResources, modernDataSources := c.listModernDocs(docsDir)
		resources = append(resources, modernResources...)
		dataSources = append(dataSources, modernDataSources...)
	}

	if isLegacy {
		legacyResources, legacyDataSources := c.listLegacyDocs(docsDir)
		resources = append(resources, legacyResources...)
		dataSources = append(dataSources, legacyDataSources...)
	}

	// Print resources
	if len(resources) > 0 {
		c.Ui.Output("\nResources:")
		sort.Strings(resources)
		for _, resource := range resources {
			c.Ui.Output(fmt.Sprintf("* %s", resource))
		}
	}

	// Print data sources
	if len(dataSources) > 0 {
		c.Ui.Output("\nData Sources:")
		sort.Strings(dataSources)
		for _, dataSource := range dataSources {
			c.Ui.Output(fmt.Sprintf("* %s", dataSource))
		}
	}

	cmdLogger.Printf("Found %d resources and %d data sources",
		len(resources), len(dataSources))
	return 0
}

func (c *CommandDocs) Help() string {
	return `
Usage: terraform docs <provider> [options] [resource_name] [search_keyword]

Shows provider documentation for resources and data sources.

Options:
  -l              List all available resources and data sources
  -r              Specify resource type documentation
  -d              Specify data source type documentation
  search_keyword  Optional keyword to search within the documentation

Examples:
  terraform docs aws -l                    # List all AWS provider resources
  terraform docs random random_id -r       # Show full documentation for random_id resource
  terraform docs aws instance -r 'Example' # Show example section for AWS instance resource
`
}

func (c *CommandDocs) handleResourceCommand(docsDir string, args []string) int {
	resourceName := args[1]
	var resourceType string
	var searchKeyword string

	// Check for resource type flag
	if len(args) > 2 {
		switch args[2] {
		case "-d":
			resourceType = "data"
			cmdLogger.Printf("Looking for data source: %s", resourceName)
		case "-r":
			resourceType = "resource"
			cmdLogger.Printf("Looking for resource: %s", resourceName)
		default:
			return c.showError(fmt.Errorf("Invalid flag. Please use -d for data source or -r for resource"))
		}

		// Check for search keyword
		if len(args) > 3 {
			searchKeyword = args[3]
			if len(searchKeyword) > 0 && (searchKeyword[0] == '\'' || searchKeyword[0] == '"') {
				searchKeyword = searchKeyword[1 : len(searchKeyword)-1]
			}
			cmdLogger.Printf("Search keyword provided: %s", searchKeyword)
		}
	}

	return c.showResourceDoc(docsDir, resourceName, resourceType, searchKeyword)
}

func (c *CommandDocs) listModernDocs(docsDir string) ([]string, []string) {
	var resources, dataSources []string

	modernPaths := map[string]*[]string{
		filepath.Join(docsDir, "docs", "resources"):    &resources,
		filepath.Join(docsDir, "docs", "data-sources"): &dataSources,
	}

	for path, slice := range modernPaths {
		cmdLogger.Printf("Checking modern path: %s", path)
		err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				cmdLogger.Printf("Error accessing %s: %s", path, err)
				return nil
			}
			if !info.IsDir() && c.isDocumentationFile(info.Name()) {
				name := strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))
				*slice = append(*slice, name)
				cmdLogger.Printf("Found: %s", name)
			}
			return nil
		})
		if err != nil {
			cmdLogger.Printf("Error walking path %s: %s", path, err)
		}
	}

	return resources, dataSources
}

func (c *CommandDocs) listLegacyDocs(docsDir string) ([]string, []string) {
	var resources, dataSources []string

	legacyPaths := map[string]*[]string{
		filepath.Join(docsDir, "website", "docs", "r"): &resources,
		filepath.Join(docsDir, "website", "docs", "d"): &dataSources,
	}

	for path, slice := range legacyPaths {
		cmdLogger.Printf("Checking legacy path: %s", path)
		err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				cmdLogger.Printf("Error accessing %s: %s", path, err)
				return nil
			}
			if !info.IsDir() && c.isDocumentationFile(info.Name()) {
				name := strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))
				name = strings.TrimSuffix(name, ".html")
				*slice = append(*slice, name)
				cmdLogger.Printf("Found: %s", name)
			}
			return nil
		})
		if err != nil {
			cmdLogger.Printf("Error walking path %s: %s", path, err)
		}
	}

	return resources, dataSources
}

func (c *CommandDocs) showResourceDoc(docsDir, resourceName, resourceType, searchKeyword string) int {
	var paths []string

	isModern, isLegacy := c.analyzeDocStructure(docsDir)

	switch resourceType {
	case "data":
		if isModern {
			paths = append(paths, filepath.Join(docsDir, "docs", "data-sources", resourceName+".md"))
		}
		if isLegacy {
			paths = append(paths,
				filepath.Join(docsDir, "website", "docs", "d", resourceName+".html.md"),
				filepath.Join(docsDir, "website", "docs", "d", resourceName+".html.markdown"))
		}
	case "resource":
		if isModern {
			paths = append(paths, filepath.Join(docsDir, "docs", "resources", resourceName+".md"))
		}
		if isLegacy {
			paths = append(paths,
				filepath.Join(docsDir, "website", "docs", "r", resourceName+".html.md"),
				filepath.Join(docsDir, "website", "docs", "r", resourceName+".html.markdown"))
		}
	default:
		if isModern {
			paths = append(paths,
				filepath.Join(docsDir, "docs", "resources", resourceName+".md"),
				filepath.Join(docsDir, "docs", "data-sources", resourceName+".md"))
		}
		if isLegacy {
			paths = append(paths,
				filepath.Join(docsDir, "website", "docs", "r", resourceName+".html.md"),
				filepath.Join(docsDir, "website", "docs", "d", resourceName+".html.md"),
				filepath.Join(docsDir, "website", "docs", "r", resourceName+".html.markdown"),
				filepath.Join(docsDir, "website", "docs", "d", resourceName+".html.markdown"))
		}
	}

	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err == nil {
			cmdLogger.Printf("Found documentation at: %s", path)

			if searchKeyword != "" {
				return c.handleDocumentSearch(content, searchKeyword, resourceName, resourceType)
			}

			c.Ui.Output(fmt.Sprintf("\n%s", string(content)))

			return 0
		}
	}

	return c.showError(fmt.Errorf("Documentation not found for %s: %s",
		resourceType, resourceName))
}

func (c *CommandDocs) handleDocumentSearch(content []byte, searchKeyword, resourceName, resourceType string) int {
	cmdLogger.Printf("Searching for section with keyword: %s", searchKeyword)
	lines := strings.Split(string(content), "\n")
	section := c.extractSection(lines, searchKeyword)

	if section != "" {
		c.Ui.Output(fmt.Sprintf("\n%s", section))
		return 0
	}

	return c.showError(fmt.Errorf("No section found matching keyword: %s", searchKeyword))
}

func (c *CommandDocs) extractSection(lines []string, keyword string) string {
	sectionPattern := fmt.Sprintf(`^#+\s*.*%s.*`, regexp.QuoteMeta(keyword))
	sectionRegex, err := regexp.Compile(sectionPattern)
	if err != nil {
		cmdLogger.Printf("Error compiling regex: %s", err)
		return ""
	}

	var extractedLines []string
	capturing := false
	currentHeadingLevel := 0

	for _, line := range lines {
		headingLevel := 0
		for i := 0; i < len(line); i++ {
			if line[i] == '#' {
				headingLevel++
			} else {
				break
			}
		}

		if sectionRegex.MatchString(line) {
			if capturing {
				break
			}
			capturing = true
			currentHeadingLevel = headingLevel
			extractedLines = append(extractedLines, line)
		} else if capturing {
			if headingLevel > 0 && headingLevel <= currentHeadingLevel {
				break
			}
			extractedLines = append(extractedLines, line)
		}
	}

	return c.cleanSection(strings.Join(extractedLines, "\n"))
}

func (c *CommandDocs) getProviderFromLockFile(providerName string) (*ProviderDetails, error) {
	content, err := os.ReadFile(".terraform.lock.hcl")
	if err != nil {
		return nil, fmt.Errorf("failed to read lock file: %w", err)
	}

	providerRegex := regexp.MustCompile(
		fmt.Sprintf(`provider "([^"]+/%s)" {[^}]*version\s*=\s*"([^"]+)"`,
			providerName))

	matches := providerRegex.FindStringSubmatch(string(content))
	if len(matches) < 3 {
		return nil, fmt.Errorf("provider %s not found in lock file", providerName)
	}

	parts := strings.Split(matches[1], "/")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid provider source format: %s", matches[1])
	}

	details := &ProviderDetails{
		Source:      matches[1],
		Version:     matches[2],
		RepoOwner:   parts[1],
		RepoName:    parts[2],
		DocsVersion: "main",
	}

	// Handle special cases
	if details.RepoOwner == "ibm" {
		details.RepoOwner = "IBM-Cloud"
	}

	// Add terraform-provider- prefix if not present
	if !strings.HasPrefix(details.RepoName, "terraform-provider-") {
		details.RepoName = "terraform-provider-" + details.RepoName
	}

	return details, nil
}

func (c *CommandDocs) cloneAndOrganizeDocs(details *ProviderDetails, docsDir string) error {
	// Create temporary directory for cloning
	tmpDir, err := os.MkdirTemp("", "terraform-provider-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	cmdLogger.Printf("Created temporary directory: %s", tmpDir)
	defer os.RemoveAll(tmpDir)

	// Construct repository URL
	repoURL := fmt.Sprintf("https://github.com/%s/%s.git",
		details.RepoOwner, details.RepoName)
	cmdLogger.Printf("Cloning from: %s", repoURL)

	// Clone repository
	cmd := exec.Command("git", "clone", "--depth", "1", repoURL, tmpDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to clone repository: %s: %s", err, string(output))
	}
	cmdLogger.Printf("Successfully cloned repository")

	// Copy documentation
	if err := c.copyDocumentation(tmpDir, docsDir); err != nil {
		return fmt.Errorf("failed to copy documentation: %w", err)
	}

	return nil
}

func (c *CommandDocs) copyDocumentation(srcDir, destDir string) error {
	docsPaths := []struct {
		src  string
		dest string
	}{
		{filepath.Join(srcDir, "docs"), filepath.Join(destDir, "docs")},
		{filepath.Join(srcDir, "website", "docs"), filepath.Join(destDir, "website", "docs")},
	}

	foundDocs := false
	for _, path := range docsPaths {
		cmdLogger.Printf("Checking for docs in: %s", path.src)
		if _, err := os.Stat(path.src); err == nil {
			cmdLogger.Printf("Found documentation in %s", path.src)
			if err := c.copyDir(path.src, path.dest); err != nil {
				cmdLogger.Printf("Error copying docs from %s: %s", path.src, err)
				continue
			}
			foundDocs = true
			cmdLogger.Printf("Copied documentation to: %s", path.dest)
		}
	}

	if !foundDocs {
		return fmt.Errorf("no documentation found in repository")
	}

	return nil
}

func (c *CommandDocs) copyDir(src, dst string) error {
	if err := c.ensureDirectory(dst); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		sourcePath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dst, entry.Name())

		fileInfo, err := entry.Info()
		if err != nil {
			return err
		}

		if fileInfo.IsDir() {
			if err := c.copyDir(sourcePath, destPath); err != nil {
				return err
			}
		} else {
			if err := c.copyFile(sourcePath, destPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *CommandDocs) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func (c *CommandDocs) cleanSection(section string) string {
	lines := strings.Split(section, "\n")

	// Trim empty lines from start
	start := 0
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}

	// Trim empty lines from end
	end := len(lines) - 1
	for end >= start && strings.TrimSpace(lines[end]) == "" {
		end--
	}

	if start <= end {
		return strings.Join(lines[start:end+1], "\n")
	}
	return ""
}

// Helper functions

func (c *CommandDocs) checkDocsVersion(docsDir string, details *ProviderDetails) bool {
	versionFile := filepath.Join(docsDir, ".version")
	content, err := os.ReadFile(versionFile)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(content)) == details.Version
}

func (c *CommandDocs) saveDocsVersion(docsDir string, details *ProviderDetails) error {
	versionFile := filepath.Join(docsDir, ".version")
	return os.WriteFile(versionFile, []byte(details.Version), 0644)
}

func (c *CommandDocs) analyzeDocStructure(docsDir string) (isModern, isLegacy bool) {
	modernPath := filepath.Join(docsDir, "docs")
	legacyPath := filepath.Join(docsDir, "website", "docs")

	if _, err := os.Stat(modernPath); err == nil {
		isModern = true
	}
	if _, err := os.Stat(legacyPath); err == nil {
		isLegacy = true
	}
	return
}

func (c *CommandDocs) ensureDirectory(path string) error {
	return os.MkdirAll(path, os.ModePerm)
}

func (c *CommandDocs) isDocumentationCached(docsDir string) bool {
	for _, subDir := range []string{"docs", "website/docs"} {
		path := filepath.Join(docsDir, subDir)
		if _, err := os.Stat(path); err == nil {
			entries, err := os.ReadDir(path)
			if err == nil && len(entries) > 0 {
				cmdLogger.Printf("Found existing documentation in: %s", path)
				return true
			}
		}
	}
	cmdLogger.Printf("No valid documentation cache found in: %s", docsDir)
	return false
}

func (c *CommandDocs) isDocumentationFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".md" || ext == ".markdown" ||
		strings.HasSuffix(filename, ".html.md") ||
		strings.HasSuffix(filename, ".html.markdown")
}

func (c *CommandDocs) showError(err error) int {
	fmt.Fprintf(os.Stderr, "Error: %s\n", err)
	return 1
}

func (c *CommandDocs) outputSection(content, keyword string) {
	c.Ui.Output(fmt.Sprintf("=== Section matching '%s' ===\n", keyword))
	c.Ui.Output(fmt.Sprint(content))
	c.Ui.Output(fmt.Sprint("\n=== End of section ==="))
}

// cleanupOldDocs removes outdated documentation
func (c *CommandDocs) cleanupOldDocs(docsDir string) error {
	if err := os.RemoveAll(docsDir); err != nil {
		return fmt.Errorf("failed to clean up old documentation: %w", err)
	}
	return nil
}

// validateDocumentation checks if the documentation is valid and complete
func (c *CommandDocs) validateDocumentation(docsDir string) error {
	isModern, isLegacy := c.analyzeDocStructure(docsDir)
	if !isModern && !isLegacy {
		return fmt.Errorf("no valid documentation structure found in %s", docsDir)
	}
	return nil
}
