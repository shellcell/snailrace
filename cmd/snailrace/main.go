package main

import (
	"fmt"
	"os"

	"github.com/shellcell/snailrace/internal/app"
)

func main() {
	if err := app.Run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "snailrace:", err)
		os.Exit(1)
	}
}
