package runner

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/shellcell/snailrace/internal/model"
)

// ErrInterrupted reports that measurement was cancelled (for example by Ctrl+C)
// after at least one full round completed. The returned benchmarks hold only the
// completed rounds; the partial final round is discarded.
var ErrInterrupted = errors.New("measurement interrupted")

func Benchmark(
	ctx context.Context,
	specs []Spec,
	config Config,
	options Options,
) ([]model.Benchmark, error) {
	if err := validateBenchmarkInputs(ctx, specs, config); err != nil {
		return nil, err
	}
	specs = append([]Spec(nil), specs...)
	for index := range specs {
		specs[index].Args = append([]string(nil), specs[index].Args...)
	}
	benchmarks := make([]model.Benchmark, len(specs))
	progress := newProgressTracker(options, specs, config)
	defer progress.finish()
	inspector := newToolInspector()
	pinned := make([]*os.File, len(specs))
	defer func() {
		for _, file := range pinned {
			if file != nil {
				_ = file.Close()
			}
		}
	}()
	for index, spec := range specs {
		benchmarks[index].Runs = make([]model.Run, 0, config.Runs)
		tool, err := inspector.inspect(ctx, spec)
		if err != nil {
			return nil, err
		}
		file, err := inspector.pin(ctx, &tool)
		if err != nil {
			return nil, fmt.Errorf("hash %q: %w", tool.Name, err)
		}
		pinned[index] = file
		pinExecution := !executableIsScript(file)
		if pinExecution {
			specs[index].executable = file
		}
		if tool.ShellTarget && pinExecution {
			if command, ok := pinShellExecutable(
				spec.Shell, pinnedExecutablePath(file),
			); ok {
				specs[index].Shell = command
				specs[index].shellTarget = true
			}
		}
		benchmarks[index].Tool = tool
		progress.setTool(index, tool)
	}
	warmupOrder := config.WarmupOrder
	if len(warmupOrder) != config.Warmups {
		warmupOrder = BalancedSchedule(
			len(specs), config.Warmups, config.OrderSeed^0x5deece66d,
		)
	}
	interrupted := false
	for warmup, order := range warmupOrder {
		for _, index := range order {
			spec := specs[index]
			progress.update(
				index, spec.Name, warmup+1, config.Warmups, true, false, nil,
			)
			warmupOptions := options
			warmupOptions.Interactive = false
			if _, err := runOnce(ctx, spec, config.Interval, warmupOptions); err != nil {
				if ctx.Err() != nil {
					interrupted = true
					break
				}
				return nil, fmt.Errorf("warmup %q: %w", spec.Name, err)
			}
			progress.update(
				index, spec.Name, warmup+1, config.Warmups, true, true, nil,
			)
		}
		if interrupted {
			break
		}
	}

	measurementOrder := config.MeasurementOrder
	if len(measurementOrder) != config.Runs {
		measurementOrder = BalancedSchedule(len(specs), config.Runs, config.OrderSeed)
	}
	completedRounds := 0
	for runIndex, order := range measurementOrder {
		if interrupted {
			break
		}
		for _, index := range order {
			progress.update(
				index, specs[index].Name, runIndex+1, config.Runs,
				false, false, benchmarks[index].Runs,
			)
			run, err := runOnce(ctx, specs[index], config.Interval, options)
			if err != nil {
				if ctx.Err() != nil {
					interrupted = true
					break
				}
				return nil, fmt.Errorf("run %q: %w", specs[index].Name, err)
			}
			run.Index = runIndex + 1
			benchmarks[index].Runs = append(benchmarks[index].Runs, run)
			progress.update(
				index, specs[index].Name, runIndex+1, config.Runs,
				false, true, benchmarks[index].Runs,
			)
		}
		if !interrupted {
			completedRounds = runIndex + 1
		}
	}
	// Discard a partial final round so every tool has the same number of runs.
	for index := range benchmarks {
		if len(benchmarks[index].Runs) > completedRounds {
			benchmarks[index].Runs = benchmarks[index].Runs[:completedRounds]
		}
	}
	if interrupted && completedRounds == 0 {
		return nil, ctx.Err()
	}
	for index := range benchmarks {
		if !interrupted {
			if err := inspector.verify(
				ctx, benchmarks[index].Tool, pinned[index],
			); err != nil {
				return nil, fmt.Errorf("verify %q: %w", benchmarks[index].Tool.Name, err)
			}
			benchmarks[index].Tool.ProvenanceVerified = true
		}
		benchmarks[index].Summary = model.Summarize(benchmarks[index].Runs)
	}
	if interrupted {
		return benchmarks, ErrInterrupted
	}
	return benchmarks, nil
}

func executableIsScript(file *os.File) bool {
	var magic [2]byte
	count, _ := file.ReadAt(magic[:], 0)
	return count == len(magic) && magic == [2]byte{'#', '!'}
}
