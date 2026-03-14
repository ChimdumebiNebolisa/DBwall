package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "dbguard",
		Short: "AST-based Go CLI that reviews AI-generated PostgreSQL SQL and blocks unsafe operations before execution",
		Long: `dbguard parses PostgreSQL SQL using a real parser, applies configurable safety rules,
and returns an allow/warn/block decision. Built for developers, CI pipelines, and agent toolchains.`,
	}
	root.AddCommand(versionCmd())
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print dbguard version",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Version will be injected by internal/version
			fmt.Println("dbguard version 0.1.0")
			return nil
		},
	}
}
