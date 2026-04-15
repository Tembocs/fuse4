package testrunner

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Tembocs/fuse4/compiler/diagnostics"
)

// PrintReport writes a colored test report to w.
func PrintReport(w io.Writer, results []TestResult, color bool, verbose bool) {
	if len(results) == 0 {
		fmt.Fprintln(w, diagnostics.Colorize("no tests found", diagnostics.ColorDim, color))
		return
	}

	var passed, failed int
	var totalDuration time.Duration

	for _, r := range results {
		totalDuration += r.Duration
		if r.Passed {
			passed++
			label := diagnostics.Colorize("  PASS  ", diagnostics.ColorBoldGreen, color)
			timing := diagnostics.Colorize(fmt.Sprintf("(%s)", formatDuration(r.Duration)), diagnostics.ColorDim, color)
			fmt.Fprintf(w, "%s %s %s\n", label, r.Name, timing)
			if verbose && r.Output != "" {
				for _, line := range strings.Split(strings.TrimSpace(r.Output), "\n") {
					fmt.Fprintf(w, "         %s\n", line)
				}
			}
		} else {
			failed++
			label := diagnostics.Colorize("  FAIL  ", diagnostics.ColorBoldRed, color)
			timing := diagnostics.Colorize(fmt.Sprintf("(%s)", formatDuration(r.Duration)), diagnostics.ColorDim, color)
			fmt.Fprintf(w, "%s %s %s\n", label, r.Name, timing)
			if r.Output != "" {
				for _, line := range strings.Split(strings.TrimSpace(r.Output), "\n") {
					fmt.Fprintf(w, "         %s\n", line)
				}
			}
		}
	}

	fmt.Fprintln(w)
	total := len(results)
	summary := fmt.Sprintf("%d test", total)
	if total != 1 {
		summary += "s"
	}
	summary += ": "

	var parts []string
	if passed > 0 {
		s := fmt.Sprintf("%d passed", passed)
		parts = append(parts, diagnostics.Colorize(s, diagnostics.ColorGreen, color))
	}
	if failed > 0 {
		s := fmt.Sprintf("%d failed", failed)
		parts = append(parts, diagnostics.Colorize(s, diagnostics.ColorBoldRed, color))
	}
	summary += strings.Join(parts, ", ")
	timing := diagnostics.Colorize(fmt.Sprintf(" (%s total)", formatDuration(totalDuration)), diagnostics.ColorDim, color)
	fmt.Fprintf(w, "%s%s\n", summary, timing)
}

func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%d\u00b5s", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}
