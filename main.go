package main

import (
	"fmt"
	"os"

	"github.com/nicholas-fedor/go-fast-cli/core"
	"github.com/spf13/cobra"
)

var version = "v0.1.0"

var RootCmd = &cobra.Command{
	Use:   "fast",
	Short: "Test your internet speed using fast.com",
	Long:  `fast-cli tests your internet download and upload speed using Netflix's fast.com.`,
	Run:   run,
}

func main() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.Flags().BoolP("json", "j", false, "Output as JSON")
	RootCmd.Flags().BoolP("simple", "s", false, "Only display the result, no progress bar")
}

func run(cmd *cobra.Command, args []string) {
	opts := core.Options{
		JSON:   cmd.Flags().Changed("json"),
		Simple: cmd.Flags().Changed("simple"),
	}

	result, err := core.RunTest(opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	if opts.JSON {
		fmt.Println(result.JSON())
	} else {
		fmt.Println(result.String(true))
	}
}
