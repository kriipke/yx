package cli

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"

	"github.com/kriipke/yiff/internal/core"
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
	if err := fs.Parse(args); err != nil {
		return err
	}
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

	switch *outputFormat {
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
		return fmt.Errorf("Unknown output format: %s. Supported: 'shell', 'yaml'", *outputFormat)
	}
	return nil
}
