package main

import (
	"fmt"
	"os"

	"github.com/ChimdumebiNebolisa/DBwall/internal/cli"
	"github.com/ChimdumebiNebolisa/DBwall/internal/version"
	"github.com/spf13/cobra"
)

var validFormats = map[string]bool{"human": true, "json": true, "sarif": true}

func validateFormat(format string) error {
	if !validFormats[format] {
		return fmt.Errorf("invalid --format %q: must be human, json, or sarif", format)
	}
	return nil
}

func main() {
	root := &cobra.Command{
		Use:   "dbguard",
		Short: "AST-based Go CLI that reviews AI-generated PostgreSQL SQL and blocks unsafe operations before execution",
		Long: `dbguard parses PostgreSQL SQL using a real parser, applies configurable safety rules,
and returns an allow/warn/block decision. Built for developers, CI pipelines, and agent toolchains.`,
	}
	root.AddCommand(versionCmd())
	root.AddCommand(reviewSQLCmd())
	root.AddCommand(reviewFileCmd())
	root.AddCommand(reviewFilesCmd())
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func reviewSQLCmd() *cobra.Command {
	var policyPath, format string
	cmd := &cobra.Command{
		Use:   "review-sql [SQL]",
		Short: "Review inline PostgreSQL SQL",
		Long:  `Parse and analyze the given SQL string. Use --policy to load a YAML policy file.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateFormat(format); err != nil {
				return err
			}
			code := cli.ReviewSQL(args[0], policyPath, format)
			os.Exit(code)
			return nil
		},
	}
	cmd.Flags().StringVar(&policyPath, "policy", "", "Path to policy YAML file")
	cmd.Flags().StringVar(&format, "format", "human", "Output format: human, json, or sarif")
	return cmd
}

func reviewFileCmd() *cobra.Command {
	var policyPath, format string
	cmd := &cobra.Command{
		Use:   "review-file [path]",
		Short: "Review a SQL file",
		Long:  `Parse and analyze SQL from a file. Use --policy to load a YAML policy file.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateFormat(format); err != nil {
				return err
			}
			code := cli.ReviewFile(args[0], policyPath, format)
			os.Exit(code)
			return nil
		},
	}
	cmd.Flags().StringVar(&policyPath, "policy", "", "Path to policy YAML file")
	cmd.Flags().StringVar(&format, "format", "human", "Output format: human, json, or sarif")
	return cmd
}

func reviewFilesCmd() *cobra.Command {
	var policyPath, format string
	cmd := &cobra.Command{
		Use:   "review-files [path...]",
		Short: "Review one or more SQL files",
		Long: `Parse and analyze SQL from multiple files, aggregating findings into a single
allow/warn/block decision (strictest wins). Empty path lists report allow.
Use --policy to load a YAML policy file.`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateFormat(format); err != nil {
				return err
			}
			code := cli.ReviewFiles(args, policyPath, format)
			os.Exit(code)
			return nil
		},
	}
	cmd.Flags().StringVar(&policyPath, "policy", "", "Path to policy YAML file")
	cmd.Flags().StringVar(&format, "format", "human", "Output format: human, json, or sarif")
	return cmd
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print dbguard version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("dbguard version %s\n", version.Version)
			return nil
		},
	}
}
