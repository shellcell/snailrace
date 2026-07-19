package platform

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseOtool(t *testing.T) {
	output := []byte(`/tmp/tool:
	@rpath/liblocal.dylib (compatibility version 1.0.0, current version 1.0.0)
	/usr/lib/libSystem.B.dylib (compatibility version 1.0.0, current version 1356.0.0)
`)
	want := []string{"@rpath/liblocal.dylib", "/usr/lib/libSystem.B.dylib"}
	if got := parseOtool(output); !reflect.DeepEqual(got, want) {
		t.Fatalf("dependencies = %#v, want %#v", got, want)
	}
}

func TestResolveDylibUsesRPathAndIdentifiesSharedCache(t *testing.T) {
	directory := t.TempDir()
	library := filepath.Join(directory, "liblocal.dylib")
	if err := os.WriteFile(library, []byte("test"), 0o600); err != nil {
		t.Fatal(err)
	}
	library, _ = filepath.EvalSymlinks(library)
	dependency, ok := resolveDylib(
		context.Background(),
		"@rpath/liblocal.dylib", "/tmp/tool", "/tmp/tool", []string{directory},
	)
	if !ok || dependency.Path != library || dependency.SharedCache {
		t.Fatalf("resolved dependency = %+v, valid=%v", dependency, ok)
	}
	dependency, ok = resolveDylib(
		context.Background(),
		"/usr/lib/libSystem.B.dylib", "/tmp/tool", "/tmp/tool", nil,
	)
	if !ok || !dependency.SharedCache {
		t.Fatalf("shared-cache dependency = %+v, valid=%v", dependency, ok)
	}
}
