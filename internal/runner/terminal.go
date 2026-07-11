package runner

import (
	"context"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/creack/pty"
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

func terminalSize(width, height uint16) *pty.Winsize {
	width, height = ResolveTerminalSize(width, height)
	return &pty.Winsize{Cols: width, Rows: height}
}

func followTerminalResize(ctx context.Context, terminal *os.File) {
	resizes := make(chan os.Signal, 1)
	signal.Notify(resizes, syscall.SIGWINCH)
	go func() {
		defer signal.Stop(resizes)
		for {
			select {
			case <-resizes:
				pty.InheritSize(os.Stdin, terminal)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func copyTerminalOutput(writer io.Writer, terminal *os.File, done chan<- struct{}) {
	defer close(done)
	io.Copy(writer, terminal)
}
