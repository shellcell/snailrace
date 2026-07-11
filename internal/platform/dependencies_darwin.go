package platform

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func LinkedFiles(executable string) []string {
	queue := []string{executable}
	seen := map[string]bool{executable: true}
	var result []string
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		output, err := exec.Command("otool", "-L", current).Output()
		if err != nil {
			continue
		}
		for _, dependency := range parseOtool(output, executable, current) {
			resolved, err := filepath.EvalSymlinks(dependency)
			if err != nil || seen[resolved] {
				continue
			}
			if info, err := os.Stat(resolved); err == nil && info.Mode().IsRegular() {
				seen[resolved] = true
				result = append(result, resolved)
				queue = append(queue, resolved)
			}
		}
	}
	return result
}

func parseOtool(output []byte, executable, loader string) []string {
	lines := strings.Split(string(output), "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		path := fields[0]
		path = strings.Replace(path, "@executable_path", filepath.Dir(executable), 1)
		path = strings.Replace(path, "@loader_path", filepath.Dir(loader), 1)
		if filepath.IsAbs(path) {
			result = append(result, path)
		}
	}
	return result
}
