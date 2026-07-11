//go:build linux || darwin

package runner

import "syscall"

func signalProcessGroup(pid int, signal syscall.Signal) error {
	return syscall.Kill(-pid, signal)
}
