package runner

import (
	"context"
	"io"
	"os/exec"
	"syscall"
)

func RunPreparation(ctx context.Context, shell string, output io.Writer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	cmd := exec.Command("/bin/sh", "-c", shell)
	configureProcessGroup(cmd)
	cmd.Stdout, cmd.Stderr = output, output
	if err := cmd.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		signalProcessGroup(cmd.Process.Pid, syscall.SIGKILL)
		return err
	case <-ctx.Done():
		signalProcessGroup(cmd.Process.Pid, syscall.SIGKILL)
		<-done
		return ctx.Err()
	}
}
