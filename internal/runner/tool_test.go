package runner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/shellcell/snailrace/internal/platform"
)

func TestInspectionHonorsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := newToolInspector().inspect(
		ctx, Spec{Name: "true", Args: []string{"/bin/true"}},
	); err == nil {
		t.Fatal("cancelled inspection should fail")
	}
}

func TestToolInspectorUsesInjectedDependencyDiscovery(t *testing.T) {
	calls := 0
	inspector := newToolInspectorWith(func(
		context.Context, string,
	) ([]platform.LinkedDependency, error) {
		calls++
		return nil, nil
	})
	for _, name := range []string{"first", "second"} {
		if _, err := inspector.inspect(
			context.Background(), Spec{Name: name, Args: []string{"/bin/true"}},
		); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 1 {
		t.Fatalf("dependency discovery calls = %d, want 1 cached call", calls)
	}
	expected := errors.New("dependency failure")
	inspector = newToolInspectorWith(func(
		context.Context, string,
	) ([]platform.LinkedDependency, error) {
		return nil, expected
	})
	if _, err := inspector.inspect(
		context.Background(), Spec{Name: "tool", Args: []string{"/bin/true"}},
	); !errors.Is(err, expected) {
		t.Fatalf("dependency error = %v, want %v", err, expected)
	}
}

func TestShellInspectionReportsTargetExecutable(t *testing.T) {
	tool, err := inspectTool(Spec{Name: "shell", Shell: "sleep 0"})
	if err != nil {
		t.Fatal(err)
	}
	want := shellExecutable("sleep 0")
	if tool.Executable != want {
		t.Fatalf("shell target executable = %q, want %q", tool.Executable, want)
	}
}

func TestShellInspectionPreservesBuiltins(t *testing.T) {
	tool, err := inspectTool(Spec{Name: "shell", Shell: "printf test"})
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks("/bin/sh")
	if err != nil {
		t.Fatal(err)
	}
	if tool.Executable != want || tool.ShellTarget {
		t.Fatalf("builtin inspection = %+v, want shell executable", tool)
	}
}

func TestPinShellExecutablePreservesArguments(t *testing.T) {
	got, ok := pinShellExecutable("  curl --url 'https://example.com/a b'", "/proc/self/fd/3")
	want := "  '/proc/self/fd/3' --url 'https://example.com/a b'"
	if !ok || got != want {
		t.Fatalf("pinned command = %q, %v; want %q, true", got, ok, want)
	}
}

func TestToolVerificationDetectsExecutableReplacement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tool")
	data, err := os.ReadFile("/bin/true")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o755); err != nil {
		t.Fatal(err)
	}
	inspector := newToolInspector()
	tool, err := inspector.inspect(
		context.Background(), Spec{Name: "tool", Args: []string{path}},
	)
	if err != nil {
		t.Fatal(err)
	}
	file, err := inspector.pin(context.Background(), &tool)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := os.Rename(path, path+".old"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := inspector.verify(context.Background(), tool, file); err == nil {
		t.Fatal("replacement executable should fail provenance verification")
	}
}
