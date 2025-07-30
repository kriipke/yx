package main

import (
	"os"
	"github.com/kriipke/yiff/internal/adapters/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
