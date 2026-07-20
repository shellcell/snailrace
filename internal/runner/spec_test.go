package runner

import (
	"context"
	"os"
	"testing"
)

func TestPreparedSpecPreservesInvocationIdentity(t *testing.T) {
	executable, err := os.Open("/bin/sh")
	if err != nil {
		t.Fatal(err)
	}
	defer executable.Close()

	direct := preparedSpec{
		Spec: Spec{Args: []string{"display-name", "argument"}}, executable: executable,
	}.command(context.Background())
	if direct.Args[0] != "display-name" || direct.Args[1] != "argument" {
		t.Fatalf("direct argv = %v", direct.Args)
	}

	shell := preparedSpec{
		Spec: Spec{Shell: "exit 0"}, executable: executable,
	}.command(context.Background())
	if shell.Args[0] != "/bin/sh" || shell.Args[1] != "-c" {
		t.Fatalf("shell argv = %v", shell.Args)
	}

	target := preparedSpec{
		Spec: Spec{Shell: "exec tool"}, executable: executable, shellTarget: true,
	}.command(context.Background())
	if target.Path != "/bin/sh" || target.Args[0] != "/bin/sh" {
		t.Fatalf("shell target path/argv = %q/%v", target.Path, target.Args)
	}
}

func TestPreparedSpecPinsResolvedPathWithoutDescriptor(t *testing.T) {
	direct := preparedSpec{
		Spec: Spec{Args: []string{"tool", "arg"}}, executablePath: "/resolved/tool",
	}.command(context.Background())
	if direct.Path != "/resolved/tool" || direct.Args[0] != "tool" {
		t.Fatalf("direct path/argv = %q/%v", direct.Path, direct.Args)
	}

	shell := preparedSpec{
		Spec: Spec{Shell: "exit 0"}, executablePath: "/resolved/sh",
	}.command(context.Background())
	if shell.Path != "/resolved/sh" || shell.Args[0] != "/bin/sh" {
		t.Fatalf("shell path/argv = %q/%v", shell.Path, shell.Args)
	}
}
