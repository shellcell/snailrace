package runner

import (
	"context"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/creack/pty"
	"golang.org/x/sys/unix"
)

func ResolveTerminalSize(width, height uint16) (uint16, uint16) {
	size := &pty.Winsize{Cols: 80, Rows: 24}
	if current, err := pty.GetsizeFull(os.Stdin); err == nil {
		if current.Cols > 0 {
			size.Cols = current.Cols
		}
		if current.Rows > 0 {
			size.Rows = current.Rows
		}
	}
	if width > 0 {
		size.Cols = width
	}
	if height > 0 {
		size.Rows = height
	}
	return size.Cols, size.Rows
}

func terminalSize(input *os.File, width, height uint16) *pty.Winsize {
	if input == nil {
		input = os.Stdin
	}
	size := &pty.Winsize{Cols: 80, Rows: 24}
	if current, err := pty.GetsizeFull(input); err == nil {
		size = current
	}
	if width > 0 {
		size.Cols = width
	}
	if height > 0 {
		size.Rows = height
	}
	return size
}

func followTerminalResize(
	ctx context.Context, input *os.File, terminal *os.File,
) <-chan struct{} {
	resizes := make(chan os.Signal, 1)
	done := make(chan struct{})
	signal.Notify(resizes, syscall.SIGWINCH)
	go func() {
		defer close(done)
		defer signal.Stop(resizes)
		for {
			select {
			case <-resizes:
				_ = pty.InheritSize(input, terminal)
			case <-ctx.Done():
				return
			}
		}
	}()
	return done
}

func copyTerminalOutput(writer io.Writer, terminal *os.File, done chan<- struct{}) {
	defer close(done)
	_, _ = io.Copy(writer, terminal)
}

func copyInteractiveTerminalOutput(
	ctx context.Context, terminalFD, outputFD int, done chan<- struct{},
) {
	defer close(done)
	buffer := make([]byte, 4096)
	scratch := make([]unix.PollFd, 1)
	for {
		revents, ok := pollDescriptor(ctx, scratch, terminalFD, unix.POLLIN)
		if !ok {
			return
		}
		if revents&unix.POLLIN == 0 {
			if revents&(unix.POLLHUP|unix.POLLERR|unix.POLLNVAL) != 0 {
				return
			}
			continue
		}
		count, err := unix.Read(terminalFD, buffer)
		if count > 0 && !writeDescriptor(ctx, scratch, outputFD, buffer[:count]) {
			return
		}
		if err != nil || count == 0 {
			return
		}
	}
}

func copyTerminalInput(
	ctx context.Context, terminal *os.File, inputFD int, done chan<- struct{},
) {
	defer close(done)
	buffer := make([]byte, 4096)
	scratch := make([]unix.PollFd, 1)
	for {
		revents, ok := pollDescriptor(ctx, scratch, inputFD, unix.POLLIN)
		if !ok {
			return
		}
		if revents&unix.POLLIN == 0 {
			if revents&(unix.POLLHUP|unix.POLLERR|unix.POLLNVAL) != 0 {
				return
			}
			continue
		}
		count, err := unix.Read(inputFD, buffer)
		if count > 0 && !writeDescriptor(ctx, scratch, int(terminal.Fd()), buffer[:count]) {
			return
		}
		if err != nil || count == 0 {
			return
		}
	}
}

func pollDescriptor(
	ctx context.Context, scratch []unix.PollFd, fd int, events int16,
) (int16, bool) {
	for {
		if ctx.Err() != nil {
			return 0, false
		}
		scratch[0] = unix.PollFd{Fd: int32(fd), Events: events}
		ready, err := unix.Poll(scratch[:1], int((100 * time.Millisecond).Milliseconds()))
		if err == unix.EINTR {
			continue
		}
		if err != nil {
			return 0, false
		}
		if ready > 0 {
			return scratch[0].Revents, true
		}
	}
}

func writeDescriptor(
	ctx context.Context, scratch []unix.PollFd, fd int, data []byte,
) bool {
	for len(data) > 0 {
		revents, ok := pollDescriptor(ctx, scratch, fd, unix.POLLOUT)
		if !ok || revents&(unix.POLLHUP|unix.POLLERR|unix.POLLNVAL) != 0 {
			return false
		}
		count, err := unix.Write(fd, data)
		if err == unix.EINTR || err == unix.EAGAIN {
			continue
		}
		if err != nil || count == 0 {
			return false
		}
		data = data[count:]
	}
	return true
}
