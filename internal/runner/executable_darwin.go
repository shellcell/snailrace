package runner

import "os"

// macOS cannot execve /dev/fd nodes and has no fexecve, so execution cannot
// be pinned to the inspected descriptor; provenance relies on the before and
// after hash verification instead.
const pinExecutionSupported = false

func pinnedExecutablePath(*os.File) string { return "" }

func pinnedExtraFiles(*os.File) []*os.File { return nil }
