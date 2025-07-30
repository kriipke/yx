package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
	"github.com/kriipke/yiff/pkg/differ"
)

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

func visibleLen(s string) int {
	return len(differ.ansiRegexp.ReplaceAllString(s, ""))
}

func spaces(n int) string {
	if n <= 0 {
		return ""
	}
	return fmt.Sprintf("%*s", n, "")
}

func main() {
	var outputFormat string
	flag.StringVar(&outputFormat, "o", "shell", "Output format: 'shell' (default), 'yaml'")
	flag.Parse()

	if flag.NArg() < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <file1.yaml> <file2.yaml>\n", filepath.Base(os.Args[0]))
		os.Exit(1)
	}
	fileA, fileB := flag.Arg(0), flag.Arg(1)

	yamlA, err := differ.LoadYAMLMap(fileA)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load %s: %v\n", fileA, err)
		os.Exit(1)
	}
	yamlB, err := differ.LoadYAMLMap(fileB)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load %s: %v\n", fileB, err)
		os.Exit(1)
	}

	diffs := differ.DiffYAML(yamlA, yamlB)

	switch outputFormat {
	case "yaml":
		out := map[string]interface{}{
			"variables": diffs,
		}
		yamlBytes, err := yaml.Marshal(out)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling YAML: %v\n", err)
			os.Exit(2)
		}
		fmt.Print(string(yamlBytes))
		return
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
			switch d.Change {
			case "MODIFIED":
				color = "\x1b[33;1m" // yellow
			case "ADDED":
				color = "\x1b[32;1m" // green
			case "REMOVED":
				color = "\x1b[31;1m" // red
			}
			if colored {
				fmt.Printf("%s%s%s%s  %s%s%s  →  %s%s%s\n",
					"\x1b[1m", varName, "\x1b[0m",
					spaces(maxVarLen-visibleLen(varName)),
					left, spaces(maxDefaultLen-visibleLen(left)), "",
					color, right, "\x1b[0m")
			} else {
				fmt.Printf("%-*s  %-*s  →  %s\n",
					maxVarLen, varName,
					maxDefaultLen, left, right)
			}
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown output format: %s. Supported: 'shell', 'yaml'\n", outputFormat)
		os.Exit(1)
	}
}
