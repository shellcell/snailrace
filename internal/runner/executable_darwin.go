package runner

import "os"

func pinnedExecutablePath(*os.File) string { return "/dev/fd/9" }

func pinnedExtraFiles(file *os.File) []*os.File {
	return []*os.File{nil, nil, nil, nil, nil, nil, file}
}
