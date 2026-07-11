package runner

import (
	"context"
	"os/exec"
)

type Spec struct {
	Name  string
	Args  []string
	Shell string
}

func (spec Spec) command(ctx context.Context) *exec.Cmd {
	if spec.Shell != "" {
		return exec.CommandContext(ctx, "/bin/sh", "-c", spec.Shell)
	}
	return exec.CommandContext(ctx, spec.Args[0], spec.Args[1:]...)
}
