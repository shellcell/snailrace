package app

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/shellcell/snailrace/internal/runner"
)

func makeSpecs(commands, arguments []string, labels []string) []runner.Spec {
	if len(commands) == 0 {
		name := filepath.Base(arguments[0])
		if len(labels) > 0 {
			name = labels[0]
		}
		return []runner.Spec{{Name: name, Args: arguments}}
	}
	defaultLabels := make([]string, len(commands))
	labelCounts := make(map[string]int)
	for index, command := range commands {
		defaultLabels[index] = commandLabel(command)
		labelCounts[defaultLabels[index]]++
	}
	seen := make(map[string]int)
	result := make([]runner.Spec, 0, len(commands))
	for index, command := range commands {
		label := defaultLabels[index]
		if len(labels) > 0 {
			label = labels[index]
		} else if labelCounts[label] > 1 {
			seen[label]++
			label += " #" + fmt.Sprint(seen[label])
		}
		result = append(result, runner.Spec{Name: label, Shell: command})
	}
	return result
}

func commandLabel(command string) string {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return "command"
	}
	name := strings.Trim(fields[0], "'\"")
	if name == "" {
		return "command"
	}
	return filepath.Base(name)
}
