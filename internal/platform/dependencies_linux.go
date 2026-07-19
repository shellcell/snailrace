package platform

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func LinkedFiles(ctx context.Context, executable string) ([]LinkedDependency, error) {
	output, _ := exec.CommandContext(ctx, "ldd", executable).CombinedOutput()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	files := parseLDD(output)
	if len(files) == 0 {
		if interpreter := shebangInterpreter(executable); interpreter != "" {
			files = append(files, interpreter)
			output, _ = exec.CommandContext(ctx, "ldd", interpreter).CombinedOutput()
			if err := ctx.Err(); err != nil {
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

func parseLDD(output []byte) []string {
	var files []string
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		candidate := ""
		if len(fields) >= 3 && fields[1] == "=>" {
			candidate = fields[2]
		} else if len(fields) >= 1 && filepath.IsAbs(fields[0]) {
			candidate = fields[0]
		}
		if filepath.IsAbs(candidate) {
			files = append(files, candidate)
		}
	}
	return files
}

func shebangInterpreter(path string) string {
	data, err := os.ReadFile(path)
	if err != nil || !strings.HasPrefix(string(data), "#!") {
		return ""
	}
	line, _, _ := strings.Cut(string(data[2:]), "\n")
	fields := strings.Fields(line)
	if len(fields) == 0 || !filepath.IsAbs(fields[0]) {
		return ""
	}
	return fields[0]
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
