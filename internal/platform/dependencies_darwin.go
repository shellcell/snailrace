package platform

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func LinkedFiles(ctx context.Context, executable string) ([]LinkedDependency, error) {
	type queuedLibrary struct {
		path   string
		rpaths []string
	}
	executableRPaths := loadRPaths(ctx, executable, executable, executable)
	queue := []queuedLibrary{{path: executable, rpaths: executableRPaths}}
	seen := map[string]bool{executable: true}
	var result []LinkedDependency
	for len(queue) > 0 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		current := queue[0]
		queue = queue[1:]
		output, err := exec.CommandContext(ctx, "/usr/bin/otool", "-L", current.path).Output()
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			continue
		}
		rpaths := append(
			loadRPaths(ctx, current.path, executable, current.path), current.rpaths...,
		)
		for _, name := range parseOtool(output) {
			dependency, ok := resolveDylib(ctx, name, executable, current.path, rpaths)
			if !ok || seen[dependency.Path] {
				continue
			}
			seen[dependency.Path] = true
			result = append(result, dependency)
			if !dependency.SharedCache {
				queue = append(queue, queuedLibrary{path: dependency.Path, rpaths: rpaths})
			}
		}
	}
	return result, nil
}

func parseOtool(output []byte) []string {
	lines := strings.Split(string(output), "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) > 0 {
			result = append(result, fields[0])
		}
	}
	return result
}

func loadRPaths(ctx context.Context, path, executable, loader string) []string {
	output, err := exec.CommandContext(ctx, "/usr/bin/otool", "-l", path).Output()
	if err != nil {
		return nil
	}
	var result []string
	wantPath := false
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "cmd LC_RPATH" {
			wantPath = true
			continue
		}
		if !wantPath || !strings.HasPrefix(line, "path ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			result = append(result, expandDylibPath(fields[1], executable, loader))
		}
		wantPath = false
	}
	return result
}

func resolveDylib(
	ctx context.Context, name, executable, loader string, rpaths []string,
) (LinkedDependency, bool) {
	var candidates []string
	if suffix, found := strings.CutPrefix(name, "@rpath/"); found {
		for _, rpath := range rpaths {
			candidates = append(candidates, filepath.Join(rpath, suffix))
		}
	} else {
		candidates = append(candidates, expandDylibPath(name, executable, loader))
	}
	sharedCacheCandidate := ""
	for _, candidate := range candidates {
		resolved, err := filepath.EvalSymlinks(candidate)
		if err != nil {
			resolved = candidate
		}
		if info, err := os.Stat(resolved); err == nil && info.Mode().IsRegular() {
			return LinkedDependency{Path: resolved}, true
		}
		if sharedCacheCandidate == "" && isSharedCachePath(resolved) {
			sharedCacheCandidate = resolved
		}
	}
	if sharedCacheCandidate != "" && inSharedCache(ctx, sharedCacheCandidate) {
		return LinkedDependency{Path: sharedCacheCandidate, SharedCache: true}, true
	}
	return LinkedDependency{}, false
}

func expandDylibPath(path, executable, loader string) string {
	path = strings.Replace(path, "@executable_path", filepath.Dir(executable), 1)
	return strings.Replace(path, "@loader_path", filepath.Dir(loader), 1)
}

func isSharedCachePath(path string) bool {
	return strings.HasPrefix(path, "/usr/lib/") ||
		strings.HasPrefix(path, "/System/Library/")
}

func inSharedCache(ctx context.Context, path string) bool {
	return exec.CommandContext(ctx, "/usr/bin/dyld_info", "-dependents", path).Run() == nil
}
