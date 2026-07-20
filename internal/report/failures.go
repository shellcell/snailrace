package report

import (
	"fmt"
	"html"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/shellcell/snailrace/internal/model"
)

type benchmarkFailure struct {
	benchmark int
	failed    int
	total     int
	exitCodes []int
}

func benchmarkFailures(report model.Report) []benchmarkFailure {
	var failures []benchmarkFailure
	for index, benchmark := range report.Benchmarks {
		failed := benchmark.FailedRunCount()
		if failed == 0 {
			continue
		}
		seen := make(map[int]bool)
		for _, run := range benchmark.Runs {
			if run.Failed() {
				seen[run.ExitCode] = true
			}
		}
		codes := make([]int, 0, len(seen))
		for code := range seen {
			codes = append(codes, code)
		}
		sort.Ints(codes)
		failures = append(failures, benchmarkFailure{
			benchmark: index, failed: failed, total: len(benchmark.Runs), exitCodes: codes,
		})
	}
	return failures
}

func failureExitCodes(failure benchmarkFailure) string {
	values := make([]string, len(failure.exitCodes))
	for index, code := range failure.exitCodes {
		values[index] = strconv.Itoa(code)
	}
	return strings.Join(values, ", ")
}

func writeTextFailures(writer io.Writer, report model.Report) {
	failures := benchmarkFailures(report)
	for _, failure := range failures {
		fmt.Fprintf(
			writer, "FAILED  %s: %d/%d measured runs exited non-zero (codes %s); excluded from rankings\n",
			report.Benchmarks[failure.benchmark].Tool.Name,
			failure.failed, failure.total, failureExitCodes(failure),
		)
	}
	if len(failures) > 0 {
		fmt.Fprintln(writer)
	}
}

func writeHTMLFailures(writer io.Writer, report model.Report) {
	failures := benchmarkFailures(report)
	if len(failures) == 0 {
		return
	}
	fmt.Fprint(writer, `<section><h2 class="bad">Command failures</h2>`)
	for _, failure := range failures {
		fmt.Fprintf(
			writer, `<p class="bad"><strong>%s</strong>: %d/%d measured runs exited `+
				`non-zero (codes %s); excluded from rankings.</p>`,
			htmlToolLabel(report, failure.benchmark, true),
			failure.failed, failure.total, html.EscapeString(failureExitCodes(failure)),
		)
	}
	fmt.Fprint(writer, `</section>`)
}

func writeMarkdownFailures(writer io.Writer, report model.Report) {
	failures := benchmarkFailures(report)
	if len(failures) == 0 {
		return
	}
	fmt.Fprintln(writer, "## Command Failures")
	for _, failure := range failures {
		fmt.Fprintf(
			writer, "\n- **%s**: %d/%d measured runs exited non-zero (codes %s); excluded from rankings.\n",
			escapeMarkdown(report.Benchmarks[failure.benchmark].Tool.Name),
			failure.failed, failure.total, failureExitCodes(failure),
		)
	}
	fmt.Fprintln(writer)
}

func failureChart(report model.Report) svgChart {
	failures := benchmarkFailures(report)
	if len(failures) == 0 {
		return svgChart{}
	}
	height := 54 + len(failures)*24
	var body strings.Builder
	body.WriteString(svgChartStyle)
	body.WriteString(`<rect width="100%" height="100%" rx="8" fill="#3b4252"/>`)
	body.WriteString(`<text x="16" y="24" class="title" style="fill:#bf616a">COMMAND FAILURES</text>`)
	for index, failure := range failures {
		fmt.Fprintf(
			&body, `<text x="16" y="%d" class="label" style="fill:#bf616a">%s: `+
				`%d/%d runs, codes %s; excluded from rankings</text>`,
			50+index*24,
			html.EscapeString(clip(report.Benchmarks[failure.benchmark].Tool.Name, 28)),
			failure.failed, failure.total, html.EscapeString(failureExitCodes(failure)),
		)
	}
	return svgChart{
		kind: "failure", title: "Command failures", slug: "command-failures",
		description: "Lists commands with non-zero measured runs that were excluded from rankings.",
		body:        body.String(), height: height,
	}
}
