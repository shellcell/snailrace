package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"perftool/internal/analysis"
	"perftool/internal/model"
	"perftool/internal/platform"
	"perftool/internal/report"
	"perftool/internal/runner"
)

const Version = "0.1.0"

func Run(arguments []string, stdout, stderr io.Writer) error {
	options, err := parseOptions(arguments, stderr)
	if err != nil {
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
		OutputMode:        outputMode(options),
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
			Output:   stderr,
			Duration: options.duration, Width: width, Height: height,
			Interactive:  interactiveTUI,
			FollowResize: followResize,
			Progress: progressRenderer(
				stderr,
				!options.showOutput && !interactiveTUI,
			),
		},
	)
	if err != nil {
		return err
	}
	if config.Baseline == 0 {
		config.Baseline = analysis.AutomaticBaseline(config, benchmarks)
	}
	_, host.MemoryAfterBytes = platform.Memory()
	result := model.Report{
		MeasuredAt: measuredAt, Config: config, Host: host,
		Benchmarks: benchmarks,
		Notes: platformNotes(
			options.tui, options.duration, len(options.specs) > 1,
			config.BaselineAutomatic, config.OrderSeed, config.OutputMode,
		),
	}
	if err := writeResult(
		stdout, stderr, options.output, options.formats, result,
	); err != nil {
		return err
	}
	return checkExitCodes(benchmarks)
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
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", command)
	cmd.Stdout, cmd.Stderr = output, output
	if err := cmd.Run(); err != nil {
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
	if err := report.Write(stdout, "text", result); err != nil {
		return err
	}
	if directory == "" {
		return nil
	}
	return saveReportFormats(stderr, directory, formats, result)
}

func checkExitCodes(benchmarks []model.Benchmark) error {
	for _, benchmark := range benchmarks {
		for _, run := range benchmark.Runs {
			if run.ExitCode != 0 && run.StopReason != "duration" {
				return fmt.Errorf(
					"%q exited unsuccessfully in one or more runs",
					benchmark.Tool.Name,
				)
			}
		}
	}
	return nil
}

func modeName(tui bool) string {
	if tui {
		return "tui"
	}
	return "command"
}
