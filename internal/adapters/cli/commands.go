package cli

import (
    "fmt"
    "os"
    "github.com/spf13/cobra"
    "github.com/kriipke/yiff/internal/app"
)

func Execute() {
    var rootCmd = &cobra.Command{
        Use:   "yiff [file1] [file2]",
        Short: "Diff two YAML files",
        Args:  cobra.ExactArgs(2),
        Run: func(cmd *cobra.Command, args []string) {
            f1, _ := os.ReadFile(args[0])
            f2, _ := os.ReadFile(args[1])
            result, _ := app.DiffYAML(f1, f2)
            fmt.Println(result)
        },
    }
    rootCmd.Execute()
}
