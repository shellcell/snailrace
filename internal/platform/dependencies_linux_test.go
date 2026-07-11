package platform

import (
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
