package cli

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
	"github.com/kriipke/yiff/pkg/differ"

	"github.com/spf13/cobra"
)

var (
	fromRef      string
	toRef        string
	diffPath     string
	outputFormat string
)

var rootCmd = &cobra.Command{
	Use:   "yiff [fileA.yaml] [fileB.yaml]",
	Short: "YAML diff tool for Helm values and GitOps",
	Long:  "Diff Helm values YAML files or directories between files or git refs. Supports per-variable, per-file, and per-directory comparisons.",
	Args:  cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Directory diff mode (git)
		if fromRef != "" && toRef != "" && diffPath != "" {
			return runGitRefDirDiff(fromRef, toRef, diffPath, outputFormat)
		}

		// Single file diff mode
		if len(args) == 2 {
			return runFileDiff(args[0], args[1], outputFormat)
		}

		// Print usage if not enough arguments
		cmd.Usage()
		return fmt.Errorf("invalid arguments")
	},
}

func init() {
	rootCmd.Flags().StringVar(&fromRef, "from", "", "Git ref/tag/commit for the base comparison (used with --to and --path)")
	rootCmd.Flags().StringVar(&toRef, "to", "", "Git ref/tag/commit for the target comparison (used with --from and --path)")
	rootCmd.Flags().StringVar(&diffPath, "path", "", "Compare all yaml files under this path between --from and --to refs")
	rootCmd.Flags().StringVarP(&outputFormat, "output", "o", "shell", "Output format: 'shell' (default) or 'yaml'")
}


// Entrypoint for Cobra
func Execute() error {
	return rootCmd.Execute()
}

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


// Update runFileDiff signature!
func runFileDiff(fileA, fileB, outputFormat string) error {
	dataA, err := os.ReadFile(fileA)
	if err != nil {
		return fmt.Errorf("Failed to load %s: %w", fileA, err)
	}
	dataB, err := os.ReadFile(fileB)
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
	return printDiffs(diffs, outputFormat)
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

// formatValuePlain returns string values as-is (no quotes).
func formatValuePlain(val interface{}) string {
	switch v := val.(type) {
	case string:
		return v
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", v)
	}
}

func colorize(s, color string) string {
	return color + s + ColorReset
}

func formatShellValue(val interface{}) string {
	if val == nil {
		return "NaN"
	}
	return fmt.Sprintf("%v", val)
}

// printDiffs outputs diff results in columns, shell, or yaml formats.
func printDiffs(diffs []differ.VariableDiff, outputFormat string) error {
	switch outputFormat {
	case "columns":
		type row struct {
			Key   string
			Old   string
			Arrow string
			New   string
		}
		rows := make([]row, 0, len(diffs))
		for _, d := range diffs {
			rows = append(rows, row{
				Key:   d.Name + ":",
				Old:   formatValuePlain(d.Default),
				Arrow: "->",
				New:   formatValuePlain(d.Value),
			})
		}
		// Find max width for each column
		maxKey, maxOld, maxArrow := 0, 0, 2
		for _, r := range rows {
			if l := len(r.Key); l > maxKey {
				maxKey = l
			}
			if l := len(r.Old); l > maxOld {
				maxOld = l
			}
		}
		// Print aligned
		for _, r := range rows {
			fmt.Printf("%-*s  %-*s  %-*s  %s\n",
				maxKey, r.Key, maxOld, r.Old, maxArrow, r.Arrow, r.New)
		}
		return nil

	case "yaml":
		out := map[string]interface{}{
			"variables": diffs,
		}
		yamlBytes, err := yaml.Marshal(out)
		if err != nil {
			return fmt.Errorf("Error marshaling YAML: %w", err)
		}
		fmt.Print(string(yamlBytes))
		return nil

	case "shell", "":
		for _, d := range diffs {
			left := formatShellValue(d.Default)
			right := formatShellValue(d.Value)
			var color string
			switch d.Status {
			case "added":
				color = ColorGreen
			case "removed":
				color = ColorRed
			case "changed":
				color = ColorYellow
			default:
				color = ""
			}
			// Bold the var name, color only the new value
			fmt.Printf("%s%s%s: %s → %s%s%s\n",
				ColorBold, d.Name, ColorReset,
				left,
				color, right, ColorReset,
			)
		}
		return nil
	default:
		return fmt.Errorf("Unknown output format: %s. Supported: 'shell', 'yaml', 'columns'", outputFormat)
	}
}
