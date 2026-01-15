package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "xml2excel",
	Short: "Convert XML files to Excel format",
	Long: `xml2excel is a high-performance CLI tool for converting large XML files to Excel format.
It uses buffered streaming to efficiently handle files with 50k-100k rows.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
