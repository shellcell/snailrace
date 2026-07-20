package app

import (
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/shellcell/snailrace/internal/platform"
	"github.com/shellcell/snailrace/internal/runner"
)

type stringList []string

func (values *stringList) String() string { return strings.Join(*values, ", ") }

func (values *stringList) Set(value string) error {
	*values = append(*values, value)
	return nil
}

type options struct {
	runs, warmups int
	baseline      int
	interval      time.Duration
	duration      time.Duration
	output        string
	formats       []string
	prepare       string
	index         []string
	verbose       bool
	showOutput    bool
	noSave        bool
	version       bool
	tui           bool
	width, height uint16
	specs         []runner.Spec
}

func parseOptions(arguments []string, stderr io.Writer) (options, error) {
	var result options
	var commands, labels stringList
	formats := newFormatValues()
	index := newIndexValues()
	var width, height uint
	if len(arguments) > 0 && arguments[0] == "tui" {
		result.tui = true
		arguments = arguments[1:]
	}
	defaultRuns, defaultWarmups := 10, 1
	if result.tui {
		defaultRuns, defaultWarmups = 1, 0
	}
	flags := flag.NewFlagSet("snailrace", flag.ContinueOnError)
	flags.SetOutput(stderr)
	// Input.
	flags.Var(&commands, "command", "shell command; repeat to compare")
	flags.Var(&commands, "c", "shell command; repeat to compare (shorthand)")
	flags.Var(&labels, "label", "display label; repeat once per command, in order")
	flags.StringVar(&result.prepare, "prepare", "", "one-time setup command")
	// Measurement.
	flags.IntVar(&result.runs, "runs", defaultRuns, "number of measured runs")
	flags.IntVar(&result.runs, "n", defaultRuns, "number of measured runs (shorthand)")
	flags.IntVar(
		&result.warmups, "warmups", defaultWarmups, "warmup runs per command",
	)
	flags.DurationVar(
		&result.interval, "interval", platform.DefaultInterval(),
		"process sampling interval",
	)
	// Ranking.
	flags.Var(
		&index, "index",
		"balanced index dimensions; comma-separate: time, cpu, ram, disk",
	)
	flags.IntVar(&result.baseline, "baseline", 0, "baseline index; 0 selects balanced winner")
	// Output.
	flags.Var(
		&formats, "format",
		"saved format; repeat or comma-separate: html, svg, markdown, json, text",
	)
	flags.Var(&formats, "f", "saved format (shorthand)")
	flags.StringVar(&result.output, "output", ".", "report directory; default current directory")
	flags.StringVar(&result.output, "o", ".", "report directory (shorthand)")
	flags.BoolVar(&result.noSave, "no-save", false, "do not save report files")
	flags.BoolVar(&result.verbose, "verbose", false, "print full statistical tables to stdout")
	flags.BoolVar(&result.showOutput, "show-output", false, "show command output")
	// TUI.
	flags.DurationVar(
		&result.duration, "duration", 0, "fixed TUI measurement duration",
	)
	flags.DurationVar(&result.duration, "d", 0, "fixed TUI measurement duration (shorthand)")
	flags.UintVar(&width, "width", 0, "TUI columns; default inherits terminal")
	flags.UintVar(&height, "height", 0, "TUI rows; default inherits terminal")
	// Misc.
	flags.BoolVar(&result.version, "version", false, "print version and exit")
	flags.BoolVar(&result.version, "v", false, "print version and exit (shorthand)")
	flags.Usage = func() { printUsage(stderr) }
	if err := flags.Parse(arguments); err != nil {
		return result, err
	}
	if result.version {
		return result, nil
	}
	result.formats = formats.values
	result.index = index.values
	if err := validateDimensions(width, height); err != nil {
		return result, err
	}
	result.width, result.height = uint16(width), uint16(height)
	if err := validateOptions(result, commands, flags.NArg()); err != nil {
		if flags.NArg() == 0 && len(commands) == 0 {
			flags.Usage()
		}
		return result, err
	}
	if err := validateLabels(labels, commands); err != nil {
		return result, err
	}
	result.specs = makeSpecs(commands, flags.Args(), labels)
	if result.baseline > len(result.specs) {
		return result, fmt.Errorf(
			"baseline %d exceeds the %d configured tools",
			result.baseline, len(result.specs),
		)
	}
	return result, nil
}

func printUsage(stderr io.Writer) {
	fmt.Fprintln(stderr, "Usage: snailrace [options] -- command [args...]")
	fmt.Fprintln(stderr, "       snailrace [options] -c 'command' [-c 'command']")
	fmt.Fprintln(stderr, "       snailrace tui [options] -- command [args...]")
	sections := []struct {
		title string
		lines []string
	}{
		{"Input", []string{
			"-c, -command string   shell command; repeat to compare",
			"-label string         display label; repeat once per command, in order",
			"-prepare string       one-time setup command",
		}},
		{"Measurement", []string{
			"-n, -runs int         number of measured runs",
			"-warmups int          warmup runs per command",
			"-interval duration    process sampling interval",
		}},
		{"Ranking", []string{
			"-index list           balanced index dimensions: time, cpu, ram, disk",
			"-baseline int         baseline index; 0 selects the balanced winner",
		}},
		{"Output", []string{
			"-f, -format list      saved format: html, svg, markdown, json, text",
			"-o, -output string    report directory; default current directory",
			"-no-save              do not save report files",
			"-verbose              print full statistical tables to stdout",
			"-show-output          forward command output to stderr",
		}},
		{"TUI (snailrace tui ...)", []string{
			"-d, -duration duration   fixed TUI measurement duration",
			"-width uint              TUI columns; default inherits terminal",
			"-height uint             TUI rows; default inherits terminal",
		}},
		{"Misc", []string{
			"-v, -version          print version and exit",
			"-h, -help             show this help",
		}},
	}
	for _, section := range sections {
		fmt.Fprintf(stderr, "\n%s:\n", section.title)
		for _, line := range section.lines {
			fmt.Fprintf(stderr, "  %s\n", line)
		}
	}
}
