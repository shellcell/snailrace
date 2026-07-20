package platform

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func LinkedFiles(ctx context.Context, executable string) ([]LinkedDependency, error) {
	output, err := runLDD(ctx, executable)
	if err != nil {
		return nil, err
	}
	files := parseLDD(output)
	if len(files) == 0 {
		for _, interpreter := range shebangExecutables(executable) {
			files = append(files, interpreter)
			output, err = runLDD(ctx, interpreter)
			if err != nil {
				return nil, err
			}
			files = append(files, parseLDD(output)...)
		}
	}
	paths := uniqueFiles(files, executable)
	result := make([]LinkedDependency, len(paths))
	for index, path := range paths {
		result[index] = LinkedDependency{Path: path}
	}
	return result, nil
}

func runLDD(ctx context.Context, executable string) ([]byte, error) {
	output, err := exec.CommandContext(ctx, "ldd", executable).CombinedOutput()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err == nil {
		return output, nil
	}
	message := strings.ToLower(string(output))
	if strings.Contains(message, "not a dynamic executable") ||
		strings.Contains(message, "statically linked") {
		return output, nil
	}
	return nil, fmt.Errorf("inspect linked files for %q: %w: %s",
		executable, err, strings.TrimSpace(string(output)))
}

func parseLDD(output []byte) []string {
	var files []string
	for _, line := range strings.Split(string(output), "\n") {
		candidate := strings.TrimSpace(line)
		if _, after, found := strings.Cut(candidate, "=>"); found {
			candidate = strings.TrimSpace(after)
		}
		if metadata := strings.LastIndex(candidate, " ("); metadata >= 0 {
			candidate = strings.TrimSpace(candidate[:metadata])
		}
		if filepath.IsAbs(candidate) {
			files = append(files, candidate)
		}
	}
	return files
}

func shebangExecutables(path string) []string {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()
	buffer := make([]byte, 4096)
	count, err := file.Read(buffer)
	if err != nil && count == 0 {
		return nil
	}
	data := string(buffer[:count])
	if !strings.HasPrefix(data, "#!") {
		return nil
	}
	line, _, _ := strings.Cut(data[2:], "\n")
	fields := strings.Fields(line)
	if len(fields) == 0 || !filepath.IsAbs(fields[0]) {
		return nil
	}
	result := []string{fields[0]}
	if filepath.Base(fields[0]) != "env" {
		return result
	}
	for index := 1; index < len(fields); index++ {
		field := fields[index]
		switch field {
		case "-u", "--unset", "-C", "--chdir", "-a", "--argv0":
			index++
			continue
		case "-S", "--split-string", "--":
			continue
		}
		if strings.HasPrefix(field, "-S") && len(field) > 2 {
			field = field[2:]
		} else if split, found := strings.CutPrefix(field, "--split-string="); found {
			field = split
		} else if strings.HasPrefix(field, "--unset=") ||
			strings.HasPrefix(field, "--chdir=") ||
			strings.HasPrefix(field, "--argv0=") {
			continue
		}
		if strings.HasPrefix(field, "-") || strings.Contains(field, "=") {
			continue
		}
		if split := strings.Fields(field); len(split) > 0 {
			field = split[0]
		}
		if executable, err := exec.LookPath(field); err == nil {
			if absolute, absErr := filepath.Abs(executable); absErr == nil {
				executable = absolute
			}
			result = append(result, executable)
		}
		break
	}
	return result
}

func uniqueFiles(paths []string, executable string) []string {
	executable, _ = filepath.EvalSymlinks(executable)
	seen := make(map[string]bool)
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil || resolved == executable || seen[resolved] {
			continue
		}
		if info, err := os.Stat(resolved); err == nil && info.Mode().IsRegular() {
			seen[resolved] = true
			result = append(result, resolved)
		}
	}
	return result
}
