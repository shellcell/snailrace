package runner

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

func validateBenchmarkInputs(ctx context.Context, specs []Spec, config Config) error {
	if ctx == nil {
		return errors.New("context cannot be nil")
	}
	if len(specs) == 0 {
		return errors.New("at least one command is required")
	}
	if config.Runs < 1 {
		return errors.New("runs must be at least 1")
	}
	if config.Warmups < 0 {
		return errors.New("warmups cannot be negative")
	}
	if config.Interval <= 0 {
		return errors.New("sampling interval must be positive")
	}
	for index, spec := range specs {
		if spec.Shell != "" && strings.TrimSpace(spec.Shell) == "" {
			return fmt.Errorf("command %d shell source cannot be blank", index+1)
		}
		hasShell := spec.Shell != ""
		hasArgs := len(spec.Args) > 0 && strings.TrimSpace(spec.Args[0]) != ""
		if hasShell == hasArgs {
			return fmt.Errorf("command %d must define either shell source or argv", index+1)
		}
	}
	if len(config.MeasurementOrder) > 0 && len(config.MeasurementOrder) != config.Runs {
		return fmt.Errorf(
			"measurement order has %d rounds, want %d",
			len(config.MeasurementOrder), config.Runs,
		)
	}
	if len(config.MeasurementOrder) == config.Runs {
		if err := validateSchedule(config.MeasurementOrder, len(specs)); err != nil {
			return fmt.Errorf("measurement order: %w", err)
		}
	}
	if len(config.WarmupOrder) > 0 && len(config.WarmupOrder) != config.Warmups {
		return fmt.Errorf(
			"warmup order has %d rounds, want %d",
			len(config.WarmupOrder), config.Warmups,
		)
	}
	if len(config.WarmupOrder) == config.Warmups {
		if err := validateSchedule(config.WarmupOrder, len(specs)); err != nil {
			return fmt.Errorf("warmup order: %w", err)
		}
	}
	return nil
}

func validateSchedule(schedule [][]int, tools int) error {
	for round, order := range schedule {
		if len(order) != tools {
			return fmt.Errorf("round %d has %d tools, want %d", round+1, len(order), tools)
		}
		seen := make([]bool, tools)
		for _, index := range order {
			if index < 0 || index >= tools {
				return fmt.Errorf("round %d contains tool index %d", round+1, index)
			}
			if seen[index] {
				return fmt.Errorf("round %d repeats tool index %d", round+1, index)
			}
			seen[index] = true
		}
	}
	return nil
}
