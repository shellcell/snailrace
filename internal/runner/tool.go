package runner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/shellcell/snailrace/internal/model"
	"github.com/shellcell/snailrace/internal/platform"
)

type toolInspector struct {
	cache     map[string]model.ToolInfo
	hashCache map[string]string
	fileCache map[string]os.FileInfo
}

func newToolInspector() *toolInspector {
	return &toolInspector{
		cache: make(map[string]model.ToolInfo), hashCache: make(map[string]string),
		fileCache: make(map[string]os.FileInfo),
	}
}

func inspectTool(spec Spec) (model.ToolInfo, error) {
	return newToolInspector().inspect(context.Background(), spec)
}

func (inspector *toolInspector) inspect(
	ctx context.Context, spec Spec,
) (model.ToolInfo, error) {
	if err := ctx.Err(); err != nil {
		return model.ToolInfo{}, err
	}
	tool := model.ToolInfo{Name: spec.Name}
	executable := "/bin/sh"
	if spec.Shell != "" {
		tool.Command = []string{spec.Shell}
		tool.ShellCommand = true
		if candidate := shellExecutable(spec.Shell); candidate != "" {
			executable = candidate
			tool.ShellTarget = true
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
	if resolved, err := filepath.EvalSymlinks(executable); err == nil {
		executable = resolved
	}
	tool.Executable = executable
	if cached, ok := inspector.cache[executable]; ok {
		cached.Name, cached.Command = tool.Name, tool.Command
		cached.ShellCommand = tool.ShellCommand
		cached.ShellTarget = tool.ShellTarget
		return cached, nil
	}
	info, err := os.Stat(executable)
	if err != nil {
		return model.ToolInfo{}, fmt.Errorf(
			"inspect executable %q: %w", executable, err,
		)
	}
	tool.SizeBytes = info.Size()
	inspector.fileCache[executable] = info
	dependencies, err := platform.LinkedFiles(ctx, executable)
	if err != nil {
		return model.ToolInfo{}, err
	}
	for _, dependency := range dependencies {
		if dependency.SharedCache {
			tool.SharedCacheFiles = append(tool.SharedCacheFiles, dependency.Path)
			continue
		}
		info, err := os.Stat(dependency.Path)
		if err != nil {
			continue
		}
		tool.LinkedFiles = append(tool.LinkedFiles, model.DiskFile{
			Path: dependency.Path, SizeBytes: info.Size(),
		})
		tool.LinkedSizeBytes += info.Size()
	}
	tool.DiskFootprintBytes = tool.SizeBytes + tool.LinkedSizeBytes
	inspector.cache[executable] = tool
	return tool, nil
}

func (inspector *toolInspector) pin(
	ctx context.Context, tool *model.ToolInfo,
) (*os.File, error) {
	file, err := os.Open(tool.Executable)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}
	if inspected := inspector.fileCache[tool.Executable]; inspected == nil ||
		!os.SameFile(inspected, info) {
		file.Close()
		return nil, errors.New("executable changed while being inspected")
	}
	inspected := inspector.fileCache[tool.Executable]
	if inspected.Size() != info.Size() || !inspected.ModTime().Equal(info.ModTime()) {
		file.Close()
		return nil, errors.New("executable contents changed while being inspected")
	}
	if hash, ok := inspector.hashCache[tool.Executable]; ok {
		tool.SHA256 = hash
	} else {
		hash, hashErr := hashOpenFile(ctx, file)
		if hashErr != nil {
			file.Close()
			return nil, hashErr
		}
		tool.SHA256 = hash
		inspector.hashCache[tool.Executable] = hash
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		file.Close()
		return nil, err
	}
	return file, nil
}

func (inspector *toolInspector) verify(
	ctx context.Context, tool model.ToolInfo, file *os.File,
) error {
	before, ok := inspector.fileCache[tool.Executable]
	if !ok {
		return errors.New("missing inspected executable identity")
	}
	after, err := os.Stat(tool.Executable)
	if err != nil {
		return err
	}
	if !os.SameFile(before, after) {
		return errors.New("executable was replaced during measurement")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	hash, err := hashOpenFile(ctx, file)
	if err != nil {
		return err
	}
	if hash != tool.SHA256 {
		return errors.New("executable changed during measurement")
	}
	return nil
}

func hashOpenFile(ctx context.Context, file *os.File) (string, error) {
	hash := sha256.New()
	buffer := make([]byte, 128*1024)
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		count, readErr := file.Read(buffer)
		if count > 0 {
			_, _ = hash.Write(buffer[:count])
		}
		if errors.Is(readErr, io.EOF) {
			return hex.EncodeToString(hash.Sum(nil)), nil
		}
		if readErr != nil {
			return "", readErr
		}
	}
}

func shellExecutable(command string) string {
	fields := strings.Fields(command)
	if len(fields) == 0 || shellBuiltin(fields[0]) ||
		strings.ContainsAny(fields[0], "'\"|&;<>()$`") {
		return ""
	}
	path, err := exec.LookPath(fields[0])
	if err != nil {
		return ""
	}
	absolute, err := filepath.Abs(path)
	if err == nil {
		path = absolute
	}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	return path
}

func shellBuiltin(name string) bool {
	builtins := map[string]bool{
		"!": true, ".": true, ":": true, "[": true, "alias": true,
		"bg": true, "break": true, "cd": true, "command": true,
		"continue": true, "echo": true, "eval": true, "exec": true,
		"exit": true, "export": true, "false": true, "fc": true,
		"fg": true, "getopts": true, "hash": true, "jobs": true,
		"kill": true, "printf": true, "pwd": true, "read": true,
		"readonly": true, "return": true, "set": true, "shift": true,
		"test": true, "time": true, "times": true, "trap": true,
		"true": true, "type": true, "ulimit": true, "umask": true,
		"unalias": true, "unset": true, "wait": true, "{": true,
		"builtin": true, "declare": true, "local": true, "source": true,
		"typeset": true,
	}
	return builtins[name]
}

func pinShellExecutable(command, executable string) (string, bool) {
	trimmed := strings.TrimLeft(command, " \t\r\n")
	if trimmed == "" {
		return command, false
	}
	end := strings.IndexAny(trimmed, " \t\r\n")
	if end < 0 {
		end = len(trimmed)
	}
	if strings.ContainsAny(trimmed[:end], "'\"|&;<>()$`") {
		return command, false
	}
	prefix := command[:len(command)-len(trimmed)]
	quoted := "'" + strings.ReplaceAll(executable, "'", `'"'"'`) + "'"
	return prefix + quoted + trimmed[end:], true
}
