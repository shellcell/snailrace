package runner

import (
	"context"
	"os"
	"os/exec"
)

type Spec struct {
	Name  string
	Args  []string
	Shell string
}

type preparedSpec struct {
	Spec
	executable  *os.File
	shellTarget bool
}

func (spec Spec) command(ctx context.Context) *exec.Cmd {
	if spec.Shell != "" {
		return exec.CommandContext(ctx, "/bin/sh", "-c", spec.Shell)
	}
	return exec.CommandContext(ctx, spec.Args[0], spec.Args[1:]...)
}

func (spec preparedSpec) command(ctx context.Context) *exec.Cmd {
	path := ""
	if spec.executable != nil {
		path = pinnedExecutablePath(spec.executable)
	}
	var command *exec.Cmd
	if spec.Shell != "" {
		shell := "/bin/sh"
		if spec.executable != nil && !spec.shellTarget {
			shell = path
		}
		command = exec.CommandContext(ctx, shell, "-c", spec.Shell)
		command.Args[0] = "/bin/sh"
	} else {
		if path == "" {
			path = spec.Args[0]
		}
		command = exec.CommandContext(ctx, path, spec.Args[1:]...)
		command.Args[0] = spec.Args[0]
	}
	if spec.executable != nil {
		command.ExtraFiles = pinnedExtraFiles(spec.executable)
	}
	return command
}

type commandSpec interface {
	command(context.Context) *exec.Cmd
}
