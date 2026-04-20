//go:build cgo

package parser

// CoverageMode reports parser/rule coverage breadth for the current build.
func CoverageMode() string {
	return "full"
}
