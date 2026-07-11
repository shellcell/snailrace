package runner

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"perftool/internal/model"
	"perftool/internal/platform"
)

type toolInspector struct {
	cache map[string]model.ToolInfo
}

func newToolInspector() *toolInspector {
	return &toolInspector{cache: make(map[string]model.ToolInfo)}
}

func inspectTool(spec Spec) (model.ToolInfo, error) {
	return newToolInspector().inspect(spec)
}

func (inspector *toolInspector) inspect(spec Spec) (model.ToolInfo, error) {
	tool := model.ToolInfo{Name: spec.Name}
	executable := "/bin/sh"
	if spec.Shell != "" {
		tool.Command = []string{spec.Shell}
		if candidate := shellExecutable(spec.Shell); candidate != "" {
			executable = candidate
		}
	} else {
		tool.Command = append([]string(nil), spec.Args...)
		path, err := exec.LookPath(spec.Args[0])
		if err != nil {
			return model.ToolInfo{}, fmt.Errorf(
				"find executable %q: %w", spec.Args[0], err,
			)
		}
		executable, _ = filepath.Abs(path)
	}
	tool.Executable = executable
	if cached, ok := inspector.cache[executable]; ok {
		cached.Name, cached.Command = tool.Name, tool.Command
		return cached, nil
	}
	info, err := os.Stat(executable)
	if err != nil {
		return model.ToolInfo{}, fmt.Errorf(
			"inspect executable %q: %w", executable, err,
		)
	}
	tool.SizeBytes = info.Size()
	file, err := os.Open(executable)
	if err != nil {
		return model.ToolInfo{}, err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return model.ToolInfo{}, err
	}
	tool.SHA256 = hex.EncodeToString(hash.Sum(nil))
	for _, path := range platform.LinkedFiles(executable) {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		tool.LinkedFiles = append(tool.LinkedFiles, model.DiskFile{
			Path: path, SizeBytes: info.Size(),
		})
		tool.LinkedSizeBytes += info.Size()
	}
	tool.DiskFootprintBytes = tool.SizeBytes + tool.LinkedSizeBytes
	inspector.cache[executable] = tool
	return tool, nil
}

func shellExecutable(command string) string {
	fields := strings.Fields(command)
	if len(fields) == 0 || strings.ContainsAny(fields[0], "'\"|&;<>()$`") {
		return ""
	}
	path, err := exec.LookPath(fields[0])
	if err != nil {
		return ""
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return absolute
}
