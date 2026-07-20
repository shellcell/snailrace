package app

import (
	"io"
	"reflect"
	"testing"
)

func TestFormatsRepeatAndCommaSeparate(t *testing.T) {
	options, err := parseOptions(
		[]string{
			"-format", "html,svg", "-format", "markdown", "--", "true",
		},
		io.Discard,
	)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"html", "svg", "markdown"}
	if !reflect.DeepEqual(options.formats, want) {
		t.Fatalf("formats = %#v, want %#v", options.formats, want)
	}
}

func TestDefaultFormatIsHTML(t *testing.T) {
	options, err := parseOptions([]string{"--", "true"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(options.formats, []string{"html"}) {
		t.Fatalf("formats = %#v, want html", options.formats)
	}
	if options.output != "." {
		t.Fatalf("output = %q, want current directory", options.output)
	}
}

func TestEmptyOutputRequiresNoSave(t *testing.T) {
	if _, err := parseOptions([]string{"-o", "", "--", "true"}, io.Discard); err == nil {
		t.Fatal("empty output should require -no-save")
	}
	options, err := parseOptions(
		[]string{"-no-save", "-o", "", "--", "true"}, io.Discard,
	)
	if err != nil || !options.noSave {
		t.Fatalf("no-save options = %+v, error = %v", options, err)
	}
}
