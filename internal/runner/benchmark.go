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
		for warmup := 0; warmup < config.Warmups; warmup++ {
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

	// Rotate execution order to distribute first/last-run bias across commands.
	for runIndex := 0; runIndex < config.Runs; runIndex++ {
		for offset := range specs {
			index := (runIndex + offset) % len(specs)
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
		benchmarks[index].Summary = model.Summarize(benchmarks[index].Runs)
	}
	return benchmarks, nil
}
