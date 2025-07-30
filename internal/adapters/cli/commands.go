package cli

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
	"github.com/kriipke/yiff/pkg/differ"
)

const (
	ColorReset  = "\x1b[0m"
	ColorGreen  = "\x1b[32;1m"
	ColorYellow = "\x1b[33;1m"
	ColorRed    = "\x1b[31;1m"
	ColorBold   = "\x1b[1m"
)

var ansiRegexp = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func visibleLen(s string) int {
	return len(ansiRegexp.ReplaceAllString(s, ""))
}

func spaces(n int) string {
	if n <= 0 {
		return ""
	}
	return fmt.Sprintf("%*s", n, "")
}

func colorEnabled() bool {
	term := os.Getenv("TERM")
	if term == "" || term == "dumb" {
		return false
	}
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func formatValue(val interface{}) string {
	switch v := val.(type) {
	case string:
		return fmt.Sprintf("“%s”", v)
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// Entrypoint for CLI
func Run(args []string) error {
	fs := flag.NewFlagSet("yiff", flag.ContinueOnError)
	outputFormat := fs.String("o", "shell", "Output format: 'shell' (default) or 'yaml'")
	fromRef := fs.String("from", "", "Git ref/tag/commit for the base comparison (used with --to and --path)")
	toRef := fs.String("to", "", "Git ref/tag/commit for the target comparison (used with --from and --path)")
	diffPath := fs.String("path", "", "Compare all yaml files under this path between --from and --to refs")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// --- New gitops diff mode ---
	if *fromRef != "" && *toRef != "" && *diffPath != "" {
		return runGitRefDirDiff(*fromRef, *toRef, *diffPath, *outputFormat)
	}

	// --- Default: file-vs-file diff ---
	return runFileDiff(fs, outputFormat)
}

// File-vs-file mode (unchanged)
func runFileDiff(fs *flag.FlagSet, outputFormat *string) error {
	if fs.NArg() < 2 {
		return errors.New(fmt.Sprintf("Usage: %s [flags] <file1.yaml> <file2.yaml>", filepath.Base(os.Args[0])))
	}
	fileA, fileB := fs.Arg(0), fs.Arg(1)
	dataA, err := ioutil.ReadFile(fileA)
	if err != nil {
		return fmt.Errorf("Failed to load %s: %w", fileA, err)
	}
	dataB, err := ioutil.ReadFile(fileB)
	if err != nil {
		return fmt.Errorf("Failed to load %s: %w", fileB, err)
	}
	yamlA, err := differ.LoadYAMLMap(dataA)
	if err != nil {
		return fmt.Errorf("Failed to parse %s: %w", fileA, err)
	}
	yamlB, err := differ.LoadYAMLMap(dataB)
	if err != nil {
		return fmt.Errorf("Failed to parse %s: %w", fileB, err)
	}
	diffs := differ.Diff(yamlA, yamlB)
	return printDiffs(diffs, *outputFormat)
}

// --- Directory diff between git refs ---
func runGitRefDirDiff(fromRef, toRef, relPath, outputFormat string) error {
	// 1. Get list of yaml files in path for both refs
	filesA, err := listGitFiles(fromRef, relPath)
	if err != nil {
		return fmt.Errorf("Failed to list files for ref %s: %w", fromRef, err)
	}
	filesB, err := listGitFiles(toRef, relPath)
	if err != nil {
		return fmt.Errorf("Failed to list files for ref %s: %w", toRef, err)
	}

	// 2. Build sets for matching, added, removed
	setA, setB := map[string]struct{}{}, map[string]struct{}{}
	for _, f := range filesA { setA[f] = struct{}{} }
	for _, f := range filesB { setB[f] = struct{}{} }

	allFiles := map[string]struct{}{}
	for f := range setA { allFiles[f] = struct{}{} }
	for f := range setB { allFiles[f] = struct{}{} }

	type perFileDiff struct {
		File  string
		Diffs []differ.VariableDiff
	}
	var changed []perFileDiff
	var added, removed []string

	for file := range allFiles {
		_, inA := setA[file]
		_, inB := setB[file]
		switch {
		case inA && inB:
			// Get file contents at both refs
			dataA, errA := gitShowFile(fromRef, file)
			dataB, errB := gitShowFile(toRef, file)
			if errA != nil || errB != nil {
				continue // skip erroring files
			}
			yamlA, err := differ.LoadYAMLMap(dataA)
			if err != nil { continue }
			yamlB, err := differ.LoadYAMLMap(dataB)
			if err != nil { continue }
			diffs := differ.Diff(yamlA, yamlB)
			if len(diffs) > 0 {
				changed = append(changed, perFileDiff{File: file, Diffs: diffs})
			}
		case inA && !inB:
			removed = append(removed, file)
		case !inA && inB:
			added = append(added, file)
		}
	}

	// 4. Print summary
	// fmt.Printf("YAML diff summary for %s between %s and %s:\n", relPath, fromRef, toRef)
	// if len(changed) > 0 {
	// 	fmt.Println("\nChanged files:")
	// 	for _, c := range changed {
	// 		fmt.Printf("  %s\n", c.File)
	// 		printDiffs(c.Diffs, outputFormat)
	// 	}
	// }
	// if len(added) > 0 {
	// 	fmt.Println("\nAdded files:")
	// 	for _, f := range added {
	// 		fmt.Printf("  %s\n", f)
	// 	}
	// }
	// if len(removed) > 0 {
	// 	fmt.Println("\nRemoved files:")
	// 	for _, f := range removed {
	// 		fmt.Printf("  %s\n", f)
	// 	}
	// }
	// if len(changed) == 0 && len(added) == 0 && len(removed) == 0 {
	// 	fmt.Println("No differences found.")
	// }
	return nil
}

// List all *.yaml and *.yml files in a given path at a specific git ref
func listGitFiles(ref, relPath string) ([]string, error) {
	cmd := exec.Command("git", "ls-tree", "-r", "--name-only", ref, relPath)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var files []string
	for _, f := range lines {
		if strings.HasSuffix(f, ".yaml") || strings.HasSuffix(f, ".yml") {
			files = append(files, f)
		}
	}
	return files, nil
}

// Get the contents of a file at a specific git ref
func gitShowFile(ref, file string) ([]byte, error) {
	cmd := exec.Command("git", "show", fmt.Sprintf("%s:%s", ref, file))
	return cmd.Output()
}

func printDiffs(diffs []differ.VariableDiff, outputFormat string) error {
	switch outputFormat {
	case "yaml":
		out := map[string]interface{}{
			"variables": diffs,
		}
		yamlBytes, err := yaml.Marshal(out)
		if err != nil {
			return fmt.Errorf("Error marshaling YAML: %w", err)
		}
		fmt.Print(string(yamlBytes))
	case "shell":
		colored := colorEnabled()
		maxVarLen := 0
		maxDefaultLen := 0
		for _, d := range diffs {
			varNameFmt := d.Name + ":"
			if l := visibleLen(varNameFmt); l > maxVarLen {
				maxVarLen = l
			}
			left := formatValue(d.Default)
			if l := visibleLen(left); l > maxDefaultLen {
				maxDefaultLen = l
			}
		}
		for _, d := range diffs {
			varName := d.Name + ":"
			left := formatValue(d.Default)
			right := formatValue(d.Value)
			color := ""
			switch d.Status {
			case "changed":
				color = ColorYellow
			case "added":
				color = ColorGreen
			case "removed":
				color = ColorRed
			}
			if colored {
				fmt.Printf("%s%s%s%s  %s%s%s  →  %s%s%s\n",
					ColorBold, varName, ColorReset,
					spaces(maxVarLen-visibleLen(varName)),
					left, spaces(maxDefaultLen-visibleLen(left)), "",
					color, right, ColorReset)
			} else {
				fmt.Printf("%-*s  %-*s  →  %s\n",
					maxVarLen, varName,
					maxDefaultLen, left, right)
			}
		}
	default:
		return fmt.Errorf("Unknown output format: %s. Supported: 'shell', 'yaml'", outputFormat)
	}
	return nil
}
