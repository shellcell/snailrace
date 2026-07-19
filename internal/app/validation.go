package app

import (
	"errors"
	"fmt"
	"time"

	"github.com/shellcell/snailrace/internal/report"
)

func validateOptions(result options, commands []string, positional int) error {
	if result.runs < 1 {
		return errors.New("runs must be at least 1")
	}
	if result.baseline < 0 {
		return errors.New("baseline cannot be negative")
	}
	if result.warmups < 0 {
		return errors.New("warmups cannot be negative")
	}
	if result.interval < time.Millisecond {
		return errors.New("interval must be at least 1ms")
	}
	if result.duration < 0 {
		return errors.New("duration cannot be negative")
	}
	if len(result.formats) == 0 {
		return errors.New("at least one output format is required")
	}
	for _, format := range result.formats {
		if !report.ValidFormat(format) {
			return fmt.Errorf("unknown report format %q", format)
		}
	}
	if len(result.index) == 0 {
		return errors.New("balanced index requires at least one dimension")
	}
	if len(commands) > 0 && positional > 0 {
		return errors.New("use positional arguments or -command, not both")
	}
	if len(commands) == 0 && positional == 0 {
		return errors.New("no command specified")
	}
	commandCount := len(commands)
	if positional > 0 {
		commandCount = 1
	}
	if !result.tui && (result.duration > 0 || result.width > 0 || result.height > 0) {
		return errors.New("duration and terminal size require the tui subcommand")
	}
	if result.tui && commandCount > 1 && result.duration <= 0 {
		return errors.New("TUI comparisons require a positive duration")
	}
	if result.tui && result.showOutput {
		return errors.New("show-output does not apply to TUI mode")
	}
	if result.tui && result.duration == 0 && result.runs != 1 {
		return errors.New("unlimited interactive TUI measurement requires one run")
	}
	if result.tui && result.duration == 0 && result.warmups != 0 {
		return errors.New("unlimited interactive TUI measurement cannot use warmups")
	}
	return nil
}

func validateDimensions(width, height uint) error {
	if width > 65535 || height > 65535 {
		return errors.New("terminal dimensions cannot exceed 65535")
	}
	return nil
}

func validateLabels(labels, commands []string) error {
	if len(labels) == 0 {
		return nil
	}
	if len(commands) > 0 {
		if len(labels) != len(commands) {
			return errors.New("label count must match command count")
		}
		return nil
	}
	if len(labels) > 1 {
		return errors.New("only one label is allowed for a single command")
	}
	return nil
}
