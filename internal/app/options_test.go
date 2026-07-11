package app

import (
	"io"
	"testing"
)

func TestParseDirectCommand(t *testing.T) {
	options, err := parseOptions(
		[]string{"-n", "3", "-warmups", "0", "--", "sleep", "0.1"},
		io.Discard,
	)
	if err != nil {
		t.Fatal(err)
	}
	if options.runs != 3 || len(options.specs) != 1 {
		t.Fatalf("unexpected options: %+v", options)
	}
	if options.specs[0].Name != "sleep" {
		t.Fatalf("name = %q, want sleep", options.specs[0].Name)
	}
}

func TestRejectsNameForComparison(t *testing.T) {
	_, err := parseOptions(
		[]string{"-name", "x", "-c", "true", "-c", "false"},
		io.Discard,
	)
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestParseFixedDurationTUIComparison(t *testing.T) {
	options, err := parseOptions(
		[]string{"tui", "-duration", "2s", "-c", "htop", "-c", "btop"},
		io.Discard,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !options.tui || options.duration.Seconds() != 2 || len(options.specs) != 2 {
		t.Fatalf("unexpected options: %+v", options)
	}
}

func TestRejectsUnlimitedTUIComparison(t *testing.T) {
	_, err := parseOptions(
		[]string{"tui", "-c", "htop", "-c", "btop"}, io.Discard,
	)
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestUnlimitedTUIDefaultsToSingleRun(t *testing.T) {
	options, err := parseOptions(
		[]string{"tui", "--", "vim"}, io.Discard,
	)
	if err != nil {
		t.Fatal(err)
	}
	if options.runs != 1 || options.warmups != 0 {
		t.Fatalf("runs/warmups = %d/%d, want 1/0", options.runs, options.warmups)
	}
}

func TestParseExplicitBaseline(t *testing.T) {
	options, err := parseOptions(
		[]string{"-baseline", "2", "-c", "one", "-c", "two"},
		io.Discard,
	)
	if err != nil {
		t.Fatal(err)
	}
	if options.baseline != 2 {
		t.Fatalf("baseline = %d, want 2", options.baseline)
	}
}

func TestRejectsBaselineOutsideTools(t *testing.T) {
	_, err := parseOptions(
		[]string{"-baseline", "3", "-c", "one", "-c", "two"},
		io.Discard,
	)
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestRepeatedExecutablesGetShortNumberedLabels(t *testing.T) {
	options, err := parseOptions(
		[]string{"-c", "sleep 1 --long-argument", "-c", "sleep 2 --other"},
		io.Discard,
	)
	if err != nil {
		t.Fatal(err)
	}
	if options.specs[0].Name != "sleep #1" || options.specs[1].Name != "sleep #2" {
		t.Fatalf("labels = %q, %q", options.specs[0].Name, options.specs[1].Name)
	}
}

func TestExplicitLabelsFollowCommandOrder(t *testing.T) {
	options, err := parseOptions(
		[]string{
			"-c", "very-long-command --one", "-c", "very-long-command --two",
			"-label", "first", "-label", "second",
		},
		io.Discard,
	)
	if err != nil {
		t.Fatal(err)
	}
	if options.specs[0].Name != "first" || options.specs[1].Name != "second" {
		t.Fatalf("labels = %q, %q", options.specs[0].Name, options.specs[1].Name)
	}
}
