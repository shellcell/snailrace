package runner

import (
	"context"
	"fmt"

	"perftool/internal/model"
)

func Benchmark(
	ctx context.Context,
	specs []Spec,
	config Config,
	options Options,
) ([]model.Benchmark, error) {
	benchmarks := make([]model.Benchmark, len(specs))
	progress := newProgressTracker(options, specs, config)
	defer progress.finish()
	inspector := newToolInspector()
	for index, spec := range specs {
		benchmarks[index].Runs = make([]model.Run, 0, config.Runs)
		tool, err := inspector.inspect(spec)
		if err != nil {
			return nil, err
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
	for warmup, order := range warmupOrder {
		for _, index := range order {
			spec := specs[index]
			progress.update(
				index, spec.Name, warmup+1, config.Warmups, true, false, nil,
			)
			warmupOptions := options
			warmupOptions.Interactive = false
			if _, err := runOnce(ctx, spec, config.Interval, warmupOptions); err != nil {
				return nil, fmt.Errorf("warmup %q: %w", spec.Name, err)
			}
			progress.update(
				index, spec.Name, warmup+1, config.Warmups, true, true, nil,
			)
		}
	}

	measurementOrder := config.MeasurementOrder
	if len(measurementOrder) != config.Runs {
		measurementOrder = BalancedSchedule(len(specs), config.Runs, config.OrderSeed)
	}
	for runIndex, order := range measurementOrder {
		for _, index := range order {
			progress.update(
				index, specs[index].Name, runIndex+1, config.Runs,
				false, false, benchmarks[index].Runs,
			)
			run, err := runOnce(ctx, specs[index], config.Interval, options)
			if err != nil {
				return nil, fmt.Errorf("run %q: %w", specs[index].Name, err)
			}
			run.Index = runIndex + 1
			benchmarks[index].Runs = append(benchmarks[index].Runs, run)
			progress.update(
				index, specs[index].Name, runIndex+1, config.Runs,
				false, true, benchmarks[index].Runs,
			)
		}
	}
	for index := range benchmarks {
		if err := inspector.addHash(&benchmarks[index].Tool); err != nil {
			return nil, fmt.Errorf("hash %q: %w", benchmarks[index].Tool.Name, err)
		}
		benchmarks[index].Summary = model.Summarize(benchmarks[index].Runs)
	}
	return benchmarks, nil
}
