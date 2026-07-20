package app

import (
	"strings"

	"github.com/shellcell/snailrace/internal/report"
)

type formatValues struct {
	values []string
	set    bool
}

func newFormatValues() formatValues {
	return formatValues{values: []string{"html"}}
}

func (formats *formatValues) String() string {
	return strings.Join(formats.values, ",")
}

func (formats *formatValues) Set(value string) error {
	if !formats.set {
		formats.values = nil
		formats.set = true
	}
	for _, item := range strings.Split(value, ",") {
		item = strings.ToLower(strings.TrimSpace(item))
		if item != "" {
			if info, ok := report.LookupFormat(item); ok {
				item = string(info.Name)
			}
			formats.values = append(formats.values, item)
		}
	}
	return nil
}
