// Package manualdocs maps rungrad command pages into Asana's family-based manual layout.
package manualdocs

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/vincentsch/rungrad"
	"github.com/vincentsch/rungrad/docsgen"
)

var commandLinkPattern = regexp.MustCompile(`\(asana_([^)]+)\.md\)`)

// CheckResult reports drift between generated command docs and committed pages.
type CheckResult struct {
	Missing  []string
	Stale    []string
	Orphaned []string
}

// OK reports whether the committed manual matches the live command tree.
func (result CheckResult) OK() bool {
	return len(result.Missing) == 0 && len(result.Stale) == 0 && len(result.Orphaned) == 0
}

// String formats drift in deterministic groups for CLI and test failures.
func (result CheckResult) String() string {
	var lines []string
	if len(result.Missing) > 0 {
		lines = append(lines, "missing: "+strings.Join(result.Missing, ", "))
	}
	if len(result.Stale) > 0 {
		lines = append(lines, "stale: "+strings.Join(result.Stale, ", "))
	}
	if len(result.Orphaned) > 0 {
		lines = append(lines, "orphaned: "+strings.Join(result.Orphaned, ", "))
	}
	return strings.Join(lines, "\n")
}

// Generate builds one page per top-level command while retaining docsgen content.
func Generate(app *rungrad.App) map[string]string {
	generated := docsgen.Generate(app)
	grouped := map[string][]string{}
	for path := range generated {
		if !strings.HasPrefix(path, "asana_") || path == "asana.md" {
			continue
		}
		commandPath := strings.TrimSuffix(strings.TrimPrefix(path, "asana_"), ".md")
		top := strings.SplitN(commandPath, "_", 2)[0]
		grouped[top] = append(grouped[top], path)
	}

	docs := make(map[string]string, len(grouped)+1)
	for top, paths := range grouped {
		sort.Strings(paths)
		var page strings.Builder
		for i, path := range paths {
			if i > 0 {
				page.WriteString("\n")
			}
			depth := strings.Count(strings.TrimSuffix(strings.TrimPrefix(path, "asana_"), ".md"), "_") + 1
			page.WriteString(nestHeadings(generated[path], depth-1))
		}
		docs[top+".md"] = page.String()
	}
	docs["index.md"] = mapIndexLinks(generated["index.md"])
	return docs
}

func nestHeadings(page string, levels int) string {
	if levels == 0 {
		return page
	}
	prefix := strings.Repeat("#", levels)
	lines := strings.SplitAfter(page, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "#") {
			lines[i] = prefix + line
		}
	}
	return strings.Join(lines, "")
}

func mapIndexLinks(index string) string {
	return commandLinkPattern.ReplaceAllStringFunc(index, func(link string) string {
		match := commandLinkPattern.FindStringSubmatch(link)
		path := match[1]
		top := strings.SplitN(path, "_", 2)[0]
		anchor := "asana-" + strings.ReplaceAll(path, "_", "-")
		return fmt.Sprintf("(%s.md#%s)", top, anchor)
	})
}

// Check compares generated pages with the committed family-based manual.
func Check(app *rungrad.App, dir string) (CheckResult, error) {
	want := Generate(app)
	var result CheckResult
	for path, content := range want {
		got, err := os.ReadFile(filepath.Join(dir, path))
		switch {
		case errors.Is(err, os.ErrNotExist):
			result.Missing = append(result.Missing, path)
		case err != nil:
			return CheckResult{}, err
		case string(got) != content:
			result.Stale = append(result.Stale, path)
		}
	}
	if err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.Type().IsRegular() || filepath.Ext(path) != ".md" {
			return nil
		}
		relative, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if _, ok := want[filepath.ToSlash(relative)]; !ok {
			result.Orphaned = append(result.Orphaned, filepath.ToSlash(relative))
		}
		return nil
	}); err != nil && !errors.Is(err, os.ErrNotExist) {
		return CheckResult{}, err
	}
	sort.Strings(result.Missing)
	sort.Strings(result.Stale)
	sort.Strings(result.Orphaned)
	return result, nil
}

// Write regenerates all mapped command pages and removes obsolete markdown pages.
func Write(app *rungrad.App, dir string) error {
	want := Generate(app)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	paths := make([]string, 0, len(want))
	for path := range want {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		if err := os.WriteFile(filepath.Join(dir, path), []byte(want[path]), 0o644); err != nil {
			return err
		}
	}
	result, err := Check(app, dir)
	if err != nil {
		return err
	}
	for _, path := range result.Orphaned {
		if err := os.Remove(filepath.Join(dir, path)); err != nil {
			return err
		}
	}
	return nil
}
