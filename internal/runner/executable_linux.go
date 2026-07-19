package runner

import (
	"fmt"
	"os"
)

func pinnedExecutablePath(file *os.File) string {
	return fmt.Sprintf("/proc/%d/fd/%d", os.Getpid(), file.Fd())
}

func pinnedExtraFiles(*os.File) []*os.File { return nil }
