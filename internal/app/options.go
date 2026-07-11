package app

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"perftool/internal/platform"
	"perftool/internal/runner"
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
	showOutput    bool
	version       bool
	tui           bool
	width, height uint16
	specs         []runner.Spec
}

func parseOptions(arguments []string, stderr io.Writer) (options, error) {
	var result options
	var commands, labels stringList
	formats := newFormatValues()
	var name string
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
	flags.IntVar(&result.runs, "runs", defaultRuns, "number of measured runs")
	flags.IntVar(&result.runs, "n", defaultRuns, "number of measured runs (shorthand)")
	flags.IntVar(
		&result.warmups, "warmups", defaultWarmups, "warmup runs per command",
	)
	flags.IntVar(&result.baseline, "baseline", 0, "baseline index; 0 selects balanced winner")
	flags.DurationVar(
		&result.interval, "interval", platform.DefaultInterval(),
		"process sampling interval",
	)
	flags.Var(&commands, "command", "shell command; repeat to compare")
	flags.Var(&commands, "c", "shell command; repeat to compare (shorthand)")
	flags.Var(&labels, "label", "short command label; repeat in command order")
	flags.StringVar(&name, "name", "", "display name for one command")
	flags.StringVar(&result.prepare, "prepare", "", "one-time setup command")
	flags.Var(
		&formats, "format",
		"saved format; repeat or comma-separate: html, svg, markdown, json, text",
	)
	flags.StringVar(&result.output, "output", "", "directory for a saved report")
	flags.BoolVar(&result.showOutput, "show-output", false, "show command output")
	flags.DurationVar(
		&result.duration, "duration", 0, "fixed TUI measurement duration",
	)
	flags.UintVar(&width, "width", 0, "TUI columns; default inherits terminal")
	flags.UintVar(&height, "height", 0, "TUI rows; default inherits terminal")
	flags.BoolVar(&result.version, "version", false, "print version and exit")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: snailrace [options] -- command [args...]")
		fmt.Fprintln(stderr, "       snailrace [options] -c 'command' [-c 'command']")
		fmt.Fprintln(stderr, "       snailrace tui [options] -- command [args...]")
		fmt.Fprintln(stderr, "\nOptions:")
		flags.PrintDefaults()
	}
	if err := flags.Parse(arguments); err != nil {
		return result, err
	}
	if result.version {
		return result, nil
	}
	result.formats = formats.values
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
	if err := validateName(name, commands); err != nil {
		return result, err
	}
	if len(labels) > 0 && len(labels) != len(commands) {
		return result, errors.New("label count must match command count")
	}
	if len(labels) > 0 && name != "" {
		return result, errors.New("use either name or labels, not both")
	}
	result.specs = makeSpecs(commands, flags.Args(), name, labels)
	if result.baseline > len(result.specs) {
		return result, fmt.Errorf(
			"baseline %d exceeds the %d configured tools",
			result.baseline, len(result.specs),
		)
	}
	return result, nil
}
