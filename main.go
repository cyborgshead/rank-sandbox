package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)


func main() {

	var rootCmd = &cobra.Command{
		Use:   "cyberrank",
		Short: "Allows you to run benchmarks of cyberrank",
	}

	rootCmd.AddCommand(RunBenchCPUCmd())
	rootCmd.AddCommand(RunBenchGPUCmd())
	rootCmd.AddCommand(RunGenGraphCmd())
	rootCmd.AddCommand(RunDiffCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
