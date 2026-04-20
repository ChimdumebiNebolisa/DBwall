package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/ChimdumebiNebolisa/DBwall/benchmark"
)

func main() {
	var opts benchmark.Options
	flag.StringVar(&opts.RepoRoot, "repo-root", ".", "Path to the repository root")
	flag.StringVar(&opts.Manifest, "manifest", "./benchmark/manifest.json", "Path to the benchmark manifest")
	flag.StringVar(&opts.Binary, "binary", "", "Path to a dbguard binary to use; if empty the runner builds one")
	flag.StringVar(&opts.JSONOut, "json-out", "./benchmark/results/benchmark_results.json", "Path to write raw benchmark JSON")
	flag.StringVar(&opts.ReportOut, "report-out", "./benchmark/reports/benchmark_report.md", "Path to write the markdown benchmark report")
	flag.Parse()

	result, err := benchmark.Run(context.Background(), opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("Benchmark completed: %d cases, accuracy %.4f, precision %.4f, recall %.4f\n",
		result.Metrics.TotalCases,
		result.Metrics.Accuracy,
		result.Metrics.Precision,
		result.Metrics.Recall,
	)
}
