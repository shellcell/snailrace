package report

import "testing"

func TestFormatCatalogCanonicalizesAliasesAndMetadata(t *testing.T) {
	tests := []struct {
		input     string
		name      Format
		extension string
		charts    bool
	}{
		{"HTML", FormatHTML, "html", false},
		{"svg", FormatSVG, "svg", true},
		{"md", FormatMarkdown, "md", true},
		{"markdown", FormatMarkdown, "md", true},
		{"json", FormatJSON, "json", false},
		{"txt", FormatText, "txt", false},
	}
	for _, test := range tests {
		info, ok := LookupFormat(test.input)
		if !ok || info.Name != test.name || info.Extension != test.extension ||
			info.NeedsCharts != test.charts {
			t.Fatalf("format %q = %+v, valid=%v", test.input, info, ok)
		}
	}
	if ValidFormat("unknown") {
		t.Fatal("unknown format should be invalid")
	}
}
