package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/shellcell/snailrace/internal/analysis"
	"github.com/shellcell/snailrace/internal/model"
	"github.com/shellcell/snailrace/internal/platform"
	"github.com/shellcell/snailrace/internal/report"
	"github.com/shellcell/snailrace/internal/runner"
)

// Version is the release tag (vX.Y.Z), injected at build time via
// -ldflags "-X github.com/shellcell/snailrace/internal/app.Version=...".
var Version = "dev"

func Run(arguments []string, stdout, stderr io.Writer) error {
	options, err := parseOptions(arguments, stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if options.version {
		fmt.Fprintf(
			stdout, "snailrace %s (%s/%s)\n", Version, runtime.GOOS, runtime.GOARCH,
		)
		return nil
	}
	ctx, cancel := signal.NotifyContext(
		context.Background(), os.Interrupt, syscall.SIGTERM,
	)
	defer cancel()
	if err := prepare(ctx, options.prepare, stderr); err != nil {
		return err
	}

	host := platform.Host()
	measuredAt := time.Now()
	width, height := options.width, options.height
	inheritedSize := options.tui && width == 0 && height == 0
	interactiveTUI := options.tui && len(options.specs) == 1 && options.duration == 0
	followResize := inheritedSize && interactiveTUI
	if options.tui {
		width, height = runner.ResolveTerminalSize(width, height)
	}
	config := model.Config{
		Runs: options.runs, Warmups: options.warmups,
		Interval:        options.interval,
		IntervalMS:      float64(options.interval) / float64(time.Millisecond),
		Mode:            modeName(options.tui),
		DurationSeconds: options.duration.Seconds(),
		TerminalWidth:   width, TerminalHeight: height,
		TerminalInherited: inheritedSize,
		Baseline:          options.baseline,
		BaselineAutomatic: options.baseline == 0,
		OrderSeed:         measuredAt.UnixNano(),
		OrderMethod:       "randomized counterbalanced cyclic blocks",
		IndexDimensions:   options.index,
		OutputMode:        outputMode(options),
	}
	if !interactiveTUI {
		printCommandLegend(stderr, options.specs)
	}
	measurementOrder := runner.BalancedSchedule(
		len(options.specs), config.Runs, config.OrderSeed,
	)
	warmupOrder := runner.BalancedSchedule(
		len(options.specs), config.Warmups, config.OrderSeed^0x5deece66d,
	)
	config.MeasurementOrder = oneBasedOrder(measurementOrder)
	config.WarmupOrder = oneBasedOrder(warmupOrder)
	benchmarks, err := runner.Benchmark(
		ctx, options.specs, runner.Config{
			Runs: config.Runs, Warmups: config.Warmups, Interval: config.Interval,
			OrderSeed:        config.OrderSeed,
			MeasurementOrder: measurementOrder,
			WarmupOrder:      warmupOrder,
		}, runner.Options{
			ShowOutput: options.showOutput, TUI: options.tui,
			Output: stderr, Input: os.Stdin, TerminalOut: os.Stdout,
			Duration: options.duration, Width: width, Height: height,
			Interactive:  interactiveTUI,
			FollowResize: followResize,
			Progress: progressRenderer(
				stderr,
				!options.showOutput && !interactiveTUI,
				specNameWidth(options.specs),
			),
		},
	)
	interrupted := errors.Is(err, runner.ErrInterrupted)
	if err != nil && !interrupted {
		return err
	}
	rankingNote := ""
	if config.Baseline == 0 {
		ranking := analysis.Calculate(config, benchmarks)
		if ranking.Available && len(ranking.Rows) > 0 {
			config.Baseline = ranking.Rows[0].Benchmark + 1
		} else {
			config.Baseline = firstEligibleBenchmark(benchmarks) + 1
			config.BaselineAutomatic = false
			rankingNote = fmt.Sprintf(
				"Balanced ranking unavailable: %s. %s is used only as the comparison baseline.",
				ranking.UnavailableReason,
				benchmarks[config.Baseline-1].Tool.Name,
			)
		}
	}
	notes := platformNotes(
		options.tui, options.duration, len(options.specs) > 1,
		config.BaselineAutomatic, config.OrderSeed, config.OutputMode,
	)
	if rankingNote != "" {
		fmt.Fprintln(stderr, rankingNote)
		notes = append([]string{rankingNote}, notes...)
	}
	if interrupted {
		completed := 0
		if len(benchmarks) > 0 {
			completed = len(benchmarks[0].Runs)
		}
		message := fmt.Sprintf(
			"Measurement interrupted; reporting %d completed round(s) of %d "+
				"and discarding the unfinished final round.",
			completed, config.Runs,
		)
		fmt.Fprintln(stderr, message)
		notes = append([]string{message}, notes...)
		config.Runs = completed
		if completed < len(config.MeasurementOrder) {
			config.MeasurementOrder = config.MeasurementOrder[:completed]
		}
	}
	if hint := report.SamplingLimitNote(config, benchmarks); hint != "" {
		fmt.Fprintln(stderr, hint)
		notes = append(notes, hint)
	}
	_, host.MemoryAfterBytes = platform.Memory()
	result := model.Report{
		MeasuredAt: measuredAt, Config: config, Host: host,
		Benchmarks: benchmarks, Verbose: options.verbose,
		Notes: notes,
	}
	outputDirectory := options.output
	if options.noSave {
		outputDirectory = ""
	}
	if err := writeResult(
		stdout, stderr, outputDirectory, options.formats, result,
	); err != nil {
		return err
	}
	return nil
}

func firstEligibleBenchmark(benchmarks []model.Benchmark) int {
	for index, benchmark := range benchmarks {
		if benchmark.EligibleForRanking() {
			return index
		}
	}
	return 0
}

func oneBasedOrder(order [][]int) [][]int {
	result := make([][]int, len(order))
	for round, tools := range order {
		result[round] = make([]int, len(tools))
		for position, tool := range tools {
			result[round][position] = tool + 1
		}
	}
	return result
}

func outputMode(options options) string {
	if options.tui {
		if len(options.specs) == 1 && options.duration == 0 {
			return "interactive PTY"
		}
		return "drained PTY"
	}
	if options.showOutput {
		return "forwarded to stderr"
	}
	return "null device"
}

func prepare(ctx context.Context, command string, output io.Writer) error {
	if command == "" {
		return nil
	}
	if err := runner.RunPreparation(ctx, command, output); err != nil {
		return fmt.Errorf("prepare command: %w", err)
	}
	return nil
}

func writeResult(
	stdout, stderr io.Writer,
	directory string,
	formats []string,
	result model.Report,
) error {
	var saveErr error
	if directory != "" {
		saveErr = saveReportFormats(stderr, directory, formats, result)
	}
	textErr := report.Write(stdout, "text", result)
	return errors.Join(saveErr, textErr)
}

func modeName(tui bool) string {
	if tui {
		return "tui"
	}
	return "command"
}
