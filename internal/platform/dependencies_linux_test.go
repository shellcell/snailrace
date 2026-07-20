package platform

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseLDD(t *testing.T) {
	output := []byte(`linux-vdso.so.1 (0x00007fff)
libc.so.6 => /lib/x86_64-linux-gnu/libc.so.6 (0x00007f)
/lib64/ld-linux-x86-64.so.2 (0x00007f)
libmissing.so => not found
`)
	want := []string{
		"/lib/x86_64-linux-gnu/libc.so.6",
		"/lib64/ld-linux-x86-64.so.2",
	}
	if got := parseLDD(output); !reflect.DeepEqual(got, want) {
		t.Fatalf("files = %#v, want %#v", got, want)
	}
}

func TestParseLDDPreservesPathsWithSpaces(t *testing.T) {
	output := []byte("libfoo.so => /tmp/lib dir/libfoo.so (0x123)\n" +
		"/tmp/loader dir/ld.so (0x456)\n")
	want := []string{"/tmp/lib dir/libfoo.so", "/tmp/loader dir/ld.so"}
	if got := parseLDD(output); !reflect.DeepEqual(got, want) {
		t.Fatalf("files = %#v, want %#v", got, want)
	}
}

func TestShebangExecutablesResolvesEnvInterpreter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "script")
	if err := os.WriteFile(path, []byte("#!/usr/bin/env -S sh -eu\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	got := shebangExecutables(path)
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Fatal(err)
	}
	sh, _ = filepath.Abs(sh)
	want := []string{"/usr/bin/env", sh}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("interpreters = %#v, want %#v", got, want)
	}
}

func TestShebangExecutablesSkipsEnvOptionOperands(t *testing.T) {
	path := filepath.Join(t.TempDir(), "script")
	if err := os.WriteFile(
		path, []byte("#!/usr/bin/env -u UNUSED -C /tmp sh -eu\n"), 0o700,
	); err != nil {
		t.Fatal(err)
	}
	got := shebangExecutables(path)
	if len(got) != 2 || filepath.Base(got[1]) != "sh" {
		t.Fatalf("interpreters = %#v, want env and sh", got)
	}
}

func TestShebangExecutablesHandlesAttachedSplitString(t *testing.T) {
	path := filepath.Join(t.TempDir(), "script")
	if err := os.WriteFile(
		path, []byte("#!/usr/bin/env -a custom --split-string=sh -eu\n"), 0o700,
	); err != nil {
		t.Fatal(err)
	}
	got := shebangExecutables(path)
	if len(got) != 2 || filepath.Base(got[1]) != "sh" {
		t.Fatalf("interpreters = %#v, want env and sh", got)
	}
}
