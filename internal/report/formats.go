package report

import "strings"

type Format string

const (
	FormatHTML     Format = "html"
	FormatSVG      Format = "svg"
	FormatMarkdown Format = "markdown"
	FormatJSON     Format = "json"
	FormatText     Format = "text"
)

type FormatInfo struct {
	Name        Format
	Extension   string
	NeedsCharts bool
}

var formatCatalog = map[Format]FormatInfo{
	FormatHTML:     {Name: FormatHTML, Extension: "html"},
	FormatSVG:      {Name: FormatSVG, Extension: "svg", NeedsCharts: true},
	FormatMarkdown: {Name: FormatMarkdown, Extension: "md", NeedsCharts: true},
	FormatJSON:     {Name: FormatJSON, Extension: "json"},
	FormatText:     {Name: FormatText, Extension: "txt"},
}

func LookupFormat(value string) (FormatInfo, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "md":
		value = string(FormatMarkdown)
	case "txt":
		value = string(FormatText)
	}
	info, ok := formatCatalog[Format(value)]
	return info, ok
}

func ValidFormat(format string) bool {
	_, ok := LookupFormat(format)
	return ok
}
