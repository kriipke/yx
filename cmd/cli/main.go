package main

import (
	"fmt"
	"os"

	"github.com/kriipke/yiff/internal/adapters/cli"
)

func main() {
	if err := cli.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
