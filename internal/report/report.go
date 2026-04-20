// Package report provides human and JSON output formatting
// with stable structure for CI and agent pipelines.
package report

// Options provides source/reporting context to output emitters.
type Options struct {
	SourcePath   string
	CoverageMode string
}
